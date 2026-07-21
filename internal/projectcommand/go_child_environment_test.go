package projectcommand

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGoProfileChildEnvironmentForcesLocalToolchainAndSelectedGo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell script")
	}

	fixture := newFixture(t, "go-child-environment", `#!/bin/sh
printf 'GOTOOLCHAIN=%s\n' "$GOTOOLCHAIN"
printf 'GOENV=%s\n' "$GOENV"
printf 'go_path=%s\n' "$(command -v go)"
printf 'go_version=%s\n' "$(go env GOVERSION)"
`)

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

	fixture.commit(t, "declare Go child environment fixture")

	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{
		"GOTOOLCHAIN=local",
		"GOENV=off",
		"go_path=" + selectedGo,
		"go_version=go1.26.5-X:nodwarf5",
	} {
		if !strings.Contains(result.Stdout.Sanitized, expected) {
			t.Fatalf("child output missing %q:\n%s", expected, result.Stdout.Sanitized)
		}
	}

	names := make(map[string]bool, len(result.EnvironmentNames))
	for _, name := range result.EnvironmentNames {
		names[name] = true
	}
	if !names["GOTOOLCHAIN"] || !names["GOENV"] {
		t.Fatalf("fixed Go environment names missing from evidence: %v", result.EnvironmentNames)
	}

	assertEvidence(t, result)
}

func TestNonGoCommandEnvironmentPreservesInheritedGoVariables(t *testing.T) {
	runDirectory := t.TempDir()
	t.Setenv("GOTOOLCHAIN", "auto")
	t.Setenv("GOENV", "/caller/goenv")

	environment, names, err := commandEnvironment(runDirectory, "/bin/sh", Request{})
	if err != nil {
		t.Fatal(err)
	}

	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			t.Fatalf("malformed environment entry %q", entry)
		}
		values[name] = value
	}

	if values["GOTOOLCHAIN"] != "auto" {
		t.Fatalf("non-Go GOTOOLCHAIN = %q, want caller value", values["GOTOOLCHAIN"])
	}
	if values["GOENV"] != "/caller/goenv" {
		t.Fatalf("non-Go GOENV = %q, want caller value", values["GOENV"])
	}

	nameSet := make(map[string]bool, len(names))
	for _, name := range names {
		nameSet[name] = true
	}
	if !nameSet["GOTOOLCHAIN"] || !nameSet["GOENV"] {
		t.Fatalf("non-Go inherited names missing: %v", names)
	}
}
