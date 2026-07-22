package projectcommand

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernedGovulncheckExecutableUsesRunnerOwnedPath(t *testing.T) {
	root := t.TempDir()
	toolRoot := t.TempDir()
	tool := filepath.Join(toolRoot, "govulncheck")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(govulncheckExecutableEnvironmentName, tool)

	got, err := governedGovulncheckExecutable(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != tool {
		t.Fatalf("resolved tool = %q, want %q", got, tool)
	}
}

func TestGovernedGovulncheckExecutableRejectsTargetOwnedPath(t *testing.T) {
	root := t.TempDir()
	tool := filepath.Join(root, ".local", "tools", "bin", "govulncheck")
	if err := os.MkdirAll(filepath.Dir(tool), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(govulncheckExecutableEnvironmentName, tool)

	_, err := governedGovulncheckExecutable(root)
	if err == nil || !strings.Contains(err.Error(), "outside the target repository") {
		t.Fatalf("error = %v, want outside-target rejection", err)
	}
}

func TestGovernedGovulncheckExecutableRejectsRelativePath(t *testing.T) {
	t.Setenv(govulncheckExecutableEnvironmentName, "tools/govulncheck")

	_, err := governedGovulncheckExecutable(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "clean absolute path") {
		t.Fatalf("error = %v, want absolute-path rejection", err)
	}
}

func TestGovernedGovulncheckExecutableRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	toolRoot := t.TempDir()
	target := filepath.Join(toolRoot, "govulncheck-real")
	link := filepath.Join(toolRoot, "govulncheck")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv(govulncheckExecutableEnvironmentName, link)

	_, err := governedGovulncheckExecutable(root)
	if err == nil || !strings.Contains(err.Error(), "regular executable file") {
		t.Fatalf("error = %v, want symlink rejection", err)
	}
}

func TestGovernedGovulncheckExecutableRetainsLocalCompatibilityFallback(t *testing.T) {
	root := t.TempDir()
	t.Setenv(govulncheckExecutableEnvironmentName, "")

	got, err := governedGovulncheckExecutable(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, filepath.FromSlash(govulncheckRuntimeExecutable))
	if got != want {
		t.Fatalf("resolved tool = %q, want %q", got, want)
	}
}

func TestHostedWorkflowKeepsGovulncheckOutsideTargetRepository(t *testing.T) {
	workflowPath := filepath.Join(
		"..",
		"..",
		".github",
		"workflows",
		"validate-project.yml",
	)
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)

	required := []string{
		`tool_root="$RUNNER_TEMP/isras-tools"`,
		`GOBIN="$tool_root/bin"`,
		`tool_path="$tool_root/bin/govulncheck"`,
		`ISRAS_GOVULNCHECK_EXECUTABLE=%s`,
	}
	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Fatalf("hosted workflow missing marker %q", marker)
		}
	}
	if strings.Contains(text, "target/.local/tools") {
		t.Fatal("hosted workflow writes validator tooling into the target repository")
	}
}
