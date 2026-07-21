package docimpact

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectComparisonUsesMergeBaseAndSortedPaths(t *testing.T) {
	root := initializeGitRepository(t)
	writeText(t, filepath.Join(root, "README.md"), "base\n")
	base := commitAll(t, root, "base")

	writeText(
		t,
		filepath.Join(root, "internal", "z", "z.go"),
		"package z\n",
	)
	writeText(
		t,
		filepath.Join(root, "internal", "a", "a.go"),
		"package a\n",
	)
	writeText(t, filepath.Join(root, "CHANGELOG.md"), "changed\n")
	head := commitAll(t, root, "head")

	comparison, err := CollectComparison(
		context.Background(),
		root,
		base,
		head,
	)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.BaseCommit != base ||
		comparison.HeadCommit != head ||
		comparison.MergeBase != base {
		t.Fatalf("comparison identity = %#v", comparison)
	}
	want := "CHANGELOG.md,internal/a/a.go,internal/z/z.go"
	if strings.Join(comparison.ChangedPaths, ",") != want {
		t.Fatalf(
			"changed paths = %q, want %q",
			strings.Join(comparison.ChangedPaths, ","),
			want,
		)
	}
}

func TestCollectComparisonRejectsNonExactCommits(t *testing.T) {
	root := initializeGitRepository(t)
	writeText(t, filepath.Join(root, "README.md"), "base\n")
	commit := commitAll(t, root, "base")

	for _, invalid := range []string{
		"HEAD",
		"main",
		commit[:12],
		strings.ToUpper(commit),
		"-" + commit[1:],
	} {
		_, err := CollectComparison(
			context.Background(),
			root,
			invalid,
			commit,
		)
		if err == nil || !strings.Contains(err.Error(), "exact 40-character") {
			t.Fatalf("commit %q error = %v", invalid, err)
		}
	}
}

func TestParseNULPathsRejectsUnsafeOutput(t *testing.T) {
	for _, value := range [][]byte{
		[]byte("not-terminated"),
		[]byte("../escape\x00"),
		[]byte(".local/evidence\x00"),
	} {
		_, err := parseNULPaths(value)
		if err == nil {
			t.Fatalf("unsafe output accepted: %q", value)
		}
	}
}

func initializeGitRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGitTest(t, root, "init", "-q")
	runGitTest(t, root, "config", "user.name", "ISRAS Test")
	runGitTest(
		t,
		root,
		"config",
		"user.email",
		"isras-test@example.invalid",
	)
	runGitTest(t, root, "config", "commit.gpgsign", "false")
	return root
}

func commitAll(t *testing.T, root, message string) string {
	t.Helper()
	runGitTest(t, root, "add", "-A")
	runGitTest(t, root, "commit", "-q", "-m", message)
	return strings.TrimSpace(
		runGitTest(t, root, "rev-parse", "HEAD"),
	)
}

func runGitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	result := execCommandForTest(t, root, "git", args...)
	return result
}

func execCommandForTest(
	t *testing.T,
	root string,
	name string,
	args ...string,
) string {
	t.Helper()
	command := exec.Command(name, append([]string{"-C", root}, args...)...)
	command.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return string(output)
}

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
