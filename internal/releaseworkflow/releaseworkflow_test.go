package releaseworkflow

import (
	"strings"
	"testing"
)

func TestStableVersionPattern(t *testing.T) {
	accepted := []string{"0.1.0", "1.0.0", "12.34.56"}
	for _, value := range accepted {
		if !stableVersionPattern.MatchString(value) {
			t.Fatalf("stable version %q was rejected", value)
		}
	}
	rejected := []string{"0.1", "v0.1.0", "0.1.0-development", "0.1.0+build", ""}
	for _, value := range rejected {
		if stableVersionPattern.MatchString(value) {
			t.Fatalf("non-stable version %q was accepted", value)
		}
	}
}

func TestParseGitHubRepository(t *testing.T) {
	tests := map[string]string{
		"git@github.com:Iron-Signal-Systems/engineering-standards.git":       "Iron-Signal-Systems/engineering-standards",
		"ssh://git@github.com/Iron-Signal-Systems/engineering-standards.git": "Iron-Signal-Systems/engineering-standards",
		"https://github.com/Iron-Signal-Systems/engineering-standards.git":   "Iron-Signal-Systems/engineering-standards",
	}
	for input, want := range tests {
		got, err := parseGitHubRepository(input)
		if err != nil {
			t.Fatalf("parse %q: %v", input, err)
		}
		if got != want {
			t.Fatalf("parse %q = %q, want %q", input, got, want)
		}
	}
}

func TestParseGitHubRepositoryRejectsUnsupportedHost(t *testing.T) {
	if _, err := parseGitHubRepository("git@example.invalid:org/repo.git"); err == nil {
		t.Fatal("unsupported remote host was accepted")
	}
}

func TestParseRemoteTagAcceptsAnnotatedTag(t *testing.T) {
	object := strings.Repeat("a", 40)
	commit := strings.Repeat("b", 40)
	output := object + "\trefs/tags/isras-v0.1.0\n" +
		commit + "\trefs/tags/isras-v0.1.0^{}\n"
	got, err := parseRemoteTag(output, "isras-v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Exists || got.ObjectSHA != object || got.CommitSHA != commit {
		t.Fatalf("unexpected parsed tag: %#v", got)
	}
}

func TestParseRemoteTagRejectsLightweightTag(t *testing.T) {
	object := strings.Repeat("a", 40)
	output := object + "\trefs/tags/isras-v0.1.0\n"
	if _, err := parseRemoteTag(output, "isras-v0.1.0"); err == nil {
		t.Fatal("lightweight remote tag was accepted")
	}
}

func TestCompareRemoteTagRequiresExactObjectAndCommit(t *testing.T) {
	commit := strings.Repeat("c", 40)
	local := remoteTag{Exists: true, ObjectSHA: strings.Repeat("a", 40), CommitSHA: commit}
	remote := remoteTag{Exists: true, ObjectSHA: local.ObjectSHA, CommitSHA: commit}
	if err := compareRemoteTag(local, remote, commit); err != nil {
		t.Fatal(err)
	}
	remote.ObjectSHA = strings.Repeat("b", 40)
	if err := compareRemoteTag(local, remote, commit); err == nil {
		t.Fatal("mismatched remote tag object was accepted")
	}
}

func TestVerifyReleaseView(t *testing.T) {
	view := releaseView{
		TagName:      "isras-v0.1.0",
		Name:         "ISRAS 0.1.0 — Solo Developer Baseline",
		IsDraft:      false,
		IsPrerelease: false,
		PublishedAt:  "2026-07-17T09:00:12Z",
		URL:          "https://github.com/example/repo/releases/tag/isras-v0.1.0",
	}
	if err := verifyReleaseView(view, view.TagName, view.Name); err != nil {
		t.Fatal(err)
	}
	view.IsPrerelease = true
	if err := verifyReleaseView(view, view.TagName, view.Name); err == nil {
		t.Fatal("prerelease was accepted")
	}
}

func TestSanitizeOriginRedactsEmbeddedCredentials(t *testing.T) {
	origin := "https://user:" + "pass" + "word" + "@github.com/org/repo.git"
	got := sanitizeOrigin(origin)
	if strings.Contains(got, "user") || strings.Contains(got, "password") {
		t.Fatalf("credentials remained in sanitized origin: %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("redaction marker missing: %q", got)
	}
}
