package projectcommand

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadOptionalGovulncheckExceptionsAbsentAndPresent(t *testing.T) {
	now := time.Date(2026, 7, 21, 11, 0, 0, 0, time.UTC)

	t.Run("absent", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(
			filepath.Join(root, ".isras"),
			0o700,
		); err != nil {
			t.Fatal(err)
		}
		source, err := loadOptionalGovulncheckExceptions(
			root,
			now,
		)
		if err != nil {
			t.Fatal(err)
		}
		if source.Present ||
			source.Path != govulncheckExceptionRelativePath ||
			source.SHA256 != "" ||
			source.Document.SchemaVersion != 1 ||
			len(source.Document.Exceptions) != 0 {
			t.Fatalf("unexpected absent source: %+v", source)
		}
	})

	t.Run("present", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(
			root,
			filepath.FromSlash(govulncheckExceptionRelativePath),
		)
		writeGovulncheckExceptionDocument(
			t,
			path,
			validGovulncheckExceptionDocument(now),
		)
		source, err := loadOptionalGovulncheckExceptions(
			root,
			now,
		)
		if err != nil {
			t.Fatal(err)
		}
		if !source.Present ||
			len(source.SHA256) != 64 ||
			len(source.Document.Exceptions) != 1 {
			t.Fatalf("unexpected present source: %+v", source)
		}
	})

	t.Run("symlink parent", func(t *testing.T) {
		root := t.TempDir()
		external := t.TempDir()
		if err := os.Symlink(
			external,
			filepath.Join(root, ".isras"),
		); err != nil {
			t.Fatal(err)
		}
		_, err := loadOptionalGovulncheckExceptions(
			root,
			now,
		)
		if err == nil ||
			!strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestProjectGovulncheckExceptionEvidenceRetainsGovernance(t *testing.T) {
	now := time.Date(2026, 7, 21, 11, 0, 0, 0, time.UTC)
	exception := validGovulncheckExceptionDocument(now).Exceptions[0]
	finding := govulncheckFindingOccurrence{
		AdvisoryID:    exception.AdvisoryID,
		GoModPath:     exception.Scope.GoModPath,
		ModulePath:    exception.Scope.ModulePath,
		PackagePath:   exception.Scope.PackagePath,
		Symbol:        exception.Scope.Symbol,
		FixedVersions: []string{"v1.2.4"},
		Occurrences:   2,
	}
	source := govulncheckExceptionSource{
		Present:     true,
		Path:        govulncheckExceptionRelativePath,
		SHA256:      strings.Repeat("a", 64),
		EvaluatedAt: now,
		Document: govulncheckExceptionDocument{
			SchemaVersion: 1,
			Exceptions:    []govulncheckException{exception},
		},
	}
	reconciliation := govulncheckExceptionReconciliation{
		Used: []govulncheckUsedException{{
			Exception: exception,
			Finding:   finding,
		}},
	}

	evidence, err := projectGovulncheckExceptionEvidence(
		source,
		reconciliation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.Present ||
		len(evidence.Used) != 1 ||
		evidence.Used[0].Exception.Approval.Record !=
			exception.Approval.Record ||
		evidence.Used[0].Finding.Occurrences != 2 {
		t.Fatalf("unexpected exception evidence: %+v", evidence)
	}

	source.Document.Exceptions[0].CompensatingControls[0] =
		"mutated control"
	finding.FixedVersions[0] = "mutated"
	if evidence.Used[0].Exception.CompensatingControls[0] ==
		"mutated control" {
		t.Fatal("compensating controls were not cloned")
	}
	if evidence.Used[0].Finding.FixedVersions[0] == "mutated" {
		t.Fatal("fixed versions were not cloned")
	}
}

func TestEvaluateGovulncheckExceptionReconciliation(t *testing.T) {
	exception := exactExceptionForFinding(
		"GO-2026-9999",
		"go.mod",
		"example.com/dep",
		"example.com/dep/pkg",
		"Open",
	)
	finding := govulncheckFindingOccurrence{
		AdvisoryID:  "GO-2026-9999",
		GoModPath:   "go.mod",
		ModulePath:  "example.com/dep",
		PackagePath: "example.com/dep/pkg",
		Symbol:      "Open",
		Occurrences: 1,
	}

	tests := []struct {
		name  string
		value govulncheckExceptionReconciliation
		want  string
	}{
		{
			name: "exact used exception passes",
			value: govulncheckExceptionReconciliation{
				Used: []govulncheckUsedException{{
					Exception: exception,
					Finding:   finding,
				}},
			},
		},
		{
			name: "unknown fails",
			value: govulncheckExceptionReconciliation{
				Unknown: []govulncheckUnknownFindingSummary{{
					GoModPath:   "go.mod",
					Occurrences: 2,
				}},
			},
			want: "unknown-level findings",
		},
		{
			name: "unexcepted fails",
			value: govulncheckExceptionReconciliation{
				Unexcepted: []govulncheckFindingOccurrence{finding},
			},
			want: "without exact governed exceptions",
		},
		{
			name: "unused fails",
			value: govulncheckExceptionReconciliation{
				Unused: []govulncheckException{exception},
			},
			want: "unused or unmatched records",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := evaluateGovulncheckExceptionReconciliation(
				testCase.value,
			)
			if testCase.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil ||
				!strings.Contains(err.Error(), testCase.want) {
				t.Fatalf(
					"error = %v, want %q",
					err,
					testCase.want,
				)
			}
		})
	}
}

func TestExecuteGovulncheckRuntimeUsesExactExceptionPolicy(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(t.TempDir(), "tool-versions.json")
	selected := goToolchainSelection{
		Executable: "/selected/go/bin/go",
		Directory:  "/selected/go/bin",
		Actual:     "go1.26.5",
		Modules: []goModuleSelection{{
			GoModPath:  "go.mod",
			Directory:  ".",
			ModulePath: "example.com/project",
		}},
	}
	tool := govulncheckToolIdentity{
		Executable: filepath.Join(
			root,
			filepath.FromSlash(govulncheckRuntimeExecutable),
		),
		CommandPackage: govulncheckCommandPackage,
		Module:         govulncheckModuleRoot,
		Version:        "v1.6.0",
		BuildGoVersion: "go1.26.5",
		SHA256:         strings.Repeat("b", 64),
	}
	run := govulncheckModuleRun{
		Tool: tool,
		Modules: []govulncheckModuleScanResult{{
			GoModPath:    "go.mod",
			Directory:    ".",
			ModulePath:   "example.com/project",
			PackageScope: "./...",
			Protocol: govulncheckProtocolSummary{
				Config: govulncheckProtocolConfig{
					ProtocolVersion: "1.0.0",
					ScannerName:     "govulncheck",
					ScannerVersion:  "v1.6.0",
					GoVersion:       "go1.26.5",
					ScanLevel:       "symbol",
					ScanMode:        "source",
				},
				MessageCount:        2,
				ConfigMessages:      1,
				SymbolLevelFindings: 1,
				ReachableFindings: []govulncheckReachableFinding{{
					AdvisoryID:   "GO-2026-9999",
					ModulePath:   "example.com/dep",
					PackagePath:  "example.com/dep/pkg",
					Symbol:       "Open",
					FixedVersion: "v1.2.4",
				}},
			},
		}},
	}
	scannerEvidence, err := projectGovulncheckEvidence(
		run,
		selected.Modules,
	)
	if err != nil {
		t.Fatal(err)
	}
	exception := exactExceptionForFinding(
		"GO-2026-9999",
		"go.mod",
		"example.com/dep",
		"example.com/dep/pkg",
		"Open",
	)
	now := time.Date(2026, 7, 21, 11, 0, 0, 0, time.UTC)

	dependencies := govulncheckRuntimeDependencies{
		selectGo: func(string) (goToolchainSelection, error) {
			return selected, nil
		},
		verifyTool: func(
			context.Context,
			string,
			string,
			string,
		) (govulncheckToolIdentity, error) {
			return tool, nil
		},
		runModules: func(
			context.Context,
			string,
			goToolchainSelection,
			govulncheckToolIdentity,
		) (govulncheckModuleRun, error) {
			return run, nil
		},
		projectEvidence: func(
			govulncheckModuleRun,
			[]goModuleSelection,
		) (GovulncheckEvidence, error) {
			return scannerEvidence, nil
		},
		loadExceptions: func(
			string,
			time.Time,
		) (govulncheckExceptionSource, error) {
			return govulncheckExceptionSource{
				Present:     true,
				Path:        govulncheckExceptionRelativePath,
				SHA256:      strings.Repeat("c", 64),
				EvaluatedAt: now,
				Document: govulncheckExceptionDocument{
					SchemaVersion: 1,
					Exceptions: []govulncheckException{
						exception,
					},
				},
			}, nil
		},
		reconcile:         reconcileGovulncheckExceptions,
		projectExceptions: projectGovulncheckExceptionEvidence,
		now:               func() time.Time { return now },
	}

	result, err := executeGovulncheckRuntimeWithDependencies(
		context.Background(),
		root,
		config,
		dependencies,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Evidence.Exceptions == nil ||
		len(result.Evidence.Exceptions.Used) != 1 ||
		len(result.Evidence.Exceptions.Unexcepted) != 0 {
		t.Fatalf("unexpected runtime evidence: %+v", result)
	}

	dependencies.loadExceptions = func(
		string,
		time.Time,
	) (govulncheckExceptionSource, error) {
		mismatch := exception
		mismatch.Scope.Symbol = "Other"
		return govulncheckExceptionSource{
			Present:     true,
			Path:        govulncheckExceptionRelativePath,
			SHA256:      strings.Repeat("d", 64),
			EvaluatedAt: now,
			Document: govulncheckExceptionDocument{
				SchemaVersion: 1,
				Exceptions:    []govulncheckException{mismatch},
			},
		}, nil
	}

	result, err = executeGovulncheckRuntimeWithDependencies(
		context.Background(),
		root,
		config,
		dependencies,
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"without exact governed exceptions",
		) {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	if result.Evidence.Exceptions == nil ||
		len(result.Evidence.Exceptions.Unexcepted) != 1 ||
		len(result.Evidence.Exceptions.Unused) != 1 {
		t.Fatalf("failure evidence missing reconciliation: %+v", result)
	}
}

func TestGovulncheckExceptionEvidenceJSONAndText(t *testing.T) {
	now := time.Date(2026, 7, 21, 11, 0, 0, 0, time.UTC)
	exception := exactExceptionForFinding(
		"GO-2026-9999",
		"go.mod",
		"example.com/dep",
		"example.com/dep/pkg",
		"Open",
	)
	evidence, err := projectGovulncheckExceptionEvidence(
		govulncheckExceptionSource{
			Present:     true,
			Path:        govulncheckExceptionRelativePath,
			SHA256:      strings.Repeat("e", 64),
			EvaluatedAt: now,
			Document: govulncheckExceptionDocument{
				SchemaVersion: 1,
				Exceptions:    []govulncheckException{exception},
			},
		},
		govulncheckExceptionReconciliation{
			Used: []govulncheckUsedException{{
				Exception: exception,
				Finding: govulncheckFindingOccurrence{
					AdvisoryID:  exception.AdvisoryID,
					GoModPath:   exception.Scope.GoModPath,
					ModulePath:  exception.Scope.ModulePath,
					PackagePath: exception.Scope.PackagePath,
					Symbol:      exception.Scope.Symbol,
					Occurrences: 1,
				},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{
		SchemaVersion: 2,
		Govulncheck: &GovulncheckEvidence{
			Exceptions: &evidence,
		},
		Stdout: emptyStreamEvidence(),
		Stderr: emptyStreamEvidence(),
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(
		string(data),
		`"exceptions":{"present":true`,
	) {
		t.Fatalf("JSON evidence = %s", data)
	}
	text := string(renderText(result))
	for _, marker := range []string{
		"Govulncheck exception document present: true",
		"Govulncheck used exception count: 1",
		"Govulncheck used exception 1 advisory: GO-2026-9999",
		"Govulncheck used exception 1 approval record: TEST-1",
	} {
		if !strings.Contains(text, marker) {
			t.Fatalf("text missing %q:\n%s", marker, text)
		}
	}
}

func TestGovulncheckExceptionEvidenceSchemaIsSynchronized(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate exception evidence test")
	}
	root := filepath.Clean(
		filepath.Join(filepath.Dir(current), "..", ".."),
	)
	schemaData, err := os.ReadFile(
		filepath.Join(
			root,
			"schemas",
			"isras-project-command-execution-v2.schema.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatal(err)
	}
	defs := schema["$defs"].(map[string]any)
	for _, name := range []string{
		"govulncheck_exceptions",
		"govulncheck_exception_record_evidence",
		"govulncheck_finding_occurrence_evidence",
		"govulncheck_used_exception_evidence",
	} {
		if _, ok := defs[name].(map[string]any); !ok {
			t.Fatalf("schema definition %q is missing", name)
		}
	}

	exampleData, err := os.ReadFile(
		filepath.Join(
			root,
			"schemas",
			"examples",
			"isras-project-command-execution-v2-govulncheck-pass.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	var example map[string]any
	if err := json.Unmarshal(exampleData, &example); err != nil {
		t.Fatal(err)
	}
	scanner := example["govulncheck"].(map[string]any)
	exceptions := scanner["exceptions"].(map[string]any)
	if exceptions["present"] != false ||
		exceptions["path"] != govulncheckExceptionRelativePath {
		t.Fatalf("unexpected example exception evidence: %#v", exceptions)
	}
}
