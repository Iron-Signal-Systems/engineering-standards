package projectcommand

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

const govulncheckRuntimeConfigurationName = "tool-versions.json"

type govulncheckRuntimeExecutor func(
	context.Context,
	string,
	string,
) (govulncheckRuntimeResult, error)

func executeGovulncheckProjectCommand(
	ctx context.Context,
	request Request,
	result Result,
	pendingPath string,
	executor govulncheckRuntimeExecutor,
) (Result, error) {
	if executor == nil {
		return finalizeGovulncheckProjectCommandFailure(
			result,
			pendingPath,
			errors.New("govulncheck runtime executor is unavailable"),
		)
	}
	if !projectUsesGoProfile(request) {
		return finalizeGovulncheckProjectCommandFailure(
			result,
			pendingPath,
			errors.New("known_vulnerabilities requires the Go profile"),
		)
	}
	if len(result.Arguments) != 2 ||
		result.Arguments[0] != projectpin.GovulncheckExecutable ||
		result.Arguments[1] != projectpin.GovulncheckPackageScope {
		return finalizeGovulncheckProjectCommandFailure(
			result,
			pendingPath,
			errors.New(
				`known_vulnerabilities must be exactly ["govulncheck", "./..."]`,
			),
		)
	}

	configuration, err := govulncheckRuntimeConfigurationPath(
		request.Root,
		request.Pin.Evidence.Directory,
	)
	if err != nil {
		return finalizeGovulncheckProjectCommandFailure(
			result,
			pendingPath,
			err,
		)
	}

	absoluteRoot, err := filepath.Abs(request.Root)
	if err != nil {
		return finalizeGovulncheckProjectCommandFailure(
			result,
			pendingPath,
			errors.New("resolve govulncheck project-command repository root"),
		)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	result.ResolvedExecutable = filepath.Join(
		absoluteRoot,
		filepath.FromSlash(govulncheckRuntimeExecutable),
	)
	result.EnvironmentNames = nil
	result.Stdout = emptyStreamEvidence()
	result.Stderr = emptyStreamEvidence()
	result.Started = time.Now().UTC()

	runtimeResult, runtimeErr := executor(
		ctx,
		absoluteRoot,
		configuration,
	)

	result.Finished = time.Now().UTC()
	result.DurationMilliseconds = result.Finished.Sub(
		result.Started,
	).Milliseconds()
	applyGovulncheckRuntimeResult(&result, runtimeResult)

	switch {
	case runtimeErr != nil:
		result.Status = "FAIL"
		result.Failure = runtimeErr.Error()
	case result.Govulncheck == nil:
		result.Status = "FAIL"
		result.Failure = "govulncheck runtime completed without typed evidence"
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

func finalizeGovulncheckProjectCommandFailure(
	result Result,
	pendingPath string,
	failure error,
) (Result, error) {
	result.Status = "FAIL"
	result.Failure = failure.Error()
	result.Finished = time.Now().UTC()
	result.DurationMilliseconds = result.Finished.Sub(
		result.Started,
	).Milliseconds()

	writeErr := finalizeEvidence(result, pendingPath)
	if writeErr != nil {
		result.Status = "FAIL"
		result.Failure = "project command evidence could not be finalized"
		return result, errors.Join(failure, writeErr)
	}
	return result, failure
}

func applyGovulncheckRuntimeResult(
	result *Result,
	runtimeResult govulncheckRuntimeResult,
) {
	if runtimeResult.SelectedGo.Executable != "" ||
		len(runtimeResult.SelectedGo.Modules) > 0 {
		result.GoToolchain = newGoToolchainEvidence(
			runtimeResult.SelectedGo,
		)
	}
	if runtimeResult.Evidence.Executable != "" ||
		len(runtimeResult.Evidence.Modules) > 0 {
		evidence := runtimeResult.Evidence
		result.Govulncheck = &evidence
	}
	if runtimeResult.Tool.Executable != "" {
		result.ResolvedExecutable = runtimeResult.Tool.Executable
	}

	if len(runtimeResult.Run.Modules) == 0 {
		result.ExitCode = -1
		return
	}

	result.ExitCode = 0
	names := make(map[string]struct{})
	for _, module := range runtimeResult.Run.Modules {
		if result.ExitCode == 0 && module.ExitCode != 0 {
			result.ExitCode = module.ExitCode
		}
		result.TimedOut = result.TimedOut || module.TimedOut
		result.OutputLimitExceeded =
			result.OutputLimitExceeded ||
				module.OutputLimitExceeded
		result.RepositoryStateChanged =
			result.RepositoryStateChanged ||
				module.RepositoryStateChanged
		for _, name := range module.EnvironmentNames {
			names[name] = struct{}{}
		}
	}

	result.EnvironmentNames = make([]string, 0, len(names))
	for name := range names {
		result.EnvironmentNames = append(
			result.EnvironmentNames,
			name,
		)
	}
	sort.Strings(result.EnvironmentNames)
}

func govulncheckRuntimeConfigurationPath(
	root string,
	evidenceDirectory string,
) (string, error) {
	if evidenceDirectory == "" ||
		filepath.IsAbs(evidenceDirectory) ||
		strings.Contains(evidenceDirectory, `\`) {
		return "", errors.New(
			"govulncheck runtime evidence directory is unsafe",
		)
	}

	cleanEvidence := filepath.ToSlash(
		filepath.Clean(
			filepath.FromSlash(evidenceDirectory),
		),
	)
	if cleanEvidence != evidenceDirectory ||
		cleanEvidence == "." ||
		cleanEvidence == ".." ||
		strings.HasPrefix(cleanEvidence, "../") {
		return "", errors.New(
			"govulncheck runtime evidence directory is unsafe",
		)
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", errors.New(
			"resolve govulncheck runtime repository root",
		)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	configuration := filepath.Join(
		absoluteRoot,
		filepath.FromSlash(cleanEvidence),
		"runtime",
		govulncheckRuntimeConfigurationName,
	)
	if !pathWithin(absoluteRoot, configuration) {
		return "", errors.New(
			"govulncheck runtime configuration escapes the repository",
		)
	}
	if err := rejectGovulncheckRuntimeConfigurationSymlinks(
		absoluteRoot,
		filepath.Dir(configuration),
	); err != nil {
		return "", err
	}
	return configuration, nil
}

func rejectGovulncheckRuntimeConfigurationSymlinks(
	root string,
	directory string,
) error {
	relative, err := filepath.Rel(root, directory)
	if err != nil ||
		relative == ".." ||
		strings.HasPrefix(
			relative,
			".."+string(filepath.Separator),
		) {
		return errors.New(
			"govulncheck runtime configuration escapes the repository",
		)
	}

	current := root
	for _, component := range strings.Split(
		relative,
		string(filepath.Separator),
	) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf(
				"govulncheck runtime configuration directory is missing: %s",
				evidenceRelativePath(root, current),
			)
		}
		if err != nil {
			return errors.New(
				"inspect govulncheck runtime configuration path",
			)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New(
				"govulncheck runtime configuration path contains a symbolic link",
			)
		}
		if !info.IsDir() {
			return errors.New(
				"govulncheck runtime configuration path contains a non-directory",
			)
		}
	}
	return nil
}

func evidenceRelativePath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}
