package repository

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFromExplicitRepositoryOutsideCurrentDirectory(t *testing.T) {
	repositoryRoot := initializeRepository(t, "first.txt")
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	identity, err := DiscoverFrom(context.Background(), repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Root != repositoryRoot {
		t.Fatalf("root = %q, want %q", identity.Root, repositoryRoot)
	}
	if len(identity.Commit) != 40 {
		t.Fatalf("commit = %q", identity.Commit)
	}
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if current != original {
		t.Fatalf("working directory changed from %q to %q", original, current)
	}
}

func TestDiscoverFromSubdirectoryReturnsCanonicalRoot(t *testing.T) {
	repositoryRoot := initializeRepository(t, "nested.txt")
	subdirectory := filepath.Join(repositoryRoot, "a", "b")
	if err := os.MkdirAll(subdirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	identity, err := DiscoverFrom(context.Background(), subdirectory)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Root != repositoryRoot {
		t.Fatalf("root = %q, want %q", identity.Root, repositoryRoot)
	}
}

func TestDiscoverFromRejectsSymlinkedTarget(t *testing.T) {
	repositoryRoot := initializeRepository(t, "symlink.txt")
	link := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(repositoryRoot, link); err != nil {
		t.Fatal(err)
	}
	_, err := DiscoverFrom(context.Background(), link)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symbolic-link rejection, got %v", err)
	}
}

func TestDiscoverFromKeepsRepositoriesIsolated(t *testing.T) {
	first := initializeRepository(t, "first.txt")
	second := initializeRepository(t, "second.txt")
	firstIdentity, err := DiscoverFrom(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	secondIdentity, err := DiscoverFrom(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if firstIdentity.Root == secondIdentity.Root {
		t.Fatalf("roots were not isolated: %q", firstIdentity.Root)
	}
	if firstIdentity.Commit == secondIdentity.Commit {
		t.Fatalf("fixture commits unexpectedly match: %q", firstIdentity.Commit)
	}
}

func TestDiscoverFromRejectsMissingAndNonDirectoryTargets(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := DiscoverFrom(context.Background(), missing); err == nil {
		t.Fatal("missing target was accepted")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("not a repository\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DiscoverFrom(context.Background(), file); err == nil {
		t.Fatal("non-directory target was accepted")
	}
}

func initializeRepository(t *testing.T, filename string) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.name", "ISRAS Fixture")
	runGit(t, root, "config", "user.email", "isras-fixture@example.invalid")
	if err := os.WriteFile(filepath.Join(root, filename), []byte(filename+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", filename)
	runGit(t, root, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "fixture")
	absolute, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(absolute)
}

func runGit(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgSign=false"}, arguments...)...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
