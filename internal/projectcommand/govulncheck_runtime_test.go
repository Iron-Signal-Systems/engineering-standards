package projectcommand

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteGovulncheckRuntimeOrchestratesExactBoundaries(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(t.TempDir(), "tool-versions.json")
	selected := goToolchainSelection{
		Executable: "/selected/go/bin/go",
		Directory:  "/selected/go/bin",
		Actual:     "go1.26.5-X:nodwarf5",
		Modules: []goModuleSelection{
			{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"},
			{GoModPath: "tools/a/go.mod", Directory: "tools/a", ModulePath: "example.com/a"},
		},
	}
	tool := govulncheckToolIdentity{
		Executable:     filepath.Join(root, filepath.FromSlash(govulncheckRuntimeExecutable)),
		Directory:      filepath.Join(root, ".local", "tools", "bin"),
		CommandPackage: govulncheckCommandPackage,
		Module:         govulncheckModuleRoot,
		Version:        "v1.6.0",
		BuildGoVersion: "go1.26.5-X:nodwarf5",
		SHA256:         strings.Repeat("a", 64),
	}
	run := govulncheckModuleRun{
		Tool: tool,
		Modules: []govulncheckModuleScanResult{
			{
				GoModPath:    "go.mod",
				Directory:    ".",
				ModulePath:   "example.com/root",
				PackageScope: "./...",
				Protocol: govulncheckProtocolSummary{
					Config: govulncheckProtocolConfig{
						ProtocolVersion: "1.0.0",
						ScannerName:     "govulncheck",
						ScannerVersion:  "v1.6.0",
						GoVersion:       "go1.26.5-X:nodwarf5",
						ScanLevel:       "symbol",
						ScanMode:        "source",
					},
					MessageCount:         2,
					ConfigMessages:       1,
					ModuleLevelFindings:  1,
					PackageLevelFindings: 1,
				},
			},
			{
				GoModPath:    "tools/a/go.mod",
				Directory:    "tools/a",
				ModulePath:   "example.com/a",
				PackageScope: "./...",
				Protocol: govulncheckProtocolSummary{
					Config: govulncheckProtocolConfig{
						ProtocolVersion: "1.0.0",
						ScannerName:     "govulncheck",
						ScannerVersion:  "v1.6.0",
						GoVersion:       "go1.26.5-X:nodwarf5",
						ScanLevel:       "symbol",
						ScanMode:        "source",
					},
					MessageCount:   1,
					ConfigMessages: 1,
				},
			},
		},
	}
	evidence, err := projectGovulncheckEvidence(run, selected.Modules)
	if err != nil {
		t.Fatal(err)
	}

	var selectedRoot string
	var verifyGo, verifyTool, verifyConfig string
	var runRoot string
	dependencies := govulncheckRuntimeDependencies{
		selectGo: func(observedRoot string) (goToolchainSelection, error) {
			selectedRoot = observedRoot
			return selected, nil
		},
		verifyTool: func(
			_ context.Context,
			goExecutable string,
			toolExecutable string,
			configuration string,
		) (govulncheckToolIdentity, error) {
			verifyGo = goExecutable
			verifyTool = toolExecutable
			verifyConfig = configuration
			return tool, nil
		},
		runModules: func(
			_ context.Context,
			observedRoot string,
			observedSelection goToolchainSelection,
			observedTool govulncheckToolIdentity,
		) (govulncheckModuleRun, error) {
			runRoot = observedRoot
			if observedSelection.Executable != selected.Executable {
				t.Fatalf("selected Go drifted: %+v", observedSelection)
			}
			if observedTool.Executable != tool.Executable {
				t.Fatalf("tool drifted: %+v", observedTool)
			}
			return run, nil
		},
		projectEvidence: func(
			observedRun govulncheckModuleRun,
			modules []goModuleSelection,
		) (GovulncheckEvidence, error) {
			if len(observedRun.Modules) != 2 || len(modules) != 2 {
				t.Fatalf("coverage drift: run=%d modules=%d", len(observedRun.Modules), len(modules))
			}
			return evidence, nil
		},
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

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if selectedRoot != absoluteRoot || runRoot != absoluteRoot {
		t.Fatalf("root propagation: select=%q run=%q want=%q", selectedRoot, runRoot, absoluteRoot)
	}
	if verifyGo != selected.Executable {
		t.Fatalf("verify selected Go = %q", verifyGo)
	}
	if verifyTool != filepath.Join(absoluteRoot, filepath.FromSlash(govulncheckRuntimeExecutable)) {
		t.Fatalf("verify tool path = %q", verifyTool)
	}
	if verifyConfig != config {
		t.Fatalf("verify configuration = %q", verifyConfig)
	}
	if len(result.Evidence.Modules) != 2 || result.Tool.Version != "v1.6.0" {
		t.Fatalf("unexpected runtime result: %+v", result)
	}
}

func TestExecuteGovulncheckRuntimePropagatesBoundaryFailures(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(t.TempDir(), "tool-versions.json")
	selected := goToolchainSelection{
		Executable: "/selected/go",
		Modules: []goModuleSelection{{
			GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root",
		}},
	}
	tool := govulncheckToolIdentity{Executable: filepath.Join(root, filepath.FromSlash(govulncheckRuntimeExecutable))}
	run := govulncheckModuleRun{Tool: tool}

	tests := []struct {
		name         string
		dependencies govulncheckRuntimeDependencies
		want         string
	}{
		{
			name: "selection",
			dependencies: govulncheckRuntimeDependencies{
				selectGo: func(string) (goToolchainSelection, error) {
					return selected, errors.New("selection failed")
				},
				verifyTool: func(context.Context, string, string, string) (govulncheckToolIdentity, error) {
					t.Fatal("verify ran after selection failure")
					return tool, nil
				},
				runModules: func(context.Context, string, goToolchainSelection, govulncheckToolIdentity) (govulncheckModuleRun, error) {
					t.Fatal("runner ran after selection failure")
					return run, nil
				},
				projectEvidence: projectGovulncheckEvidence,
			},
			want: "select govulncheck Go toolchain",
		},
		{
			name: "verification",
			dependencies: govulncheckRuntimeDependencies{
				selectGo: func(string) (goToolchainSelection, error) { return selected, nil },
				verifyTool: func(context.Context, string, string, string) (govulncheckToolIdentity, error) {
					return tool, errors.New("verification failed")
				},
				runModules: func(context.Context, string, goToolchainSelection, govulncheckToolIdentity) (govulncheckModuleRun, error) {
					t.Fatal("runner ran after verification failure")
					return run, nil
				},
				projectEvidence: projectGovulncheckEvidence,
			},
			want: "verify governed govulncheck tool",
		},
		{
			name: "runner",
			dependencies: govulncheckRuntimeDependencies{
				selectGo:   func(string) (goToolchainSelection, error) { return selected, nil },
				verifyTool: func(context.Context, string, string, string) (govulncheckToolIdentity, error) { return tool, nil },
				runModules: func(context.Context, string, goToolchainSelection, govulncheckToolIdentity) (govulncheckModuleRun, error) {
					return run, errors.New("runner failed")
				},
				projectEvidence: func(govulncheckModuleRun, []goModuleSelection) (GovulncheckEvidence, error) {
					t.Fatal("projection ran after runner failure")
					return GovulncheckEvidence{}, nil
				},
			},
			want: "execute governed govulncheck modules",
		},
		{
			name: "projection",
			dependencies: govulncheckRuntimeDependencies{
				selectGo:   func(string) (goToolchainSelection, error) { return selected, nil },
				verifyTool: func(context.Context, string, string, string) (govulncheckToolIdentity, error) { return tool, nil },
				runModules: func(context.Context, string, goToolchainSelection, govulncheckToolIdentity) (govulncheckModuleRun, error) {
					return run, nil
				},
				projectEvidence: func(govulncheckModuleRun, []goModuleSelection) (GovulncheckEvidence, error) {
					return GovulncheckEvidence{}, errors.New("projection failed")
				},
			},
			want: "project governed govulncheck evidence",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := executeGovulncheckRuntimeWithDependencies(
				context.Background(),
				root,
				config,
				test.dependencies,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestEvaluateGovulncheckFindings(t *testing.T) {
	tests := []struct {
		name string
		run  govulncheckModuleRun
		want string
	}{
		{
			name: "module and package findings are recorded without reachable failure",
			run: govulncheckModuleRun{Modules: []govulncheckModuleScanResult{{
				GoModPath: "go.mod",
				Protocol: govulncheckProtocolSummary{
					ModuleLevelFindings:  2,
					PackageLevelFindings: 1,
					FindingAdvisoryIDs:   []string{"GO-2026-0001"},
				},
			}}},
		},
		{
			name: "reachable finding",
			run: govulncheckModuleRun{Modules: []govulncheckModuleScanResult{{
				GoModPath: "tools/a/go.mod",
				Protocol: govulncheckProtocolSummary{
					SymbolLevelFindings: 2,
					FindingAdvisoryIDs:  []string{"GO-2026-0002", "GO-2026-0001"},
				},
			}}},
			want: "reachable vulnerabilities without governed exceptions: tools/a/go.mod=2[GO-2026-0001,GO-2026-0002]",
		},
		{
			name: "unknown finding",
			run: govulncheckModuleRun{Modules: []govulncheckModuleScanResult{{
				GoModPath: "go.mod",
				Protocol: govulncheckProtocolSummary{
					UnknownLevelFindings: 1,
				},
			}}},
			want: "unknown-level findings: go.mod=1",
		},
		{
			name: "unknown and reachable",
			run: govulncheckModuleRun{Modules: []govulncheckModuleScanResult{
				{
					GoModPath: "z/go.mod",
					Protocol: govulncheckProtocolSummary{
						UnknownLevelFindings: 1,
					},
				},
				{
					GoModPath: "a/go.mod",
					Protocol: govulncheckProtocolSummary{
						SymbolLevelFindings: 1,
						FindingAdvisoryIDs:  []string{"GO-2026-0003"},
					},
				},
			}},
			want: "unknown-level findings: z/go.mod=1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := evaluateGovulncheckFindings(test.run)
			if test.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestExecuteGovulncheckRuntimeRejectsRelativeConfiguration(t *testing.T) {
	_, err := executeGovulncheckRuntimeWithDependencies(
		context.Background(),
		t.TempDir(),
		"validation/tool-versions.json",
		govulncheckRuntimeDependencies{
			selectGo: func(string) (goToolchainSelection, error) {
				t.Fatal("selection ran for relative configuration")
				return goToolchainSelection{}, nil
			},
			verifyTool: func(context.Context, string, string, string) (govulncheckToolIdentity, error) {
				return govulncheckToolIdentity{}, nil
			},
			runModules: func(context.Context, string, goToolchainSelection, govulncheckToolIdentity) (govulncheckModuleRun, error) {
				return govulncheckModuleRun{}, nil
			},
			projectEvidence: projectGovulncheckEvidence,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("error = %v", err)
	}
}
