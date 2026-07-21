package projectcommand

import (
	"strings"
	"testing"
	"time"
)

func TestProjectGovulncheckEvidenceProjectsEveryGovernedModule(t *testing.T) {
	started := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Second)
	run := govulncheckModuleRun{
		Tool: govulncheckToolIdentity{
			Executable:     "/repo/.local/tools/bin/govulncheck",
			Directory:      "/repo/.local/tools/bin",
			CommandPackage: "golang.org/x/vuln/cmd/govulncheck",
			Module:         "golang.org/x/vuln",
			Version:        "v1.6.0",
			BuildGoVersion: "go1.26.5",
			SHA256:         strings.Repeat("a", 64),
		},
		Modules: []govulncheckModuleScanResult{
			{
				GoModPath:            "tools/a/go.mod",
				Directory:            "tools/a",
				ModulePath:           "example.com/a",
				PackageScope:         "./...",
				EnvironmentNames:     []string{"GOENV", "GOTOOLCHAIN", "PATH"},
				Started:              started,
				Finished:             finished,
				DurationMilliseconds: 2000,
				ExitCode:             0,
				Protocol: govulncheckProtocolSummary{
					Config: govulncheckProtocolConfig{
						ProtocolVersion: "1.0.0",
						ScannerName:     "govulncheck",
						ScannerVersion:  "v1.6.0",
						ScanLevel:       "symbol",
						ScanMode:        "source",
					},
					MessageCount:        2,
					ConfigMessages:      1,
					SBOMMessages:        1,
					SBOMRoots:           []string{"example.com/a"},
					SBOMModules:         []govulncheckProtocolModule{{Path: "example.com/a"}},
					OSVAdvisoryIDs:      []string{"GO-2026-0001"},
					FindingAdvisoryIDs:  []string{"GO-2026-0001"},
					SymbolLevelFindings: 1,
				},
			},
			{
				GoModPath:            "go.mod",
				Directory:            ".",
				ModulePath:           "example.com/root",
				PackageScope:         "./...",
				EnvironmentNames:     []string{"GOENV", "GOTOOLCHAIN", "PATH"},
				Started:              started,
				Finished:             finished,
				DurationMilliseconds: 2000,
				ExitCode:             0,
				Protocol: govulncheckProtocolSummary{
					Config: govulncheckProtocolConfig{
						ProtocolVersion: "1.0.0",
						ScannerName:     "govulncheck",
						ScannerVersion:  "v1.6.0",
						ScanLevel:       "symbol",
						ScanMode:        "source",
					},
					MessageCount:   1,
					ConfigMessages: 1,
				},
			},
		},
	}
	expected := []goModuleSelection{
		{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"},
		{GoModPath: "tools/a/go.mod", Directory: "tools/a", ModulePath: "example.com/a"},
	}

	evidence, err := projectGovulncheckEvidence(run, expected)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Version != "v1.6.0" || evidence.GOTOOLCHAINEffective != "local" || evidence.GOENVEffective != "off" {
		t.Fatalf("unexpected tool evidence: %+v", evidence)
	}
	if len(evidence.Modules) != 2 {
		t.Fatalf("module evidence count = %d", len(evidence.Modules))
	}
	if evidence.Modules[0].GoModPath != "go.mod" || evidence.Modules[1].GoModPath != "tools/a/go.mod" {
		t.Fatalf("module order = %#v", []string{evidence.Modules[0].GoModPath, evidence.Modules[1].GoModPath})
	}
	if evidence.Modules[1].Protocol.SymbolLevelFindings != 1 {
		t.Fatalf("symbol findings = %d", evidence.Modules[1].Protocol.SymbolLevelFindings)
	}

	run.Modules[0].EnvironmentNames[0] = "MUTATED"
	run.Modules[0].Protocol.SBOMRoots[0] = "mutated"
	if evidence.Modules[1].EnvironmentNames[0] == "MUTATED" {
		t.Fatal("environment names were not cloned")
	}
	if evidence.Modules[1].Protocol.SBOMRoots[0] == "mutated" {
		t.Fatal("protocol roots were not cloned")
	}
}

func TestProjectGovulncheckEvidenceRejectsCoverageDrift(t *testing.T) {
	baseRun := govulncheckModuleRun{
		Tool: govulncheckToolIdentity{Executable: "/tool/govulncheck"},
		Modules: []govulncheckModuleScanResult{{
			GoModPath:    "go.mod",
			Directory:    ".",
			ModulePath:   "example.com/root",
			PackageScope: "./...",
		}},
	}
	expected := []goModuleSelection{{
		GoModPath:  "go.mod",
		Directory:  ".",
		ModulePath: "example.com/root",
	}}

	cases := []struct {
		name     string
		run      govulncheckModuleRun
		expected []goModuleSelection
		contains string
	}{
		{
			name:     "empty expected inventory",
			run:      baseRun,
			expected: nil,
			contains: "at least one governed Go module",
		},
		{
			name:     "count mismatch",
			run:      govulncheckModuleRun{Tool: baseRun.Tool},
			expected: expected,
			contains: "module count",
		},
		{
			name: "ungoverned result",
			run: govulncheckModuleRun{
				Tool: baseRun.Tool,
				Modules: []govulncheckModuleScanResult{{
					GoModPath: "other/go.mod", Directory: "other", ModulePath: "example.com/other", PackageScope: "./...",
				}},
			},
			expected: expected,
			contains: "ungoverned module",
		},
		{
			name: "directory mismatch",
			run: govulncheckModuleRun{
				Tool: baseRun.Tool,
				Modules: []govulncheckModuleScanResult{{
					GoModPath: "go.mod", Directory: "wrong", ModulePath: "example.com/root", PackageScope: "./...",
				}},
			},
			expected: expected,
			contains: "does not match governed directory",
		},
		{
			name: "module path mismatch",
			run: govulncheckModuleRun{
				Tool: baseRun.Tool,
				Modules: []govulncheckModuleScanResult{{
					GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/wrong", PackageScope: "./...",
				}},
			},
			expected: expected,
			contains: "does not match governed module path",
		},
		{
			name: "package scope mismatch",
			run: govulncheckModuleRun{
				Tool: baseRun.Tool,
				Modules: []govulncheckModuleScanResult{{
					GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root", PackageScope: "./cmd/...",
				}},
			},
			expected: expected,
			contains: "must be ./...",
		},
		{
			name: "duplicate expected path",
			run: govulncheckModuleRun{
				Tool: baseRun.Tool,
				Modules: []govulncheckModuleScanResult{
					baseRun.Modules[0],
					baseRun.Modules[0],
				},
			},
			expected: append(expected, expected[0]),
			contains: "duplicate path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := projectGovulncheckEvidence(tc.run, tc.expected)
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("error = %v, want substring %q", err, tc.contains)
			}
		})
	}
}
