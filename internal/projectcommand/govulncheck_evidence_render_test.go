package projectcommand

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGovulncheckEvidenceV2JSONAndTextRendering(t *testing.T) {
	empty := emptyStreamEvidence()
	result := Result{
		SchemaVersion:        2,
		RunID:                "0123456789abcdef01234567",
		Authorization:        "GRANTED",
		Status:               "PASS",
		Mode:                 "commit",
		CommandName:          "known_vulnerabilities",
		Arguments:            []string{"govulncheck", "./..."},
		ResolvedExecutable:   "/workspace/target/.local/tools/bin/govulncheck",
		WorkingDirectory:     "/workspace/target",
		EnvironmentNames:     []string{"GOENV", "GOTOOLCHAIN", "PATH"},
		TimeoutSeconds:       1200,
		OutputLimitBytes:     1048576,
		Started:              time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC),
		Finished:             time.Date(2026, 7, 21, 10, 0, 1, 0, time.UTC),
		DurationMilliseconds: 1000,
		ExitCode:             0,
		Validator: IdentityEvidence{
			Profile:          "ISRAS-SD",
			Version:          "0.1.5",
			ReleaseTag:       "isras-v0.1.5",
			SourceRepository: "github.com/Iron-Signal-Systems/engineering-standards",
			SourceCommit:     strings.Repeat("a", 40),
		},
		Target: TargetEvidence{
			Repository: "github.com/Iron-Signal-Systems/example-project",
			Root:       "/workspace/target",
			Commit:     strings.Repeat("b", 40),
			Branch:     "dev",
			Origin:     "git@github.com:Iron-Signal-Systems/example-project.git",
		},
		Govulncheck: &GovulncheckEvidence{
			Executable:             "/workspace/target/.local/tools/bin/govulncheck",
			Directory:              "/workspace/target/.local/tools/bin",
			ApprovedCommandPackage: "golang.org/x/vuln/cmd/govulncheck",
			EmbeddedModule:         "golang.org/x/vuln",
			Version:                "v1.6.0",
			BuildGoVersion:         "go1.26.5-X:nodwarf5",
			SHA256:                 strings.Repeat("c", 64),
			PackageScope:           "./...",
			GOTOOLCHAINEffective:   "local",
			GOENVEffective:         "off",
			Modules: []GovulncheckModuleEvidence{{
				GoModPath:            "go.mod",
				Directory:            ".",
				ModulePath:           "github.com/Iron-Signal-Systems/example-project",
				PackageScope:         "./...",
				EnvironmentNames:     []string{"GOCACHE", "GOENV", "GOPATH", "GOTOOLCHAIN", "HOME", "PATH", "TMPDIR", "XDG_CACHE_HOME"},
				Started:              "2026-07-21T10:00:00Z",
				Finished:             "2026-07-21T10:00:01Z",
				DurationMilliseconds: 1000,
				ExitCode:             0,
				Protocol: GovulncheckProtocolEvidence{
					ProtocolVersion:      "1.0.0",
					ScannerName:          "govulncheck",
					ScannerVersion:       "v1.6.0",
					Database:             "https://vuln.go.dev",
					DatabaseLastModified: "2026-07-21T09:00:00Z",
					GoVersion:            "go1.26.5-X:nodwarf5",
					ScanLevel:            "symbol",
					ScanMode:             "source",
					MessageCount:         3,
					ConfigMessages:       1,
					ProgressMessages:     1,
					SBOMMessages:         1,
					SBOMRoots:            []string{"github.com/Iron-Signal-Systems/example-project"},
					SBOMModules:          []GovulncheckProtocolModule{{Path: "github.com/Iron-Signal-Systems/example-project"}},
				},
				Stdout: empty,
				Stderr: empty,
			}},
		},
		Stdout: empty,
		Stderr: empty,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatal(err)
	}
	scanner, ok := encoded["govulncheck"].(map[string]any)
	if !ok || scanner["version"] != "v1.6.0" || scanner["GOTOOLCHAIN_effective"] != "local" {
		t.Fatalf("encoded scanner evidence = %#v", encoded["govulncheck"])
	}
	text := string(renderText(result))
	for _, expected := range []string{
		"Govulncheck executable: /workspace/target/.local/tools/bin/govulncheck",
		"Govulncheck approved command package: golang.org/x/vuln/cmd/govulncheck",
		"Govulncheck version: v1.6.0",
		"Govulncheck module count: 1",
		"Govulncheck module 1 go.mod: go.mod",
		"Govulncheck module 1 protocol version: 1.0.0",
		"Govulncheck module 1 scan level: symbol",
		"Govulncheck module 1 symbol-level findings: 0",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("text evidence missing %q:\n%s", expected, text)
		}
	}
}

func TestGovulncheckEvidenceSchemaV2IsSynchronized(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	schemaData, err := os.ReadFile(filepath.Join(root, "schemas", "isras-project-command-execution-v2.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	govulncheck := properties["govulncheck"].(map[string]any)
	if govulncheck["$ref"] != "#/$defs/govulncheck" {
		t.Fatalf("schema govulncheck property = %#v", govulncheck)
	}
	defs := schema["$defs"].(map[string]any)
	for _, name := range []string{"govulncheck", "govulncheck_module", "govulncheck_protocol", "govulncheck_protocol_module"} {
		if _, ok := defs[name]; !ok {
			t.Fatalf("schema definition %q missing", name)
		}
	}
	exampleData, err := os.ReadFile(filepath.Join(root, "schemas", "examples", "isras-project-command-execution-v2-govulncheck-pass.json"))
	if err != nil {
		t.Fatal(err)
	}
	var example map[string]any
	if err := json.Unmarshal(exampleData, &example); err != nil {
		t.Fatal(err)
	}
	if example["command_name"] != "known_vulnerabilities" || example["govulncheck"] == nil {
		t.Fatalf("unexpected governed example: %#v", example)
	}
}
