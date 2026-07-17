package releaseartifact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

const maxGHOutput = 4 * 1024 * 1024

type CommandRunner func(context.Context, ...string) ([]byte, error)

type GitHubClient struct {
	Run CommandRunner
}

type releaseRecord struct {
	TagName    string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type gitReference struct {
	Ref    string `json:"ref"`
	Object struct {
		Type string `json:"type"`
		SHA  string `json:"sha"`
	} `json:"object"`
}

type annotatedTag struct {
	Tag    string `json:"tag"`
	Object struct {
		Type string `json:"type"`
		SHA  string `json:"sha"`
	} `json:"object"`
	Verification struct {
		Verified   bool   `json:"verified"`
		Reason     string `json:"reason"`
		Signature  string `json:"signature"`
		Payload    string `json:"payload"`
		VerifiedAt string `json:"verified_at"`
	} `json:"verification"`
}

func VerifyGitHub(ctx context.Context, pin projectpin.Pin) (Report, error) {
	client := GitHubClient{Run: runGH}
	return client.Verify(ctx, pin)
}

func (client GitHubClient) Verify(ctx context.Context, pin projectpin.Pin) (Report, error) {
	report := newReport("github-release", pin.Standard.SourceRepository+"@"+pin.Standard.ReleaseTag, pin.Standard.ReleaseTag, pin.Standard.SourceCommit, time.Now().UTC())
	if client.Run == nil {
		return finishFailure(report, "GitHub command runner is unavailable")
	}

	remoteSizes, err := client.verifyReleaseIdentity(ctx, pin, &report)
	if err != nil {
		return finishFailure(report, err.Error())
	}

	directory, err := os.MkdirTemp("", "isras-release-artifacts-")
	if err != nil {
		return finishFailure(report, "create temporary artifact directory")
	}
	defer os.RemoveAll(directory)

	args := []string{"release", "download", pin.Standard.ReleaseTag, "--repo", repositorySlug(pin.Standard.SourceRepository), "--dir", directory}
	artifacts := append([]projectpin.Artifact(nil), pin.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	for _, artifact := range artifacts {
		args = append(args, "--pattern", artifact.Name)
	}
	if _, err := client.Run(ctx, args...); err != nil {
		report.AssetAcquisition = StatusFail
		return finishFailure(report, "download declared release assets")
	}
	report.AssetAcquisition = StatusPass

	local, err := VerifyDirectory(pin, directory)
	report.AssetInventory = local.AssetInventory
	report.PinDigests = local.PinDigests
	report.SHA256Manifest = local.SHA256Manifest
	report.SHA512Manifest = local.SHA512Manifest
	report.Provenance = local.Provenance
	report.Artifacts = local.Artifacts
	for index := range report.Artifacts {
		report.Artifacts[index].RemoteSize = remoteSizes[report.Artifacts[index].Name]
		if report.Artifacts[index].Size != report.Artifacts[index].RemoteSize {
			report.AssetInventory = StatusFail
			return finishFailure(report, "downloaded artifact size does not match the release record")
		}
	}
	if err != nil {
		return finishFailure(report, err.Error())
	}

	finalSizes, err := client.verifyReleaseIdentity(ctx, pin, &report)
	if err != nil {
		return finishFailure(report, "release identity changed or failed final verification")
	}
	if !sameRemoteSizes(remoteSizes, finalSizes) {
		report.AssetInventory = StatusFail
		return finishFailure(report, "release asset inventory changed during verification")
	}

	report.ExecutionAuthorization = AuthorizationGranted
	report.FinishedAt = time.Now().UTC()
	return report, nil
}

func (client GitHubClient) verifyReleaseIdentity(ctx context.Context, pin projectpin.Pin, report *Report) (map[string]int64, error) {
	slug := repositorySlug(pin.Standard.SourceRepository)
	data, err := client.Run(ctx, "api", "repos/"+slug+"/releases/tags/"+pin.Standard.ReleaseTag)
	if err != nil {
		return nil, errors.New("read the pinned GitHub release record")
	}
	var release releaseRecord
	if err := decodeGitHubJSON(data, &release); err != nil {
		return nil, errors.New("parse the pinned GitHub release record")
	}
	if release.TagName != pin.Standard.ReleaseTag || release.Draft || release.Prerelease {
		report.ReleaseRecord = StatusFail
		return nil, errors.New("pinned GitHub release is missing, draft, prerelease, or has the wrong tag")
	}
	report.ReleaseRecord = StatusPass

	declared := make(map[string]projectpin.Artifact, len(pin.Artifacts))
	for _, artifact := range pin.Artifacts {
		declared[artifact.Name] = artifact
	}
	if len(release.Assets) != len(declared) {
		report.AssetInventory = StatusFail
		return nil, errors.New("GitHub release asset inventory does not match the project pin")
	}
	remoteSizes := make(map[string]int64, len(release.Assets))
	for _, asset := range release.Assets {
		artifact, ok := declared[asset.Name]
		if !ok || asset.State != "uploaded" || asset.Size <= 0 || asset.Size > maxArtifactSize {
			report.AssetInventory = StatusFail
			return nil, errors.New("GitHub release contains an undeclared, unavailable, or oversized asset")
		}
		if _, duplicate := remoteSizes[asset.Name]; duplicate {
			report.AssetInventory = StatusFail
			return nil, errors.New("GitHub release contains a duplicate asset name")
		}
		if asset.Digest != "sha256:"+artifact.SHA256 {
			report.AssetInventory = StatusFail
			return nil, errors.New("GitHub release asset digest does not match the project pin")
		}
		remoteSizes[asset.Name] = asset.Size
	}
	data, err = client.Run(ctx, "api", "repos/"+slug+"/git/ref/tags/"+pin.Standard.ReleaseTag)
	if err != nil {
		return nil, errors.New("read the pinned release tag reference")
	}
	var reference gitReference
	if err := decodeGitHubJSON(data, &reference); err != nil {
		return nil, errors.New("parse the pinned release tag reference")
	}
	if reference.Ref != "refs/tags/"+pin.Standard.ReleaseTag || reference.Object.Type != "tag" || reference.Object.SHA == "" {
		report.SignedTag = StatusFail
		return nil, errors.New("pinned release tag is not an annotated tag object")
	}

	data, err = client.Run(ctx, "api", "repos/"+slug+"/git/tags/"+reference.Object.SHA)
	if err != nil {
		return nil, errors.New("read the annotated release tag object")
	}
	var tag annotatedTag
	if err := decodeGitHubJSON(data, &tag); err != nil {
		return nil, errors.New("parse the annotated release tag object")
	}
	if tag.Tag != pin.Standard.ReleaseTag || tag.Object.Type != "commit" || tag.Object.SHA != pin.Standard.SourceCommit {
		report.SignedTag = StatusFail
		return nil, errors.New("signed release tag does not point to the pinned source commit")
	}
	if !tag.Verification.Verified || tag.Verification.Reason != "valid" || tag.Verification.Signature == "" || tag.Verification.Payload == "" || tag.Verification.VerifiedAt == "" {
		report.SignedTag = StatusFail
		return nil, errors.New("annotated release tag is not verified by GitHub")
	}
	report.SignedTag = StatusPass
	return remoteSizes, nil
}

func sameRemoteSizes(first, second map[string]int64) bool {
	if len(first) != len(second) {
		return false
	}
	for name, size := range first {
		if second[name] != size {
			return false
		}
	}
	return true
}

func repositorySlug(repository string) string {
	return strings.TrimPrefix(repository, "github.com/")
}

func decodeGitHubJSON(data []byte, target any) error {
	if err := rejectDuplicateJSONFields(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return requireJSONEOF(decoder)
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (value *boundedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := value.limit - value.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
			value.truncated = true
		}
		_, _ = value.buffer.Write(data)
	} else if original > 0 {
		value.truncated = true
	}
	return original, nil
}

func runGH(ctx context.Context, args ...string) ([]byte, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, errors.New("GitHub CLI is unavailable")
	}
	stdout := &boundedBuffer{limit: maxGHOutput}
	stderr := &boundedBuffer{limit: maxGHOutput}
	command := exec.CommandContext(ctx, "gh", args...)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("GitHub CLI command failed: %w", err)
	}
	if stdout.truncated || stderr.truncated {
		return nil, errors.New("GitHub CLI output exceeded its safety boundary")
	}
	return stdout.buffer.Bytes(), nil
}
