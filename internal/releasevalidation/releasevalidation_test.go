package releasevalidation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseRemoteTipAcceptsExactBranch(t *testing.T) {
	commit := strings.Repeat("a", 40)
	got, err := parseRemoteTip(commit+"\trefs/heads/dev\n", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if got != commit {
		t.Fatalf("commit = %q, want %q", got, commit)
	}
}

func TestParseRemoteTipRejectsMissingBranch(t *testing.T) {
	if _, err := parseRemoteTip("", "dev"); err == nil {
		t.Fatal("missing remote branch was accepted")
	}
}

func TestParseRemoteTipRejectsUnexpectedRef(t *testing.T) {
	commit := strings.Repeat("b", 40)
	if _, err := parseRemoteTip(commit+"\trefs/heads/main\n", "dev"); err == nil {
		t.Fatal("unexpected remote ref was accepted")
	}
}

func TestSanitizeOriginRedactsEmbeddedCredentials(t *testing.T) {
	got := sanitizeOrigin("https://user:" + "pass" + "word" + "@example.invalid/org/repo.git")
	if strings.Contains(got, "user") || strings.Contains(got, "password") {
		t.Fatalf("credentials remained in sanitized origin: %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("sanitized origin did not mark redaction: %q", got)
	}
}

func TestReadToolVersionRequiresExactVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool-versions.json")
	data := `{"tools":{"govulncheck":{"version":"latest"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readToolVersion(path, "govulncheck"); err == nil {
		t.Fatal("non-exact tool version was accepted")
	}
}

func TestReadToolVersionReturnsPinnedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool-versions.json")
	data := `{"tools":{"govulncheck":{"version":"v1.6.0"}}}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readToolVersion(path, "govulncheck")
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.6.0" {
		t.Fatalf("version = %q, want v1.6.0", got)
	}
}

func TestBuildRunIDUsesUTCAndShortCommit(t *testing.T) {
	at := time.Date(2026, 7, 17, 0, 15, 30, 0, time.UTC)
	commit := "0123456789abcdef0123456789abcdef01234567"
	got := buildRunID(at, commit)
	if got != "20260717T001530Z-0123456789ab" {
		t.Fatalf("run ID = %q", got)
	}
}
