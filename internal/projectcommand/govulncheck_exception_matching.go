package projectcommand

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type govulncheckFindingOccurrence struct {
	AdvisoryID    string   `json:"advisory_id"`
	GoModPath     string   `json:"go_mod_path"`
	ModulePath    string   `json:"module_path"`
	PackagePath   string   `json:"package_path"`
	Symbol        string   `json:"symbol"`
	FixedVersions []string `json:"fixed_versions"`
	Occurrences   int      `json:"occurrences"`
}

type govulncheckUsedException struct {
	Exception govulncheckException         `json:"exception"`
	Finding   govulncheckFindingOccurrence `json:"finding"`
}

type govulncheckUnknownFindingSummary struct {
	GoModPath   string `json:"go_mod_path"`
	Occurrences int    `json:"occurrences"`
}

type govulncheckExceptionReconciliation struct {
	Used       []govulncheckUsedException         `json:"used"`
	Unused     []govulncheckException             `json:"unused"`
	Unexcepted []govulncheckFindingOccurrence     `json:"unexcepted"`
	Unknown    []govulncheckUnknownFindingSummary `json:"unknown"`
}

func reconcileGovulncheckExceptions(
	run govulncheckModuleRun,
	document govulncheckExceptionDocument,
) (govulncheckExceptionReconciliation, error) {
	var reconciliation govulncheckExceptionReconciliation

	if document.SchemaVersion !=
		govulncheckExceptionSchemaVersion {
		return reconciliation, fmt.Errorf(
			"govulncheck exception schema_version must be %d",
			govulncheckExceptionSchemaVersion,
		)
	}

	exceptions := make(
		map[string]govulncheckException,
		len(document.Exceptions),
	)
	for _, exception := range document.Exceptions {
		key := govulncheckExceptionKey(exception)
		if _, duplicate := exceptions[key]; duplicate {
			return reconciliation, fmt.Errorf(
				"govulncheck exception reconciliation contains duplicate exact scope %q",
				key,
			)
		}
		exceptions[key] = exception
	}

	modules := append(
		[]govulncheckModuleScanResult(nil),
		run.Modules...,
	)
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].GoModPath < modules[j].GoModPath
	})

	seenModules := make(map[string]struct{}, len(modules))
	occurrences := make(
		map[string]*govulncheckFindingOccurrence,
	)
	for _, module := range modules {
		if module.GoModPath == "" ||
			module.ModulePath == "" {
			return reconciliation, errors.New(
				"govulncheck exception reconciliation requires exact governed module identity",
			)
		}
		if _, duplicate := seenModules[module.GoModPath]; duplicate {
			return reconciliation, fmt.Errorf(
				"govulncheck exception reconciliation contains duplicate module %q",
				module.GoModPath,
			)
		}
		seenModules[module.GoModPath] = struct{}{}

		if module.Protocol.SymbolLevelFindings !=
			len(module.Protocol.ReachableFindings) {
			return reconciliation, fmt.Errorf(
				"govulncheck reachable finding detail count %d does not match symbol-level count %d for %s",
				len(module.Protocol.ReachableFindings),
				module.Protocol.SymbolLevelFindings,
				module.GoModPath,
			)
		}
		if module.Protocol.UnknownLevelFindings > 0 {
			reconciliation.Unknown = append(
				reconciliation.Unknown,
				govulncheckUnknownFindingSummary{
					GoModPath: module.GoModPath,
					Occurrences: module.Protocol.
						UnknownLevelFindings,
				},
			)
		}

		for _, finding := range module.Protocol.ReachableFindings {
			if finding.AdvisoryID == "" ||
				finding.ModulePath == "" ||
				finding.PackagePath == "" ||
				finding.Symbol == "" {
				return reconciliation, fmt.Errorf(
					"govulncheck reachable finding for %s lacks exact scope",
					module.GoModPath,
				)
			}
			key := govulncheckFindingScopeKey(
				module.GoModPath,
				finding.AdvisoryID,
				finding.ModulePath,
				finding.PackagePath,
				finding.Symbol,
			)
			entry := occurrences[key]
			if entry == nil {
				entry = &govulncheckFindingOccurrence{
					AdvisoryID:  finding.AdvisoryID,
					GoModPath:   module.GoModPath,
					ModulePath:  finding.ModulePath,
					PackagePath: finding.PackagePath,
					Symbol:      finding.Symbol,
				}
				occurrences[key] = entry
			}
			entry.Occurrences++
			if finding.FixedVersion != "" &&
				!containsString(
					entry.FixedVersions,
					finding.FixedVersion,
				) {
				entry.FixedVersions = append(
					entry.FixedVersions,
					finding.FixedVersion,
				)
			}
		}
	}

	keys := make([]string, 0, len(occurrences))
	for key := range occurrences {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	used := make(map[string]struct{})
	for _, key := range keys {
		finding := *occurrences[key]
		sort.Strings(finding.FixedVersions)

		exception, matched := exceptions[key]
		if !matched {
			reconciliation.Unexcepted = append(
				reconciliation.Unexcepted,
				finding,
			)
			continue
		}
		used[key] = struct{}{}
		reconciliation.Used = append(
			reconciliation.Used,
			govulncheckUsedException{
				Exception: exception,
				Finding:   finding,
			},
		)
	}

	exceptionKeys := make([]string, 0, len(exceptions))
	for key := range exceptions {
		exceptionKeys = append(exceptionKeys, key)
	}
	sort.Strings(exceptionKeys)
	for _, key := range exceptionKeys {
		if _, matched := used[key]; matched {
			continue
		}
		reconciliation.Unused = append(
			reconciliation.Unused,
			exceptions[key],
		)
	}

	sort.Slice(reconciliation.Unknown, func(i, j int) bool {
		return reconciliation.Unknown[i].GoModPath <
			reconciliation.Unknown[j].GoModPath
	})
	return reconciliation, nil
}

func govulncheckFindingScopeKey(
	goModPath string,
	advisoryID string,
	modulePath string,
	packagePath string,
	symbol string,
) string {
	return strings.Join(
		[]string{
			advisoryID,
			goModPath,
			modulePath,
			packagePath,
			symbol,
		},
		"\x00",
	)
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
