package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifact"
)

func TestParseProjectArtifactArgs(t *testing.T) {
	if value, err := parseProjectArtifactArgs(nil); err != nil || value != "" {
		t.Fatalf("default parse = %q, %v", value, err)
	}
	if value, err := parseProjectArtifactArgs([]string{"--source-directory", "/tmp/assets"}); err != nil || value != "/tmp/assets" {
		t.Fatalf("directory parse = %q, %v", value, err)
	}
	if _, err := parseProjectArtifactArgs([]string{"--source-directory"}); err == nil {
		t.Fatal("expected missing source directory failure")
	}
}

func TestRenderProjectArtifactVerificationReportsAuthorizationWithoutDigests(t *testing.T) {
	fullDigest := strings.Repeat("a", 64)
	report := releaseartifact.Report{
		SourceMode: "github-release", ReleaseTag: "isras-v0.1.5",
		SourceCommit: strings.Repeat("b", 40), ReleaseRecord: releaseartifact.StatusPass,
		SignedTag: releaseartifact.StatusPass, AssetAcquisition: releaseartifact.StatusPass,
		AssetInventory: releaseartifact.StatusPass, PinDigests: releaseartifact.StatusPass,
		SHA256Manifest: releaseartifact.StatusPass, SHA512Manifest: releaseartifact.StatusPass,
		Provenance:             releaseartifact.StatusPass,
		ExecutionAuthorization: releaseartifact.AuthorizationGranted,
		Artifacts: []releaseartifact.ArtifactResult{{
			Kind: "validator", Name: "isras-validator-linux-amd64", OS: "linux", Arch: "amd64",
			Size: 100, ExpectedSHA256: fullDigest, ObservedSHA256: fullDigest,
			SHA256Status: releaseartifact.StatusPass, SHA512Status: releaseartifact.StatusPass,
			SHA256Manifest: releaseartifact.StatusPass, SHA512Manifest: releaseartifact.StatusPass,
			ProvenanceBinding: releaseartifact.StatusPass,
		}},
	}
	var output bytes.Buffer
	renderProjectArtifactVerification(&output, "/repo", report, "/repo/.local/validation/a.json", "/repo/.local/validation/a.txt")
	rendered := output.String()
	for _, expected := range []string{
		"ISRAS RELEASE ARTIFACT VERIFICATION",
		"Release record:          PASS",
		"Signed annotated tag:    PASS",
		"Pin SHA-256/SHA-512:     PASS",
		"Execution authorization: GRANTED",
		"validator (linux/amd64): isras-validator-linux-amd64",
		"Evidence JSON:           .local/validation/a.json",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("output missing %q:\n%s", expected, rendered)
		}
	}
	if strings.Contains(rendered, fullDigest) {
		t.Fatalf("terminal output exposed complete digest:\n%s", rendered)
	}
}
