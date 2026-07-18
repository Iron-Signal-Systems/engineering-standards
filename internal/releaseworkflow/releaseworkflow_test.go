package releaseworkflow

import (
	"strings"
	"testing"
)

func TestStableVersionPattern(t *testing.T) {
	accepted := []string{"0.1.0", "0.1.1", "1.0.0", "12.34.56"}
	for _, value := range accepted {
		if !stableVersionPattern.MatchString(value) {
			t.Fatalf("stable version %q was rejected", value)
		}
	}
	rejected := []string{"0.1", "v0.1.0", "0.1.0-development", "0.1.1-development", "0.1.0+build", ""}
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

func TestValidateTagIdentityRejectsPublishedVersionBoundToDifferentCommit(t *testing.T) {
	candidate := "a58ea97fb881a2582a1fe5e24436513c2d99a2a3"
	published := "96d0bbae212027ef2c74d4d90dc3fe1df981bd58"
	object := "23963864b9b35f7ca6317d8b074cf4ed76200fdc"
	local := remoteTag{Exists: true, ObjectSHA: object, CommitSHA: published}
	remote := remoteTag{Exists: true, ObjectSHA: object, CommitSHA: published}

	err := validateTagIdentity(local, remote, candidate, "isras-v0.1.0")
	if err == nil {
		t.Fatal("published isras-v0.1.0 tag was accepted for a different release candidate")
	}
	if !strings.Contains(err.Error(), "advance VERSION") {
		t.Fatalf("conflict did not provide version-advance guidance: %v", err)
	}
}

func TestValidateTagIdentityAcceptsExactPublishedRelease(t *testing.T) {
	commit := strings.Repeat("c", 40)
	object := strings.Repeat("d", 40)
	local := remoteTag{Exists: true, ObjectSHA: object, CommitSHA: commit}
	remote := remoteTag{Exists: true, ObjectSHA: object, CommitSHA: commit}

	if err := validateTagIdentity(local, remote, commit, "isras-v1.2.3"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTagIdentityRejectsRemoteTagWithoutLocalTag(t *testing.T) {
	commit := strings.Repeat("e", 40)
	remote := remoteTag{
		Exists:    true,
		ObjectSHA: strings.Repeat("f", 40),
		CommitSHA: commit,
	}

	if err := validateTagIdentity(remoteTag{}, remote, commit, "isras-v1.2.3"); err == nil {
		t.Fatal("remote tag without a corresponding local tag was accepted")
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

func TestValidateActionRejectsLegacyPublish(t *testing.T) {
	err := validateAction(ActionPublish)
	if err == nil || !strings.Contains(err.Error(), "legacy release publication is disabled") {
		t.Fatalf("legacy publication action was not disabled: %v", err)
	}
}
