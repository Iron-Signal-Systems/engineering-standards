package projectadoption

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectorigin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

func TestProjectAdoptionUsesSharedCanonicalOrigin(t *testing.T) {
	got, err := projectorigin.Canonical("git@github.com:Iron-Signal-Systems/iron-atlas.git")
	if err != nil {
		t.Fatal(err)
	}
	if got != "github.com/Iron-Signal-Systems/iron-atlas" {
		t.Fatalf("canonical origin = %q", got)
	}
}

func TestInitializeRequiresExactReleaseValidatorBeforeNetwork(t *testing.T) {
	called := false
	_, err := initializeWithBootstrap(context.Background(), Request{
		Root: "/does/not/matter", ReleaseTag: "isras-v0.1.2", GoDefaults: true,
		Validator: validatoridentity.Identity{Metadata: validatoridentity.Metadata{
			Profile: projectpin.Profile, StandardVersion: "0.1.2",
			Ownership:        validatoridentity.OwnershipReference,
			SourceRepository: projectpin.SourceRepository,
		}},
	}, func(context.Context, string) (releaseartifact.Bootstrap, error) {
		called = true
		return releaseartifact.Bootstrap{}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "release validator") {
		t.Fatalf("authority error = %v", err)
	}
	if called {
		t.Fatal("bootstrap network boundary ran before validator authority was established")
	}
}

func TestInitializeTwiceIsActuallyIdempotentWithFreshReports(t *testing.T) {
	root := initializeTestRepository(t)
	call := 0
	bootstrap := func(context.Context, string) (releaseartifact.Bootstrap, error) {
		call++
		return validBootstrap(time.Date(2026, 7, 18, 19, 0, call, 0, time.UTC)), nil
	}
	request := Request{
		Root: root, ReleaseTag: "isras-v0.1.2", GoDefaults: true,
		Validator: releaseValidatorIdentity(),
	}
	first, err := initializeWithBootstrap(context.Background(), request, bootstrap)
	if err != nil {
		t.Fatalf("first initialization: %v", err)
	}
	if !first.Changed {
		t.Fatal("first initialization reported no change")
	}
	evidencePath := filepath.Join(root, filepath.FromSlash(AdoptionEvidencePath))
	firstEvidence, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(firstEvidence, []byte("started_at")) || bytes.Contains(firstEvidence, []byte("finished_at")) {
		t.Fatal("durable adoption evidence contains volatile timestamps")
	}

	second, err := initializeWithBootstrap(context.Background(), request, bootstrap)
	if err != nil {
		t.Fatalf("second initialization: %v", err)
	}
	if second.Changed {
		t.Fatal("second initialization was not idempotent")
	}
	secondEvidence, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstEvidence, secondEvidence) {
		t.Fatal("fresh verification report changed durable adoption evidence")
	}
	if call != 2 {
		t.Fatalf("bootstrap calls = %d, want 2", call)
	}
}

func TestInitializeRejectsValidatorThatDoesNotMatchVerifiedRelease(t *testing.T) {
	root := initializeTestRepository(t)
	request := Request{
		Root: root, ReleaseTag: "isras-v0.1.2", GoDefaults: true,
		Validator: releaseValidatorIdentity(),
	}
	bootstrap := validBootstrap(time.Now().UTC())
	bootstrap.Standard.SourceCommit = strings.Repeat("b", 40)
	_, err := initializeWithBootstrap(context.Background(), request, func(context.Context, string) (releaseartifact.Bootstrap, error) {
		return bootstrap, nil
	})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("identity mismatch error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".isras")); !os.IsNotExist(statErr) {
		t.Fatal("identity mismatch modified the target")
	}
}

func TestValidateEvidenceBoundaryRejectsTrackedOrSymlinkedPath(t *testing.T) {
	t.Run("tracked", func(t *testing.T) {
		root := initializeTestRepository(t)
		path := filepath.Join(root, ".local", "isras", "record.txt")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("tracked\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		runGit(t, root, "add", ".local/isras/record.txt")
		runGit(t, root, "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "track evidence")
		if err := validateEvidenceBoundary(context.Background(), root); err == nil || !strings.Contains(err.Error(), "tracked") {
			t.Fatalf("tracked evidence error = %v", err)
		}
	})
	t.Run("symlink", func(t *testing.T) {
		root := initializeTestRepository(t)
		if err := os.MkdirAll(filepath.Join(root, ".local"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(t.TempDir(), filepath.Join(root, ".local", "isras")); err != nil {
			t.Fatal(err)
		}
		if err := validateEvidenceBoundary(context.Background(), root); err == nil || !strings.Contains(err.Error(), "unsafe") {
			t.Fatalf("symlink evidence error = %v", err)
		}
	})
}

func releaseValidatorIdentity() validatoridentity.Identity {
	commit := strings.Repeat("a", 40)
	return validatoridentity.Identity{
		Metadata: validatoridentity.Metadata{
			SchemaVersion: 1, Profile: projectpin.Profile, StandardVersion: "0.1.2",
			Ownership:        validatoridentity.OwnershipReleaseArtifact,
			SourceRepository: projectpin.SourceRepository, SourceCommit: commit,
		},
		ReleaseTag: "isras-v0.1.2", RepositoryCommit: commit,
	}
}

func validBootstrap(now time.Time) releaseartifact.Bootstrap {
	commit := strings.Repeat("a", 40)
	standard := projectpin.Standard{
		Profile: projectpin.Profile, Version: "0.1.2", ReleaseTag: "isras-v0.1.2",
		SourceRepository: projectpin.SourceRepository, SourceCommit: commit,
	}
	specs := []struct{ name, kind, goos, arch string }{
		{"SHA256SUMS", "sha256-manifest", "", ""},
		{"SHA512SUMS", "sha512-manifest", "", ""},
		{"isras-contracts.tar.gz", "contracts", "", ""},
		{"isras-project-framework.tar.gz", "framework", "", ""},
		{"isras-validator-linux-amd64", "validator", "linux", "amd64"},
		{"provenance.json", "provenance", "", ""},
	}
	artifacts := make([]projectpin.Artifact, 0, len(specs))
	results := make([]releaseartifact.ArtifactResult, 0, len(specs))
	for index, spec := range specs {
		d256 := strings.Repeat(string(rune('a'+index)), 64)
		d512 := strings.Repeat(string(rune('a'+index)), 128)
		artifacts = append(artifacts, projectpin.Artifact{
			Kind: spec.kind, Name: spec.name, OS: spec.goos, Arch: spec.arch,
			SHA256: d256, SHA512: d512,
		})
		results = append(results, releaseartifact.ArtifactResult{
			Kind: spec.kind, Name: spec.name, OS: spec.goos, Arch: spec.arch,
			Size: int64(100 + index), RemoteSize: int64(100 + index),
			ExpectedSHA256: d256, ObservedSHA256: d256,
			ExpectedSHA512: d512, ObservedSHA512: d512,
			SHA256Status: releaseartifact.StatusPass, SHA512Status: releaseartifact.StatusPass,
			SHA256Manifest: releaseartifact.StatusPass, SHA512Manifest: releaseartifact.StatusPass,
			ProvenanceBinding: releaseartifact.StatusPass,
		})
	}
	return releaseartifact.Bootstrap{
		Standard: standard, Artifacts: artifacts,
		Report: releaseartifact.Report{
			SchemaVersion: 1, StartedAt: now, FinishedAt: now.Add(time.Second),
			SourceMode: "github-release-bootstrap", SourceLocation: projectpin.SourceRepository + "@isras-v0.1.2",
			ReleaseTag: "isras-v0.1.2", SourceCommit: commit,
			ReleaseRecord: releaseartifact.StatusPass, SignedTag: releaseartifact.StatusPass,
			AssetAcquisition: releaseartifact.StatusPass, AssetInventory: releaseartifact.StatusPass,
			PinDigests: releaseartifact.StatusPass, SHA256Manifest: releaseartifact.StatusPass,
			SHA512Manifest: releaseartifact.StatusPass, Provenance: releaseartifact.StatusPass,
			ExecutionAuthorization: releaseartifact.AuthorizationGranted, Artifacts: results,
		},
	}
}

func TestGeneratedGoFormatCheckPropagatesGitFailure(t *testing.T) {
	text := string(goFormatCheck())
	if strings.Contains(text, "< <(git ls-files") {
		t.Fatal("format checker still hides git ls-files failure in process substitution")
	}
	for _, required := range []string{"git ls-files -z", ">\"$list_file\"", "mapfile -d '' files <\"$list_file\""} {
		if !strings.Contains(text, required) {
			t.Fatalf("format checker is missing fail-closed boundary %q", required)
		}
	}
	root := t.TempDir()
	path := filepath.Join(root, "check-go-format")
	if err := os.WriteFile(path, goFormatCheck(), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(path)
	command.Dir = root
	if output, err := command.CombinedOutput(); err == nil {
		t.Fatalf("format checker succeeded outside a Git repository:\n%s", output)
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
	runGit(t, root, "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test")
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
