package projectadoption

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReusableWorkflowUsesImmutableCalledSourceIdentity(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	path := filepath.Join(root, ".github", "workflows", "validate-project.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read reusable workflow: %v", err)
	}
	text := string(data)
	for _, required := range []string{
		"workflow_call:",
		"repository: ${{ job.workflow_repository }}",
		"ref: ${{ job.workflow_sha }}",
		"--mode commit",
		"project-pin verify-artifacts",
		"project-command run",
		"actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd",
		"actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("reusable workflow is missing required boundary %q", required)
		}
	}
	for _, forbidden := range []string{
		"actions/checkout@main",
		"actions/checkout@v",
		"actions/setup-go@main",
		"actions/setup-go@v",
		"ref: ${{ github.sha }}",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("reusable workflow contains floating or caller-owned identity %q", forbidden)
		}
	}
}

func TestReusableWorkflowIsInReleaseFramework(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "release", "framework-files.txt"))
	if err != nil {
		t.Fatalf("read framework file list: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	wanted := ".github/workflows/validate-project.yml"
	count := 0
	previous := ""
	for _, line := range lines {
		if previous != "" && line <= previous {
			t.Fatalf("framework file list is not strictly sorted: %q then %q", previous, line)
		}
		if line == wanted {
			count++
		}
		previous = line
	}
	if count != 1 {
		t.Fatalf("reusable workflow occurrence count = %d, want 1", count)
	}
}
