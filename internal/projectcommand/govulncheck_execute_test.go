package projectcommand

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

func TestExecuteDispatchesKnownVulnerabilitiesThroughRuntime(t *testing.T) {
	fixture := newFixture(t, "govulncheck-dispatch", "#!/bin/sh\nexit 99\n")
	fixture.setCommand(t, "known_vulnerabilities", []string{
		projectpin.GovulncheckExecutable,
		projectpin.GovulncheckPackageScope,
	})
	fixture.commit(t, "baseline")
	stageGovulncheckRuntimeConfiguration(t, fixture.root, []byte(`{"version":1,"tools":{"govulncheck":{"module":"golang.org/x/vuln/cmd/govulncheck","version":"v1.6.0"}}}`+"\n"))

	selected := goToolchainSelection{
		Executable:       "/selected/go/bin/go",
		Directory:        "/selected/go/bin",
		Minimum:          "go1.25.12",
		Actual:           "go1.26.5-X:nodwarf5",
		MinimumSatisfied: true,
		Modules: []goModuleSelection{{
			GoModPath:        "go.mod",
			Directory:        ".",
			ModulePath:       "github.com/Iron-Signal-Systems/govulncheck-dispatch",
			Minimum:          "go1.25.12",
			MinimumSatisfied: true,
		}},
	}
	toolPath := filepath.Join(fixture.root, filepath.FromSlash(govulncheckRuntimeExecutable))
	empty := emptyStreamEvidence()
	evidence := GovulncheckEvidence{
		Executable:             toolPath,
		Directory:              filepath.Dir(toolPath),
		ApprovedCommandPackage: govulncheckCommandPackage,
		EmbeddedModule:         govulncheckModuleRoot,
		Version:                "v1.6.0",
		BuildGoVersion:         "go1.26.5-X:nodwarf5",
		SHA256:                 strings.Repeat("a", 64),
		PackageScope:           "./...",
		GOTOOLCHAINEffective:   "local",
		GOENVEffective:         "off",
		Modules: []GovulncheckModuleEvidence{{
			GoModPath:        "go.mod",
			Directory:        ".",
			ModulePath:       "github.com/Iron-Signal-Systems/govulncheck-dispatch",
			PackageScope:     "./...",
			EnvironmentNames: []string{"GOENV", "GOTOOLCHAIN", "PATH"},
			Started:          "2026-07-21T10:00:00Z",
			Finished:         "2026-07-21T10:00:01Z",
			ExitCode:         0,
			Protocol: GovulncheckProtocolEvidence{
				ProtocolVersion: "1.0.0",
				ScannerName:     "govulncheck",
				ScannerVersion:  "v1.6.0",
				GoVersion:       "go1.26.5-X:nodwarf5",
				ScanLevel:       "symbol",
				ScanMode:        "source",
				MessageCount:    1,
				ConfigMessages:  1,
			},
			Stdout: empty,
			Stderr: empty,
		}},
	}
	run := govulncheckModuleRun{
		Tool: govulncheckToolIdentity{
			Executable:     toolPath,
			CommandPackage: govulncheckCommandPackage,
			Module:         govulncheckModuleRoot,
			Version:        "v1.6.0",
			BuildGoVersion: "go1.26.5-X:nodwarf5",
			SHA256:         strings.Repeat("a", 64),
		},
		Modules: []govulncheckModuleScanResult{{
			GoModPath:        "go.mod",
			Directory:        ".",
			ModulePath:       "github.com/Iron-Signal-Systems/govulncheck-dispatch",
			PackageScope:     "./...",
			EnvironmentNames: []string{"PATH", "GOENV", "GOTOOLCHAIN"},
			ExitCode:         0,
		}},
	}

	var called bool
	executor := func(_ context.Context, root, configuration string) (govulncheckRuntimeResult, error) {
		called = true
		if root != fixture.root {
			t.Fatalf("runtime root = %q, want %q", root, fixture.root)
		}
		wantConfiguration := filepath.Join(fixture.root, projectpin.RuntimeEvidenceDirectory, "runtime", govulncheckRuntimeConfigurationName)
		if configuration != wantConfiguration {
			t.Fatalf("runtime configuration = %q, want %q", configuration, wantConfiguration)
		}
		return govulncheckRuntimeResult{SelectedGo: selected, Tool: run.Tool, Run: run, Evidence: evidence}, nil
	}

	result, err := executeProjectCommand(context.Background(), fixture.request(t, "known_vulnerabilities"), executor)
	if err != nil {
		t.Fatal(err)
	}
	if !called || result.Status != "PASS" || result.Govulncheck == nil || result.GoToolchain == nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	assertEvidence(t, result)
	data, err := os.ReadFile(result.EvidenceJSON)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document["schema_version"] != float64(2) {
		t.Fatalf("schema version = %#v", document["schema_version"])
	}
	if _, ok := document["govulncheck"].(map[string]any); !ok {
		t.Fatalf("JSON evidence lacks govulncheck section: %s", data)
	}
}

func TestExecuteGovulncheckFailureFinalizesTypedEvidence(t *testing.T) {
	fixture := newFixture(t, "govulncheck-reachable", "#!/bin/sh\nexit 99\n")
	fixture.setCommand(t, "known_vulnerabilities", []string{
		projectpin.GovulncheckExecutable,
		projectpin.GovulncheckPackageScope,
	})
	fixture.commit(t, "baseline")
	stageGovulncheckRuntimeConfiguration(t, fixture.root, []byte(`{"version":1,"tools":{"govulncheck":{"module":"golang.org/x/vuln/cmd/govulncheck","version":"v1.6.0"}}}`+"\n"))

	toolPath := filepath.Join(fixture.root, filepath.FromSlash(govulncheckRuntimeExecutable))
	evidence := GovulncheckEvidence{
		Executable:             toolPath,
		Directory:              filepath.Dir(toolPath),
		ApprovedCommandPackage: govulncheckCommandPackage,
		EmbeddedModule:         govulncheckModuleRoot,
		Version:                "v1.6.0",
		BuildGoVersion:         "go1.26.5",
		SHA256:                 strings.Repeat("b", 64),
		PackageScope:           "./...",
		GOTOOLCHAINEffective:   "local",
		GOENVEffective:         "off",
		Modules: []GovulncheckModuleEvidence{{
			GoModPath:    "go.mod",
			Directory:    ".",
			ModulePath:   "github.com/Iron-Signal-Systems/govulncheck-reachable",
			PackageScope: "./...",
			Started:      "2026-07-21T10:00:00Z",
			Finished:     "2026-07-21T10:00:01Z",
			ExitCode:     0,
			Protocol: GovulncheckProtocolEvidence{
				ProtocolVersion:     "1.0.0",
				ScannerName:         "govulncheck",
				ScannerVersion:      "v1.6.0",
				GoVersion:           "go1.26.5",
				ScanLevel:           "symbol",
				ScanMode:            "source",
				MessageCount:        2,
				ConfigMessages:      1,
				FindingMessages:     1,
				FindingAdvisoryIDs:  []string{"GO-2026-9999"},
				SymbolLevelFindings: 1,
			},
			Stdout: emptyStreamEvidence(),
			Stderr: emptyStreamEvidence(),
		}},
	}
	run := govulncheckModuleRun{
		Tool: govulncheckToolIdentity{Executable: toolPath, CommandPackage: govulncheckCommandPackage, Module: govulncheckModuleRoot, Version: "v1.6.0", SHA256: strings.Repeat("b", 64)},
		Modules: []govulncheckModuleScanResult{{
			GoModPath: "go.mod", Directory: ".", ModulePath: "github.com/Iron-Signal-Systems/govulncheck-reachable", PackageScope: "./...", EnvironmentNames: []string{"GOENV", "GOTOOLCHAIN", "PATH"}, ExitCode: 0,
		}},
	}
	executor := func(context.Context, string, string) (govulncheckRuntimeResult, error) {
		return govulncheckRuntimeResult{Tool: run.Tool, Run: run, Evidence: evidence}, errors.New("govulncheck found reachable vulnerabilities without governed exceptions: go.mod=1[GO-2026-9999]")
	}

	result, err := executeProjectCommand(context.Background(), fixture.request(t, "known_vulnerabilities"), executor)
	if err == nil || !strings.Contains(err.Error(), "reachable vulnerabilities") {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	if result.Status != "FAIL" || result.ExitCode != 0 || result.Govulncheck == nil {
		t.Fatalf("unexpected failure evidence: %+v", result)
	}
	assertEvidence(t, result)
}

func TestExecuteNonVulnerabilityCommandDoesNotCallRuntime(t *testing.T) {
	fixture := newFixture(t, "ordinary-dispatch", "#!/bin/sh\nexit 0\n")
	fixture.commit(t, "baseline")
	result, err := executeProjectCommand(context.Background(), fixture.request(t, "test"), func(context.Context, string, string) (govulncheckRuntimeResult, error) {
		t.Fatal("govulncheck runtime called for ordinary command")
		return govulncheckRuntimeResult{}, nil
	})
	if err != nil || result.Status != "PASS" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestGovulncheckRuntimeConfigurationPathRejectsUnsafeBoundary(t *testing.T) {
	root := t.TempDir()
	for _, unsafe := range []string{"", ".", "../escape", "/absolute", `windows\path`} {
		if _, err := govulncheckRuntimeConfigurationPath(root, unsafe); err == nil {
			t.Fatalf("unsafe evidence directory accepted: %q", unsafe)
		}
	}
	runtimeDirectory := filepath.Join(root, ".local", "isras", "runtime")
	if err := os.MkdirAll(filepath.Dir(runtimeDirectory), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), runtimeDirectory); err != nil {
		t.Fatal(err)
	}
	_, err := govulncheckRuntimeConfigurationPath(root, ".local/isras")
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlink boundary error = %v", err)
	}
}

func TestExecuteGovulncheckLiveCandidate(t *testing.T) {
	if os.Getenv("ISRAS_RUN_LIVE_GOVULNCHECK_EXECUTE") != "1" {
		t.Skip("set ISRAS_RUN_LIVE_GOVULNCHECK_EXECUTE=1")
	}
	sourceTool := os.Getenv("ISRAS_LIVE_GOVULNCHECK")
	sourceConfiguration := os.Getenv("ISRAS_LIVE_TOOL_VERSIONS")
	if sourceTool == "" || sourceConfiguration == "" {
		t.Fatal("live scanner inputs are required")
	}

	fixture := newFixture(t, "govulncheck-live-execute", "#!/bin/sh\nexit 99\n")
	fixture.setCommand(t, "known_vulnerabilities", []string{projectpin.GovulncheckExecutable, projectpin.GovulncheckPackageScope})
	if err := os.WriteFile(filepath.Join(fixture.root, "live.go"), []byte("package livecandidate\n\nfunc Value() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.commit(t, "live candidate")

	copyFileForGovulncheckLiveTest(t, sourceTool, filepath.Join(fixture.root, filepath.FromSlash(govulncheckRuntimeExecutable)), 0o755)
	configuration, err := os.ReadFile(sourceConfiguration)
	if err != nil {
		t.Fatal(err)
	}
	stageGovulncheckRuntimeConfiguration(t, fixture.root, configuration)

	started := time.Now()
	result, err := Execute(context.Background(), fixture.request(t, "known_vulnerabilities"))
	t.Logf("live Execute duration=%s status=%s evidence=%s", time.Since(started), result.Status, result.EvidenceJSON)
	if err != nil {
		t.Fatalf("live Execute failed: result=%+v error=%v", result, err)
	}
	if result.Status != "PASS" || result.Govulncheck == nil || len(result.Govulncheck.Modules) != 1 {
		t.Fatalf("unexpected live evidence: %+v", result)
	}
	assertEvidence(t, result)
}

func stageGovulncheckRuntimeConfiguration(t *testing.T, root string, content []byte) {
	t.Helper()
	directory := filepath.Join(root, projectpin.RuntimeEvidenceDirectory, "runtime")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, govulncheckRuntimeConfigurationName), content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func copyFileForGovulncheckLiveTest(t *testing.T, source, destination string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, mode); err != nil {
		t.Fatal(err)
	}
}
