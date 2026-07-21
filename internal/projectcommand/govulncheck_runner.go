package projectcommand

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
)

const govulncheckPackageScope = "./..."

var (
	govulncheckModuleTimeout           = ExecutionTimeout
	govulncheckModuleOutputLimit int64 = MaxOutputBytes
	errGovulncheckOutputLimit          = errors.New("govulncheck output exceeded the configured limit")
)

type govulncheckModuleScanResult struct {
	GoModPath              string
	Directory              string
	ModulePath             string
	PackageScope           string
	EnvironmentNames       []string
	Started                time.Time
	Finished               time.Time
	DurationMilliseconds   int64
	ExitCode               int
	TimedOut               bool
	OutputLimitExceeded    bool
	RepositoryStateChanged bool
	Stdout                 StreamEvidence
	Stderr                 StreamEvidence
	Protocol               govulncheckProtocolSummary
}

type govulncheckModuleRun struct {
	Tool    govulncheckToolIdentity
	Modules []govulncheckModuleScanResult
}

type govulncheckScanCapture struct {
	mu       sync.Mutex
	limit    int64
	bytes    int64
	sha256   hash.Hash
	sha512   hash.Hash
	raw      bytes.Buffer
	exceeded bool
	cancel   context.CancelFunc
}

func newGovulncheckScanCapture(limit int64, cancel context.CancelFunc) *govulncheckScanCapture {
	return &govulncheckScanCapture{
		limit:  limit,
		sha256: sha256.New(),
		sha512: sha512.New(),
		cancel: cancel,
	}
}

func (capture *govulncheckScanCapture) Write(data []byte) (int, error) {
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
		_, _ = capture.raw.Write(chunk)
		capture.bytes += allowed
	}
	if allowed < int64(len(data)) {
		capture.exceeded = true
		capture.cancel()
		return int(allowed), errGovulncheckOutputLimit
	}
	return len(data), nil
}

func (capture *govulncheckScanCapture) rawBytes() []byte {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return append([]byte(nil), capture.raw.Bytes()...)
}

func (capture *govulncheckScanCapture) evidence() StreamEvidence {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return StreamEvidence{
		Bytes:          capture.bytes,
		SHA256:         hex.EncodeToString(capture.sha256.Sum(nil)),
		SHA512:         hex.EncodeToString(capture.sha512.Sum(nil)),
		Sanitized:      redact.Sanitize(capture.raw.String()),
		LimitExceeded:  capture.exceeded,
		RedactionBound: "credential-shaped values are redacted; raw output is not retained",
	}
}

func runGovulncheckModules(
	ctx context.Context,
	root string,
	selectedGo goToolchainSelection,
	tool govulncheckToolIdentity,
) (govulncheckModuleRun, error) {
	var run govulncheckModuleRun
	if ctx == nil {
		return run, errors.New("govulncheck scan context is required")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return run, errors.New("resolve govulncheck repository root")
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	rootInfo, err := os.Lstat(absoluteRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return run, errors.New("govulncheck repository root must be a nonsymlink directory")
	}
	if len(selectedGo.Modules) == 0 {
		return run, errors.New("govulncheck requires at least one governed Go module")
	}
	selectedGo.Executable, err = exactRegularExecutable(selectedGo.Executable, "selected Go executable")
	if err != nil {
		return run, err
	}
	selectedGo.Directory = filepath.Dir(selectedGo.Executable)
	tool.Executable, err = exactRegularExecutable(tool.Executable, "govulncheck executable")
	if err != nil {
		return run, err
	}
	tool.Directory = filepath.Dir(tool.Executable)
	if tool.CommandPackage != govulncheckCommandPackage || tool.Module != govulncheckModuleRoot || !exactGovulncheckVersionPattern.MatchString(tool.Version) || tool.SHA256 == "" {
		return run, errors.New("govulncheck tool identity is incomplete or unapproved")
	}
	run.Tool = tool

	baseline, err := snapshot(ctx, absoluteRoot, ".local/isras")
	if err != nil {
		return run, errors.New("capture repository state before govulncheck execution")
	}

	environmentRoot, err := os.MkdirTemp("", "isras-govulncheck-")
	if err != nil {
		return run, errors.New("create isolated govulncheck environment")
	}
	defer os.RemoveAll(environmentRoot)
	if err := os.Chmod(environmentRoot, 0o700); err != nil {
		return run, errors.New("secure isolated govulncheck environment")
	}

	modules := append([]goModuleSelection(nil), selectedGo.Modules...)
	sort.Slice(modules, func(left, right int) bool {
		return modules[left].GoModPath < modules[right].GoModPath
	})
	seen := make(map[string]struct{}, len(modules))
	moduleDirectories := make([]string, len(modules))
	for index, module := range modules {
		if _, ok := seen[module.GoModPath]; ok {
			return run, fmt.Errorf("duplicate govulncheck module inventory path %q", module.GoModPath)
		}
		seen[module.GoModPath] = struct{}{}

		moduleDirectory, err := governedGovulncheckModuleDirectory(absoluteRoot, module)
		if err != nil {
			return run, err
		}
		moduleDirectories[index] = moduleDirectory
	}

	for index, module := range modules {
		moduleDirectory := moduleDirectories[index]
		moduleEnvironmentRoot := filepath.Join(environmentRoot, fmt.Sprintf("module-%03d", index+1))
		environment, names, err := govulncheckModuleEnvironment(moduleEnvironmentRoot, selectedGo, tool)
		if err != nil {
			return run, err
		}

		result, err := runGovulncheckModule(ctx, absoluteRoot, moduleDirectory, module, selectedGo, tool, environment, names, baseline)
		run.Modules = append(run.Modules, result)
		if err != nil {
			return run, err
		}
	}

	if len(run.Modules) != len(modules) {
		return run, errors.New("govulncheck module execution count does not match the governed inventory")
	}
	return run, nil
}

func governedGovulncheckModuleDirectory(root string, module goModuleSelection) (string, error) {
	if module.GoModPath == "" || filepath.IsAbs(module.GoModPath) || strings.Contains(module.GoModPath, "\\") {
		return "", errors.New("govulncheck module go.mod path is unsafe")
	}
	if module.Directory == "" || filepath.IsAbs(module.Directory) || strings.Contains(module.Directory, "\\") {
		return "", errors.New("govulncheck module directory is unsafe")
	}
	cleanDirectory := filepath.ToSlash(filepath.Clean(filepath.FromSlash(module.Directory)))
	if cleanDirectory != module.Directory || cleanDirectory == ".." || strings.HasPrefix(cleanDirectory, "../") {
		return "", errors.New("govulncheck module directory is unsafe")
	}
	moduleDirectory := root
	if module.Directory != "." {
		moduleDirectory = filepath.Join(root, filepath.FromSlash(module.Directory))
	}
	if !pathWithin(root, moduleDirectory) {
		return "", errors.New("govulncheck module directory escapes the repository")
	}
	if err := rejectGovulncheckDirectorySymlinks(root, moduleDirectory); err != nil {
		return "", err
	}
	info, err := os.Lstat(moduleDirectory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("govulncheck module directory must be a nonsymlink directory")
	}
	goModPath := filepath.Join(root, filepath.FromSlash(module.GoModPath))
	if !pathWithin(root, goModPath) {
		return "", errors.New("govulncheck module go.mod path escapes the repository")
	}
	goModInfo, err := os.Lstat(goModPath)
	if err != nil || goModInfo.Mode()&os.ModeSymlink != 0 || !goModInfo.Mode().IsRegular() {
		return "", errors.New("govulncheck module go.mod must be a regular nonsymlink file")
	}
	if filepath.Clean(filepath.Dir(goModPath)) != filepath.Clean(moduleDirectory) {
		return "", errors.New("govulncheck module directory does not match go.mod path")
	}
	return moduleDirectory, nil
}

func rejectGovulncheckDirectorySymlinks(root, directory string) error {
	relative, err := filepath.Rel(root, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("govulncheck module directory escapes the repository")
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
		if err != nil {
			return errors.New("inspect govulncheck module directory")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("govulncheck module directory contains a symbolic link")
		}
	}
	return nil
}

func govulncheckModuleEnvironment(root string, selectedGo goToolchainSelection, tool govulncheckToolIdentity) ([]string, []string, error) {
	values := map[string]string{
		"HOME":           filepath.Join(root, "home"),
		"TMPDIR":         filepath.Join(root, "tmp"),
		"XDG_CACHE_HOME": filepath.Join(root, "cache"),
		"GOCACHE":        filepath.Join(root, "go-cache"),
		"GOPATH":         filepath.Join(root, "go"),
		"LANG":           "C",
		"LC_ALL":         "C",
		"GOTOOLCHAIN":    "local",
		"GOENV":          "off",
		"PATH":           sanitizedCommandPath(tool.Executable, selectedGo.Directory, tool.Directory),
	}
	for _, path := range []string{values["HOME"], values["TMPDIR"], values["XDG_CACHE_HOME"], values["GOCACHE"], values["GOPATH"]} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return nil, nil, errors.New("create isolated govulncheck environment")
		}
	}
	for _, name := range inheritedEnvironmentNames {
		if value, ok := os.LookupEnv(name); ok && value != "" {
			if strings.ContainsAny(value, "\x00\r\n") {
				return nil, nil, fmt.Errorf("environment variable %s contains a prohibited control character", name)
			}
			values[name] = value
		}
	}
	values["GOTOOLCHAIN"] = "local"
	values["GOENV"] = "off"
	if err := validatePath(values["PATH"]); err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	environment := make([]string, 0, len(names))
	for _, name := range names {
		environment = append(environment, name+"="+values[name])
	}
	return environment, names, nil
}

func runGovulncheckModule(
	ctx context.Context,
	root string,
	moduleDirectory string,
	module goModuleSelection,
	selectedGo goToolchainSelection,
	tool govulncheckToolIdentity,
	environment []string,
	environmentNames []string,
	baseline gitSnapshot,
) (govulncheckModuleScanResult, error) {
	result := govulncheckModuleScanResult{
		GoModPath:        module.GoModPath,
		Directory:        module.Directory,
		ModulePath:       module.ModulePath,
		PackageScope:     govulncheckPackageScope,
		EnvironmentNames: append([]string(nil), environmentNames...),
		ExitCode:         -1,
		Stdout:           emptyStreamEvidence(),
		Stderr:           emptyStreamEvidence(),
	}

	commandContext, cancel := context.WithTimeout(ctx, govulncheckModuleTimeout)
	defer cancel()
	stdout := newGovulncheckScanCapture(govulncheckModuleOutputLimit, cancel)
	stderr := newGovulncheckScanCapture(govulncheckModuleOutputLimit, cancel)
	command := exec.CommandContext(commandContext, tool.Executable, "-json", govulncheckPackageScope)
	command.Dir = moduleDirectory
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

	post, snapshotErr := snapshot(ctx, root, ".local/isras")
	if snapshotErr != nil || post != baseline {
		result.RepositoryStateChanged = true
	}

	switch {
	case result.OutputLimitExceeded:
		return result, fmt.Errorf("govulncheck module %s output exceeded the configured limit", module.GoModPath)
	case result.TimedOut:
		return result, fmt.Errorf("govulncheck module %s exceeded the configured timeout", module.GoModPath)
	case result.RepositoryStateChanged:
		return result, fmt.Errorf("govulncheck module %s changed Git-visible repository state or HEAD", module.GoModPath)
	case runErr != nil:
		return result, fmt.Errorf("govulncheck module %s execution failed with exit code %d", module.GoModPath, result.ExitCode)
	}

	protocol, err := parseGovulncheckProtocol(stdout.rawBytes())
	if err != nil {
		return result, fmt.Errorf("parse govulncheck module %s protocol: %w", module.GoModPath, err)
	}
	if protocol.Config.ScannerName != "" && protocol.Config.ScannerName != "govulncheck" {
		return result, fmt.Errorf("govulncheck module %s reported unexpected scanner name %q", module.GoModPath, protocol.Config.ScannerName)
	}
	if protocol.Config.ScannerVersion != "" && protocol.Config.ScannerVersion != tool.Version {
		return result, fmt.Errorf("govulncheck module %s reported scanner version %q instead of %q", module.GoModPath, protocol.Config.ScannerVersion, tool.Version)
	}
	if protocol.Config.ScanMode != "" && protocol.Config.ScanMode != "source" {
		return result, fmt.Errorf("govulncheck module %s reported unsupported scan mode %q", module.GoModPath, protocol.Config.ScanMode)
	}
	if protocol.Config.ScanLevel != "" && protocol.Config.ScanLevel != "symbol" {
		return result, fmt.Errorf("govulncheck module %s reported unsupported scan level %q", module.GoModPath, protocol.Config.ScanLevel)
	}
	if protocol.Config.GoVersion != "" && selectedGo.Actual != "" && protocol.Config.GoVersion != selectedGo.Actual {
		return result, fmt.Errorf("govulncheck module %s used Go version %q instead of selected %q", module.GoModPath, protocol.Config.GoVersion, selectedGo.Actual)
	}
	result.Protocol = protocol
	return result, nil
}
