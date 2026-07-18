package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

func TestRenderProjectPinValidationReportsMetadataOnly(t *testing.T) {
	pin := projectpin.Pin{
		Project: projectpin.Project{Repository: "github.com/Iron-Signal-Systems/iron-atlas"},
		Standard: projectpin.Standard{
			ReleaseTag:   "isras-v0.1.5",
			SourceCommit: "0123456789abcdef0123456789abcdef01234567",
		},
	}

	var output bytes.Buffer
	renderProjectPinValidation(&output, pin)
	rendered := output.String()

	for _, expected := range []string{
		"PROJECT PIN DECLARATION VALIDATION",
		"Declaration status:    PASS",
		"Validation scope:      metadata structure and identity only",
		"Artifact verification: NOT PERFORMED",
		projectpin.MetadataPath,
		"isras-v0.1.5",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("validation output missing %q:\n%s", expected, rendered)
		}
	}
	if strings.Contains(rendered, "Artifact verification: PASS") {
		t.Fatalf("validation output made a false artifact-verification claim:\n%s", rendered)
	}
}

func TestRenderProjectPinReportsDeclarationWithoutVerificationClaim(t *testing.T) {
	sha256 := strings.Repeat("1", 64)
	sha512 := strings.Repeat("2", 128)
	pin := projectpin.Pin{
		SchemaVersion: 1,
		Project:       projectpin.Project{Repository: "github.com/Iron-Signal-Systems/iron-atlas"},
		Standard: projectpin.Standard{
			Profile: projectpin.Profile, Version: "0.1.5", ReleaseTag: "isras-v0.1.5",
			SourceRepository: projectpin.SourceRepository,
			SourceCommit:     "0123456789abcdef0123456789abcdef01234567",
		},
		Artifacts: []projectpin.Artifact{{
			Kind: "validator", OS: "linux", Arch: "amd64", Name: "isras-validator-linux-amd64",
			SHA256: sha256, SHA512: sha512,
		}},
		Workflow: projectpin.Workflow{
			Repository: projectpin.SourceRepository, Path: projectpin.ReusableWorkflowPath,
			Commit: "0123456789abcdef0123456789abcdef01234567",
		},
		Profiles: []string{"go"},
		Commands: map[string][]string{
			"test":  {"go", "test", "runtime-only-value"},
			"build": {"go", "build", "./..."},
		},
		Evidence: projectpin.Evidence{Directory: projectpin.RuntimeEvidenceDirectory},
	}

	var output bytes.Buffer
	renderProjectPin(&output, pin)
	rendered := output.String()

	for _, expected := range []string{
		"ISRAS PROJECT PIN DECLARATION",
		"Declaration status:      PASS",
		"Validation scope:        metadata structure and identity only",
		"Artifact verification:   NOT PERFORMED",
		"Verification reason:     artifact bytes were not acquired or hashed",
		"github.com/Iron-Signal-Systems/iron-atlas",
		"isras-v0.1.5",
		"Profiles:                go",
		"Commands declared:       build, test",
		"Artifacts declared:      1",
		"validator (linux/amd64): isras-validator-linux-amd64",
		"Declared SHA-256: 111111111111...111111111111",
		"Declared SHA-512: 222222222222...222222222222",
		"Byte verification: NOT PERFORMED",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("output missing %q:\n%s", expected, rendered)
		}
	}

	for _, prohibited := range []string{
		"runtime-only-value",
		sha256,
		sha512,
		"Artifact verification:   PASS",
		"Byte verification: PASS",
		"Artifacts:       ",
		"\n     SHA-256: ",
		"\n     SHA-512: ",
	} {
		if strings.Contains(rendered, prohibited) {
			t.Fatalf("project pin inspection exposed or implied %q:\n%s", prohibited, rendered)
		}
	}
}

func TestAbbreviatedDigest(t *testing.T) {
	full := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if got, want := abbreviatedDigest(full), "0123456789ab...456789abcdef"; got != want {
		t.Fatalf("abbreviated digest = %q, want %q", got, want)
	}
	if got := abbreviatedDigest("short"); got != "short" {
		t.Fatalf("short digest changed: %q", got)
	}
}
