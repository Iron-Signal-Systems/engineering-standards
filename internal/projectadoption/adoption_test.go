package projectadoption

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
)

func TestCanonicalProjectOrigin(t *testing.T) {
	for _, test := range []struct {
		origin string
		want   string
	}{
		{"git@github.com:Iron-Signal-Systems/iron-atlas.git", "github.com/Iron-Signal-Systems/iron-atlas"},
		{"https://github.com/Iron-Signal-Systems/iron-atlas.git", "github.com/Iron-Signal-Systems/iron-atlas"},
		{"ssh://git@github.com/Iron-Signal-Systems/iron-atlas.git", "github.com/Iron-Signal-Systems/iron-atlas"},
	} {
		got, err := canonicalProjectOrigin(test.origin)
		if err != nil {
			t.Fatalf("canonicalize %q: %v", test.origin, err)
		}
		if got != test.want {
			t.Fatalf("canonicalize %q = %q, want %q", test.origin, got, test.want)
		}
	}
}

func TestCanonicalProjectOriginRejectsWrongBoundary(t *testing.T) {
	for _, origin := range []string{
		"",
		"git@gitlab.com:Iron-Signal-Systems/iron-atlas.git",
		"git@github.com:Other/iron-atlas.git",
		"https://github.com/Iron-Signal-Systems/too/many.git",
		"https://token@github.com/Iron-Signal-Systems/iron-atlas.git",
		"file://github.com/Iron-Signal-Systems/iron-atlas.git",
		"https://github.com/Iron-Signal-Systems/../iron-atlas.git",
	} {
		if _, err := canonicalProjectOrigin(origin); err == nil {
			t.Fatalf("unsafe origin %q was accepted", origin)
		}
	}
}

func TestValidateEvidenceDirectory(t *testing.T) {
	for _, value := range []string{".local/isras", "validation/evidence"} {
		if err := validateEvidenceDirectory(value); err != nil {
			t.Fatalf("valid evidence directory %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{"", ".", "..", "../outside", "/absolute", ".git/isras", `windows\path`} {
		if err := validateEvidenceDirectory(value); err == nil {
			t.Fatalf("invalid evidence directory %q accepted", value)
		}
	}
}

func TestCallerWorkflowPinsExactSourceCommit(t *testing.T) {
	commit := strings.Repeat("a", 40)
	workflow := string(callerWorkflow(commit))
	if !strings.Contains(workflow, "validate-project.yml@"+commit) {
		t.Fatal("caller workflow does not pin the exact source commit")
	}
	if strings.Contains(workflow, "@dev") || strings.Contains(workflow, "@main") {
		t.Fatal("caller workflow contains a floating branch")
	}
}

func TestInstallCreatesExactSetAndIsIdempotent(t *testing.T) {
	root := initializeTestRepository(t)
	identity, err := repository.DiscoverFrom(context.Background(), root)
	if err != nil {
		t.Fatalf("discover test repository: %v", err)
	}
	files := []installFile{
		{Path: ".isras/project.json", Data: []byte("pin\n"), Mode: 0o644},
		{Path: ".github/workflows/isras-validation.yml", Data: []byte("workflow\n"), Mode: 0o644},
		{Path: ".isras/adoption-verification.json", Data: []byte("evidence\n"), Mode: 0o644},
		{Path: ".isras/check-go-format", Data: []byte("#!/bin/sh\n"), Mode: 0o755},
	}
	changed, err := install(context.Background(), identity, files)
	if err != nil {
		t.Fatalf("install adoption files: %v", err)
	}
	if !changed {
		t.Fatal("first installation reported no change")
	}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatalf("read installed file %s: %v", file.Path, err)
		}
		if string(data) != string(file.Data) {
			t.Fatalf("installed file %s changed", file.Path)
		}
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != file.Mode.Perm() {
			t.Fatalf("installed file %s mode = %o, want %o", file.Path, info.Mode().Perm(), file.Mode.Perm())
		}
	}

	changed, err = install(context.Background(), identity, files)
	if err != nil {
		t.Fatalf("repeat exact installation: %v", err)
	}
	if changed {
		t.Fatal("repeat exact installation was not idempotent")
	}
}

func TestInstallRejectsConflictWithoutOverwrite(t *testing.T) {
	root := initializeTestRepository(t)
	identity, err := repository.DiscoverFrom(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".isras"), 0o755); err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(root, ".isras", "project.json")
	if err := os.WriteFile(conflict, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []installFile{
		{Path: ".isras/project.json", Data: []byte("replacement\n")},
		{Path: ".github/workflows/isras-validation.yml", Data: []byte("workflow\n")},
		{Path: ".isras/adoption-verification.json", Data: []byte("evidence\n")},
	}
	if _, err := install(context.Background(), identity, files); err == nil {
		t.Fatal("conflicting adoption state was accepted")
	}
	data, err := os.ReadFile(conflict)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing\n" {
		t.Fatal("conflicting file was overwritten")
	}
	if _, err := os.Stat(filepath.Join(root, ".github", "workflows", "isras-validation.yml")); !os.IsNotExist(err) {
		t.Fatal("partial workflow was created after conflict")
	}
}

func TestInstallRejectsDirtyRepository(t *testing.T) {
	root := initializeTestRepository(t)
	identity, err := repository.DiscoverFrom(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []installFile{{Path: ".isras/project.json", Data: []byte("pin\n")}}
	if _, err := install(context.Background(), identity, files); err == nil || !strings.Contains(err.Error(), "clean") {
		t.Fatalf("dirty repository error = %v", err)
	}
}

func TestInstallRejectsSymlinkedAdoptionDirectory(t *testing.T) {
	root := initializeTestRepository(t)
	identity, err := repository.DiscoverFrom(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".isras")); err != nil {
		t.Fatal(err)
	}
	files := []installFile{{Path: ".isras/project.json", Data: []byte("pin\n"), Mode: 0o644}}
	if _, err := install(context.Background(), identity, files); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlinked adoption directory error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "project.json")); !os.IsNotExist(err) {
		t.Fatal("installer wrote through a symbolic-link directory")
	}
}

func TestInstallRollsBackAfterPublicationFailure(t *testing.T) {
	root := initializeTestRepository(t)
	identity, err := repository.DiscoverFrom(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	files := []installFile{
		{Path: ".isras/project.json", Data: []byte("first\n"), Mode: 0o644},
		{Path: ".isras/project.json", Data: []byte("second\n"), Mode: 0o644},
	}
	if _, err := install(context.Background(), identity, files); err == nil {
		t.Fatal("duplicate publication path was accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".isras", "project.json")); !os.IsNotExist(err) {
		t.Fatal("failed installation left a published file")
	}
	if _, err := os.Stat(filepath.Join(root, ".isras")); !os.IsNotExist(err) {
		t.Fatal("failed installation left an adoption directory")
	}
}

func TestInstallRejectsModeDriftAsConflict(t *testing.T) {
	root := initializeTestRepository(t)
	identity, err := repository.DiscoverFrom(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	files := []installFile{{Path: ".isras/check-go-format", Data: []byte("#!/bin/sh\n"), Mode: 0o755}}
	if _, err := install(context.Background(), identity, files); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".isras", "check-go-format")
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := install(context.Background(), identity, files); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("mode drift error = %v", err)
	}
}

func initializeTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "--quiet")
	runGit(t, root, "config", "user.name", "ISRAS Test")
	runGit(t, root, "config", "user.email", "isras-test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "--quiet", "-m", "test")
	runGit(t, root, "remote", "add", "origin", "git@github.com:Iron-Signal-Systems/test-project.git")
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
