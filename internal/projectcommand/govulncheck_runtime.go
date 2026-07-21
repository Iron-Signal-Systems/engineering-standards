package projectcommand

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const govulncheckRuntimeExecutable = ".local/tools/bin/govulncheck"

type govulncheckRuntimeResult struct {
	SelectedGo goToolchainSelection
	Tool       govulncheckToolIdentity
	Run        govulncheckModuleRun
	Evidence   GovulncheckEvidence
}

type govulncheckRuntimeDependencies struct {
	selectGo          func(string) (goToolchainSelection, error)
	verifyTool        func(context.Context, string, string, string) (govulncheckToolIdentity, error)
	runModules        func(context.Context, string, goToolchainSelection, govulncheckToolIdentity) (govulncheckModuleRun, error)
	projectEvidence   func(govulncheckModuleRun, []goModuleSelection) (GovulncheckEvidence, error)
	loadExceptions    func(string, time.Time) (govulncheckExceptionSource, error)
	reconcile         func(govulncheckModuleRun, govulncheckExceptionDocument) (govulncheckExceptionReconciliation, error)
	projectExceptions func(govulncheckExceptionSource, govulncheckExceptionReconciliation) (GovulncheckExceptionsEvidence, error)
	now               func() time.Time
}

func defaultGovulncheckRuntimeDependencies() govulncheckRuntimeDependencies {
	return govulncheckRuntimeDependencies{
		selectGo:          selectGoToolchain,
		verifyTool:        verifyGovulncheckTool,
		runModules:        runGovulncheckModules,
		projectEvidence:   projectGovulncheckEvidence,
		loadExceptions:    loadOptionalGovulncheckExceptions,
		reconcile:         reconcileGovulncheckExceptions,
		projectExceptions: projectGovulncheckExceptionEvidence,
		now:               func() time.Time { return time.Now().UTC() },
	}
}

func executeGovulncheckRuntime(
	ctx context.Context,
	root string,
	toolVersionConfiguration string,
) (govulncheckRuntimeResult, error) {
	return executeGovulncheckRuntimeWithDependencies(
		ctx,
		root,
		toolVersionConfiguration,
		defaultGovulncheckRuntimeDependencies(),
	)
}

func executeGovulncheckRuntimeWithDependencies(
	ctx context.Context,
	root string,
	toolVersionConfiguration string,
	dependencies govulncheckRuntimeDependencies,
) (govulncheckRuntimeResult, error) {
	var result govulncheckRuntimeResult
	if ctx == nil {
		return result, errors.New("govulncheck runtime context is required")
	}
	if dependencies.selectGo == nil ||
		dependencies.verifyTool == nil ||
		dependencies.runModules == nil ||
		dependencies.projectEvidence == nil {
		return result, errors.New("govulncheck runtime dependencies are incomplete")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return result, errors.New("resolve govulncheck runtime repository root")
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	if !filepath.IsAbs(toolVersionConfiguration) {
		return result, errors.New("govulncheck tool-version configuration path must be absolute")
	}
	toolVersionConfiguration = filepath.Clean(toolVersionConfiguration)

	selectedGo, err := dependencies.selectGo(absoluteRoot)
	result.SelectedGo = selectedGo
	if err != nil {
		return result, fmt.Errorf("select govulncheck Go toolchain: %w", err)
	}
	if len(selectedGo.Modules) == 0 {
		return result, errors.New("govulncheck runtime selected no governed Go modules")
	}

	toolExecutable := filepath.Join(
		absoluteRoot,
		filepath.FromSlash(govulncheckRuntimeExecutable),
	)
	tool, err := dependencies.verifyTool(
		ctx,
		selectedGo.Executable,
		toolExecutable,
		toolVersionConfiguration,
	)
	result.Tool = tool
	if err != nil {
		return result, fmt.Errorf("verify governed govulncheck tool: %w", err)
	}

	run, err := dependencies.runModules(
		ctx,
		absoluteRoot,
		selectedGo,
		tool,
	)
	result.Run = run
	if err != nil {
		return result, fmt.Errorf("execute governed govulncheck modules: %w", err)
	}

	evidence, err := dependencies.projectEvidence(
		run,
		selectedGo.Modules,
	)
	result.Evidence = evidence
	if err != nil {
		return result, fmt.Errorf("project governed govulncheck evidence: %w", err)
	}

	exceptionAware := dependencies.loadExceptions != nil ||
		dependencies.reconcile != nil ||
		dependencies.projectExceptions != nil ||
		dependencies.now != nil
	if !exceptionAware {
		if err := evaluateGovulncheckFindings(run); err != nil {
			return result, err
		}
		return result, nil
	}
	if dependencies.loadExceptions == nil ||
		dependencies.reconcile == nil ||
		dependencies.projectExceptions == nil ||
		dependencies.now == nil {
		return result, errors.New(
			"govulncheck exception-aware runtime dependencies are incomplete",
		)
	}

	evaluatedAt := dependencies.now().UTC()
	if evaluatedAt.IsZero() {
		return result, errors.New(
			"govulncheck exception-aware runtime evaluation time is required",
		)
	}
	source, err := dependencies.loadExceptions(
		absoluteRoot,
		evaluatedAt,
	)
	if err != nil {
		return result, fmt.Errorf(
			"load governed govulncheck exceptions: %w",
			err,
		)
	}
	reconciliation, err := dependencies.reconcile(
		run,
		source.Document,
	)
	if err != nil {
		return result, fmt.Errorf(
			"reconcile governed govulncheck exceptions: %w",
			err,
		)
	}
	exceptionEvidence, err := dependencies.projectExceptions(
		source,
		reconciliation,
	)
	if err != nil {
		return result, fmt.Errorf(
			"project governed govulncheck exception evidence: %w",
			err,
		)
	}
	evidence.Exceptions = &exceptionEvidence
	result.Evidence = evidence

	if err := evaluateGovulncheckExceptionReconciliation(
		reconciliation,
	); err != nil {
		return result, err
	}
	return result, nil
}

func evaluateGovulncheckFindings(run govulncheckModuleRun) error {
	var unknown []string
	var reachable []string

	for _, module := range run.Modules {
		if module.Protocol.UnknownLevelFindings > 0 {
			unknown = append(
				unknown,
				fmt.Sprintf(
					"%s=%d",
					module.GoModPath,
					module.Protocol.UnknownLevelFindings,
				),
			)
		}
		if module.Protocol.SymbolLevelFindings > 0 {
			advisories := append(
				[]string(nil),
				module.Protocol.FindingAdvisoryIDs...,
			)
			sort.Strings(advisories)
			advisoryText := strings.Join(advisories, ",")
			if advisoryText == "" {
				advisoryText = "unidentified"
			}
			reachable = append(
				reachable,
				fmt.Sprintf(
					"%s=%d[%s]",
					module.GoModPath,
					module.Protocol.SymbolLevelFindings,
					advisoryText,
				),
			)
		}
	}

	sort.Strings(unknown)
	sort.Strings(reachable)

	var failures []error
	if len(unknown) > 0 {
		failures = append(
			failures,
			fmt.Errorf(
				"govulncheck produced unknown-level findings: %s",
				strings.Join(unknown, "; "),
			),
		)
	}
	if len(reachable) > 0 {
		failures = append(
			failures,
			fmt.Errorf(
				"govulncheck found reachable vulnerabilities without governed exceptions: %s",
				strings.Join(reachable, "; "),
			),
		)
	}
	return errors.Join(failures...)
}
