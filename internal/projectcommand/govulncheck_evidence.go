package projectcommand

import (
	"fmt"
	"path/filepath"
	"sort"
)

type GovulncheckEvidence struct {
	Executable             string                         `json:"executable"`
	Directory              string                         `json:"directory"`
	ApprovedCommandPackage string                         `json:"approved_command_package"`
	EmbeddedModule         string                         `json:"embedded_module"`
	Version                string                         `json:"version"`
	BuildGoVersion         string                         `json:"build_go_version"`
	SHA256                 string                         `json:"sha256"`
	PackageScope           string                         `json:"package_scope"`
	GOTOOLCHAINEffective   string                         `json:"GOTOOLCHAIN_effective"`
	GOENVEffective         string                         `json:"GOENV_effective"`
	Modules                []GovulncheckModuleEvidence    `json:"modules"`
	Exceptions             *GovulncheckExceptionsEvidence `json:"exceptions,omitempty"`
}

type GovulncheckModuleEvidence struct {
	GoModPath              string                      `json:"go_mod_path"`
	Directory              string                      `json:"directory"`
	ModulePath             string                      `json:"module_path"`
	PackageScope           string                      `json:"package_scope"`
	EnvironmentNames       []string                    `json:"environment_names"`
	Started                string                      `json:"started"`
	Finished               string                      `json:"finished"`
	DurationMilliseconds   int64                       `json:"duration_milliseconds"`
	ExitCode               int                         `json:"exit_code"`
	TimedOut               bool                        `json:"timed_out"`
	OutputLimitExceeded    bool                        `json:"output_limit_exceeded"`
	RepositoryStateChanged bool                        `json:"repository_state_changed"`
	Protocol               GovulncheckProtocolEvidence `json:"protocol"`
	Stdout                 StreamEvidence              `json:"stdout"`
	Stderr                 StreamEvidence              `json:"stderr"`
}

type GovulncheckProtocolEvidence struct {
	ProtocolVersion      string                      `json:"protocol_version"`
	ScannerName          string                      `json:"scanner_name"`
	ScannerVersion       string                      `json:"scanner_version"`
	Database             string                      `json:"database"`
	DatabaseLastModified string                      `json:"database_last_modified"`
	GoVersion            string                      `json:"go_version"`
	ScanLevel            string                      `json:"scan_level"`
	ScanMode             string                      `json:"scan_mode"`
	MessageCount         int                         `json:"message_count"`
	ConfigMessages       int                         `json:"config_messages"`
	ProgressMessages     int                         `json:"progress_messages"`
	SBOMMessages         int                         `json:"sbom_messages"`
	OSVMessages          int                         `json:"osv_messages"`
	FindingMessages      int                         `json:"finding_messages"`
	SBOMRoots            []string                    `json:"sbom_roots"`
	SBOMModules          []GovulncheckProtocolModule `json:"sbom_modules"`
	OSVAdvisoryIDs       []string                    `json:"osv_advisory_ids"`
	FindingAdvisoryIDs   []string                    `json:"finding_advisory_ids"`
	ModuleLevelFindings  int                         `json:"module_level_findings"`
	PackageLevelFindings int                         `json:"package_level_findings"`
	SymbolLevelFindings  int                         `json:"symbol_level_findings"`
	UnknownLevelFindings int                         `json:"unknown_level_findings"`
}

type GovulncheckProtocolModule struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

func projectGovulncheckEvidence(
	run govulncheckModuleRun,
	expectedModules []goModuleSelection,
) (GovulncheckEvidence, error) {
	if len(expectedModules) == 0 {
		return GovulncheckEvidence{}, fmt.Errorf("govulncheck evidence requires at least one governed Go module")
	}
	if len(run.Modules) != len(expectedModules) {
		return GovulncheckEvidence{}, fmt.Errorf(
			"govulncheck evidence module count %d does not match governed module count %d",
			len(run.Modules),
			len(expectedModules),
		)
	}

	expectedByGoMod := make(map[string]goModuleSelection, len(expectedModules))
	for _, module := range expectedModules {
		if _, exists := expectedByGoMod[module.GoModPath]; exists {
			return GovulncheckEvidence{}, fmt.Errorf(
				"governed Go module inventory contains duplicate path %q",
				module.GoModPath,
			)
		}
		expectedByGoMod[module.GoModPath] = module
	}

	results := append([]govulncheckModuleScanResult(nil), run.Modules...)
	sort.Slice(results, func(i, j int) bool {
		return results[i].GoModPath < results[j].GoModPath
	})

	seen := make(map[string]struct{}, len(results))
	modules := make([]GovulncheckModuleEvidence, 0, len(results))
	for _, result := range results {
		expected, ok := expectedByGoMod[result.GoModPath]
		if !ok {
			return GovulncheckEvidence{}, fmt.Errorf(
				"govulncheck result references ungoverned module %q",
				result.GoModPath,
			)
		}
		if _, duplicate := seen[result.GoModPath]; duplicate {
			return GovulncheckEvidence{}, fmt.Errorf(
				"govulncheck result duplicates module %q",
				result.GoModPath,
			)
		}
		seen[result.GoModPath] = struct{}{}

		if result.Directory != expected.Directory {
			return GovulncheckEvidence{}, fmt.Errorf(
				"govulncheck result directory %q does not match governed directory %q for %s",
				result.Directory,
				expected.Directory,
				result.GoModPath,
			)
		}
		if result.ModulePath != expected.ModulePath {
			return GovulncheckEvidence{}, fmt.Errorf(
				"govulncheck result module path %q does not match governed module path %q for %s",
				result.ModulePath,
				expected.ModulePath,
				result.GoModPath,
			)
		}
		if result.PackageScope != "./..." {
			return GovulncheckEvidence{}, fmt.Errorf(
				"govulncheck result package scope for %s must be ./...",
				result.GoModPath,
			)
		}

		modules = append(modules, GovulncheckModuleEvidence{
			GoModPath:              result.GoModPath,
			Directory:              result.Directory,
			ModulePath:             result.ModulePath,
			PackageScope:           result.PackageScope,
			EnvironmentNames:       append([]string(nil), result.EnvironmentNames...),
			Started:                result.Started.UTC().Format("2006-01-02T15:04:05.999999999Z"),
			Finished:               result.Finished.UTC().Format("2006-01-02T15:04:05.999999999Z"),
			DurationMilliseconds:   result.DurationMilliseconds,
			ExitCode:               result.ExitCode,
			TimedOut:               result.TimedOut,
			OutputLimitExceeded:    result.OutputLimitExceeded,
			RepositoryStateChanged: result.RepositoryStateChanged,
			Protocol:               projectGovulncheckProtocolEvidence(result.Protocol),
			Stdout:                 result.Stdout,
			Stderr:                 result.Stderr,
		})
	}

	for path := range expectedByGoMod {
		if _, ok := seen[path]; !ok {
			return GovulncheckEvidence{}, fmt.Errorf(
				"govulncheck evidence is missing governed module %q",
				path,
			)
		}
	}

	return GovulncheckEvidence{
		Executable:             run.Tool.Executable,
		Directory:              filepath.Dir(run.Tool.Executable),
		ApprovedCommandPackage: run.Tool.CommandPackage,
		EmbeddedModule:         run.Tool.Module,
		Version:                run.Tool.Version,
		BuildGoVersion:         run.Tool.BuildGoVersion,
		SHA256:                 run.Tool.SHA256,
		PackageScope:           "./...",
		GOTOOLCHAINEffective:   "local",
		GOENVEffective:         "off",
		Modules:                modules,
	}, nil
}

func projectGovulncheckProtocolEvidence(
	summary govulncheckProtocolSummary,
) GovulncheckProtocolEvidence {
	modules := make([]GovulncheckProtocolModule, len(summary.SBOMModules))
	for index, module := range summary.SBOMModules {
		modules[index] = GovulncheckProtocolModule{
			Path:    module.Path,
			Version: module.Version,
		}
	}

	return GovulncheckProtocolEvidence{
		ProtocolVersion:      summary.Config.ProtocolVersion,
		ScannerName:          summary.Config.ScannerName,
		ScannerVersion:       summary.Config.ScannerVersion,
		Database:             summary.Config.Database,
		DatabaseLastModified: summary.Config.DatabaseLastModified,
		GoVersion:            summary.Config.GoVersion,
		ScanLevel:            summary.Config.ScanLevel,
		ScanMode:             summary.Config.ScanMode,
		MessageCount:         summary.MessageCount,
		ConfigMessages:       summary.ConfigMessages,
		ProgressMessages:     summary.ProgressMessages,
		SBOMMessages:         summary.SBOMMessages,
		OSVMessages:          summary.OSVMessages,
		FindingMessages:      summary.FindingMessages,
		SBOMRoots:            append([]string(nil), summary.SBOMRoots...),
		SBOMModules:          modules,
		OSVAdvisoryIDs:       append([]string(nil), summary.OSVAdvisoryIDs...),
		FindingAdvisoryIDs:   append([]string(nil), summary.FindingAdvisoryIDs...),
		ModuleLevelFindings:  summary.ModuleLevelFindings,
		PackageLevelFindings: summary.PackageLevelFindings,
		SymbolLevelFindings:  summary.SymbolLevelFindings,
		UnknownLevelFindings: summary.UnknownLevelFindings,
	}
}
