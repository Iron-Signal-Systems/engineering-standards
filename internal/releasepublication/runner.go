package releasepublication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

type releaseAssetUploadCommand struct {
	Repository string
	ReleaseID  int64
	AssetName  string
	InputPath  string
}

func (runner OSRunner) Run(ctx context.Context, dir string, environment []string, name string, args ...string) CommandResult {
	upload, matched, err := parseReleaseAssetUploadCommand(name, args)
	if err != nil {
		return CommandResult{ExitCode: -1, Err: err}
	}
	if matched {
		return runner.runReleaseAssetUpload(ctx, dir, environment, upload)
	}
	return runner.runCommand(ctx, dir, environment, name, args...)
}

func (OSRunner) runCommand(ctx context.Context, dir string, environment []string, name string, args ...string) CommandResult {
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

func parseReleaseAssetUploadCommand(name string, args []string) (releaseAssetUploadCommand, bool, error) {
	if name != "gh" || len(args) == 0 || args[0] != "api" {
		return releaseAssetUploadCommand{}, false, nil
	}
	endpoint := args[len(args)-1]
	if !strings.Contains(endpoint, "/releases/") || !strings.Contains(endpoint, "/assets?") {
		return releaseAssetUploadCommand{}, false, nil
	}
	if len(args) != 8 || args[1] != "--method" || args[2] != "POST" || args[3] != "-H" || args[4] != "Content-Type: application/octet-stream" || args[5] != "--input" {
		return releaseAssetUploadCommand{}, true, errors.New("release asset upload command does not match the controlled transport shape")
	}
	parts := strings.SplitN(endpoint, "?", 2)
	if len(parts) != 2 {
		return releaseAssetUploadCommand{}, true, errors.New("release asset upload endpoint has no query")
	}
	segments := strings.Split(parts[0], "/")
	if len(segments) != 6 || segments[0] != "repos" || segments[1] == "" || segments[2] == "" || segments[3] != "releases" || segments[5] != "assets" {
		return releaseAssetUploadCommand{}, true, errors.New("release asset upload endpoint is invalid")
	}
	releaseID, err := strconv.ParseInt(segments[4], 10, 64)
	if err != nil || releaseID <= 0 {
		return releaseAssetUploadCommand{}, true, errors.New("release asset upload release ID is invalid")
	}
	query, err := url.ParseQuery(parts[1])
	if err != nil || len(query) != 1 || len(query["name"]) != 1 || query.Get("name") == "" {
		return releaseAssetUploadCommand{}, true, errors.New("release asset upload name is invalid")
	}
	assetName := query.Get("name")
	inputPath := args[6]
	if filepath.Base(inputPath) != assetName {
		return releaseAssetUploadCommand{}, true, errors.New("release asset upload path does not match the declared asset name")
	}
	return releaseAssetUploadCommand{
		Repository: segments[1] + "/" + segments[2],
		ReleaseID:  releaseID,
		AssetName:  assetName,
		InputPath:  inputPath,
	}, true, nil
}

func (runner OSRunner) runReleaseAssetUpload(ctx context.Context, dir string, environment []string, upload releaseAssetUploadCommand) CommandResult {
	before, result := runner.readReleaseForUpload(ctx, dir, environment, upload)
	if result.Err != nil {
		return result
	}
	for _, asset := range before.Assets {
		if asset.Name == upload.AssetName {
			return CommandResult{ExitCode: -1, Err: errors.New("release asset already exists; clobbering is denied")}
		}
	}

	uploadResult := runner.runCommand(
		ctx,
		dir,
		environment,
		"gh",
		"release",
		"upload",
		before.TagName,
		upload.InputPath,
		"--repo",
		upload.Repository,
	)

	after, authoritative := runner.readReleaseForUpload(ctx, dir, environment, upload)
	if authoritative.Err != nil {
		if uploadResult.Err != nil {
			return uploadResult
		}
		return authoritative
	}
	var observed *githubAsset
	for index := range after.Assets {
		if after.Assets[index].Name != upload.AssetName {
			continue
		}
		if observed != nil {
			return CommandResult{ExitCode: -1, Err: errors.New("release contains duplicate uploaded asset names")}
		}
		observed = &after.Assets[index]
	}
	if observed == nil {
		if uploadResult.Err != nil {
			return uploadResult
		}
		return CommandResult{ExitCode: -1, Err: errors.New("release asset upload completed without authoritative asset state")}
	}
	data, err := json.Marshal(observed)
	if err != nil {
		return CommandResult{ExitCode: -1, Err: errors.New("encode authoritative release asset state")}
	}
	return CommandResult{Stdout: data, ExitCode: 0}
}

func (runner OSRunner) readReleaseForUpload(ctx context.Context, dir string, environment []string, upload releaseAssetUploadCommand) (githubRelease, CommandResult) {
	endpoint := "repos/" + upload.Repository + "/releases/" + strconv.FormatInt(upload.ReleaseID, 10)
	result := runner.runCommand(ctx, dir, environment, "gh", "api", "--method", "GET", endpoint)
	if result.Err != nil {
		return githubRelease{}, result
	}
	var release githubRelease
	if err := json.Unmarshal(result.Stdout, &release); err != nil {
		return githubRelease{}, CommandResult{ExitCode: -1, Err: errors.New("parse authoritative GitHub Release during asset upload")}
	}
	if release.ID != upload.ReleaseID || release.TagName == "" || !release.Draft || release.Prerelease {
		return githubRelease{}, CommandResult{ExitCode: -1, Err: errors.New("authoritative GitHub Release changed during asset upload")}
	}
	return release, CommandResult{ExitCode: 0}
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
