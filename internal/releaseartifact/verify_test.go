package releaseartifact

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

func TestVerifyDirectoryPassesCompleteArtifactSet(t *testing.T) {
	directory, pin := buildFixture(t)
	report, err := VerifyDirectory(pin, directory)
	if err != nil {
		t.Fatal(err)
	}
	if report.PinDigests != StatusPass || report.SHA256Manifest != StatusPass || report.SHA512Manifest != StatusPass || report.Provenance != StatusPass {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.ExecutionAuthorization != AuthorizationDenied {
		t.Fatalf("local directory must not authorize execution: %s", report.ExecutionAuthorization)
	}
}

func TestVerifyDirectoryRejectsMutatedArtifact(t *testing.T) {
	directory, pin := buildFixture(t)
	if err := os.WriteFile(filepath.Join(directory, "isras-validator-linux-amd64"), []byte("mutated"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := VerifyDirectory(pin, directory)
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
	if report.PinDigests != StatusFail || report.ExecutionAuthorization != AuthorizationDenied {
		t.Fatalf("unexpected failure report: %+v", report)
	}
}

func TestVerifyDirectoryRejectsManifestMismatchAfterManifestRepin(t *testing.T) {
	directory, pin := buildFixture(t)
	manifest := filepath.Join(directory, "SHA256SUMS")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), pin.Artifacts[0].SHA256, strings.Repeat("a", 64), 1))
	if err := os.WriteFile(manifest, data, 0o600); err != nil {
		t.Fatal(err)
	}
	updateArtifactDigests(t, directory, &pin, "SHA256SUMS")

	report, err := VerifyDirectory(pin, directory)
	if err == nil || !strings.Contains(err.Error(), "manifest digest") {
		t.Fatalf("expected manifest mismatch, got %v", err)
	}
	if report.SHA256Manifest != StatusFail {
		t.Fatalf("SHA-256 manifest status = %s", report.SHA256Manifest)
	}
}

func TestVerifyDirectoryRejectsProvenanceIdentityDrift(t *testing.T) {
	directory, pin := buildFixture(t)
	provenancePath := filepath.Join(directory, "provenance.json")
	var value provenance
	data, err := os.ReadFile(provenancePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	value.SourceCommit = strings.Repeat("a", 40)
	writeJSON(t, provenancePath, value)
	updateArtifactDigests(t, directory, &pin, "provenance.json")
	rebuildManifests(t, directory, &pin)

	report, err := VerifyDirectory(pin, directory)
	if err == nil || !strings.Contains(err.Error(), "provenance identity") {
		t.Fatalf("expected provenance identity mismatch, got %v", err)
	}
	if report.Provenance != StatusFail {
		t.Fatalf("provenance status = %s", report.Provenance)
	}
}

func TestVerifyDirectoryRejectsExtraEntry(t *testing.T) {
	directory, pin := buildFixture(t)
	if err := os.WriteFile(filepath.Join(directory, "extra.bin"), []byte("extra"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := VerifyDirectory(pin, directory)
	if err == nil || !strings.Contains(err.Error(), "exact declared file set") {
		t.Fatalf("expected inventory failure, got %v", err)
	}
}

func TestVerifyDirectoryRejectsSymlink(t *testing.T) {
	directory, pin := buildFixture(t)
	target := filepath.Join(directory, "isras-project-framework.tar.gz")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("isras-contracts.tar.gz", target); err != nil {
		t.Fatal(err)
	}
	_, err := VerifyDirectory(pin, directory)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink failure, got %v", err)
	}
}

func TestWriteEvidenceRetainsFullExpectedAndObservedDigests(t *testing.T) {
	directory, pin := buildFixture(t)
	report, err := VerifyDirectory(pin, directory)
	if err != nil {
		t.Fatal(err)
	}
	report.FinishedAt = time.Date(2026, 7, 17, 20, 0, 0, 1, time.UTC)
	root := t.TempDir()
	jsonPath, textPath, err := WriteEvidence(root, ".local/validation", report)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{jsonPath, textPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), report.Artifacts[0].ExpectedSHA256) || !strings.Contains(string(data), report.Artifacts[0].ObservedSHA512) {
			t.Fatalf("evidence missing complete digests: %s", path)
		}
	}
}

func buildFixture(t *testing.T) (string, projectpin.Pin) {
	t.Helper()
	directory := t.TempDir()
	core := map[string][]byte{
		"isras-validator-linux-amd64":    []byte("validator-v0.1.5\n"),
		"isras-project-framework.tar.gz": []byte("framework-v0.1.5\n"),
		"isras-contracts.tar.gz":         []byte("contracts-v0.1.5\n"),
	}
	for name, data := range core {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	pin := projectpin.Pin{
		SchemaVersion: projectpin.SchemaVersion,
		Project:       projectpin.Project{Repository: "github.com/Iron-Signal-Systems/iron-atlas"},
		Standard: projectpin.Standard{
			Profile: projectpin.Profile, Version: "0.1.5", ReleaseTag: "isras-v0.1.5",
			SourceRepository: projectpin.SourceRepository,
			SourceCommit:     "0123456789abcdef0123456789abcdef01234567",
		},
		Workflow: projectpin.Workflow{
			Repository: projectpin.SourceRepository,
			Path:       projectpin.ReusableWorkflowPath,
			Commit:     "0123456789abcdef0123456789abcdef01234567",
		},
		Profiles: []string{"go"},
		Commands: map[string][]string{
			"format_check":          {"gofmt", "-l", "."},
			"static_analysis":       {"go", "vet", "./..."},
			"test":                  {"go", "test", "./..."},
			"build":                 {"go", "build", "./..."},
			"module_consistency":    {"go", "mod", "tidy", "-diff"},
			"module_integrity":      {"go", "mod", "verify"},
			"known_vulnerabilities": {"govulncheck", "./..."},
		},
		Evidence: projectpin.Evidence{Directory: ".local/validation"},
	}

	pin.Artifacts = []projectpin.Artifact{
		newArtifact(t, directory, "validator", "linux", "amd64", "isras-validator-linux-amd64"),
		newArtifact(t, directory, "framework", "", "", "isras-project-framework.tar.gz"),
		newArtifact(t, directory, "contracts", "", "", "isras-contracts.tar.gz"),
	}

	prov := provenance{
		SchemaVersion:    1,
		Profile:          pin.Standard.Profile,
		Version:          pin.Standard.Version,
		ReleaseTag:       pin.Standard.ReleaseTag,
		SourceRepository: pin.Standard.SourceRepository,
		SourceCommit:     pin.Standard.SourceCommit,
		Build:            provenanceBuild{GoVersion: "go1.25.12", GOOS: "linux", GOARCH: "amd64"},
		Validation:       provenanceValidation{Campaign: "isras-release", Commit: pin.Standard.SourceCommit, Status: "PASS"},
		PublishedAt:      "2026-07-17T20:00:00Z",
		ReleaseAuthority: "Iron Signal Systems",
		Limitations:      []string{"self-validated"},
	}
	for _, artifact := range sortedArtifacts(pin.Artifacts) {
		prov.Artifacts = append(prov.Artifacts, provenanceArtifact{
			Kind: artifact.Kind, OS: artifact.OS, Arch: artifact.Arch, Name: artifact.Name,
			SHA256: artifact.SHA256, SHA512: artifact.SHA512,
		})
	}
	writeJSON(t, filepath.Join(directory, "provenance.json"), prov)
	pin.Artifacts = append(pin.Artifacts, newArtifact(t, directory, "provenance", "", "", "provenance.json"))

	rebuildManifests(t, directory, &pin)
	return directory, pin
}

func rebuildManifests(t *testing.T, directory string, pin *projectpin.Pin) {
	t.Helper()
	pin.Artifacts = removeByKind(pin.Artifacts, "sha256-manifest")
	pin.Artifacts = removeByKind(pin.Artifacts, "sha512-manifest")
	artifacts := sortedArtifacts(pin.Artifacts)
	var sha256Text strings.Builder
	var sha512Text strings.Builder
	for _, artifact := range artifacts {
		sha256Text.WriteString(artifact.SHA256 + "  " + artifact.Name + "\n")
		sha512Text.WriteString(artifact.SHA512 + "  " + artifact.Name + "\n")
	}
	if err := os.WriteFile(filepath.Join(directory, "SHA256SUMS"), []byte(sha256Text.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "SHA512SUMS"), []byte(sha512Text.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	pin.Artifacts = append(pin.Artifacts,
		newArtifact(t, directory, "sha256-manifest", "", "", "SHA256SUMS"),
		newArtifact(t, directory, "sha512-manifest", "", "", "SHA512SUMS"),
	)
}

func newArtifact(t *testing.T, directory, kind, operatingSystem, architecture, name string) projectpin.Artifact {
	t.Helper()
	sha256Value, sha512Value := fileDigests(t, filepath.Join(directory, name))
	return projectpin.Artifact{
		Kind: kind, OS: operatingSystem, Arch: architecture, Name: name,
		SHA256: sha256Value, SHA512: sha512Value,
	}
}

func updateArtifactDigests(t *testing.T, directory string, pin *projectpin.Pin, name string) {
	t.Helper()
	sha256Value, sha512Value := fileDigests(t, filepath.Join(directory, name))
	for index := range pin.Artifacts {
		if pin.Artifacts[index].Name == name {
			pin.Artifacts[index].SHA256 = sha256Value
			pin.Artifacts[index].SHA512 = sha512Value
			return
		}
	}
	t.Fatalf("artifact not found: %s", name)
}

func fileDigests(t *testing.T, path string) (string, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h256 := sha256.Sum256(data)
	h512 := sha512.Sum512(data)
	return hex.EncodeToString(h256[:]), hex.EncodeToString(h512[:])
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func sortedArtifacts(values []projectpin.Artifact) []projectpin.Artifact {
	out := append([]projectpin.Artifact(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func removeByKind(values []projectpin.Artifact, kind string) []projectpin.Artifact {
	out := values[:0]
	for _, value := range values {
		if value.Kind != kind {
			out = append(out, value)
		}
	}
	return out
}
