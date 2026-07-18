package releaseartifact

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

var bootstrapReleaseTagPattern = regexp.MustCompile(`^isras-v([0-9]+\.[0-9]+\.[0-9]+)$`)

type bootstrapAssetSpec struct {
	Kind string
	OS   string
	Arch string
}

var bootstrapAssetSpecs = map[string]bootstrapAssetSpec{
	"SHA256SUMS":                     {Kind: "sha256-manifest"},
	"SHA512SUMS":                     {Kind: "sha512-manifest"},
	"isras-contracts.tar.gz":         {Kind: "contracts"},
	"isras-project-framework.tar.gz": {Kind: "framework"},
	"isras-validator-linux-amd64":    {Kind: "validator", OS: "linux", Arch: "amd64"},
	"provenance.json":                {Kind: "provenance"},
}

// Bootstrap contains release identity and artifact declarations that have been
// discovered from, downloaded from, and fully verified against one immutable
// GitHub release. A caller may use this data to construct a new project pin.
type Bootstrap struct {
	Standard  projectpin.Standard
	Artifacts []projectpin.Artifact
	Report    Report
}

// BootstrapGitHub discovers and verifies the exact release asset set without
// requiring a pre-existing project pin. This closes the initialization
// chicken-and-egg boundary while preserving the same verification checks used
// after adoption.
func BootstrapGitHub(ctx context.Context, releaseTag string) (Bootstrap, error) {
	return (GitHubClient{Run: runGH}).Bootstrap(ctx, releaseTag)
}

func (client GitHubClient) Bootstrap(ctx context.Context, releaseTag string) (Bootstrap, error) {
	match := bootstrapReleaseTagPattern.FindStringSubmatch(releaseTag)
	if match == nil {
		return Bootstrap{}, errors.New("release tag must be isras-vMAJOR.MINOR.PATCH")
	}
	if client.Run == nil {
		return Bootstrap{}, errors.New("GitHub command runner is unavailable")
	}

	version := match[1]
	report := newReport(
		"github-release-bootstrap",
		projectpin.SourceRepository+"@"+releaseTag,
		releaseTag,
		"",
		time.Now().UTC(),
	)

	_, remoteSHA256, remoteSizes, err := client.inspectBootstrapRelease(ctx, releaseTag)
	if err != nil {
		report.ReleaseRecord = StatusFail
		report.Failure = err.Error()
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Report: report}, err
	}
	report.ReleaseRecord = StatusPass
	report.AssetInventory = StatusPass

	sourceCommit, err := client.inspectBootstrapTag(ctx, releaseTag)
	if err != nil {
		report.SignedTag = StatusFail
		report.Failure = err.Error()
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Report: report}, err
	}
	report.SourceCommit = sourceCommit
	report.SignedTag = StatusPass

	directory, err := os.MkdirTemp("", "isras-release-bootstrap-")
	if err != nil {
		return Bootstrap{Report: report}, errors.New("create temporary release directory")
	}
	defer os.RemoveAll(directory)

	names := bootstrapAssetNames()
	args := []string{"release", "download", releaseTag, "--repo", repositorySlug(projectpin.SourceRepository), "--dir", directory}
	for _, name := range names {
		args = append(args, "--pattern", name)
	}
	if _, err := client.Run(ctx, args...); err != nil {
		report.AssetAcquisition = StatusFail
		report.Failure = "download exact release asset set"
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Report: report}, errors.New(report.Failure)
	}
	report.AssetAcquisition = StatusPass

	artifacts, err := inspectBootstrapDirectory(directory, remoteSHA256, remoteSizes)
	if err == nil {
		err = verifyBootstrapFramework(filepath.Join(directory, "isras-project-framework.tar.gz"))
	}
	if err != nil {
		report.AssetInventory = StatusFail
		report.Failure = err.Error()
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Report: report}, err
	}

	standard := projectpin.Standard{
		Profile:          projectpin.Profile,
		Version:          version,
		ReleaseTag:       releaseTag,
		SourceRepository: projectpin.SourceRepository,
		SourceCommit:     sourceCommit,
	}
	candidate := projectpin.Pin{
		SchemaVersion: projectpin.SchemaVersion,
		Project:       projectpin.Project{Repository: "github.com/Iron-Signal-Systems/bootstrap-verification"},
		Standard:      standard,
		Artifacts:     artifacts,
		Workflow: projectpin.Workflow{
			Repository: projectpin.SourceRepository,
			Path:       projectpin.ReusableWorkflowPath,
			Commit:     sourceCommit,
		},
		Profiles: []string{"go"},
		Commands: projectpin.DefaultGoCommands(),
		Evidence: projectpin.Evidence{Directory: ".local/isras"},
	}
	if _, err := projectpin.CanonicalJSON(candidate); err != nil {
		report.Failure = err.Error()
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Standard: standard, Artifacts: artifacts, Report: report}, err
	}

	local, err := VerifyDirectory(candidate, directory)
	copyLocalVerification(&report, local)
	if err != nil {
		report.Failure = err.Error()
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Standard: standard, Artifacts: artifacts, Report: report}, err
	}

	finalSizes, err := client.verifyReleaseIdentity(ctx, candidate, &report)
	if err != nil {
		report.Failure = "release identity changed or failed final verification"
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Standard: standard, Artifacts: artifacts, Report: report}, errors.New(report.Failure)
	}
	if !sameRemoteSizes(remoteSizes, finalSizes) {
		report.AssetInventory = StatusFail
		report.Failure = "release asset inventory changed during bootstrap verification"
		report.FinishedAt = time.Now().UTC()
		return Bootstrap{Standard: standard, Artifacts: artifacts, Report: report}, errors.New(report.Failure)
	}
	for index := range report.Artifacts {
		report.Artifacts[index].RemoteSize = finalSizes[report.Artifacts[index].Name]
		if report.Artifacts[index].Size != report.Artifacts[index].RemoteSize {
			report.AssetInventory = StatusFail
			report.Failure = "downloaded artifact size does not match the final release record"
			report.FinishedAt = time.Now().UTC()
			return Bootstrap{Standard: standard, Artifacts: artifacts, Report: report}, errors.New(report.Failure)
		}
	}

	report.ExecutionAuthorization = AuthorizationGranted
	report.FinishedAt = time.Now().UTC()
	return Bootstrap{Standard: standard, Artifacts: artifacts, Report: report}, nil
}

func (client GitHubClient) inspectBootstrapRelease(ctx context.Context, tag string) (releaseRecord, map[string]string, map[string]int64, error) {
	data, err := client.Run(ctx, "api", "repos/"+repositorySlug(projectpin.SourceRepository)+"/releases/tags/"+tag)
	if err != nil {
		return releaseRecord{}, nil, nil, errors.New("read GitHub release record")
	}
	var release releaseRecord
	if err := decodeGitHubJSON(data, &release); err != nil {
		return releaseRecord{}, nil, nil, errors.New("parse GitHub release record")
	}
	if release.TagName != tag || release.Draft || release.Prerelease {
		return releaseRecord{}, nil, nil, errors.New("release is missing, draft, prerelease, or has the wrong tag")
	}
	if len(release.Assets) != len(bootstrapAssetSpecs) {
		return releaseRecord{}, nil, nil, errors.New("release does not contain the exact six-asset bootstrap set")
	}

	sha256Values := make(map[string]string, len(release.Assets))
	sizes := make(map[string]int64, len(release.Assets))
	for _, asset := range release.Assets {
		if _, ok := bootstrapAssetSpecs[asset.Name]; !ok {
			return releaseRecord{}, nil, nil, errors.New("release contains an unexpected asset")
		}
		if _, duplicate := sizes[asset.Name]; duplicate {
			return releaseRecord{}, nil, nil, errors.New("release contains a duplicate asset")
		}
		if asset.State != "uploaded" || asset.Size <= 0 || asset.Size > maxArtifactSize {
			return releaseRecord{}, nil, nil, errors.New("release contains an unavailable or oversized asset")
		}
		if !strings.HasPrefix(asset.Digest, "sha256:") {
			return releaseRecord{}, nil, nil, errors.New("release asset is missing its GitHub SHA-256 digest")
		}
		digest := strings.TrimPrefix(asset.Digest, "sha256:")
		if !validLowerHex(digest, sha256.Size*2) {
			return releaseRecord{}, nil, nil, errors.New("release asset has an invalid GitHub SHA-256 digest")
		}
		sha256Values[asset.Name] = digest
		sizes[asset.Name] = asset.Size
	}
	return release, sha256Values, sizes, nil
}

func (client GitHubClient) inspectBootstrapTag(ctx context.Context, tagName string) (string, error) {
	slug := repositorySlug(projectpin.SourceRepository)
	data, err := client.Run(ctx, "api", "repos/"+slug+"/git/ref/tags/"+tagName)
	if err != nil {
		return "", errors.New("read release tag reference")
	}
	var reference gitReference
	if err := decodeGitHubJSON(data, &reference); err != nil {
		return "", errors.New("parse release tag reference")
	}
	if reference.Ref != "refs/tags/"+tagName || reference.Object.Type != "tag" || reference.Object.SHA == "" {
		return "", errors.New("release tag is not an annotated tag object")
	}
	data, err = client.Run(ctx, "api", "repos/"+slug+"/git/tags/"+reference.Object.SHA)
	if err != nil {
		return "", errors.New("read annotated release tag object")
	}
	var tag annotatedTag
	if err := decodeGitHubJSON(data, &tag); err != nil {
		return "", errors.New("parse annotated release tag object")
	}
	if tag.Tag != tagName || tag.Object.Type != "commit" || !validLowerHex(tag.Object.SHA, 40) {
		return "", errors.New("annotated release tag has an invalid target")
	}
	if !tag.Verification.Verified || tag.Verification.Reason != "valid" || tag.Verification.Signature == "" || tag.Verification.Payload == "" || tag.Verification.VerifiedAt == "" {
		return "", errors.New("annotated release tag is not verified by GitHub")
	}
	return tag.Object.SHA, nil
}

func verifyBootstrapFramework(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return errors.New("open project framework archive")
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return errors.New("open project framework gzip stream")
	}
	defer compressed.Close()

	reader := tar.NewReader(compressed)
	found := false
	seen := make(map[string]bool)
	entries := 0
	var total int64
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.New("read project framework archive")
		}
		entries++
		if entries > 4096 {
			return errors.New("project framework archive contains too many entries")
		}
		name := header.Name
		if name == "" || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.HasPrefix(name, "../") || strings.Contains(name, "\\") {
			return errors.New("project framework archive contains an unsafe path")
		}
		if !strings.HasPrefix(name, "isras-project-framework/") || seen[name] {
			return errors.New("project framework archive contains an unexpected or duplicate path")
		}
		seen[name] = true
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return errors.New("project framework archive contains a non-regular entry")
		}
		if header.Size < 0 || header.Size > 8*1024*1024 {
			return errors.New("project framework archive entry violates the size boundary")
		}
		total += header.Size
		if total > 128*1024*1024 {
			return errors.New("project framework archive exceeds the expanded size boundary")
		}
		if name == "isras-project-framework/"+projectpin.ReusableWorkflowPath {
			if found {
				return errors.New("project framework archive contains a duplicate reusable workflow")
			}
			if header.Size == 0 {
				return errors.New("project framework reusable workflow is empty")
			}
			found = true
		}
	}
	if !found {
		return errors.New("release framework does not contain the pinned reusable validation workflow")
	}
	return nil
}

func inspectBootstrapDirectory(directory string, remoteSHA256 map[string]string, remoteSizes map[string]int64) ([]projectpin.Artifact, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, errors.New("read downloaded release directory")
	}
	if len(entries) != len(bootstrapAssetSpecs) {
		return nil, errors.New("downloaded release directory does not contain the exact six-asset set")
	}
	for _, entry := range entries {
		if _, ok := bootstrapAssetSpecs[entry.Name()]; !ok {
			return nil, errors.New("downloaded release directory contains an unexpected entry")
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil, errors.New("downloaded release directory contains a directory or symbolic link")
		}
	}

	names := bootstrapAssetNames()
	artifacts := make([]projectpin.Artifact, 0, len(names))
	var total int64
	for _, name := range names {
		d256, d512, size, err := digestBootstrapFile(filepath.Join(directory, name))
		if err != nil {
			return nil, fmt.Errorf("inspect release asset %s: %w", name, err)
		}
		if size != remoteSizes[name] {
			return nil, errors.New("downloaded release asset size does not match the release record")
		}
		if d256 != remoteSHA256[name] {
			return nil, errors.New("downloaded release asset SHA-256 does not match the release record")
		}
		total += size
		if total > maxTotalArtifactSize {
			return nil, errors.New("downloaded release asset set exceeds the total size limit")
		}
		spec := bootstrapAssetSpecs[name]
		artifacts = append(artifacts, projectpin.Artifact{
			Kind: spec.Kind, OS: spec.OS, Arch: spec.Arch, Name: name,
			SHA256: d256, SHA512: d512,
		})
	}
	return artifacts, nil
}

func digestBootstrapFile(path string) (string, string, int64, error) {
	before, err := os.Lstat(path)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return "", "", 0, errors.New("asset is not a regular file")
	}
	if before.Size() <= 0 || before.Size() > maxArtifactSize {
		return "", "", 0, errors.New("asset violates the size boundary")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", "", 0, errors.New("open asset")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return "", "", 0, errors.New("asset changed during open")
	}
	h256 := sha256.New()
	h512 := sha512.New()
	written, err := io.Copy(io.MultiWriter(h256, h512), io.LimitReader(file, maxArtifactSize+1))
	if err != nil || written != before.Size() || written > maxArtifactSize {
		return "", "", 0, errors.New("asset changed during hashing")
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() {
		return "", "", 0, errors.New("asset changed during hashing")
	}
	return hex.EncodeToString(h256.Sum(nil)), hex.EncodeToString(h512.Sum(nil)), written, nil
}

func bootstrapAssetNames() []string {
	names := make([]string, 0, len(bootstrapAssetSpecs))
	for name := range bootstrapAssetSpecs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func validLowerHex(value string, length int) bool {
	if len(value) != length || strings.ToLower(value) != value || strings.Trim(value, "0") == "" {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func copyLocalVerification(target *Report, source Report) {
	target.AssetInventory = source.AssetInventory
	target.PinDigests = source.PinDigests
	target.SHA256Manifest = source.SHA256Manifest
	target.SHA512Manifest = source.SHA512Manifest
	target.Provenance = source.Provenance
	target.Artifacts = source.Artifacts
}
