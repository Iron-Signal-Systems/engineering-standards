package projectpin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCommittedRequiresIdenticalHeadIndexAndWorkingTree(t *testing.T) {
	root := committedFixture(t)
	pin, err := LoadCommitted(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if pin.Project.Repository != validPin().Project.Repository {
		t.Fatalf("unexpected project: %s", pin.Project.Repository)
	}
}

func TestLoadCommittedRejectsWorkingTreeDrift(t *testing.T) {
	root := committedFixture(t)
	path := filepath.Join(root, filepath.FromSlash(MetadataPath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadCommitted(context.Background(), root)
	if err == nil || !strings.Contains(err.Error(), "working-tree project pin differs") {
		t.Fatalf("expected working-tree drift failure, got %v", err)
	}
}

func TestLoadCommittedRejectsIndexDrift(t *testing.T) {
	root := committedFixture(t)
	path := filepath.Join(root, filepath.FromSlash(MetadataPath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	committedGit(t, root, "add", MetadataPath)
	_, err = LoadCommitted(context.Background(), root)
	if err == nil || !strings.Contains(err.Error(), "staged project pin differs") {
		t.Fatalf("expected index drift failure, got %v", err)
	}
}

func TestLoadCommittedRejectsSymlinkPath(t *testing.T) {
	root := committedFixture(t)
	path := filepath.Join(root, filepath.FromSlash(MetadataPath))
	external := filepath.Join(t.TempDir(), "project.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(external, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, path); err != nil {
		t.Fatal(err)
	}
	_, err = LoadCommitted(context.Background(), root)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symlink failure, got %v", err)
	}
}

func committedFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	committedGit(t, root, "init", "-q")
	committedGit(t, root, "config", "user.name", "ISRAS Test")
	committedGit(t, root, "config", "user.email", "isras-test@example.invalid")
	path := filepath.Join(root, filepath.FromSlash(MetadataPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, mustJSON(t, validPin()), 0o644); err != nil {
		t.Fatal(err)
	}
	committedGit(t, root, "add", MetadataPath)
	committedGit(t, root, "commit", "-q", "-m", "commit project pin")
	return root
}

func committedGit(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", arguments, err, output)
	}
}
