package releaseartifactbuild

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const maxCommandOutput = 4 * 1024 * 1024

type commandRunner interface {
	Run(ctx context.Context, directory string, environment []string, name string, arguments ...string) (string, string, error)
}

type osCommandRunner struct{}

func (osCommandRunner) Run(ctx context.Context, directory string, environment []string, name string, arguments ...string) (string, string, error) {
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	if environment != nil {
		command.Env = environment
	}
	var stdout limitedBuffer
	var stderr limitedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if stdout.Overflow || stderr.Overflow {
		return "", "", errors.New("command output exceeded the bounded capture limit")
	}
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("command failed: %s", name)
	}
	return stdout.String(), stderr.String(), nil
}

type limitedBuffer struct {
	bytes.Buffer
	Overflow bool
}

func (buffer *limitedBuffer) Write(data []byte) (int, error) {
	if buffer.Overflow {
		return len(data), nil
	}
	remaining := maxCommandOutput - buffer.Len()
	if remaining <= 0 {
		buffer.Overflow = true
		return len(data), nil
	}
	if len(data) > remaining {
		_, _ = buffer.Buffer.Write(data[:remaining])
		buffer.Overflow = true
		return len(data), nil
	}
	return buffer.Buffer.Write(data)
}

func sanitizedEnvironment(extra ...string) []string {
	allowed := map[string]bool{
		"HOME":          true,
		"PATH":          true,
		"TMPDIR":        true,
		"USER":          true,
		"LOGNAME":       true,
		"SSH_AUTH_SOCK": true,
		"GPG_TTY":       true,
		"GNUPGHOME":     true,
		"GOCACHE":       true,
		"GOMODCACHE":    true,
		"GOPATH":        true,
		"GOPROXY":       true,
		"GOSUMDB":       true,
		"GONOSUMDB":     true,
		"GOPRIVATE":     true,
		"GONOPROXY":     true,
		"GOTOOLCHAIN":   true,
	}
	out := make([]string, 0, len(allowed)+len(extra)+4)
	for _, entry := range os.Environ() {
		name, _, found := strings.Cut(entry, "=")
		if found && allowed[name] {
			out = append(out, entry)
		}
	}
	out = append(out,
		"LC_ALL=C",
		"LANG=C",
		"TZ=UTC",
		"SOURCE_DATE_EPOCH=0",
	)
	out = append(out, extra...)
	return out
}
