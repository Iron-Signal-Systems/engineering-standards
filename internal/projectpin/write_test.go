package projectpin

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonicalJSONProducesValidatedStablePin(t *testing.T) {
	pin := Pin{
		SchemaVersion: SchemaVersion,
		Project:       Project{Repository: "github.com/Iron-Signal-Systems/iron-atlas"},
		Standard: Standard{
			Profile: Profile, Version: "0.1.2", ReleaseTag: "isras-v0.1.2",
			SourceRepository: SourceRepository,
			SourceCommit:     strings.Repeat("1", 40),
		},
		Artifacts: []Artifact{
			{Kind: "sha256-manifest", Name: "SHA256SUMS", SHA256: strings.Repeat("1", 64), SHA512: strings.Repeat("1", 128)},
			{Kind: "sha512-manifest", Name: "SHA512SUMS", SHA256: strings.Repeat("2", 64), SHA512: strings.Repeat("2", 128)},
			{Kind: "contracts", Name: "isras-contracts.tar.gz", SHA256: strings.Repeat("3", 64), SHA512: strings.Repeat("3", 128)},
			{Kind: "framework", Name: "isras-project-framework.tar.gz", SHA256: strings.Repeat("4", 64), SHA512: strings.Repeat("4", 128)},
			{Kind: "validator", OS: "linux", Arch: "amd64", Name: "isras-validator-linux-amd64", SHA256: strings.Repeat("5", 64), SHA512: strings.Repeat("5", 128)},
			{Kind: "provenance", Name: "provenance.json", SHA256: strings.Repeat("6", 64), SHA512: strings.Repeat("6", 128)},
		},
		Workflow: Workflow{Repository: SourceRepository, Path: ReusableWorkflowPath, Commit: strings.Repeat("1", 40)},
		Profiles: []string{"go"},
		Commands: DefaultGoCommands(),
		Evidence: Evidence{Directory: ".local/isras"},
	}

	first, err := CanonicalJSON(pin)
	if err != nil {
		t.Fatalf("canonicalize project pin: %v", err)
	}
	if !bytes.HasSuffix(first, []byte("\n")) {
		t.Fatal("canonical project pin is not newline terminated")
	}
	parsed, err := Parse(first)
	if err != nil {
		t.Fatalf("parse canonical project pin: %v", err)
	}
	second, err := CanonicalJSON(parsed)
	if err != nil {
		t.Fatalf("re-canonicalize project pin: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("canonical project pin changed after parse")
	}
}

func TestDefaultGoCommandsReturnsIndependentMaps(t *testing.T) {
	first := DefaultGoCommands()
	second := DefaultGoCommands()
	first["build"][0] = "changed"
	delete(first, "test")
	if second["build"][0] != "go" {
		t.Fatal("default command arguments share mutable backing storage")
	}
	if _, ok := second["test"]; !ok {
		t.Fatal("default command maps share mutable state")
	}
}
