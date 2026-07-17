package validatoridentity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testRepositoryCommit = "0123456789abcdef0123456789abcdef01234567"
const testSourceCommit = "89abcdef0123456789abcdef0123456789abcdef"

func TestReferenceIdentityUsesVersionAndRepositoryCommit(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "0.1.1-development", `{
  "schema_version": 1,
  "profile": "ISRAS-SD",
  "standard_version": "0.1.1-development",
  "ownership": "reference-repository",
  "source_repository": "github.com/Iron-Signal-Systems/engineering-standards"
}
`)

	identity, err := Load(root, testRepositoryCommit)
	if err != nil {
		t.Fatal(err)
	}
	if identity.SourceCommit != testRepositoryCommit {
		t.Fatalf("reference source commit mismatch: %s", identity.SourceCommit)
	}
	if identity.RepositoryCommit != testRepositoryCommit {
		t.Fatalf("repository commit mismatch: %s", identity.RepositoryCommit)
	}
	if identity.Header() != "ISRAS-SD 0.1.1-development [reference]" {
		t.Fatalf("unexpected header: %s", identity.Header())
	}
}

func TestReferenceIdentityRejectsVersionDrift(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "0.1.1-development", `{
  "schema_version": 1,
  "profile": "ISRAS-SD",
  "standard_version": "0.1.0-development",
  "ownership": "reference-repository",
  "source_repository": "github.com/Iron-Signal-Systems/engineering-standards"
}
`)

	_, err := Load(root, testRepositoryCommit)
	if err == nil || !strings.Contains(err.Error(), "does not match VERSION") {
		t.Fatalf("expected version-drift failure, got %v", err)
	}
}

func TestProjectOwnedExportPreservesSourceAndTargetIdentity(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "", `{
  "schema_version": 1,
  "profile": "ISRAS-SD",
  "standard_version": "0.1.1-development",
  "ownership": "project-owned-export",
  "source_repository": "github.com/Iron-Signal-Systems/engineering-standards",
  "source_commit": "89abcdef0123456789abcdef0123456789abcdef",
  "target_module": "github.com/Iron-Signal-Systems/iron-atlas"
}
`)

	identity, err := Load(root, testRepositoryCommit)
	if err != nil {
		t.Fatal(err)
	}
	if identity.SourceCommit != testSourceCommit {
		t.Fatalf("source commit mismatch: %s", identity.SourceCommit)
	}
	if identity.TargetModule != "github.com/Iron-Signal-Systems/iron-atlas" {
		t.Fatalf("target module mismatch: %s", identity.TargetModule)
	}
	if identity.RepositoryCommit != testRepositoryCommit {
		t.Fatalf("repository commit mismatch: %s", identity.RepositoryCommit)
	}
	if identity.Header() != "ISRAS-SD 0.1.1-development [project-owned export]" {
		t.Fatalf("unexpected header: %s", identity.Header())
	}
}

func TestProjectOwnedExportRequiresExactSourceCommit(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "", `{
  "schema_version": 1,
  "profile": "ISRAS-SD",
  "standard_version": "0.1.1-development",
  "ownership": "project-owned-export",
  "source_repository": "github.com/Iron-Signal-Systems/engineering-standards",
  "source_commit": "unknown",
  "target_module": "example.com/target"
}
`)

	_, err := Load(root, testRepositoryCommit)
	if err == nil || !strings.Contains(err.Error(), "invalid exported validator source commit") {
		t.Fatalf("expected source-commit failure, got %v", err)
	}
}

func TestIdentityRejectsUnknownMetadataFields(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "0.1.1-development", `{
  "schema_version": 1,
  "profile": "ISRAS-SD",
  "standard_version": "0.1.1-development",
  "ownership": "reference-repository",
  "source_repository": "github.com/Iron-Signal-Systems/engineering-standards",
  "ambiguous_version": true
}
`)

	_, err := Load(root, testRepositoryCommit)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field failure, got %v", err)
	}
}

func TestIdentityRejectsInvalidRepositoryCommit(t *testing.T) {
	root := t.TempDir()
	writeIdentityFixture(t, root, "0.1.1-development", `{
  "schema_version": 1,
  "profile": "ISRAS-SD",
  "standard_version": "0.1.1-development",
  "ownership": "reference-repository",
  "source_repository": "github.com/Iron-Signal-Systems/engineering-standards"
}
`)

	_, err := Load(root, "HEAD")
	if err == nil || !strings.Contains(err.Error(), "invalid repository commit identity") {
		t.Fatalf("expected repository-commit failure, got %v", err)
	}
}

func writeIdentityFixture(t *testing.T, root, version, metadata string) {
	t.Helper()
	if version != "" {
		if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte(version+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, filepath.FromSlash(MetadataPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(metadata), 0o644); err != nil {
		t.Fatal(err)
	}
}
