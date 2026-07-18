package releasepublication

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
)

const (
	maxCommandOutput = 4 * 1024 * 1024
	maxFileOutput    = int64(512*1024*1024 + 1)
)

type CommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

type Runner interface {
	Run(context.Context, string, []string, string, ...string) CommandResult
	RunToFile(context.Context, string, []string, string, string, ...string) CommandResult
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, dir string, environment []string, name string, args ...string) CommandResult {
	stdout := &boundedBuffer{limit: maxCommandOutput}
	stderr := &boundedBuffer{limit: maxCommandOutput}
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	if environment != nil {
		command.Env = environment
	}
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	result := commandResult(err, stdout.bytes(), stderr.bytes())
	if stdout.overflow || stderr.overflow {
		result.Err = errors.New("command output exceeded the publication safety boundary")
		result.ExitCode = -1
	}
	return result
}

func (OSRunner) RunToFile(ctx context.Context, dir string, environment []string, outputPath, name string, args ...string) CommandResult {
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return CommandResult{ExitCode: -1, Err: errors.New("create command output file")}
	}
	stderr := &boundedBuffer{limit: maxCommandOutput}
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	if environment != nil {
		command.Env = environment
	}
	limited := &limitedFileWriter{file: file, remaining: maxFileOutput}
	command.Stdout = limited
	command.Stderr = stderr
	runErr := command.Run()
	closeErr := file.Close()
	if runErr == nil && closeErr != nil {
		runErr = closeErr
	}
	result := commandResult(runErr, nil, stderr.bytes())
	if limited.overflow {
		result.Err = errors.New("downloaded command output exceeded the release asset size boundary")
		result.ExitCode = -1
	}
	if stderr.overflow {
		result.Err = errors.New("command error output exceeded the publication safety boundary")
		result.ExitCode = -1
	}
	if result.Err != nil {
		_ = os.Remove(outputPath)
	}
	return result
}

func commandResult(err error, stdout, stderr []byte) CommandResult {
	result := CommandResult{Stdout: stdout, Stderr: stderr, ExitCode: 0, Err: err}
	if err == nil {
		return result
	}
	result.ExitCode = -1
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
	}
	return result
}

type limitedFileWriter struct {
	file      *os.File
	remaining int64
	overflow  bool
}

func (writer *limitedFileWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > writer.remaining {
		writer.overflow = true
		return 0, errors.New("file output exceeds limit")
	}
	written, err := writer.file.Write(data)
	writer.remaining -= int64(written)
	return written, err
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
}

func (value *boundedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := value.limit - value.buffer.Len()
	if remaining <= 0 {
		if original > 0 {
			value.overflow = true
		}
		return original, nil
	}
	if len(data) > remaining {
		value.overflow = true
		data = data[:remaining]
	}
	_, _ = value.buffer.Write(data)
	return original, nil
}

func (value *boundedBuffer) bytes() []byte {
	return append([]byte(nil), value.buffer.Bytes()...)
}

func commandFailure(label string, result CommandResult) error {
	if result.Err == nil {
		return nil
	}
	detail := redact.Sanitize(string(result.Stderr))
	if len(detail) > 2048 {
		detail = detail[:2048] + " [truncated]"
	}
	if detail == "" {
		return fmt.Errorf("%s: %w", label, result.Err)
	}
	return fmt.Errorf("%s: %w: %s", label, result.Err, detail)
}

func minimalValidatorEnvironment(root string) []string {
	return []string{
		"HOME=" + root,
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"PATH=/usr/bin:/bin",
		"TZ=UTC",
	}
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if written > 0 {
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func nowUTC() time.Time { return time.Now().UTC() }
