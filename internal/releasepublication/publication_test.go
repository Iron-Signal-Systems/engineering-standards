package releasepublication

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifactbuild"
)

type publicationFixture struct {
	root      string
	commit    string
	version   string
	tag       string
	artifacts string
	evidence  string
	notes     string
	report    releaseartifactbuild.Result
	options   Options
	runner    *fakeRunner
}

type fakeRunner struct {
	base            OSRunner
	root            string
	commit          string
	tag             string
	repository      string
	artifactDir     string
	report          releaseartifactbuild.Result
	release         *githubRelease
	uploaded        map[int64][]byte
	calls           []string
	nextAssetID     int64
	failUpload      string
	corruptDownload bool
	verifiedTag     bool
}

func TestCheckPerformsNoRemoteMutation(t *testing.T) {
	fixture := newPublicationFixture(t)
	result, err := (Publisher{Runner: fixture.runner, Now: fixedNow}).Run(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if result.LocalVerification != StatusPass || result.RemoteTag != StatusPass || result.ReleaseAbsence != StatusPass {
		t.Fatalf("unexpected check result: %#v", result)
	}
	if result.DraftCreation != StatusNotPerformed || result.Publication != StatusNotPerformed {
		t.Fatalf("check claimed remote mutation: %#v", result)
	}
	for _, call := range fixture.runner.calls {
		if strings.Contains(call, "--method POST") || strings.Contains(call, "--method PATCH") || strings.Contains(call, "--method DELETE") {
			t.Fatalf("check performed a write call: %s", call)
		}
	}
	assertPrivateEvidence(t, result.EvidenceJSON)
	assertPrivateEvidence(t, result.EvidenceText)
}

func TestPublishUploadsAndReverifiesExactAssets(t *testing.T) {
	fixture := newPublicationFixture(t)
	fixture.options.Action = ActionPublish
	fixture.options.Confirm = true
	result, err := (Publisher{Runner: fixture.runner, Now: fixedNow}).Run(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.runner.release == nil || fixture.runner.release.Draft || fixture.runner.release.PublishedAt == nil {
		t.Fatalf("release was not published: %#v", fixture.runner.release)
	}
	if result.AssetUpload != StatusPass || result.DraftVerification != StatusPass || result.Publication != StatusPass || result.FinalVerification != StatusPass {
		t.Fatalf("publication stages did not pass: %#v", result)
	}
	if len(result.Artifacts) != 6 || len(fixture.runner.release.Assets) != 6 {
		t.Fatalf("unexpected asset count: result=%d release=%d", len(result.Artifacts), len(fixture.runner.release.Assets))
	}
	for _, artifact := range result.Artifacts {
		if artifact.UploadStatus != StatusPass || artifact.DownloadStatus != StatusPass || artifact.RemoteAssetID <= 0 {
			t.Fatalf("artifact was not fully verified: %#v", artifact)
		}
	}
	uploadCalls := 0
	for _, call := range fixture.runner.calls {
		if strings.Contains(call, "git push") || strings.Contains(call, "git tag") || strings.Contains(call, "refs/heads/main") {
			t.Fatalf("publisher crossed the ref-mutation boundary: %s", call)
		}
		if strings.Contains(call, "/assets?name=") {
			uploadCalls++
			if !strings.Contains(call, "gh api --hostname uploads.github.com --method POST") {
				t.Fatalf("release asset upload did not use uploads.github.com: %s", call)
			}
		}
	}
	if uploadCalls != 6 {
		t.Fatalf("upload call count = %d, want 6", uploadCalls)
	}
}

func TestPublishCleansExactDraftAfterUploadFailure(t *testing.T) {
	fixture := newPublicationFixture(t)
	fixture.options.Action = ActionPublish
	fixture.options.Confirm = true
	fixture.runner.failUpload = releaseartifactbuild.FrameworkName
	result, err := (Publisher{Runner: fixture.runner, Now: fixedNow}).Run(context.Background(), fixture.options)
	if err == nil {
		t.Fatal("upload failure was accepted")
	}
	if fixture.runner.release != nil {
		t.Fatalf("incomplete draft was not cleaned up: %#v", fixture.runner.release)
	}
	if result.Cleanup != StatusPass {
		t.Fatalf("cleanup was not recorded as passing: %#v", result)
	}
}

func TestPublishRejectsExistingRelease(t *testing.T) {
	fixture := newPublicationFixture(t)
	fixture.runner.release = &githubRelease{ID: 55, TagName: fixture.tag, Name: releaseTitle("", fixture.version), Draft: true, HTMLURL: "https://example.invalid/draft"}
	fixture.options.Action = ActionPublish
	fixture.options.Confirm = true
	_, err := (Publisher{Runner: fixture.runner, Now: fixedNow}).Run(context.Background(), fixture.options)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing release was not rejected: %v", err)
	}
	if fixture.runner.release == nil || fixture.runner.release.ID != 55 {
		t.Fatal("preexisting release was modified")
	}
}

func TestPublishRejectsUnverifiedRemoteTag(t *testing.T) {
	fixture := newPublicationFixture(t)
	fixture.runner.verifiedTag = false
	fixture.options.Action = ActionPublish
	fixture.options.Confirm = true
	_, err := (Publisher{Runner: fixture.runner, Now: fixedNow}).Run(context.Background(), fixture.options)
	if err == nil || !strings.Contains(err.Error(), "not verified") {
		t.Fatalf("unverified remote tag was not rejected: %v", err)
	}
	if fixture.runner.release != nil {
		t.Fatal("release was created despite unverified tag")
	}
}

func TestPublishRejectsAlteredRemoteBytesAndCleansDraft(t *testing.T) {
	fixture := newPublicationFixture(t)
	fixture.runner.corruptDownload = true
	fixture.options.Action = ActionPublish
	fixture.options.Confirm = true
	result, err := (Publisher{Runner: fixture.runner, Now: fixedNow}).Run(context.Background(), fixture.options)
	if err == nil || !strings.Contains(err.Error(), "downloaded remote release assets") {
		t.Fatalf("altered remote bytes were not rejected: %v", err)
	}
	if fixture.runner.release != nil || result.Cleanup != StatusPass {
		t.Fatalf("altered draft was not cleaned up safely: release=%#v result=%#v", fixture.runner.release, result)
	}
}

func TestRepositoryFromOriginRejectsCredentials(t *testing.T) {
	passwordValue := strings.Join([]string{"pass", "word"}, "")
	tokenUser := strings.Join([]string{"to", "ken"}, "")
	for _, value := range []string{
		fmt.Sprintf("https://user:%s@github.com/Iron-Signal-Systems/engineering-standards.git", passwordValue),
		fmt.Sprintf("https://%s@github.com/Iron-Signal-Systems/engineering-standards.git", tokenUser),
		"git@example.invalid:Iron-Signal-Systems/engineering-standards.git",
	} {
		if _, err := repositoryFromOrigin(value); err == nil {
			t.Fatalf("unsafe origin was accepted: %s", value)
		}
	}
}

func TestVerifyReleaseRecordRejectsExtraAsset(t *testing.T) {
	fixture := newPublicationFixture(t)
	boundary := sourceBoundary{Tag: fixture.tag, Title: releaseTitle("", fixture.version), NotesBody: "fixture notes"}
	release := githubRelease{ID: 1, TagName: fixture.tag, Name: boundary.Title, Body: boundary.NotesBody, Draft: true}
	release.Assets = []githubAsset{{ID: 1, Name: "extra", State: "uploaded", Size: 1, Digest: "sha256:" + strings.Repeat("a", 64)}}
	if err := verifyReleaseRecord(release, boundary, true, false, fixture.report.Artifacts); err == nil {
		t.Fatal("extra release asset was accepted")
	}
}

func newPublicationFixture(t *testing.T) publicationFixture {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-b", "dev")
	runGit(t, root, "config", "user.name", "ISRAS Test")
	runGit(t, root, "config", "user.email", "isras-test@example.invalid")
	version := "1.2.3"
	tag := "isras-v" + version
	writeFile(t, filepath.Join(root, ".gitignore"), []byte(".local/\n"), 0o644)
	writeFile(t, filepath.Join(root, "VERSION"), []byte(version+"\n"), 0o644)
	writeFile(t, filepath.Join(root, "CHANGELOG.md"), []byte("# Changelog\n\n## "+version+" — Test release\n\n- Test.\n"), 0o644)
	notes := filepath.Join(root, "docs", "releases", version+".md")
	writeFile(t, notes, []byte("# ISRAS "+version+"\n\nTest release notes.\n"), 0o644)
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "test release source")
	commit := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	runGit(t, root, "remote", "add", "origin", "git@github.com:Iron-Signal-Systems/engineering-standards.git")

	artifacts := filepath.Join(root, ".local", "releases", tag, "assets")
	if err := os.MkdirAll(artifacts, 0o700); err != nil {
		t.Fatal(err)
	}
	validator := "#!/bin/sh\ncat <<'OUT'\n" +
		"Standard version:  " + version + "\n" +
		"Ownership:         release-artifact\n" +
		"Release tag:       " + tag + "\n" +
		"Source repository: " + releaseartifactbuild.SourceRepository + "\n" +
		"Source commit:     " + commit + "\nOUT\n"
	writeFile(t, filepath.Join(artifacts, releaseartifactbuild.ValidatorName), []byte(validator), 0o755)
	writeFile(t, filepath.Join(artifacts, releaseartifactbuild.FrameworkName), []byte("framework archive\n"), 0o644)
	writeFile(t, filepath.Join(artifacts, releaseartifactbuild.ContractsName), []byte("contracts archive\n"), 0o644)

	core := []releaseartifactbuild.ArtifactRecord{
		hashRecord(t, artifacts, "contracts", releaseartifactbuild.ContractsName, "", ""),
		hashRecord(t, artifacts, "framework", releaseartifactbuild.FrameworkName, "", ""),
		hashRecord(t, artifacts, "validator", releaseartifactbuild.ValidatorName, "linux", "amd64"),
	}
	sort.Slice(core, func(i, j int) bool { return core[i].Name < core[j].Name })
	provenanceArtifacts := make([]map[string]any, 0, len(core))
	for _, record := range core {
		entry := map[string]any{"kind": record.Kind, "name": record.Name, "sha256": record.SHA256, "sha512": record.SHA512}
		if record.OS != "" {
			entry["os"] = record.OS
			entry["arch"] = record.Arch
		}
		provenanceArtifacts = append(provenanceArtifacts, entry)
	}
	provenance := map[string]any{
		"schema_version":    1,
		"profile":           releaseartifactbuild.Profile,
		"version":           version,
		"release_tag":       tag,
		"source_repository": releaseartifactbuild.SourceRepository,
		"source_commit":     commit,
		"build":             map[string]any{"go_version": "go1.25.12", "goos": "linux", "goarch": "amd64"},
		"validation":        map[string]any{"campaign": "fixture-campaign", "commit": commit, "status": "PASS"},
		"published_at":      "2026-07-18T12:00:00Z",
		"release_authority": "fixture-authority",
		"limitations":       []string{"fixture evidence"},
		"artifacts":         provenanceArtifacts,
	}
	writeJSONFile(t, filepath.Join(artifacts, releaseartifactbuild.ProvenanceName), provenance, 0o644)
	nonManifest := append(append([]releaseartifactbuild.ArtifactRecord(nil), core...), hashRecord(t, artifacts, "provenance", releaseartifactbuild.ProvenanceName, "", ""))
	sort.Slice(nonManifest, func(i, j int) bool { return nonManifest[i].Name < nonManifest[j].Name })
	writeManifestFixture(t, filepath.Join(artifacts, releaseartifactbuild.SHA256Name), nonManifest, true)
	writeManifestFixture(t, filepath.Join(artifacts, releaseartifactbuild.SHA512Name), nonManifest, false)

	records := append([]releaseartifactbuild.ArtifactRecord(nil), nonManifest...)
	records = append(records,
		hashRecord(t, artifacts, "sha256-manifest", releaseartifactbuild.SHA256Name, "", ""),
		hashRecord(t, artifacts, "sha512-manifest", releaseartifactbuild.SHA512Name, "", ""),
	)
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	evidence := filepath.Join(root, ".local", "validation", "releases", tag, "artifact-build.json")
	textEvidence := filepath.Join(filepath.Dir(evidence), "artifact-build.txt")
	report := releaseartifactbuild.Result{
		SchemaVersion: 1, GeneratedAt: fixedNow(), Profile: releaseartifactbuild.Profile,
		Version: version, ReleaseTag: tag, SourceRepository: releaseartifactbuild.SourceRepository,
		SourceCommit: commit, GoVersion: "go1.25.12", OutputDirectory: artifacts,
		Artifacts: records, EvidenceJSON: evidence, EvidenceText: textEvidence,
	}
	writeJSONFile(t, evidence, report, 0o600)
	writeFile(t, textEvidence, []byte("artifact build evidence\n"), 0o600)

	runner := &fakeRunner{
		root: root, commit: commit, tag: tag,
		repository:  "Iron-Signal-Systems/engineering-standards",
		artifactDir: artifacts, report: report, uploaded: map[int64][]byte{},
		nextAssetID: 100, verifiedTag: true,
	}
	return publicationFixture{
		root: root, commit: commit, version: version, tag: tag,
		artifacts: artifacts, evidence: evidence, notes: notes, report: report,
		options: Options{Root: root, Action: ActionCheck, ExpectedVersion: version, Branch: "dev", Remote: "origin", ArtifactDirectory: artifacts, BuildEvidence: evidence, NotesFile: notes},
		runner:  runner,
	}
}

func (runner *fakeRunner) Run(ctx context.Context, dir string, environment []string, name string, args ...string) CommandResult {
	runner.calls = append(runner.calls, name+" "+strings.Join(args, " "))
	if name == "git" && len(args) > 0 {
		switch args[0] {
		case "verify-commit":
			return CommandResult{ExitCode: 0}
		case "ls-remote":
			return CommandResult{Stdout: []byte(runner.commit + "\trefs/heads/dev\n"), ExitCode: 0}
		}
	}
	if name != "gh" {
		return runner.base.Run(ctx, dir, environment, name, args...)
	}
	return runner.runGH(args)
}

func (runner *fakeRunner) RunToFile(ctx context.Context, dir string, environment []string, outputPath, name string, args ...string) CommandResult {
	runner.calls = append(runner.calls, name+" "+strings.Join(args, " ")+" > "+outputPath)
	if name != "gh" {
		return runner.base.RunToFile(ctx, dir, environment, outputPath, name, args...)
	}
	endpoint := args[len(args)-1]
	parts := strings.Split(endpoint, "/")
	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return failedCommand("invalid asset id")
	}
	data, ok := runner.uploaded[id]
	if !ok {
		return failedCommand("asset missing")
	}
	if runner.corruptDownload {
		data = append([]byte(nil), data...)
		data[0] ^= 0xff
		runner.corruptDownload = false
	}
	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return CommandResult{ExitCode: -1, Err: err}
	}
	return CommandResult{ExitCode: 0}
}

func (runner *fakeRunner) runGH(args []string) CommandResult {
	if len(args) < 4 || args[0] != "api" {
		return failedCommand("unsupported gh call")
	}
	method := optionAfter(args, "--method")
	endpoint := args[len(args)-1]
	switch {
	case method == "GET" && strings.Contains(endpoint, "/git/ref/tags/"):
		value := map[string]any{"ref": "refs/tags/" + runner.tag, "url": "https://api.invalid/ref", "object": map[string]any{"type": "tag", "sha": strings.Repeat("d", 40), "url": "https://api.invalid/tag"}}
		return jsonCommand(value)
	case method == "GET" && strings.Contains(endpoint, "/git/tags/"):
		value := map[string]any{"tag": runner.tag, "node_id": "fixture", "object": map[string]any{"type": "commit", "sha": runner.commit, "url": "https://api.invalid/commit"}, "verification": map[string]any{"verified": runner.verifiedTag, "reason": map[bool]string{true: "valid", false: "unsigned"}[runner.verifiedTag], "signature": "fixture-signature", "payload": "fixture-payload", "verified_at": "2026-07-18T12:00:00Z"}}
		return jsonCommand(value)
	case method == "GET" && strings.Contains(endpoint, "/releases/tags/"):
		if runner.release == nil {
			return CommandResult{ExitCode: 1, Err: errors.New("exit status 1"), Stderr: []byte("HTTP 404: Not Found")}
		}
		return jsonCommand(runner.release)
	case method == "POST" && strings.HasSuffix(endpoint, "/releases"):
		if runner.release != nil {
			return failedCommand("release already exists")
		}
		var payload releasePayload
		if err := readInputJSON(args, &payload); err != nil {
			return failedCommand(err.Error())
		}
		runner.release = &githubRelease{ID: 77, TagName: payload.TagName, Target: payload.TargetCommitish, Name: payload.Name, Body: payload.Body, Draft: true, Prerelease: false, HTMLURL: "https://github.com/fixture/releases/tag/" + runner.tag}
		return jsonCommand(runner.release)
	case method == "POST" && strings.Contains(endpoint, "/assets?name="):
		nameValue, _ := url.QueryUnescape(strings.SplitN(endpoint, "?name=", 2)[1])
		if nameValue == runner.failUpload {
			return failedCommand("injected upload failure")
		}
		path := optionAfter(args, "--input")
		data, err := os.ReadFile(path)
		if err != nil {
			return failedCommand(err.Error())
		}
		record, ok := artifactRecord(runner.report.Artifacts, nameValue)
		if !ok {
			return failedCommand("undeclared upload")
		}
		runner.nextAssetID++
		asset := githubAsset{ID: runner.nextAssetID, Name: nameValue, State: "uploaded", Size: int64(len(data)), Digest: "sha256:" + record.SHA256}
		runner.uploaded[asset.ID] = append([]byte(nil), data...)
		runner.release.Assets = append(runner.release.Assets, asset)
		return jsonCommand(asset)
	case method == "PATCH" && strings.Contains(endpoint, "/releases/"):
		if runner.release == nil {
			return failedCommand("release missing")
		}
		published := "2026-07-18T12:10:00Z"
		runner.release.Draft = false
		runner.release.PublishedAt = &published
		return jsonCommand(runner.release)
	case method == "DELETE" && strings.Contains(endpoint, "/releases/"):
		runner.release = nil
		runner.uploaded = map[int64][]byte{}
		return CommandResult{ExitCode: 0}
	default:
		return failedCommand("unsupported gh api endpoint: " + endpoint)
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 7, 18, 12, 0, 0, 123456789, time.UTC)
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	result := (OSRunner{}).Run(context.Background(), root, nil, "git", append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgSign=false"}, args...)...)
	if result.Err != nil {
		t.Fatalf("git %v: %v: %s", args, result.Err, result.Stderr)
	}
	return string(result.Stdout)
}

func writeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func writeJSONFile(t *testing.T, path string, value any, mode os.FileMode) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, append(data, '\n'), mode)
}

func hashRecord(t *testing.T, directory, kind, name, goos, arch string) releaseartifactbuild.ArtifactRecord {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		t.Fatal(err)
	}
	h256 := sha256.Sum256(data)
	h512 := sha512.Sum512(data)
	return releaseartifactbuild.ArtifactRecord{Kind: kind, OS: goos, Arch: arch, Name: name, Size: int64(len(data)), SHA256: hex.EncodeToString(h256[:]), SHA512: hex.EncodeToString(h512[:])}
}

func writeManifestFixture(t *testing.T, path string, records []releaseartifactbuild.ArtifactRecord, sha256Manifest bool) {
	t.Helper()
	var builder strings.Builder
	for _, record := range records {
		digest := record.SHA512
		if sha256Manifest {
			digest = record.SHA256
		}
		fmt.Fprintf(&builder, "%s  %s\n", digest, record.Name)
	}
	writeFile(t, path, []byte(builder.String()), 0o644)
}

func optionAfter(args []string, option string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == option {
			return args[index+1]
		}
	}
	return ""
}

func readInputJSON(args []string, target any) error {
	path := optionAfter(args, "--input")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func jsonCommand(value any) CommandResult {
	data, err := json.Marshal(value)
	if err != nil {
		return CommandResult{ExitCode: -1, Err: err}
	}
	return CommandResult{Stdout: data, ExitCode: 0}
}

func failedCommand(message string) CommandResult {
	return CommandResult{ExitCode: 1, Err: errors.New("exit status 1"), Stderr: []byte(message)}
}

func artifactRecord(records []releaseartifactbuild.ArtifactRecord, name string) (releaseartifactbuild.ArtifactRecord, bool) {
	for _, record := range records {
		if record.Name == name {
			return record, true
		}
	}
	return releaseartifactbuild.ArtifactRecord{}, false
}

func assertPrivateEvidence(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("evidence mode = %o, want 600", info.Mode().Perm())
	}
}
