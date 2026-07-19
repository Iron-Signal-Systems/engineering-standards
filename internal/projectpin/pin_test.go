package projectpin

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSourceCommit = "0123456789abcdef0123456789abcdef01234567"

func TestParseValidPin(t *testing.T) {
	pin, err := Parse(mustJSON(t, validPin()))
	if err != nil {
		t.Fatal(err)
	}
	if pin.Standard.ReleaseTag != "isras-v0.1.5" {
		t.Fatalf("unexpected release tag: %s", pin.Standard.ReleaseTag)
	}
	if pin.Project.Repository != "github.com/Iron-Signal-Systems/iron-atlas" {
		t.Fatalf("unexpected project: %s", pin.Project.Repository)
	}
}

func TestLoadUsesCanonicalProjectPinPath(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, filepath.FromSlash(MetadataPath))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, mustJSON(t, validPin()), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatal(err)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	data := mustJSON(t, validPin())
	data = bytes.Replace(data, []byte(`"schema_version":1`), []byte(`"schema_version":1,"unknown":true`), 1)
	requireError(t, data, "unknown field")
}

func TestParseRejectsDuplicateNestedFields(t *testing.T) {
	data := mustJSON(t, validPin())
	old := []byte(`"repository":"github.com/Iron-Signal-Systems/iron-atlas"`)
	newValue := []byte(`"repository":"github.com/Iron-Signal-Systems/iron-atlas","repository":"github.com/Iron-Signal-Systems/other"`)
	data = bytes.Replace(data, old, newValue, 1)
	requireError(t, data, "duplicate field")
}

func TestParseRejectsMultipleJSONValues(t *testing.T) {
	data := append(mustJSON(t, validPin()), []byte("\n{}\n")...)
	requireError(t, data, "trailing JSON value")
}

func TestParseRejectsDevelopmentVersion(t *testing.T) {
	pin := validPin()
	pin.Standard.Version = "0.1.5-development"
	pin.Standard.ReleaseTag = "isras-v0.1.5-development"
	requirePinError(t, pin, "stable MAJOR.MINOR.PATCH")
}

func TestParseRejectsReleaseTagDrift(t *testing.T) {
	pin := validPin()
	pin.Standard.ReleaseTag = "isras-v0.1.4"
	requirePinError(t, pin, "does not match the pinned version")
}

func TestParseRejectsWorkflowSourceDrift(t *testing.T) {
	pin := validPin()
	pin.Workflow.Commit = "89abcdef0123456789abcdef0123456789abcdef"
	requirePinError(t, pin, "workflow commit does not match")
}

func TestParseRejectsDuplicateValidatorPlatform(t *testing.T) {
	pin := validPin()
	pin.Artifacts = append(pin.Artifacts, Artifact{
		Kind: "validator", OS: "linux", Arch: "amd64", Name: "isras-validator-linux-amd64-copy",
		SHA256: strings.Repeat("1", 64), SHA512: strings.Repeat("2", 128),
	})
	requirePinError(t, pin, "duplicate validator artifact platform")
}

func TestParseRejectsMissingRequiredArtifact(t *testing.T) {
	pin := validPin()
	pin.Artifacts = removeArtifactKind(pin.Artifacts, "contracts")
	requirePinError(t, pin, "requires exactly one contracts artifact")
}

func TestParseRejectsUnknownArtifactKind(t *testing.T) {
	pin := validPin()
	pin.Artifacts = append(pin.Artifacts, artifact("unknown-kind", "", "", "unknown.bin", "7"))
	requirePinError(t, pin, "unsupported kind")
}

func TestParseRejectsAllZeroDigest(t *testing.T) {
	pin := validPin()
	pin.Artifacts[0].SHA256 = strings.Repeat("0", 64)
	requirePinError(t, pin, "invalid artifact SHA-256")
}

func TestParseRejectsPlatformOnNonValidatorArtifact(t *testing.T) {
	pin := validPin()
	pin.Artifacts[1].OS = "linux"
	pin.Artifacts[1].Arch = "amd64"
	requirePinError(t, pin, "must not declare os or arch")
}

func TestParseRejectsUnsupportedProfile(t *testing.T) {
	pin := validPin()
	pin.Profiles = []string{"rust"}
	requirePinError(t, pin, "unsupported project profile")
}

func TestParseRejectsDuplicateProfile(t *testing.T) {
	pin := validPin()
	pin.Profiles = []string{"go", "go"}
	requirePinError(t, pin, "duplicate project profile")
}

func TestParseRejectsMissingGoCommand(t *testing.T) {
	pin := validPin()
	delete(pin.Commands, "known_vulnerabilities")
	requirePinError(t, pin, `Go profile requires command "known_vulnerabilities"`)
}

func TestParseRejectsOpaqueExecutableString(t *testing.T) {
	pin := validPin()
	pin.Commands["test"] = []string{"go test ./..."}
	requirePinError(t, pin, "executable must be one argument")
}

func TestParseRejectsControlCharactersInArguments(t *testing.T) {
	pin := validPin()
	pin.Commands["test"] = []string{"go", "test\n./..."}
	requirePinError(t, pin, "prohibited control character")
}

func TestParseRejectsUnsafeEvidencePath(t *testing.T) {
	pin := validPin()
	pin.Evidence.Directory = ".isras"
	requirePinError(t, pin, "evidence directory must be")
}

func TestParseErrorsDoNotEchoUntrustedFieldValues(t *testing.T) {
	pin := validPin()
	untrusted := "SensitiveBoundaryValue987"
	pin.Project.Repository = untrusted
	_, err := Parse(mustJSON(t, pin))
	if err == nil {
		t.Fatal("expected invalid project identity")
	}
	if strings.Contains(err.Error(), untrusted) {
		t.Fatalf("parser error exposed untrusted field value: %v", err)
	}
}

func TestParseRejectsOversizedEvidencePath(t *testing.T) {
	pin := validPin()
	pin.Evidence.Directory = strings.Repeat("a", maxRelativePathBytes+1)
	requirePinError(t, pin, "evidence directory must be")
}

func TestParseRejectsOversizedPin(t *testing.T) {
	data := bytes.Repeat([]byte("x"), MaxFileSize+1)
	requireError(t, data, "exceeds")
}

func validPin() Pin {
	artifacts := []Artifact{
		artifact("validator", "linux", "amd64", "isras-validator-linux-amd64", "1"),
		artifact("framework", "", "", "isras-project-framework.tar.gz", "2"),
		artifact("contracts", "", "", "isras-contracts.tar.gz", "3"),
		artifact("provenance", "", "", "provenance.json", "4"),
		artifact("sha256-manifest", "", "", "SHA256SUMS", "5"),
		artifact("sha512-manifest", "", "", "SHA512SUMS", "6"),
	}
	return Pin{
		SchemaVersion: SchemaVersion,
		Project:       Project{Repository: "github.com/Iron-Signal-Systems/iron-atlas"},
		Standard: Standard{
			Profile: Profile, Version: "0.1.5", ReleaseTag: "isras-v0.1.5",
			SourceRepository: SourceRepository, SourceCommit: testSourceCommit,
		},
		Artifacts: artifacts,
		Workflow: Workflow{
			Repository: SourceRepository, Path: ReusableWorkflowPath, Commit: testSourceCommit,
		},
		Profiles: []string{"go"},
		Commands: map[string][]string{
			"format_check":          {"./tools/check-go-formatting.sh"},
			"static_analysis":       {"go", "vet", "./..."},
			"test":                  {"go", "test", "./..."},
			"build":                 {"go", "build", "./..."},
			"module_consistency":    {"go", "mod", "tidy", "-diff"},
			"module_integrity":      {"go", "mod", "verify"},
			"known_vulnerabilities": {"govulncheck", "./..."},
		},
		Evidence: Evidence{Directory: RuntimeEvidenceDirectory},
	}
}

func artifact(kind, operatingSystem, architecture, name, seed string) Artifact {
	return Artifact{
		Kind: kind, OS: operatingSystem, Arch: architecture, Name: name,
		SHA256: strings.Repeat(seed, 64), SHA512: strings.Repeat(seed, 128),
	}
}

func removeArtifactKind(artifacts []Artifact, kind string) []Artifact {
	out := make([]Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Kind != kind {
			out = append(out, artifact)
		}
	}
	return out
}

func mustJSON(t *testing.T, pin Pin) []byte {
	t.Helper()
	data, err := json.Marshal(pin)
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func requirePinError(t *testing.T, pin Pin, expected string) {
	t.Helper()
	requireError(t, mustJSON(t, pin), expected)
}

func requireError(t *testing.T, data []byte, expected string) {
	t.Helper()
	_, err := Parse(data)
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}
