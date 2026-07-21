package projectcommand

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	govulncheckToolConfigVersion        = 1
	govulncheckCommandPackage           = "golang.org/x/vuln/cmd/govulncheck"
	govulncheckModuleRoot               = "golang.org/x/vuln"
	maxGovulncheckToolConfigBytes       = 64 * 1024
	maxGovulncheckBuildInformationBytes = 256 * 1024
)

var exactGovulncheckVersionPattern = regexp.MustCompile(
	`^v[0-9]+\.[0-9]+\.[0-9]+$`,
)

type govulncheckApproval struct {
	CommandPackage string
	Module         string
	Version        string
}

type govulncheckToolIdentity struct {
	Executable     string
	Directory      string
	CommandPackage string
	Module         string
	Version        string
	BuildGoVersion string
	SHA256         string
}

type govulncheckToolConfiguration struct {
	Version int                        `json:"version"`
	Tools   map[string]json.RawMessage `json:"tools"`
}

type govulncheckToolDeclaration struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

type boundedIdentityBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (buffer *boundedIdentityBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining <= 0 {
		buffer.exceeded = true
		return len(data), nil
	}
	if len(data) > remaining {
		_, _ = buffer.buffer.Write(data[:remaining])
		buffer.exceeded = true
		return len(data), nil
	}
	_, _ = buffer.buffer.Write(data)
	return len(data), nil
}

func (buffer *boundedIdentityBuffer) Bytes() []byte {
	return buffer.buffer.Bytes()
}

func loadGovulncheckApproval(path string) (govulncheckApproval, error) {
	if !filepath.IsAbs(path) {
		return govulncheckApproval{}, errors.New(
			"govulncheck tool-version configuration path must be absolute",
		)
	}

	info, err := os.Lstat(path)
	if err != nil {
		return govulncheckApproval{}, fmt.Errorf(
			"inspect govulncheck tool-version configuration: %w",
			err,
		)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return govulncheckApproval{}, errors.New(
			"govulncheck tool-version configuration must not be a symbolic link",
		)
	}
	if !info.Mode().IsRegular() {
		return govulncheckApproval{}, errors.New(
			"govulncheck tool-version configuration must be a regular file",
		)
	}
	if info.Size() > maxGovulncheckToolConfigBytes {
		return govulncheckApproval{}, errors.New(
			"govulncheck tool-version configuration exceeds the size limit",
		)
	}

	file, err := os.Open(path)
	if err != nil {
		return govulncheckApproval{}, fmt.Errorf(
			"open govulncheck tool-version configuration: %w",
			err,
		)
	}
	defer file.Close()

	limited := io.LimitReader(
		file,
		maxGovulncheckToolConfigBytes+1,
	)
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	var configuration govulncheckToolConfiguration
	if err := decoder.Decode(&configuration); err != nil {
		return govulncheckApproval{}, fmt.Errorf(
			"decode govulncheck tool-version configuration: %w",
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return govulncheckApproval{}, errors.New(
				"govulncheck tool-version configuration contains multiple JSON values",
			)
		}
		return govulncheckApproval{}, fmt.Errorf(
			"decode trailing govulncheck tool-version configuration: %w",
			err,
		)
	}

	if configuration.Version != govulncheckToolConfigVersion {
		return govulncheckApproval{}, fmt.Errorf(
			"unsupported govulncheck tool-version configuration version %d",
			configuration.Version,
		)
	}
	raw, ok := configuration.Tools["govulncheck"]
	if !ok {
		return govulncheckApproval{}, errors.New(
			"govulncheck tool-version declaration is missing",
		)
	}

	var declaration govulncheckToolDeclaration
	declarationDecoder := json.NewDecoder(bytes.NewReader(raw))
	declarationDecoder.DisallowUnknownFields()
	if err := declarationDecoder.Decode(&declaration); err != nil {
		return govulncheckApproval{}, fmt.Errorf(
			"decode govulncheck tool-version declaration: %w",
			err,
		)
	}
	if err := declarationDecoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return govulncheckApproval{}, errors.New(
				"govulncheck tool-version declaration contains multiple JSON values",
			)
		}
		return govulncheckApproval{}, fmt.Errorf(
			"decode trailing govulncheck declaration: %w",
			err,
		)
	}

	if declaration.Module != govulncheckCommandPackage {
		return govulncheckApproval{}, fmt.Errorf(
			"govulncheck command package must be %q",
			govulncheckCommandPackage,
		)
	}
	if !exactGovulncheckVersionPattern.MatchString(declaration.Version) {
		return govulncheckApproval{}, errors.New(
			"govulncheck approved version must be an exact semantic version",
		)
	}

	return govulncheckApproval{
		CommandPackage: declaration.Module,
		Module:         govulncheckModuleRoot,
		Version:        declaration.Version,
	}, nil
}

func verifyGovulncheckTool(
	ctx context.Context,
	selectedGoExecutable string,
	toolExecutable string,
	toolVersionConfiguration string,
) (govulncheckToolIdentity, error) {
	if ctx == nil {
		return govulncheckToolIdentity{}, errors.New(
			"govulncheck identity context is required",
		)
	}

	approval, err := loadGovulncheckApproval(
		toolVersionConfiguration,
	)
	if err != nil {
		return govulncheckToolIdentity{}, err
	}

	selectedGoExecutable, err = exactRegularExecutable(
		selectedGoExecutable,
		"selected Go executable",
	)
	if err != nil {
		return govulncheckToolIdentity{}, err
	}
	toolExecutable, err = exactRegularExecutable(
		toolExecutable,
		"govulncheck executable",
	)
	if err != nil {
		return govulncheckToolIdentity{}, err
	}

	digest, err := executableSHA256(toolExecutable)
	if err != nil {
		return govulncheckToolIdentity{}, err
	}

	probeContext, cancel := context.WithTimeout(
		ctx,
		30*time.Second,
	)
	defer cancel()

	command := exec.CommandContext(
		probeContext,
		selectedGoExecutable,
		"version",
		"-m",
		toolExecutable,
	)
	command.Env = []string{
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"LANG=C",
		"LC_ALL=C",
		"PATH=" + boundedToolIdentityPath(
			selectedGoExecutable,
		),
	}

	stdout := boundedIdentityBuffer{
		limit: maxGovulncheckBuildInformationBytes,
	}
	stderr := boundedIdentityBuffer{
		limit: maxGovulncheckBuildInformationBytes,
	}
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		if probeContext.Err() != nil {
			return govulncheckToolIdentity{}, fmt.Errorf(
				"inspect govulncheck build identity: %w",
				probeContext.Err(),
			)
		}
		return govulncheckToolIdentity{}, fmt.Errorf(
			"inspect govulncheck build identity: %w",
			err,
		)
	}
	if stdout.exceeded || stderr.exceeded {
		return govulncheckToolIdentity{}, errors.New(
			"govulncheck build identity exceeds the output limit",
		)
	}

	observed, err := parseGovulncheckBuildIdentity(
		stdout.Bytes(),
		toolExecutable,
	)
	if err != nil {
		return govulncheckToolIdentity{}, err
	}
	if observed.CommandPackage != approval.CommandPackage {
		return govulncheckToolIdentity{}, fmt.Errorf(
			"govulncheck command package mismatch: observed %q",
			observed.CommandPackage,
		)
	}
	if observed.Module != approval.Module {
		return govulncheckToolIdentity{}, fmt.Errorf(
			"govulncheck module mismatch: observed %q",
			observed.Module,
		)
	}
	if observed.Version != approval.Version {
		return govulncheckToolIdentity{}, fmt.Errorf(
			"govulncheck version mismatch: observed %q",
			observed.Version,
		)
	}

	observed.Executable = toolExecutable
	observed.Directory = filepath.Dir(toolExecutable)
	observed.SHA256 = digest
	return observed, nil
}

func exactRegularExecutable(path string, label string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s path must be absolute", label)
	}
	cleaned := filepath.Clean(path)
	info, err := os.Lstat(cleaned)
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s must not be a symbolic link", label)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s must be a regular file", label)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s is not executable", label)
	}
	return cleaned, nil
}

func executableSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf(
			"open govulncheck executable for hashing: %w",
			err,
		)
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf(
			"hash govulncheck executable: %w",
			err,
		)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func parseGovulncheckBuildIdentity(
	output []byte,
	toolExecutable string,
) (govulncheckToolIdentity, error) {
	var identity govulncheckToolIdentity

	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(
		make([]byte, 4096),
		maxGovulncheckBuildInformationBytes,
	)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			prefix := toolExecutable + ": "
			if !strings.HasPrefix(line, prefix) {
				return govulncheckToolIdentity{}, errors.New(
					"govulncheck build identity has an unexpected header",
				)
			}
			identity.BuildGoVersion = strings.TrimPrefix(
				line,
				prefix,
			)
			if identity.BuildGoVersion == "" {
				return govulncheckToolIdentity{}, errors.New(
					"govulncheck build Go version is missing",
				)
			}
			continue
		}

		fields := strings.Split(strings.TrimLeft(line, "\t"), "\t")
		if len(fields) >= 2 && fields[0] == "path" {
			identity.CommandPackage = fields[1]
		}
		if len(fields) >= 3 && fields[0] == "mod" {
			identity.Module = fields[1]
			identity.Version = fields[2]
		}
	}
	if err := scanner.Err(); err != nil {
		return govulncheckToolIdentity{}, fmt.Errorf(
			"read govulncheck build identity: %w",
			err,
		)
	}
	if first {
		return govulncheckToolIdentity{}, errors.New(
			"govulncheck build identity is empty",
		)
	}
	if identity.CommandPackage == "" {
		return govulncheckToolIdentity{}, errors.New(
			"govulncheck command package identity is missing",
		)
	}
	if identity.Module == "" || identity.Version == "" {
		return govulncheckToolIdentity{}, errors.New(
			"govulncheck module identity is missing",
		)
	}
	return identity, nil
}

func boundedToolIdentityPath(selectedGoExecutable string) string {
	directories := []string{
		filepath.Dir(selectedGoExecutable),
		"/usr/local/sbin",
		"/usr/local/bin",
		"/usr/sbin",
		"/usr/bin",
		"/sbin",
		"/bin",
	}
	seen := make(map[string]struct{}, len(directories))
	accepted := make([]string, 0, len(directories))
	for _, directory := range directories {
		cleaned := filepath.Clean(directory)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		accepted = append(accepted, cleaned)
	}
	return strings.Join(accepted, string(os.PathListSeparator))
}
