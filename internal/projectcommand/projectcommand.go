package projectcommand

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectorigin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

const (
	EvidenceSchemaVersion = 1
	ExecutionTimeout      = 20 * time.Minute
	MaxOutputBytes        = 1024 * 1024
	maxEvidenceNameBytes  = 96
)

var (
	executionTimeout       = ExecutionTimeout
	maxOutputBytes   int64 = MaxOutputBytes
	errOutputLimit         = errors.New("project command output exceeded the configured limit")
)

var inheritedEnvironmentNames = []string{
	"CC",
	"CGO_ENABLED",
	"CXX",
	"GOENV",
	"GOEXPERIMENT",
	"GOFLAGS",
	"GONOPROXY",
	"GONOSUMDB",
	"GOPRIVATE",
	"GOPROXY",
	"GOSUMDB",
	"GOTOOLCHAIN",
	"PKG_CONFIG",
	"PKG_CONFIG_PATH",
	"SOURCE_DATE_EPOCH",
	"SSL_CERT_DIR",
	"SSL_CERT_FILE",
	"TZ",
}

type Request struct {
	Root      string
	Mode      string
	Target    repository.Identity
	Validator validatoridentity.Identity
	Pin       projectpin.Pin
	Name      string
}

type IdentityEvidence struct {
	Profile          string `json:"profile"`
	Version          string `json:"version"`
	ReleaseTag       string `json:"release_tag"`
	SourceRepository string `json:"source_repository"`
	SourceCommit     string `json:"source_commit"`
}

type TargetEvidence struct {
	Repository string `json:"repository"`
	Root       string `json:"root"`
	Commit     string `json:"commit"`
	Branch     string `json:"branch,omitempty"`
	Origin     string `json:"origin"`
}

type StreamEvidence struct {
	Bytes          int64  `json:"bytes"`
	SHA256         string `json:"sha256"`
	SHA512         string `json:"sha512"`
	Sanitized      string `json:"sanitized_output"`
	LimitExceeded  bool   `json:"limit_exceeded"`
	RedactionBound string `json:"redaction_boundary"`
}

type Result struct {
	SchemaVersion          int              `json:"schema_version"`
	RunID                  string           `json:"run_id"`
	Authorization          string           `json:"authorization"`
	Status                 string           `json:"status"`
	Failure                string           `json:"failure,omitempty"`
	Mode                   string           `json:"mode"`
	CommandName            string           `json:"command_name"`
	Arguments              []string         `json:"arguments"`
	ResolvedExecutable     string           `json:"resolved_executable"`
	WorkingDirectory       string           `json:"working_directory"`
	EnvironmentNames       []string         `json:"environment_names"`
	TimeoutSeconds         int64            `json:"timeout_seconds"`
	OutputLimitBytes       int64            `json:"output_limit_bytes_per_stream"`
	Started                time.Time        `json:"started"`
	Finished               time.Time        `json:"finished"`
	DurationMilliseconds   int64            `json:"duration_milliseconds"`
	ExitCode               int              `json:"exit_code"`
	TimedOut               bool             `json:"timed_out"`
	OutputLimitExceeded    bool             `json:"output_limit_exceeded"`
	RepositoryStateChanged bool             `json:"repository_state_changed"`
	Validator              IdentityEvidence `json:"validator"`
	Target                 TargetEvidence   `json:"target"`
	Stdout                 StreamEvidence   `json:"stdout"`
	Stderr                 StreamEvidence   `json:"stderr"`
	EvidenceJSON           string           `json:"-"`
	EvidenceText           string           `json:"-"`
}

type gitSnapshot struct {
	Commit string
	Status string
}

func Execute(ctx context.Context, request Request) (Result, error) {
	result := newResult(request)
	arguments, err := authorize(ctx, request)
	if err != nil {
		result.Authorization = "DENIED"
		result.Status = "DENIED"
		result.Failure = err.Error()
		result.Finished = time.Now().UTC()
		result.DurationMilliseconds = result.Finished.Sub(result.Started).Milliseconds()
		return result, err
	}
	result.Arguments = sanitizedArguments(arguments)

	baseline, err := snapshot(ctx, request.Root, request.Pin.Evidence.Directory)
	if err != nil {
		return result, fmt.Errorf("capture repository state before project command: %w", err)
	}
	if request.Mode != "development" && baseline.Status != "" {
		return result, errors.New("commit and release modes require a clean target repository before project command execution")
	}
	if baseline.Commit != request.Target.Commit {
		return result, errors.New("target repository HEAD changed after target discovery")
	}

	runDirectory, pendingPath, err := prepareEvidence(request.Root, request.Pin.Evidence.Directory, request.Name, result.RunID)
	if err != nil {
		return result, err
	}
	result.EvidenceJSON = filepath.Join(runDirectory, "execution.json")
	result.EvidenceText = filepath.Join(runDirectory, "execution.txt")

	resolved, err := resolveExecutable(ctx, request.Root, arguments[0])
	if err != nil {
		result.Status = "FAIL"
		result.Failure = err.Error()
		result.Finished = time.Now().UTC()
		result.DurationMilliseconds = result.Finished.Sub(result.Started).Milliseconds()
		writeErr := finalizeEvidence(result, pendingPath)
		return result, errors.Join(err, writeErr)
	}
	result.ResolvedExecutable = resolved
	if err := rejectOpaqueLauncher(resolved, arguments[1:]); err != nil {
		result.Status = "FAIL"
		result.Failure = err.Error()
		result.Finished = time.Now().UTC()
		result.DurationMilliseconds = result.Finished.Sub(result.Started).Milliseconds()
		writeErr := finalizeEvidence(result, pendingPath)
		return result, errors.Join(err, writeErr)
	}

	environmentRoot, err := os.MkdirTemp("", "isras-project-command-")
	if err != nil {
		result.Status = "FAIL"
		result.Failure = "create isolated project command environment"
		result.Finished = time.Now().UTC()
		result.DurationMilliseconds = result.Finished.Sub(result.Started).Milliseconds()
		writeErr := finalizeEvidence(result, pendingPath)
		return result, errors.Join(errors.New(result.Failure), writeErr)
	}
	if err := os.Chmod(environmentRoot, 0o700); err != nil {
		_ = os.RemoveAll(environmentRoot)
		result.Status = "FAIL"
		result.Failure = "secure isolated project command environment"
		result.Finished = time.Now().UTC()
		result.DurationMilliseconds = result.Finished.Sub(result.Started).Milliseconds()
		writeErr := finalizeEvidence(result, pendingPath)
		return result, errors.Join(errors.New(result.Failure), writeErr)
	}
	defer os.RemoveAll(environmentRoot)

	environment, environmentNames, err := commandEnvironment(environmentRoot, resolved, request)
	if err != nil {
		result.Status = "FAIL"
		result.Failure = err.Error()
		result.Finished = time.Now().UTC()
		result.DurationMilliseconds = result.Finished.Sub(result.Started).Milliseconds()
		writeErr := finalizeEvidence(result, pendingPath)
		return result, errors.Join(err, writeErr)
	}
	result.EnvironmentNames = environmentNames

	commandContext, cancel := context.WithTimeout(ctx, executionTimeout)
	defer cancel()

	stdout := newCapture(maxOutputBytes, cancel)
	stderr := newCapture(maxOutputBytes, cancel)
	command := exec.CommandContext(commandContext, resolved, arguments[1:]...)
	command.Dir = request.Root
	command.Env = environment
	command.Stdout = stdout
	command.Stderr = stderr
	if runtime.GOOS == "linux" {
		command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		command.Cancel = func() error {
			if command.Process == nil {
				return os.ErrProcessDone
			}
			if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				return err
			}
			return nil
		}
		command.WaitDelay = 2 * time.Second
	}

	result.Started = time.Now().UTC()
	runErr := command.Run()
	if killErr := killProcessGroup(command); killErr != nil && runErr == nil {
		runErr = killErr
	}
	result.Finished = time.Now().UTC()
	result.DurationMilliseconds = result.Finished.Sub(result.Started).Milliseconds()
	result.Stdout = stdout.evidence()
	result.Stderr = stderr.evidence()
	result.OutputLimitExceeded = result.Stdout.LimitExceeded || result.Stderr.LimitExceeded
	result.TimedOut = errors.Is(commandContext.Err(), context.DeadlineExceeded) && !result.OutputLimitExceeded
	result.ExitCode = exitCode(command, runErr)

	post, snapshotErr := snapshot(ctx, request.Root, request.Pin.Evidence.Directory)
	if snapshotErr != nil {
		result.RepositoryStateChanged = true
		result.Failure = "repository state could not be verified after project command execution"
	} else if post != baseline {
		result.RepositoryStateChanged = true
		result.Failure = "project command changed Git-visible repository state or HEAD"
	}

	switch {
	case result.OutputLimitExceeded:
		result.Status = "FAIL"
		result.Failure = errOutputLimit.Error()
	case result.TimedOut:
		result.Status = "FAIL"
		result.Failure = "project command exceeded the configured timeout"
	case result.RepositoryStateChanged:
		result.Status = "FAIL"
	case runErr != nil:
		result.Status = "FAIL"
		result.Failure = commandFailure(runErr)
	default:
		result.Status = "PASS"
	}

	writeErr := finalizeEvidence(result, pendingPath)
	if writeErr != nil {
		result.Status = "FAIL"
		result.Failure = "project command evidence could not be finalized"
		return result, writeErr
	}
	if result.Status != "PASS" {
		return result, errors.New(result.Failure)
	}
	return result, nil
}

func newResult(request Request) Result {
	emptyStream := emptyStreamEvidence()
	return Result{
		SchemaVersion:    EvidenceSchemaVersion,
		RunID:            newRunID(),
		Authorization:    "GRANTED",
		Status:           "PENDING",
		Mode:             request.Mode,
		CommandName:      request.Name,
		WorkingDirectory: request.Root,
		TimeoutSeconds:   int64(executionTimeout / time.Second),
		OutputLimitBytes: maxOutputBytes,
		Started:          time.Now().UTC(),
		ExitCode:         -1,
		Validator: IdentityEvidence{
			Profile:          request.Validator.Profile,
			Version:          request.Validator.StandardVersion,
			ReleaseTag:       request.Validator.ReleaseTag,
			SourceRepository: request.Validator.SourceRepository,
			SourceCommit:     request.Validator.SourceCommit,
		},
		Target: TargetEvidence{
			Repository: request.Pin.Project.Repository,
			Root:       request.Target.Root,
			Commit:     request.Target.Commit,
			Branch:     request.Target.Branch,
			Origin:     sanitizeOrigin(request.Target.Origin),
		},
		Stdout: emptyStream,
		Stderr: emptyStream,
	}
}

func emptyStreamEvidence() StreamEvidence {
	sha256Value := sha256.Sum256(nil)
	sha512Value := sha512.Sum512(nil)
	return StreamEvidence{
		SHA256:         hex.EncodeToString(sha256Value[:]),
		SHA512:         hex.EncodeToString(sha512Value[:]),
		RedactionBound: "credential-shaped values are redacted; raw output is not retained",
	}
}

func authorize(ctx context.Context, request Request) ([]string, error) {
	if request.Root == "" || request.Root != request.Target.Root {
		return nil, errors.New("project command target root is inconsistent")
	}
	if request.Mode != "development" && request.Mode != "commit" && request.Mode != "release" {
		return nil, errors.New("project command validation mode is invalid")
	}
	if request.Name == "" || len(request.Name) > maxEvidenceNameBytes || strings.ContainsAny(request.Name, "\x00\r\n/\\") {
		return nil, errors.New("project command name is invalid")
	}
	arguments, ok := request.Pin.Commands[request.Name]
	if !ok {
		return nil, fmt.Errorf("project pin does not declare command %q", request.Name)
	}
	if len(arguments) == 0 {
		return nil, fmt.Errorf("project command %q has no executable", request.Name)
	}
	if request.Validator.Ownership != validatoridentity.OwnershipReleaseArtifact {
		return nil, errors.New("project command execution requires a linker-bound release validator")
	}
	if request.Validator.Profile != request.Pin.Standard.Profile ||
		request.Validator.StandardVersion != request.Pin.Standard.Version ||
		request.Validator.ReleaseTag != request.Pin.Standard.ReleaseTag ||
		request.Validator.SourceRepository != request.Pin.Standard.SourceRepository ||
		request.Validator.SourceCommit != request.Pin.Standard.SourceCommit {
		return nil, errors.New("validator release identity does not match the committed project pin")
	}
	canonicalOrigin, err := canonicalRepository(request.Target.Origin)
	if err != nil {
		return nil, err
	}
	if canonicalOrigin != request.Pin.Project.Repository {
		return nil, errors.New("target repository origin does not match the committed project pin")
	}
	committedPin, err := projectpin.LoadCommitted(ctx, request.Root)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(committedPin, request.Pin) {
		return nil, errors.New("project command request does not match the committed project pin")
	}
	if err := requireUntrackedEvidenceBoundary(ctx, request.Root, request.Pin.Evidence.Directory); err != nil {
		return nil, err
	}
	return append([]string(nil), arguments...), nil
}

func requireUntrackedEvidenceBoundary(ctx context.Context, root, evidenceDirectory string) error {
	if evidenceDirectory != projectpin.RuntimeEvidenceDirectory {
		return errors.New("project command evidence directory is not the fixed runtime boundary")
	}
	tracked, err := runGitOutput(ctx, root, "ls-files", "--", evidenceDirectory, evidenceDirectory+"/**")
	if err != nil {
		return errors.New("inspect project command evidence tracking state")
	}
	if strings.TrimSpace(tracked) != "" {
		return errors.New("project command evidence directory must not contain tracked paths")
	}
	return nil
}

func resolveExecutable(ctx context.Context, root, declared string) (string, error) {
	if strings.ContainsAny(declared, "\x00\r\n") {
		return "", errors.New("project command executable contains a prohibited control character")
	}
	if strings.ContainsRune(declared, filepath.Separator) || strings.Contains(declared, "/") || strings.Contains(declared, "\\") {
		return resolveRepositoryExecutable(ctx, root, declared)
	}
	pathValue := os.Getenv("PATH")
	if pathValue == "" {
		return "", errors.New("PATH is unavailable for project command executable resolution")
	}
	for _, directory := range filepath.SplitList(pathValue) {
		if directory == "" || !filepath.IsAbs(directory) {
			return "", errors.New("PATH contains an empty or relative component")
		}
		candidate := filepath.Join(directory, declared)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			return "", errors.New("resolve project command executable")
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return "", errors.New("resolve absolute project command executable")
		}
		return filepath.Clean(resolved), nil
	}
	return "", fmt.Errorf("project command executable %q was not found in the sanitized PATH", declared)
}

func resolveRepositoryExecutable(ctx context.Context, root, declared string) (string, error) {
	if filepath.IsAbs(declared) || strings.Contains(declared, "\\") {
		return "", errors.New("project-owned executable must be a relative slash-separated path")
	}
	normalized := strings.TrimPrefix(declared, "./")
	if normalized == "" || filepath.Clean(normalized) != normalized || normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) {
		return "", errors.New("project-owned executable path is unsafe")
	}
	candidate := filepath.Join(root, filepath.FromSlash(normalized))
	if !pathWithin(root, candidate) {
		return "", errors.New("project-owned executable escapes the target repository")
	}
	if err := rejectSymlinkPath(root, candidate); err != nil {
		return "", err
	}
	info, err := os.Lstat(candidate)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("project-owned executable is not a regular executable file")
	}
	if err := runGitQuiet(ctx, root, "ls-files", "--error-unmatch", "--", filepath.ToSlash(normalized)); err != nil {
		return "", errors.New("project-owned executable is not tracked")
	}
	if err := runGitQuiet(ctx, root, "diff", "--quiet", "HEAD", "--", filepath.ToSlash(normalized)); err != nil {
		return "", errors.New("project-owned executable differs from the committed target")
	}
	return candidate, nil
}

func rejectOpaqueLauncher(executable string, arguments []string) error {
	base := strings.ToLower(filepath.Base(executable))
	switch base {
	case "env", "sudo", "doas", "su", "nohup", "setsid", "xargs":
		return fmt.Errorf("project command launcher %q is prohibited", base)
	}
	if isShell(base) {
		for _, argument := range arguments {
			switch strings.ToLower(argument) {
			case "-c", "/c", "-command", "-encodedcommand":
				return errors.New("project command must not supply an opaque shell command string")
			}
		}
	}
	return nil
}

func isShell(base string) bool {
	switch base {
	case "sh", "bash", "dash", "zsh", "ksh", "fish", "csh", "tcsh", "pwsh", "powershell", "powershell.exe", "cmd", "cmd.exe":
		return true
	default:
		return false
	}
}

func commandEnvironment(runDirectory, resolvedExecutable string, request Request) ([]string, []string, error) {
	isolated := map[string]string{
		"HOME":                          filepath.Join(runDirectory, "home"),
		"TMPDIR":                        filepath.Join(runDirectory, "tmp"),
		"XDG_CACHE_HOME":                filepath.Join(runDirectory, "cache"),
		"GOCACHE":                       filepath.Join(runDirectory, "go-cache"),
		"GOPATH":                        filepath.Join(runDirectory, "go"),
		"LANG":                          "C",
		"LC_ALL":                        "C",
		"PATH":                          sanitizedCommandPath(resolvedExecutable),
		"ISRAS_PROJECT_ROOT":            request.Root,
		"ISRAS_PROJECT_COMMAND":         request.Name,
		"ISRAS_VALIDATOR_SOURCE_COMMIT": request.Validator.SourceCommit,
	}
	for _, path := range []string{isolated["HOME"], isolated["TMPDIR"], isolated["XDG_CACHE_HOME"], isolated["GOCACHE"], isolated["GOPATH"]} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return nil, nil, errors.New("create isolated project command environment")
		}
	}
	for _, name := range inheritedEnvironmentNames {
		if value, ok := os.LookupEnv(name); ok && value != "" {
			if strings.ContainsAny(value, "\x00\r\n") {
				return nil, nil, fmt.Errorf("environment variable %s contains a prohibited control character", name)
			}
			isolated[name] = value
		}
	}
	if err := validatePath(isolated["PATH"]); err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(isolated))
	for name := range isolated {
		names = append(names, name)
	}
	sort.Strings(names)
	environment := make([]string, 0, len(names))
	for _, name := range names {
		environment = append(environment, name+"="+isolated[name])
	}
	return environment, names, nil
}

func sanitizedCommandPath(resolvedExecutable string) string {
	candidates := []string{filepath.Dir(resolvedExecutable), "/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}
	seen := make(map[string]bool)
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" || !filepath.IsAbs(candidate) {
			continue
		}
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func validatePath(value string) error {
	if value == "" || strings.ContainsAny(value, "\x00\r\n") {
		return errors.New("PATH is unavailable or invalid")
	}
	for _, component := range filepath.SplitList(value) {
		if component == "" || !filepath.IsAbs(component) {
			return errors.New("PATH contains an empty or relative component")
		}
	}
	return nil
}

func snapshot(ctx context.Context, root, evidenceDirectory string) (gitSnapshot, error) {
	commit, err := runGitOutput(ctx, root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return gitSnapshot{}, err
	}
	arguments := []string{"status", "--porcelain=v1", "--untracked-files=all", "--", "."}
	if evidenceDirectory != "" {
		arguments = append(arguments, ":(exclude)"+evidenceDirectory, ":(exclude)"+evidenceDirectory+"/**")
	}
	status, err := runGitOutput(ctx, root, arguments...)
	if err != nil {
		return gitSnapshot{}, err
	}
	return gitSnapshot{Commit: strings.TrimSpace(commit), Status: strings.TrimSpace(status)}, nil
}

func runGitOutput(ctx context.Context, root string, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", arguments...)
	command.Dir = root
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %s", arguments[0], strings.TrimSpace(redact.Sanitize(stderr.String())))
	}
	return stdout.String(), nil
}

func runGitQuiet(ctx context.Context, root string, arguments ...string) error {
	command := exec.CommandContext(ctx, "git", arguments...)
	command.Dir = root
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Run()
}

func canonicalRepository(origin string) (string, error) {
	return projectorigin.Canonical(origin)
}

func canonicalRepositoryPath(value string) (string, error) {
	return projectorigin.Canonical("git@github.com:" + value)
}

func sanitizeOrigin(origin string) string {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.User == nil {
		return redact.Sanitize(origin)
	}
	parsed.User = url.User(parsed.User.Username())
	return redact.Sanitize(parsed.String())
}

func prepareEvidence(root, relativeDirectory, commandName, runID string) (string, string, error) {
	base, err := secureDirectory(root, relativeDirectory)
	if err != nil {
		return "", "", err
	}
	runName := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + commandName + "-" + runID
	runDirectory := filepath.Join(base, "project-commands", runName)
	if err := secureMkdirAll(root, filepath.Dir(runDirectory)); err != nil {
		return "", "", err
	}
	if err := os.Mkdir(runDirectory, 0o700); err != nil {
		return "", "", errors.New("create private project command evidence run directory")
	}
	pending := filepath.Join(runDirectory, "execution.pending")
	file, err := os.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", "", errors.New("create project command evidence preflight marker")
	}
	_, writeErr := file.WriteString("project command evidence pending\n")
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return "", "", errors.New("write project command evidence preflight marker")
	}
	return runDirectory, pending, nil
}

func secureDirectory(root, relative string) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", errors.New("resolve project command repository root")
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	directory := filepath.Join(absoluteRoot, filepath.FromSlash(relative))
	if !pathWithin(absoluteRoot, directory) {
		return "", errors.New("project command evidence directory escapes the target repository")
	}
	if err := secureMkdirAll(absoluteRoot, directory); err != nil {
		return "", err
	}
	return directory, nil
}

func secureMkdirAll(root, directory string) error {
	relative, err := filepath.Rel(root, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("project command evidence path escapes the target repository")
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return errors.New("create private project command evidence directory")
			}
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("project command evidence path contains a non-directory or symbolic link")
		}
	}
	return nil
}

func finalizeEvidence(result Result, pendingPath string) error {
	result.Failure = redact.Sanitize(result.Failure)
	if err := secureMkdirAll(result.WorkingDirectory, filepath.Dir(result.EvidenceJSON)); err != nil {
		return errors.New("project command evidence path became unsafe")
	}
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return errors.New("encode project command JSON evidence")
	}
	jsonData = append(jsonData, '\n')
	if err := writeAtomic(result.EvidenceJSON, jsonData); err != nil {
		return errors.New("write project command JSON evidence")
	}
	if err := writeAtomic(result.EvidenceText, renderText(result)); err != nil {
		_ = os.Remove(result.EvidenceJSON)
		return errors.New("write project command text evidence")
	}
	if err := os.Remove(pendingPath); err != nil {
		return errors.New("remove project command evidence preflight marker")
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(temporary)
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(temporary)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(temporary)
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		os.Remove(temporary)
		return err
	}
	return nil
}

func renderText(result Result) []byte {
	var builder strings.Builder
	fmt.Fprintf(&builder, "PROJECT COMMAND EXECUTION EVIDENCE\n")
	fmt.Fprintf(&builder, "==================================\n")
	fmt.Fprintf(&builder, "Run ID: %s\n", result.RunID)
	fmt.Fprintf(&builder, "Authorization: %s\n", result.Authorization)
	fmt.Fprintf(&builder, "Status: %s\n", result.Status)
	if result.Failure != "" {
		fmt.Fprintf(&builder, "Failure: %s\n", redact.Sanitize(result.Failure))
	}
	fmt.Fprintf(&builder, "Command name: %s\n", result.CommandName)
	fmt.Fprintf(&builder, "Arguments: %s\n", safeArguments(result.Arguments))
	fmt.Fprintf(&builder, "Resolved executable: %s\n", result.ResolvedExecutable)
	fmt.Fprintf(&builder, "Working directory: %s\n", result.WorkingDirectory)
	fmt.Fprintf(&builder, "Target repository: %s\n", result.Target.Repository)
	fmt.Fprintf(&builder, "Target commit: %s\n", result.Target.Commit)
	fmt.Fprintf(&builder, "Validator release: %s\n", result.Validator.ReleaseTag)
	fmt.Fprintf(&builder, "Validator source commit: %s\n", result.Validator.SourceCommit)
	fmt.Fprintf(&builder, "Started: %s\n", result.Started.Format(time.RFC3339Nano))
	fmt.Fprintf(&builder, "Finished: %s\n", result.Finished.Format(time.RFC3339Nano))
	fmt.Fprintf(&builder, "Duration milliseconds: %d\n", result.DurationMilliseconds)
	fmt.Fprintf(&builder, "Exit code: %d\n", result.ExitCode)
	fmt.Fprintf(&builder, "Timed out: %t\n", result.TimedOut)
	fmt.Fprintf(&builder, "Output limit exceeded: %t\n", result.OutputLimitExceeded)
	fmt.Fprintf(&builder, "Repository state changed: %t\n", result.RepositoryStateChanged)
	fmt.Fprintf(&builder, "Environment names: %s\n", strings.Join(result.EnvironmentNames, ", "))
	fmt.Fprintf(&builder, "Stdout bytes: %d\n", result.Stdout.Bytes)
	fmt.Fprintf(&builder, "Stdout SHA-256: %s\n", result.Stdout.SHA256)
	fmt.Fprintf(&builder, "Stdout SHA-512: %s\n", result.Stdout.SHA512)
	fmt.Fprintf(&builder, "Stderr bytes: %d\n", result.Stderr.Bytes)
	fmt.Fprintf(&builder, "Stderr SHA-256: %s\n", result.Stderr.SHA256)
	fmt.Fprintf(&builder, "Stderr SHA-512: %s\n", result.Stderr.SHA512)
	fmt.Fprintf(&builder, "\nSANITIZED STDOUT\n----------------\n%s", result.Stdout.Sanitized)
	if !strings.HasSuffix(result.Stdout.Sanitized, "\n") {
		builder.WriteByte('\n')
	}
	fmt.Fprintf(&builder, "\nSANITIZED STDERR\n----------------\n%s", result.Stderr.Sanitized)
	if !strings.HasSuffix(result.Stderr.Sanitized, "\n") {
		builder.WriteByte('\n')
	}
	return []byte(builder.String())
}

func sanitizedArguments(arguments []string) []string {
	out := make([]string, len(arguments))
	for index, argument := range arguments {
		out[index] = redact.Sanitize(argument)
	}
	return out
}

func safeArguments(arguments []string) string {
	parts := make([]string, len(arguments))
	for index, argument := range arguments {
		parts[index] = strconv.Quote(redact.Sanitize(argument))
	}
	return strings.Join(parts, " ")
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func rejectSymlinkPath(root, candidate string) error {
	relative, err := filepath.Rel(root, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("project-owned executable escapes the target repository")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New("inspect project-owned executable path")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("project-owned executable path contains a symbolic link")
		}
	}
	return nil
}

func killProcessGroup(command *exec.Cmd) error {
	if runtime.GOOS != "linux" || command.Process == nil {
		return nil
	}
	if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return errors.New("terminate project command process group")
	}
	return nil
}

func newRunID() string {
	data := make([]byte, 12)
	if _, err := rand.Read(data); err == nil {
		return hex.EncodeToString(data)
	}
	return fmt.Sprintf("fallback-%d", time.Now().UTC().UnixNano())
}

func exitCode(command *exec.Cmd, runErr error) int {
	if command.ProcessState != nil {
		return command.ProcessState.ExitCode()
	}
	var exitError *exec.ExitError
	if errors.As(runErr, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

func commandFailure(err error) string {
	if err == nil {
		return ""
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return fmt.Sprintf("project command exited with status %d", exitError.ExitCode())
	}
	return redact.Sanitize(err.Error())
}

type capture struct {
	mu       sync.Mutex
	limit    int64
	bytes    int64
	sha256   hash.Hash
	sha512   hash.Hash
	buffer   bytes.Buffer
	redactor *redact.Writer
	exceeded bool
	cancel   context.CancelFunc
}

func newCapture(limit int64, cancel context.CancelFunc) *capture {
	capture := &capture{
		limit:  limit,
		sha256: sha256.New(),
		sha512: sha512.New(),
		cancel: cancel,
	}
	capture.redactor = redact.NewWriter(&capture.buffer)
	return capture
}

func (capture *capture) Write(data []byte) (int, error) {
	capture.mu.Lock()
	defer capture.mu.Unlock()

	allowed := int64(len(data))
	if remaining := capture.limit - capture.bytes; allowed > remaining {
		allowed = remaining
	}
	if allowed > 0 {
		chunk := data[:allowed]
		_, _ = capture.sha256.Write(chunk)
		_, _ = capture.sha512.Write(chunk)
		_, _ = capture.redactor.Write(chunk)
		capture.bytes += allowed
	}
	if allowed < int64(len(data)) {
		capture.exceeded = true
		capture.cancel()
		return int(allowed), errOutputLimit
	}
	return len(data), nil
}

func (capture *capture) evidence() StreamEvidence {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	_ = capture.redactor.Flush()
	return StreamEvidence{
		Bytes:          capture.bytes,
		SHA256:         hex.EncodeToString(capture.sha256.Sum(nil)),
		SHA512:         hex.EncodeToString(capture.sha512.Sum(nil)),
		Sanitized:      capture.buffer.String(),
		LimitExceeded:  capture.exceeded,
		RedactionBound: "credential-shaped values are redacted; raw output is not retained",
	}
}
