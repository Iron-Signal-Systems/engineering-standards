package releaseartifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestGitHubBootstrapDiscoversAndVerifiesRelease(t *testing.T) {
	fixture := newBootstrapFixture(t, true)
	bootstrap, err := (GitHubClient{Run: fixture.run}).Bootstrap(context.Background(), fixture.tag)
	if err != nil {
		t.Fatalf("bootstrap release: %v", err)
	}
	if bootstrap.Standard.Version != "0.1.2" || bootstrap.Standard.ReleaseTag != fixture.tag || bootstrap.Standard.SourceCommit != fixture.commit {
		t.Fatalf("unexpected standard identity: %#v", bootstrap.Standard)
	}
	if len(bootstrap.Artifacts) != 6 {
		t.Fatalf("artifact count = %d, want 6", len(bootstrap.Artifacts))
	}
	if bootstrap.Report.ExecutionAuthorization != AuthorizationGranted {
		t.Fatalf("execution authorization = %q", bootstrap.Report.ExecutionAuthorization)
	}
	for _, status := range []string{
		bootstrap.Report.ReleaseRecord,
		bootstrap.Report.SignedTag,
		bootstrap.Report.AssetAcquisition,
		bootstrap.Report.AssetInventory,
		bootstrap.Report.PinDigests,
		bootstrap.Report.SHA256Manifest,
		bootstrap.Report.SHA512Manifest,
		bootstrap.Report.Provenance,
	} {
		if status != StatusPass {
			t.Fatalf("bootstrap status = %q, want PASS", status)
		}
	}
}

func TestGitHubBootstrapRejectsReleaseWithoutReusableWorkflow(t *testing.T) {
	fixture := newBootstrapFixture(t, false)
	bootstrap, err := (GitHubClient{Run: fixture.run}).Bootstrap(context.Background(), fixture.tag)
	if err == nil {
		t.Fatal("release without reusable workflow was accepted")
	}
	if !strings.Contains(err.Error(), "reusable validation workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
	if bootstrap.Report.ExecutionAuthorization == AuthorizationGranted {
		t.Fatal("failed bootstrap granted execution authorization")
	}
}

type bootstrapFixture struct {
	t           *testing.T
	directory   string
	tag         string
	commit      string
	tagObject   string
	releaseJSON []byte
	refJSON     []byte
	tagJSON     []byte
}

func newBootstrapFixture(t *testing.T, includeWorkflow bool) bootstrapFixture {
	t.Helper()
	directory := t.TempDir()
	tag := "isras-v0.1.2"
	commit := strings.Repeat("a", 40)
	tagObject := strings.Repeat("b", 40)

	core := map[string][]byte{
		"isras-validator-linux-amd64": []byte("validator\n"),
		"isras-contracts.tar.gz":      []byte("contracts\n"),
	}
	core["isras-project-framework.tar.gz"] = frameworkArchive(t, includeWorkflow)
	for name, data := range core {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	coreNames := make([]string, 0, len(core))
	for name := range core {
		coreNames = append(coreNames, name)
	}
	sort.Strings(coreNames)
	provenanceArtifacts := make([]provenanceArtifact, 0, len(coreNames))
	for _, name := range coreNames {
		d256, d512 := byteDigests(core[name])
		spec := bootstrapAssetSpecs[name]
		provenanceArtifacts = append(provenanceArtifacts, provenanceArtifact{
			Kind: spec.Kind, OS: spec.OS, Arch: spec.Arch, Name: name,
			SHA256: d256, SHA512: d512,
		})
	}
	provenanceData, err := json.MarshalIndent(provenance{
		SchemaVersion:    1,
		Profile:          "ISRAS-SD",
		Version:          "0.1.2",
		ReleaseTag:       tag,
		SourceRepository: "github.com/Iron-Signal-Systems/engineering-standards",
		SourceCommit:     commit,
		Build:            provenanceBuild{GoVersion: "go1.25.12", GOOS: "linux", GOARCH: "amd64"},
		Validation:       provenanceValidation{Campaign: "test", Commit: commit, Status: "PASS"},
		PublishedAt:      time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		ReleaseAuthority: "test",
		Limitations:      []string{"test evidence"},
		Artifacts:        provenanceArtifacts,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	provenanceData = append(provenanceData, '\n')
	if err := os.WriteFile(filepath.Join(directory, "provenance.json"), provenanceData, 0o644); err != nil {
		t.Fatal(err)
	}

	nonManifest := append(coreNames, "provenance.json")
	sort.Strings(nonManifest)
	writeTestManifest(t, directory, "SHA256SUMS", nonManifest, true)
	writeTestManifest(t, directory, "SHA512SUMS", nonManifest, false)

	assets := make([]releaseAsset, 0, 6)
	for _, name := range bootstrapAssetNames() {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		d256, _ := byteDigests(data)
		assets = append(assets, releaseAsset{Name: name, State: "uploaded", Size: int64(len(data)), Digest: "sha256:" + d256})
	}
	releaseJSON, _ := json.Marshal(releaseRecord{TagName: tag, Assets: assets})
	reference := gitReference{Ref: "refs/tags/" + tag}
	reference.Object.Type = "tag"
	reference.Object.SHA = tagObject
	refJSON, _ := json.Marshal(reference)
	annotated := annotatedTag{Tag: tag}
	annotated.Object.Type = "commit"
	annotated.Object.SHA = commit
	annotated.Verification.Verified = true
	annotated.Verification.Reason = "valid"
	annotated.Verification.Signature = "signature"
	annotated.Verification.Payload = "payload"
	annotated.Verification.VerifiedAt = "2026-07-18T00:00:00Z"
	tagJSON, _ := json.Marshal(annotated)

	return bootstrapFixture{t: t, directory: directory, tag: tag, commit: commit, tagObject: tagObject, releaseJSON: releaseJSON, refJSON: refJSON, tagJSON: tagJSON}
}

func (fixture bootstrapFixture) run(_ context.Context, args ...string) ([]byte, error) {
	joined := strings.Join(args, " ")
	switch {
	case joined == "api repos/Iron-Signal-Systems/engineering-standards/releases/tags/"+fixture.tag:
		return fixture.releaseJSON, nil
	case joined == "api repos/Iron-Signal-Systems/engineering-standards/git/ref/tags/"+fixture.tag:
		return fixture.refJSON, nil
	case joined == "api repos/Iron-Signal-Systems/engineering-standards/git/tags/"+fixture.tagObject:
		return fixture.tagJSON, nil
	case len(args) >= 5 && args[0] == "release" && args[1] == "download":
		directory := ""
		for index := range args {
			if args[index] == "--dir" && index+1 < len(args) {
				directory = args[index+1]
			}
		}
		if directory == "" {
			return nil, fmt.Errorf("download directory missing")
		}
		for _, name := range bootstrapAssetNames() {
			data, err := os.ReadFile(filepath.Join(fixture.directory, name))
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(directory, name), data, 0o644); err != nil {
				return nil, err
			}
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected command: %s", joined)
	}
}

func frameworkArchive(t *testing.T, includeWorkflow bool) []byte {
	t.Helper()
	var output bytes.Buffer
	compressed := gzip.NewWriter(&output)
	archive := tar.NewWriter(compressed)
	entries := map[string][]byte{
		"isras-project-framework/integration-guides/PROJECT-ADOPTION.md": []byte("guide\n"),
	}
	if includeWorkflow {
		entries["isras-project-framework/.github/workflows/validate-project.yml"] = []byte("on:\n  workflow_call:\n")
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data := entries[name]
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func writeTestManifest(t *testing.T, directory, name string, artifacts []string, useSHA256 bool) {
	t.Helper()
	var output strings.Builder
	for _, artifact := range artifacts {
		data, err := os.ReadFile(filepath.Join(directory, artifact))
		if err != nil {
			t.Fatal(err)
		}
		d256, d512 := byteDigests(data)
		digest := d512
		if useSHA256 {
			digest = d256
		}
		fmt.Fprintf(&output, "%s  %s\n", digest, artifact)
	}
	if err := os.WriteFile(filepath.Join(directory, name), []byte(output.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func byteDigests(data []byte) (string, string) {
	d256 := sha256.Sum256(data)
	d512 := sha512.Sum512(data)
	return hex.EncodeToString(d256[:]), hex.EncodeToString(d512[:])
}
