package projectcommand

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGoProfileEvidenceV2RecordsSelectedToolchain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX executable paths")
	}
	fixture := newFixture(t, "go-evidence-v2", "#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(filepath.Join(fixture.root, "go.mod"), []byte("module github.com/Iron-Signal-Systems/go-evidence-v2\n\ngo 1.25.12\ntoolchain default\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.commit(t, "declare evidence fixture")

	selectedBin := filepath.Join(t.TempDir(), "selected-go", "bin")
	if err := os.MkdirAll(selectedBin, 0o700); err != nil {
		t.Fatal(err)
	}
	selectedGo := filepath.Join(selectedBin, "go")
	writeFakeGo(t, selectedGo, "go1.26.5-X:nodwarf5")
	originalPath := os.Getenv("PATH")
	if originalPath == "" {
		t.Fatal("test requires a caller PATH")
	}
	t.Setenv("PATH", selectedBin+string(os.PathListSeparator)+originalPath)
	t.Setenv("GOTOOLCHAIN", "auto")
	t.Setenv("GOENV", filepath.Join(t.TempDir(), "caller-goenv"))

	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != 2 || result.GoToolchain == nil {
		t.Fatalf("unexpected v2 result: %+v", result)
	}
	goEvidence := result.GoToolchain
	checks := map[string]string{
		"selected executable": goEvidence.SelectedGoExecutable,
		"selected directory":  goEvidence.SelectedGoDirectory,
		"selected version":    goEvidence.SelectedGoVersion,
		"project minimum":     goEvidence.ProjectGoMinimum,
		"toolchain directive": goEvidence.ProjectToolchainDirective,
		"GOTOOLCHAIN":         goEvidence.GOTOOLCHAINEffective,
		"GOENV":               goEvidence.GOENVEffective,
	}
	wants := map[string]string{
		"selected executable": selectedGo,
		"selected directory":  selectedBin,
		"selected version":    "go1.26.5-X:nodwarf5",
		"project minimum":     "go1.25.12",
		"toolchain directive": "default",
		"GOTOOLCHAIN":         "local",
		"GOENV":               "off",
	}
	for name, value := range checks {
		if value != wants[name] {
			t.Fatalf("%s = %q, want %q", name, value, wants[name])
		}
	}
	if !goEvidence.GoMinimumSatisfied {
		t.Fatal("minimum satisfaction was not recorded")
	}

	data, err := os.ReadFile(result.EvidenceJSON)
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatal(err)
	}
	if encoded["schema_version"] != float64(2) {
		t.Fatalf("encoded schema version = %#v", encoded["schema_version"])
	}
	encodedGo, ok := encoded["go_toolchain"].(map[string]any)
	if !ok || encodedGo["selected_go_executable"] != selectedGo || encodedGo["go_minimum_satisfied"] != true {
		t.Fatalf("encoded Go evidence = %#v", encoded["go_toolchain"])
	}

	text, err := os.ReadFile(result.EvidenceText)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Selected Go executable: " + selectedGo,
		"Selected Go directory: " + selectedBin,
		"Selected Go version: go1.26.5-X:nodwarf5",
		"Project Go minimum: go1.25.12",
		"Project toolchain directive: default",
		"GOTOOLCHAIN effective: local",
		"GOENV effective: off",
		"Go minimum satisfied: true",
	} {
		if !strings.Contains(string(text), expected) {
			t.Fatalf("text evidence missing %q:\n%s", expected, text)
		}
	}
	assertEvidence(t, result)
}

func TestGoProfileEvidenceV2RetainsBelowMinimumFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX executable paths")
	}
	fixture := newFixture(t, "go-evidence-below-minimum", "#!/bin/sh\nprintf 'ran\\n' > command-ran\n")
	fixture.commit(t, "declare below-minimum fixture")
	selectedBin := filepath.Join(t.TempDir(), "old-go", "bin")
	if err := os.MkdirAll(selectedBin, 0o700); err != nil {
		t.Fatal(err)
	}
	selectedGo := filepath.Join(selectedBin, "go")
	writeFakeGo(t, selectedGo, "go1.24.13")
	originalPath := os.Getenv("PATH")
	if originalPath == "" {
		t.Fatal("test requires a caller PATH")
	}
	t.Setenv("PATH", selectedBin+string(os.PathListSeparator)+originalPath)

	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || !strings.Contains(err.Error(), "below project minimum") {
		t.Fatalf("expected below-minimum failure, got %v", err)
	}
	if result.GoToolchain == nil || result.GoToolchain.SelectedGoExecutable != selectedGo || result.GoToolchain.SelectedGoVersion != "go1.24.13" || result.GoToolchain.ProjectGoMinimum != "go1.25.12" || result.GoToolchain.GoMinimumSatisfied {
		t.Fatalf("unexpected negative Go evidence: %+v", result.GoToolchain)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.root, "command-ran")); !os.IsNotExist(statErr) {
		t.Fatal("project command ran despite below-minimum Go")
	}
	data, readErr := os.ReadFile(result.EvidenceJSON)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(data), `"go_minimum_satisfied": false`) {
		t.Fatalf("negative evidence missing false result:\n%s", data)
	}
	assertEvidence(t, result)
}

func TestProjectCommandEvidenceSchemaRevisionIsSynchronized(t *testing.T) {
	root := projectCommandRepositoryRoot(t)
	v1 := readJSONMap(t, filepath.Join(root, "schemas", "isras-project-command-execution-v1.schema.json"))
	v2 := readJSONMap(t, filepath.Join(root, "schemas", "isras-project-command-execution-v2.schema.json"))
	example := readJSONMap(t, filepath.Join(root, "schemas", "examples", "isras-project-command-execution-v2-go-pass.json"))
	if schemaVersionConst(t, v1) != float64(1) || schemaVersionConst(t, v2) != float64(2) || example["schema_version"] != float64(2) {
		t.Fatal("schema revision identities are not synchronized")
	}
	properties := jsonObject(t, v2["properties"], "v2 properties")
	if _, ok := properties["go_toolchain"]; !ok {
		t.Fatal("v2 schema is missing go_toolchain")
	}
	definitions := jsonObject(t, v2["$defs"], "v2 definitions")
	goDefinition := jsonObject(t, definitions["go_toolchain"], "go_toolchain definition")
	required := jsonStringSet(t, goDefinition["required"], "go_toolchain required")
	for _, key := range []string{"selected_go_executable", "selected_go_directory", "selected_go_version", "project_go_minimum", "GOTOOLCHAIN_effective", "GOENV_effective", "go_minimum_satisfied"} {
		if !required[key] {
			t.Fatalf("v2 schema does not require %q", key)
		}
	}
	exampleGo := jsonObject(t, example["go_toolchain"], "example go_toolchain")
	if exampleGo["GOTOOLCHAIN_effective"] != "local" || exampleGo["GOENV_effective"] != "off" || exampleGo["go_minimum_satisfied"] != true {
		t.Fatalf("v2 example drifted: %#v", exampleGo)
	}
}

func projectCommandRepositoryRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return value
}

func jsonObject(t *testing.T, value any, label string) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v", label, value)
	}
	return object
}

func jsonStringSet(t *testing.T, value any, label string) map[string]bool {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v", label, value)
	}
	result := make(map[string]bool, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("%s item = %#v", label, item)
		}
		result[text] = true
	}
	return result
}

func schemaVersionConst(t *testing.T, schema map[string]any) any {
	t.Helper()
	properties := jsonObject(t, schema["properties"], "schema properties")
	version := jsonObject(t, properties["schema_version"], "schema_version")
	return version["const"]
}
