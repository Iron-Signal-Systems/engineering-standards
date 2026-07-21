package projectcommand

import (
	"strings"
	"testing"
)

func TestReconcileGovulncheckExceptionsMatchesOnlyExactScope(
	t *testing.T,
) {
	run := govulncheckModuleRun{
		Modules: []govulncheckModuleScanResult{{
			GoModPath:  "go.mod",
			ModulePath: "example.com/project",
			Protocol: govulncheckProtocolSummary{
				SymbolLevelFindings: 2,
				ReachableFindings: []govulncheckReachableFinding{
					{
						AdvisoryID:   "GO-2026-9999",
						ModulePath:   "example.com/dep",
						PackagePath:  "example.com/dep/service",
						Symbol:       "(*Service).Handle",
						FixedVersion: "v1.2.4",
					},
					{
						AdvisoryID:   "GO-2026-9999",
						ModulePath:   "example.com/dep",
						PackagePath:  "example.com/dep/service",
						Symbol:       "(*Service).Handle",
						FixedVersion: "v1.2.5",
					},
				},
			},
		}},
	}
	exception := exactExceptionForFinding(
		"GO-2026-9999",
		"go.mod",
		"example.com/dep",
		"example.com/dep/service",
		"(*Service).Handle",
	)
	document := govulncheckExceptionDocument{
		SchemaVersion: 1,
		Exceptions:    []govulncheckException{exception},
	}

	reconciliation, err := reconcileGovulncheckExceptions(
		run,
		document,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(reconciliation.Used) != 1 ||
		len(reconciliation.Unused) != 0 ||
		len(reconciliation.Unexcepted) != 0 {
		t.Fatalf("reconciliation = %#v", reconciliation)
	}
	finding := reconciliation.Used[0].Finding
	if finding.Occurrences != 2 ||
		strings.Join(finding.FixedVersions, ",") != "v1.2.4,v1.2.5" {
		t.Fatalf("finding occurrence = %#v", finding)
	}
}

func TestReconcileGovulncheckExceptionsRejectsScopeDrift(
	t *testing.T,
) {
	baseRun := govulncheckModuleRun{
		Modules: []govulncheckModuleScanResult{{
			GoModPath:  "go.mod",
			ModulePath: "example.com/project",
			Protocol: govulncheckProtocolSummary{
				SymbolLevelFindings: 1,
				ReachableFindings: []govulncheckReachableFinding{{
					AdvisoryID:  "GO-2026-9999",
					ModulePath:  "example.com/dep",
					PackagePath: "example.com/dep/service",
					Symbol:      "(*Service).Handle",
				}},
			},
		}},
	}
	base := exactExceptionForFinding(
		"GO-2026-9999",
		"go.mod",
		"example.com/dep",
		"example.com/dep/service",
		"(*Service).Handle",
	)

	tests := []struct {
		name   string
		mutate func(*govulncheckException)
	}{
		{
			name: "advisory",
			mutate: func(exception *govulncheckException) {
				exception.AdvisoryID = "GO-2026-0001"
			},
		},
		{
			name: "go mod",
			mutate: func(exception *govulncheckException) {
				exception.Scope.GoModPath = "tools/a/go.mod"
			},
		},
		{
			name: "module",
			mutate: func(exception *govulncheckException) {
				exception.Scope.ModulePath = "example.com/other"
			},
		},
		{
			name: "package",
			mutate: func(exception *govulncheckException) {
				exception.Scope.PackagePath = "example.com/dep/other"
			},
		},
		{
			name: "symbol",
			mutate: func(exception *govulncheckException) {
				exception.Scope.Symbol = "(*Service).Other"
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			exception := base
			testCase.mutate(&exception)
			reconciliation, err := reconcileGovulncheckExceptions(
				baseRun,
				govulncheckExceptionDocument{
					SchemaVersion: 1,
					Exceptions: []govulncheckException{
						exception,
					},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(reconciliation.Used) != 0 ||
				len(reconciliation.Unused) != 1 ||
				len(reconciliation.Unexcepted) != 1 {
				t.Fatalf(
					"scope drift reconciled unexpectedly: %#v",
					reconciliation,
				)
			}
		})
	}
}

func TestReconcileGovulncheckExceptionsIsDeterministic(
	t *testing.T,
) {
	run := govulncheckModuleRun{
		Modules: []govulncheckModuleScanResult{
			{
				GoModPath:  "tools/z/go.mod",
				ModulePath: "example.com/z",
				Protocol: govulncheckProtocolSummary{
					SymbolLevelFindings: 1,
					ReachableFindings: []govulncheckReachableFinding{{
						AdvisoryID:  "GO-2026-0002",
						ModulePath:  "example.com/zdep",
						PackagePath: "example.com/zdep/pkg",
						Symbol:      "Z",
					}},
				},
			},
			{
				GoModPath:  "go.mod",
				ModulePath: "example.com/root",
				Protocol: govulncheckProtocolSummary{
					SymbolLevelFindings:  1,
					UnknownLevelFindings: 2,
					ReachableFindings: []govulncheckReachableFinding{{
						AdvisoryID:  "GO-2026-0001",
						ModulePath:  "example.com/adep",
						PackagePath: "example.com/adep/pkg",
						Symbol:      "A",
					}},
				},
			},
		},
	}
	document := govulncheckExceptionDocument{
		SchemaVersion: 1,
		Exceptions: []govulncheckException{
			exactExceptionForFinding(
				"GO-2026-9999",
				"go.mod",
				"example.com/unused",
				"example.com/unused/pkg",
				"Unused",
			),
			exactExceptionForFinding(
				"GO-2026-0002",
				"tools/z/go.mod",
				"example.com/zdep",
				"example.com/zdep/pkg",
				"Z",
			),
		},
	}

	reconciliation, err := reconcileGovulncheckExceptions(
		run,
		document,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(reconciliation.Used) != 1 ||
		len(reconciliation.Unused) != 1 ||
		len(reconciliation.Unexcepted) != 1 ||
		len(reconciliation.Unknown) != 1 {
		t.Fatalf("reconciliation = %#v", reconciliation)
	}
	if reconciliation.Unexcepted[0].GoModPath != "go.mod" ||
		reconciliation.Used[0].Finding.GoModPath != "tools/z/go.mod" ||
		reconciliation.Unknown[0].GoModPath != "go.mod" ||
		reconciliation.Unknown[0].Occurrences != 2 {
		t.Fatalf("deterministic order/content = %#v", reconciliation)
	}
}

func TestReconcileGovulncheckExceptionsRejectsDetailDrift(
	t *testing.T,
) {
	run := govulncheckModuleRun{
		Modules: []govulncheckModuleScanResult{{
			GoModPath:  "go.mod",
			ModulePath: "example.com/project",
			Protocol: govulncheckProtocolSummary{
				SymbolLevelFindings: 1,
			},
		}},
	}
	_, err := reconcileGovulncheckExceptions(
		run,
		govulncheckExceptionDocument{SchemaVersion: 1},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "detail count") {
		t.Fatalf("error = %v", err)
	}
}

func TestReconcileGovulncheckExceptionsRejectsDuplicateBoundaries(
	t *testing.T,
) {
	finding := govulncheckReachableFinding{
		AdvisoryID:  "GO-2026-9999",
		ModulePath:  "example.com/dep",
		PackagePath: "example.com/dep/pkg",
		Symbol:      "Open",
	}
	run := govulncheckModuleRun{
		Modules: []govulncheckModuleScanResult{
			{
				GoModPath:  "go.mod",
				ModulePath: "example.com/project",
				Protocol: govulncheckProtocolSummary{
					SymbolLevelFindings: 1,
					ReachableFindings: []govulncheckReachableFinding{
						finding,
					},
				},
			},
			{
				GoModPath:  "go.mod",
				ModulePath: "example.com/project",
				Protocol: govulncheckProtocolSummary{
					SymbolLevelFindings: 1,
					ReachableFindings: []govulncheckReachableFinding{
						finding,
					},
				},
			},
		},
	}
	_, err := reconcileGovulncheckExceptions(
		run,
		govulncheckExceptionDocument{SchemaVersion: 1},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "duplicate module") {
		t.Fatalf("error = %v", err)
	}

	exception := exactExceptionForFinding(
		"GO-2026-9999",
		"go.mod",
		"example.com/dep",
		"example.com/dep/pkg",
		"Open",
	)
	_, err = reconcileGovulncheckExceptions(
		govulncheckModuleRun{},
		govulncheckExceptionDocument{
			SchemaVersion: 1,
			Exceptions: []govulncheckException{
				exception,
				exception,
			},
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "duplicate exact scope") {
		t.Fatalf("error = %v", err)
	}
}

func exactExceptionForFinding(
	advisory string,
	goModPath string,
	modulePath string,
	packagePath string,
	symbol string,
) govulncheckException {
	return govulncheckException{
		AdvisoryID: advisory,
		Scope: govulncheckExceptionScope{
			GoModPath:   goModPath,
			ModulePath:  modulePath,
			PackagePath: packagePath,
			Symbol:      symbol,
		},
		Justification: "Exact synthetic exception used only for reconciliation testing.",
		CompensatingControls: []string{
			"Synthetic tests constrain the exact vulnerable path.",
		},
		Owner: "owner@example.invalid",
		Approval: govulncheckExceptionApproval{
			ApprovedBy: "approver@example.invalid",
			ApprovedAt: "2026-07-21T09:00:00Z",
			Record:     "TEST-1",
		},
		ExpiresAt: "2026-07-28T09:00:00Z",
		Remediation: govulncheckExceptionRemediation{
			Owner:      "remediation@example.invalid",
			TargetDate: "2026-07-28",
			Plan:       "Replace the affected synthetic dependency and remove the exception.",
		},
	}
}
