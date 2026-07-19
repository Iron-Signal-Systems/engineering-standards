package releasepublication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifactbuild"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
)

const (
	defaultBranch = "dev"
	defaultRemote = "origin"
	maxNotesSize  = 1024 * 1024
	maxReportSize = 2 * 1024 * 1024
)

var (
	stableVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	commitPattern        = regexp.MustCompile(`^[0-9a-f]{40}$`)
	digest256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	digest512Pattern     = regexp.MustCompile(`^[0-9a-f]{128}$`)
	repositoryPattern    = regexp.MustCompile(`^Iron-Signal-Systems/[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
)

type Publisher struct {
	Runner Runner
	Now    func() time.Time
}

type sourceBoundary struct {
	Root             string
	Branch           string
	Remote           string
	Commit           string
	Origin           string
	Version          string
	Tag              string
	GitHubRepository string
	Title            string
	NotesFile        string
	NotesBody        string
	Artifacts        string
	BuildEvidence    string
	BuildReport      releaseartifactbuild.Result
	Pin              projectpin.Pin
}

type githubReference struct {
	Ref    string `json:"ref"`
	Object struct {
		Type string `json:"type"`
		SHA  string `json:"sha"`
	} `json:"object"`
}

type githubTag struct {
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

type githubRelease struct {
	ID          int64         `json:"id"`
	TagName     string        `json:"tag_name"`
	Target      string        `json:"target_commitish"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt *string       `json:"published_at"`
	HTMLURL     string        `json:"html_url"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type releasePayload struct {
	TagName         string `json:"tag_name,omitempty"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	Name            string `json:"name,omitempty"`
	Body            string `json:"body,omitempty"`
	Draft           *bool  `json:"draft,omitempty"`
	Prerelease      *bool  `json:"prerelease,omitempty"`
	MakeLatest      string `json:"make_latest,omitempty"`
}

func Run(ctx context.Context, options Options) (Result, error) {
	publisher := Publisher{Runner: OSRunner{}, Now: nowUTC}
	return publisher.Run(ctx, options)
}

func (publisher Publisher) Run(ctx context.Context, options Options) (result Result, runErr error) {
	if publisher.Runner == nil {
		publisher.Runner = OSRunner{}
	}
	if publisher.Now == nil {
		publisher.Now = nowUTC
	}
	applyDefaults(&options)
	if err := validateOptions(options); err != nil {
		return Result{}, err
	}
	started := publisher.Now().UTC()
	result = newResult(options.Action, started)

	boundary, err := publisher.inspectLocal(ctx, options, &result)
	if err != nil {
		result.Failure = safeFailure(err)
		result.FinishedAt = publisher.Now().UTC()
		return result, err
	}
	result.RepositoryRoot = boundary.Root
	result.SourceRepository = releaseartifactbuild.SourceRepository
	result.SourceCommit = boundary.Commit
	result.Version = boundary.Version
	result.ReleaseTag = boundary.Tag
	result.GitHubRepository = boundary.GitHubRepository
	result.ArtifactDirectory = boundary.Artifacts
	result.BuildEvidence = boundary.BuildEvidence
	result.NotesFile = boundary.NotesFile
	result.Title = boundary.Title
	result.Artifacts = artifactResults(boundary.BuildReport.Artifacts)

	if err := prepareEvidence(&result, boundary.Root, boundary.Tag, started); err != nil {
		return result, err
	}
	if err := writeEvidence(result); err != nil {
		return result, err
	}

	defer func() {
		if runErr != nil {
			result.Failure = safeFailure(runErr)
		}
		result.FinishedAt = publisher.Now().UTC()
		if evidenceErr := writeEvidence(result); evidenceErr != nil && runErr == nil {
			runErr = evidenceErr
			result.Failure = safeFailure(evidenceErr)
		}
	}()

	if err := publisher.verifyRemoteSource(ctx, boundary); err != nil {
		return result, err
	}
	result.RemoteTag = StatusPass
	if err := writeEvidence(result); err != nil {
		return result, err
	}

	release, exists, err := publisher.readRelease(ctx, boundary)
	if err != nil {
		return result, err
	}
	if exists {
		return result, fmt.Errorf("GitHub Release %s already exists and will not be replaced", boundary.Tag)
	}
	result.ReleaseAbsence = StatusPass
	if err := writeEvidence(result); err != nil {
		return result, err
	}

	if options.Action == ActionCheck {
		result.FinishedAt = publisher.Now().UTC()
		return result, nil
	}
	if !options.Confirm {
		return result, errors.New("publish requires --confirm because it creates and publishes a GitHub Release")
	}

	draftCreated := false
	published := false
	defer func() {
		if runErr == nil || !draftCreated || published {
			return
		}
		cleanupErr := publisher.cleanupDraft(ctx, boundary, result.ReleaseID)
		if cleanupErr != nil {
			result.Cleanup = StatusFail
			runErr = errors.Join(runErr, cleanupErr)
		} else {
			result.Cleanup = StatusPass
		}
		_ = writeEvidence(result)
	}()

	release, err = publisher.createDraft(ctx, boundary)
	if err != nil {
		observed, observedExists, readErr := publisher.readRelease(ctx, boundary)
		if readErr == nil && observedExists && verifyCleanupDraft(observed, boundary, boundary.BuildReport.Artifacts) == nil {
			draftCreated = true
			result.ReleaseID = observed.ID
			result.ReleaseURL = observed.HTMLURL
		}
		return result, err
	}
	draftCreated = true
	result.ReleaseID = release.ID
	result.ReleaseURL = release.HTMLURL
	result.DraftCreation = StatusPass
	if err := writeEvidence(result); err != nil {
		return result, err
	}

	if err := verifyReleaseRecord(release, boundary, true, false, nil); err != nil {
		return result, err
	}
	if err := publisher.uploadAssets(ctx, boundary, &result); err != nil {
		return result, err
	}
	result.AssetUpload = StatusPass
	if err := writeEvidence(result); err != nil {
		return result, err
	}

	release, exists, err = publisher.readReleaseByID(ctx, boundary, result.ReleaseID)
	if err != nil || !exists {
		if err == nil {
			err = errors.New("draft release disappeared after asset upload")
		}
		return result, err
	}
	if err := verifyReleaseRecord(release, boundary, true, false, boundary.BuildReport.Artifacts); err != nil {
		return result, err
	}
	if err := publisher.downloadAndVerify(ctx, boundary, release, &result); err != nil {
		return result, err
	}
	result.DraftVerification = StatusPass
	if err := writeEvidence(result); err != nil {
		return result, err
	}
	prePublishBoundary, err := publisher.inspectLocal(ctx, options, &Result{})
	if err != nil || !sameSourceBoundary(boundary, prePublishBoundary) {
		if err == nil {
			err = errors.New("local release source or artifact boundary changed before publication")
		}
		return result, err
	}

	if err := publisher.publishDraft(ctx, boundary, release.ID); err != nil {
		observed, observedExists, readErr := publisher.readReleaseByID(ctx, boundary, release.ID)
		if readErr == nil && observedExists && verifyReleaseRecord(observed, boundary, false, true, boundary.BuildReport.Artifacts) == nil {
			release = observed
		} else {
			return result, err
		}
	}
	published = true
	result.Publication = StatusPass

	release, exists, err = publisher.readReleaseByID(ctx, boundary, result.ReleaseID)
	if err != nil || !exists {
		if err == nil {
			err = errors.New("published release was not found during final verification")
		}
		return result, err
	}
	if err := verifyReleaseRecord(release, boundary, false, true, boundary.BuildReport.Artifacts); err != nil {
		return result, err
	}
	if err := publisher.downloadAndVerify(ctx, boundary, release, &result); err != nil {
		return result, err
	}
	finalBoundary, err := publisher.inspectLocal(ctx, options, &Result{})
	if err != nil || !sameSourceBoundary(boundary, finalBoundary) {
		if err == nil {
			err = errors.New("local release source or artifact boundary changed during publication")
		}
		return result, err
	}
	if err := publisher.verifyRemoteSource(ctx, boundary); err != nil {
		return result, errors.New("release source identity changed during publication")
	}
	result.FinalVerification = StatusPass
	result.ReleaseURL = release.HTMLURL
	result.FinishedAt = publisher.Now().UTC()
	return result, nil
}

func applyDefaults(options *Options) {
	if options.Branch == "" {
		options.Branch = defaultBranch
	}
	if options.Remote == "" {
		options.Remote = defaultRemote
	}
}

func validateOptions(options Options) error {
	if options.Action != ActionCheck && options.Action != ActionPublish {
		return fmt.Errorf("unsupported publication action %q", options.Action)
	}
	if options.ExpectedVersion != "" && !stableVersionPattern.MatchString(options.ExpectedVersion) {
		return errors.New("expected version must be MAJOR.MINOR.PATCH")
	}
	for name, value := range map[string]string{"branch": options.Branch, "remote": options.Remote, "GitHub repository": options.GitHubRepository, "title": options.Title} {
		if len(value) > 512 || strings.ContainsAny(value, "\x00\r\n") {
			return fmt.Errorf("%s is invalid or oversized", name)
		}
	}
	return nil
}

func newResult(action Action, started time.Time) Result {
	return Result{
		SchemaVersion:     1,
		StartedAt:         started,
		FinishedAt:        started,
		Action:            string(action),
		LocalVerification: StatusNotPerformed,
		RemoteTag:         StatusNotPerformed,
		ReleaseAbsence:    StatusNotPerformed,
		DraftCreation:     StatusNotPerformed,
		AssetUpload:       StatusNotPerformed,
		DraftVerification: StatusNotPerformed,
		Publication:       StatusNotPerformed,
		FinalVerification: StatusNotPerformed,
		Cleanup:           StatusNotPerformed,
	}
}

func (publisher Publisher) inspectLocal(ctx context.Context, options Options, result *Result) (sourceBoundary, error) {
	identity, err := repository.DiscoverFrom(ctx, options.Root)
	if err != nil {
		return sourceBoundary{}, err
	}
	if identity.Branch != options.Branch {
		return sourceBoundary{}, fmt.Errorf("current branch is %q; publication requires %q", identity.Branch, options.Branch)
	}
	status := publisher.Runner.Run(ctx, identity.Root, nil, "git", "status", "--porcelain=v1", "--untracked-files=all")
	if status.Err != nil {
		return sourceBoundary{}, commandFailure("inspect repository status", status)
	}
	if strings.TrimSpace(string(status.Stdout)) != "" {
		return sourceBoundary{}, errors.New("publication requires a completely clean repository")
	}
	if !commitPattern.MatchString(identity.Commit) {
		return sourceBoundary{}, errors.New("repository returned an invalid source commit")
	}
	verified := publisher.Runner.Run(ctx, identity.Root, nil, "git", "verify-commit", identity.Commit)
	if verified.Err != nil {
		return sourceBoundary{}, commandFailure("verify exact source commit signature", verified)
	}

	versionBytes, err := readBoundedRegular(filepath.Join(identity.Root, "VERSION"), 128)
	if err != nil {
		return sourceBoundary{}, errors.New("read VERSION")
	}
	version := strings.TrimSpace(string(versionBytes))
	if !stableVersionPattern.MatchString(version) {
		return sourceBoundary{}, fmt.Errorf("VERSION %q is not a stable MAJOR.MINOR.PATCH release", version)
	}
	if options.ExpectedVersion != "" && version != options.ExpectedVersion {
		return sourceBoundary{}, fmt.Errorf("VERSION is %s, but --version requested %s", version, options.ExpectedVersion)
	}
	tag := "isras-v" + version

	githubRepository, err := githubRepository(options.GitHubRepository, identity.Origin)
	if err != nil {
		return sourceBoundary{}, err
	}
	if githubRepository != "Iron-Signal-Systems/engineering-standards" {
		return sourceBoundary{}, errors.New("publication repository is not the canonical Engineering Standards repository")
	}

	notesPath := options.NotesFile
	if notesPath == "" {
		notesPath = filepath.Join(identity.Root, "docs", "releases", version+".md")
	}
	notesPath, err = secureRegularPath(identity.Root, notesPath, maxNotesSize)
	if err != nil {
		return sourceBoundary{}, fmt.Errorf("release notes: %w", err)
	}
	notes, err := readBoundedRegular(notesPath, maxNotesSize)
	if err != nil || len(bytes.TrimSpace(notes)) == 0 {
		return sourceBoundary{}, errors.New("release notes are unavailable or empty")
	}
	changelog, err := readBoundedRegular(filepath.Join(identity.Root, "CHANGELOG.md"), maxNotesSize)
	if err != nil || !bytes.Contains(changelog, []byte("## "+version+" —")) {
		return sourceBoundary{}, errors.New("CHANGELOG.md does not contain the exact stable release heading")
	}

	artifactDirectory := options.ArtifactDirectory
	if artifactDirectory == "" {
		artifactDirectory = filepath.Join(identity.Root, ".local", "releases", tag, "assets")
	}
	artifactDirectory, err = secureDirectoryPath(identity.Root, artifactDirectory)
	if err != nil {
		return sourceBoundary{}, fmt.Errorf("artifact directory: %w", err)
	}
	buildEvidence := options.BuildEvidence
	if buildEvidence == "" {
		buildEvidence = filepath.Join(identity.Root, ".local", "validation", "releases", tag, "artifact-build.json")
	}
	buildEvidence, err = secureRegularPath(identity.Root, buildEvidence, maxReportSize)
	if err != nil {
		return sourceBoundary{}, fmt.Errorf("artifact build evidence: %w", err)
	}
	report, err := loadBuildReport(buildEvidence)
	if err != nil {
		return sourceBoundary{}, err
	}
	reportedEvidence, evidenceErr := filepath.Abs(report.EvidenceJSON)
	if evidenceErr != nil || filepath.Clean(reportedEvidence) != filepath.Clean(buildEvidence) {
		return sourceBoundary{}, errors.New("artifact build evidence does not identify the selected evidence file")
	}
	textEvidence, textErr := secureRegularPath(identity.Root, report.EvidenceText, maxReportSize)
	if textErr != nil || filepath.Dir(textEvidence) != filepath.Dir(buildEvidence) {
		return sourceBoundary{}, errors.New("artifact build text evidence is unavailable or outside the evidence boundary")
	}
	if !pathWithin(identity.Root, artifactDirectory) || !pathWithin(identity.Root, buildEvidence) || !pathWithin(identity.Root, notesPath) {
		return sourceBoundary{}, errors.New("release inputs must remain inside the canonical repository")
	}
	if err := verifyBuildReport(report, identity, version, tag, artifactDirectory); err != nil {
		return sourceBoundary{}, err
	}
	pin := pinFromBuildReport(report)
	if _, err := releaseartifact.VerifyDirectory(pin, artifactDirectory); err != nil {
		return sourceBoundary{}, fmt.Errorf("verify deterministic artifact directory: %w", err)
	}
	if err := publisher.verifyValidatorIdentity(ctx, identity.Root, artifactDirectory, version, tag, identity.Commit); err != nil {
		return sourceBoundary{}, err
	}

	result.LocalVerification = StatusPass
	return sourceBoundary{
		Root:             identity.Root,
		Branch:           identity.Branch,
		Remote:           options.Remote,
		Commit:           identity.Commit,
		Origin:           identity.Origin,
		Version:          version,
		Tag:              tag,
		GitHubRepository: githubRepository,
		Title:            releaseTitle(options.Title, version),
		NotesFile:        notesPath,
		NotesBody:        string(notes),
		Artifacts:        artifactDirectory,
		BuildEvidence:    buildEvidence,
		BuildReport:      report,
		Pin:              pin,
	}, nil
}

func (publisher Publisher) verifyValidatorIdentity(ctx context.Context, root, artifacts, version, tag, commit string) error {
	path := filepath.Join(artifacts, releaseartifactbuild.ValidatorName)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o111 == 0 {
		return errors.New("release validator is not a regular executable file")
	}
	result := publisher.Runner.Run(ctx, root, minimalValidatorEnvironment(root), path, "version")
	if result.Err != nil {
		return commandFailure("execute release validator identity check", result)
	}
	output := string(result.Stdout)
	for _, expected := range []string{
		"Standard version:  " + version,
		"Ownership:         release-artifact",
		"Release tag:       " + tag,
		"Source repository: " + releaseartifactbuild.SourceRepository,
		"Source commit:     " + commit,
	} {
		if !strings.Contains(output, expected) {
			return errors.New("release validator embedded identity does not match the publication source")
		}
	}
	return nil
}

func (publisher Publisher) verifyRemoteSource(ctx context.Context, boundary sourceBoundary) error {
	branch := publisher.Runner.Run(ctx, boundary.Root, nil, "git", "ls-remote", "--heads", boundary.Remote, "refs/heads/"+boundary.Branch)
	if branch.Err != nil {
		return commandFailure("read authoritative remote branch", branch)
	}
	fields := strings.Fields(string(branch.Stdout))
	if len(fields) != 2 || fields[0] != boundary.Commit || fields[1] != "refs/heads/"+boundary.Branch {
		return errors.New("authoritative remote branch does not identify the exact publication commit")
	}

	referenceData, err := publisher.ghJSON(ctx, boundary, "read remote annotated tag reference", "repos/"+boundary.GitHubRepository+"/git/ref/tags/"+boundary.Tag)
	if err != nil {
		return err
	}
	var reference githubReference
	if err := decodeGitHubJSON(referenceData, &reference); err != nil {
		return errors.New("parse remote annotated tag reference")
	}
	if reference.Ref != "refs/tags/"+boundary.Tag || reference.Object.Type != "tag" || !commitPattern.MatchString(reference.Object.SHA) {
		return errors.New("remote release tag is not an annotated tag object")
	}
	tagData, err := publisher.ghJSON(ctx, boundary, "read remote annotated tag object", "repos/"+boundary.GitHubRepository+"/git/tags/"+reference.Object.SHA)
	if err != nil {
		return err
	}
	var tag githubTag
	if err := decodeGitHubJSON(tagData, &tag); err != nil {
		return errors.New("parse remote annotated tag object")
	}
	if tag.Tag != boundary.Tag || tag.Object.Type != "commit" || tag.Object.SHA != boundary.Commit {
		return errors.New("remote signed release tag does not point directly to the publication commit")
	}
	if !tag.Verification.Verified || tag.Verification.Reason != "valid" || tag.Verification.Signature == "" || tag.Verification.Payload == "" || tag.Verification.VerifiedAt == "" {
		return errors.New("remote annotated release tag is not verified by GitHub")
	}
	return nil
}

func (publisher Publisher) ghJSON(ctx context.Context, boundary sourceBoundary, label, endpoint string) ([]byte, error) {
	result := publisher.Runner.Run(ctx, boundary.Root, nil, "gh", "api", "--method", "GET", endpoint)
	if result.Err != nil {
		return nil, commandFailure(label, result)
	}
	return result.Stdout, nil
}

func (publisher Publisher) readRelease(ctx context.Context, boundary sourceBoundary) (githubRelease, bool, error) {
	endpoint := "repos/" + boundary.GitHubRepository + "/releases?per_page=100"
	result := publisher.Runner.Run(ctx, boundary.Root, nil, "gh", "api", "--method", "GET", "--paginate", "--slurp", endpoint)
	if result.Err != nil {
		return githubRelease{}, false, commandFailure("list GitHub Releases including drafts", result)
	}
	var pages [][]githubRelease
	if err := decodeGitHubJSON(result.Stdout, &pages); err != nil {
		return githubRelease{}, false, errors.New("parse paginated GitHub Release records")
	}
	var matched *githubRelease
	for _, page := range pages {
		for _, release := range page {
			if release.TagName != boundary.Tag {
				continue
			}
			if matched != nil {
				return githubRelease{}, false, errors.New("multiple GitHub Releases identify the selected tag")
			}
			copyValue := release
			matched = &copyValue
		}
	}
	if matched == nil {
		return githubRelease{}, false, nil
	}
	return *matched, true, nil
}

func (publisher Publisher) readReleaseByID(ctx context.Context, boundary sourceBoundary, releaseID int64) (githubRelease, bool, error) {
	if releaseID <= 0 {
		return githubRelease{}, false, errors.New("release ID is invalid")
	}
	endpoint := "repos/" + boundary.GitHubRepository + "/releases/" + strconv.FormatInt(releaseID, 10)
	result := publisher.Runner.Run(ctx, boundary.Root, nil, "gh", "api", "--method", "GET", endpoint)
	if result.Err != nil {
		text := string(result.Stderr)
		if result.ExitCode == 1 && (strings.Contains(text, "HTTP 404") || strings.Contains(strings.ToLower(text), "not found")) {
			return githubRelease{}, false, nil
		}
		return githubRelease{}, false, commandFailure("read GitHub Release by ID", result)
	}
	var release githubRelease
	if err := decodeGitHubJSON(result.Stdout, &release); err != nil {
		return githubRelease{}, false, errors.New("parse GitHub Release record by ID")
	}
	if release.ID != releaseID {
		return githubRelease{}, false, errors.New("GitHub Release ID response does not match the requested release")
	}
	return release, true, nil
}

func (publisher Publisher) createDraft(ctx context.Context, boundary sourceBoundary) (githubRelease, error) {
	trueValue, falseValue := true, false
	payload := releasePayload{
		TagName:         boundary.Tag,
		TargetCommitish: boundary.Commit,
		Name:            boundary.Title,
		Body:            boundary.NotesBody,
		Draft:           &trueValue,
		Prerelease:      &falseValue,
		MakeLatest:      "false",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return githubRelease{}, errors.New("encode draft release request")
	}
	temporary, err := writeTemporaryJSON(boundary.Root, data)
	if err != nil {
		return githubRelease{}, err
	}
	defer os.Remove(temporary)
	endpoint := "repos/" + boundary.GitHubRepository + "/releases"
	result := publisher.Runner.Run(ctx, boundary.Root, nil, "gh", "api", "--method", "POST", "--input", temporary, endpoint)
	if result.Err != nil {
		return githubRelease{}, commandFailure("create draft GitHub Release", result)
	}
	var release githubRelease
	if err := decodeGitHubJSON(result.Stdout, &release); err != nil {
		return githubRelease{}, errors.New("parse created draft GitHub Release")
	}
	if err := verifyReleaseRecord(release, boundary, true, false, nil); err != nil {
		return githubRelease{}, err
	}
	return release, nil
}

func (publisher Publisher) uploadAssets(ctx context.Context, boundary sourceBoundary, result *Result) error {
	artifacts := append([]releaseartifactbuild.ArtifactRecord(nil), boundary.BuildReport.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	for _, artifact := range artifacts {
		path := filepath.Join(boundary.Artifacts, artifact.Name)
		command := publisher.Runner.Run(ctx, boundary.Root, nil,
			"gh", "release", "upload", boundary.Tag, path,
			"--repo", boundary.GitHubRepository,
		)
		observed, exists, readErr := publisher.readReleaseByID(ctx, boundary, result.ReleaseID)
		if readErr != nil {
			if command.Err != nil {
				return errors.Join(commandFailure("upload release asset "+artifact.Name, command), readErr)
			}
			return readErr
		}
		if !exists {
			if command.Err != nil {
				return commandFailure("upload release asset "+artifact.Name, command)
			}
			return errors.New("draft release disappeared after release asset upload")
		}
		if err := verifyCleanupDraft(observed, boundary, artifacts); err != nil {
			if command.Err != nil {
				return errors.Join(commandFailure("upload release asset "+artifact.Name, command), err)
			}
			return err
		}
		remote, ok := releaseAssetByName(observed.Assets, artifact.Name)
		if !ok {
			if command.Err != nil {
				return commandFailure("upload release asset "+artifact.Name, command)
			}
			return fmt.Errorf("uploaded release asset %s is absent from the exact draft", artifact.Name)
		}
		if err := verifyRemoteAsset(remote, artifact); err != nil {
			if command.Err != nil {
				return errors.Join(commandFailure("upload release asset "+artifact.Name, command), err)
			}
			return err
		}
		updateArtifactResult(result, remote, StatusPass, "")
		if err := writeEvidence(*result); err != nil {
			return err
		}
	}
	return nil
}

func (publisher Publisher) downloadAndVerify(ctx context.Context, boundary sourceBoundary, release githubRelease, result *Result) error {
	directory, err := os.MkdirTemp(filepath.Join(boundary.Root, ".local", "validation", "releases", boundary.Tag), ".remote-assets-")
	if err != nil {
		return errors.New("create private remote-asset verification directory")
	}
	defer os.RemoveAll(directory)
	if err := os.Chmod(directory, 0o700); err != nil {
		return errors.New("secure remote-asset verification directory")
	}
	byName := make(map[string]githubAsset, len(release.Assets))
	for _, asset := range release.Assets {
		byName[asset.Name] = asset
	}
	for _, artifact := range boundary.BuildReport.Artifacts {
		asset, ok := byName[artifact.Name]
		if !ok {
			return fmt.Errorf("remote release is missing asset %s", artifact.Name)
		}
		path := filepath.Join(directory, artifact.Name)
		endpoint := "repos/" + boundary.GitHubRepository + "/releases/assets/" + strconv.FormatInt(asset.ID, 10)
		command := publisher.Runner.RunToFile(ctx, boundary.Root, nil, path, "gh", "api", "--method", "GET", "-H", "Accept: application/octet-stream", endpoint)
		if command.Err != nil {
			return commandFailure("download remote release asset "+artifact.Name, command)
		}
		updateArtifactResult(result, asset, StatusPass, StatusPass)
	}
	if _, err := releaseartifact.VerifyDirectory(boundary.Pin, directory); err != nil {
		return fmt.Errorf("verify downloaded remote release assets: %w", err)
	}
	return nil
}

func (publisher Publisher) publishDraft(ctx context.Context, boundary sourceBoundary, releaseID int64) error {
	falseValue := false
	payload := releasePayload{Draft: &falseValue, Prerelease: &falseValue, MakeLatest: "true"}
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.New("encode release publication request")
	}
	temporary, err := writeTemporaryJSON(boundary.Root, data)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	endpoint := "repos/" + boundary.GitHubRepository + "/releases/" + strconv.FormatInt(releaseID, 10)
	result := publisher.Runner.Run(ctx, boundary.Root, nil, "gh", "api", "--method", "PATCH", "--input", temporary, endpoint)
	if result.Err != nil {
		return commandFailure("publish verified draft GitHub Release", result)
	}
	return nil
}

func (publisher Publisher) cleanupDraft(ctx context.Context, boundary sourceBoundary, releaseID int64) error {
	observed, exists, err := publisher.readReleaseByID(ctx, boundary, releaseID)
	if err != nil {
		return fmt.Errorf("inspect incomplete draft by ID before cleanup: %w", err)
	}
	if !exists {
		return nil
	}
	if verifyCleanupDraft(observed, boundary, boundary.BuildReport.Artifacts) != nil {
		return errors.New("incomplete release is not the exact draft created by this run; automatic cleanup denied")
	}
	endpoint := "repos/" + boundary.GitHubRepository + "/releases/" + strconv.FormatInt(releaseID, 10)
	result := publisher.Runner.Run(ctx, boundary.Root, nil, "gh", "api", "--method", "DELETE", endpoint)
	if result.Err != nil {
		return commandFailure("delete incomplete draft GitHub Release", result)
	}
	_, exists, err = publisher.readReleaseByID(ctx, boundary, releaseID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("incomplete draft still exists after ID-based cleanup")
	}
	return nil
}

func sameSourceBoundary(first, second sourceBoundary) bool {
	if first.Root != second.Root || first.Branch != second.Branch || first.Remote != second.Remote || first.Commit != second.Commit || first.Version != second.Version || first.Tag != second.Tag || first.GitHubRepository != second.GitHubRepository || first.Title != second.Title || first.NotesFile != second.NotesFile || first.NotesBody != second.NotesBody || first.Artifacts != second.Artifacts || first.BuildEvidence != second.BuildEvidence {
		return false
	}
	if len(first.BuildReport.Artifacts) != len(second.BuildReport.Artifacts) {
		return false
	}
	for index := range first.BuildReport.Artifacts {
		if first.BuildReport.Artifacts[index] != second.BuildReport.Artifacts[index] {
			return false
		}
	}
	return true
}

func verifyReleaseRecord(release githubRelease, boundary sourceBoundary, draft, published bool, artifacts []releaseartifactbuild.ArtifactRecord) error {
	if release.ID <= 0 || release.TagName != boundary.Tag || release.Name != boundary.Title || release.Body != boundary.NotesBody || release.Draft != draft || release.Prerelease {
		return errors.New("GitHub Release identity, title, draft state, or prerelease state is invalid")
	}
	if published {
		if release.PublishedAt == nil || strings.TrimSpace(*release.PublishedAt) == "" {
			return errors.New("published GitHub Release has no publication timestamp")
		}
		if _, err := time.Parse(time.RFC3339, *release.PublishedAt); err != nil {
			return errors.New("published GitHub Release has an invalid publication timestamp")
		}
	} else if release.PublishedAt != nil && strings.TrimSpace(*release.PublishedAt) != "" {
		return errors.New("draft GitHub Release unexpectedly has a publication timestamp")
	}
	if artifacts == nil {
		if len(release.Assets) != 0 {
			return errors.New("new draft GitHub Release was not empty")
		}
		return nil
	}
	if len(release.Assets) != len(artifacts) {
		return errors.New("GitHub Release asset inventory does not match the deterministic artifact set")
	}
	declared := make(map[string]releaseartifactbuild.ArtifactRecord, len(artifacts))
	for _, artifact := range artifacts {
		declared[artifact.Name] = artifact
	}
	seen := make(map[string]bool, len(release.Assets))
	for _, asset := range release.Assets {
		artifact, ok := declared[asset.Name]
		if !ok || seen[asset.Name] {
			return errors.New("GitHub Release contains an undeclared or duplicate asset")
		}
		seen[asset.Name] = true
		if err := verifyRemoteAsset(asset, artifact); err != nil {
			return err
		}
	}
	return nil
}

func releaseAssetByName(assets []githubAsset, name string) (githubAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func verifyRemoteAsset(asset githubAsset, artifact releaseartifactbuild.ArtifactRecord) error {
	if asset.ID <= 0 || asset.Name != artifact.Name || asset.State != "uploaded" || asset.Size != artifact.Size {
		return fmt.Errorf("remote asset %s does not match the expected identity, state, or size", artifact.Name)
	}
	if asset.Digest != "sha256:"+artifact.SHA256 {
		return fmt.Errorf("remote asset %s does not match the expected SHA-256 digest", artifact.Name)
	}
	return nil
}

func verifyCleanupDraft(release githubRelease, boundary sourceBoundary, artifacts []releaseartifactbuild.ArtifactRecord) error {
	if release.ID <= 0 || release.TagName != boundary.Tag || release.Name != boundary.Title || release.Body != boundary.NotesBody || !release.Draft || release.Prerelease {
		return errors.New("draft is not the exact release created by this publication run")
	}
	declared := make(map[string]releaseartifactbuild.ArtifactRecord, len(artifacts))
	for _, artifact := range artifacts {
		declared[artifact.Name] = artifact
	}
	seen := map[string]bool{}
	for _, asset := range release.Assets {
		artifact, ok := declared[asset.Name]
		if !ok || seen[asset.Name] || verifyRemoteAsset(asset, artifact) != nil {
			return errors.New("draft contains an unexpected, duplicate, or altered asset")
		}
		seen[asset.Name] = true
	}
	return nil
}

func releaseTitle(requested, version string) string {
	if strings.TrimSpace(requested) != "" {
		return strings.TrimSpace(requested)
	}
	return "ISRAS " + version + " — Solo Developer Baseline"
}

func githubRepository(requested, origin string) (string, error) {
	derived, err := repositoryFromOrigin(origin)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(requested) == "" {
		return derived, nil
	}
	if !repositoryPattern.MatchString(requested) {
		return "", errors.New("GitHub repository must be canonical owner/name syntax")
	}
	if requested != derived {
		return "", errors.New("--github-repo does not match the credential-free origin repository")
	}
	return requested, nil
}

func repositoryFromOrigin(origin string) (string, error) {
	value := strings.TrimSpace(origin)
	if value == "" || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("origin is empty or invalid")
	}
	var path string
	switch {
	case strings.HasPrefix(value, "git@github.com:"):
		path = strings.TrimPrefix(value, "git@github.com:")
	case strings.HasPrefix(value, "ssh://git@github.com/"):
		path = strings.TrimPrefix(value, "ssh://git@github.com/")
	case strings.HasPrefix(value, "https://github.com/"):
		parsed, err := url.Parse(value)
		if err != nil || parsed.User != nil || parsed.Host != "github.com" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", errors.New("origin is not a credential-free canonical GitHub URL")
		}
		path = strings.TrimPrefix(parsed.Path, "/")
	default:
		return "", errors.New("origin is not a supported GitHub repository URL")
	}
	path = strings.TrimSuffix(path, ".git")
	if !repositoryPattern.MatchString(path) {
		return "", errors.New("origin does not identify a canonical GitHub owner/name repository")
	}
	return path, nil
}

func loadBuildReport(path string) (releaseartifactbuild.Result, error) {
	data, err := readBoundedRegular(path, maxReportSize)
	if err != nil {
		return releaseartifactbuild.Result{}, errors.New("read artifact build evidence")
	}
	var report releaseartifactbuild.Result
	if err := decodeStrictJSON(data, &report); err != nil {
		return releaseartifactbuild.Result{}, errors.New("parse artifact build evidence")
	}
	return report, nil
}

func verifyBuildReport(report releaseartifactbuild.Result, identity repository.Identity, version, tag, artifactDirectory string) error {
	if report.SchemaVersion != 1 || report.Profile != releaseartifactbuild.Profile || report.Version != version || report.ReleaseTag != tag || report.SourceRepository != releaseartifactbuild.SourceRepository || report.SourceCommit != identity.Commit {
		return errors.New("artifact build evidence identity does not match the publication source")
	}
	if !commitPattern.MatchString(report.SourceCommit) || report.GoVersion == "" || len(report.GoVersion) > 64 || strings.ContainsAny(report.GoVersion, "\x00\r\n") {
		return errors.New("artifact build evidence contains an invalid source or toolchain identity")
	}
	reportedOutput, err := filepath.Abs(report.OutputDirectory)
	if err != nil || filepath.Clean(reportedOutput) != filepath.Clean(artifactDirectory) {
		return errors.New("artifact build evidence output directory does not match the selected artifact directory")
	}
	if report.GeneratedAt.IsZero() || report.GeneratedAt.Location() == nil {
		return errors.New("artifact build evidence has an invalid generation time")
	}
	if len(report.Artifacts) != 6 {
		return errors.New("artifact build evidence does not declare exactly six core artifacts")
	}
	expected := map[string]string{
		releaseartifactbuild.ValidatorName:  "validator",
		releaseartifactbuild.FrameworkName:  "framework",
		releaseartifactbuild.ContractsName:  "contracts",
		releaseartifactbuild.ProvenanceName: "provenance",
		releaseartifactbuild.SHA256Name:     "sha256-manifest",
		releaseartifactbuild.SHA512Name:     "sha512-manifest",
	}
	seen := make(map[string]bool, len(report.Artifacts))
	previous := ""
	for _, artifact := range report.Artifacts {
		if expected[artifact.Name] != artifact.Kind || seen[artifact.Name] {
			return errors.New("artifact build evidence contains an unexpected or duplicate artifact")
		}
		if previous != "" && artifact.Name <= previous {
			return errors.New("artifact build evidence artifacts must be unique and sorted by name")
		}
		previous = artifact.Name
		seen[artifact.Name] = true
		if artifact.Size <= 0 || artifact.Size > 512*1024*1024 || !digest256Pattern.MatchString(artifact.SHA256) || !digest512Pattern.MatchString(artifact.SHA512) {
			return errors.New("artifact build evidence contains an invalid size or digest")
		}
		if artifact.Kind == "validator" && (artifact.OS != "linux" || artifact.Arch != "amd64") {
			return errors.New("validator artifact platform identity is invalid")
		}
		if artifact.Kind != "validator" && (artifact.OS != "" || artifact.Arch != "") {
			return errors.New("non-validator artifact unexpectedly declares a platform")
		}
	}
	if len(seen) != len(expected) {
		return errors.New("artifact build evidence is missing a required artifact")
	}
	return nil
}

func pinFromBuildReport(report releaseartifactbuild.Result) projectpin.Pin {
	artifacts := make([]projectpin.Artifact, 0, len(report.Artifacts))
	for _, artifact := range report.Artifacts {
		artifacts = append(artifacts, projectpin.Artifact{
			Kind: artifact.Kind, OS: artifact.OS, Arch: artifact.Arch,
			Name: artifact.Name, SHA256: artifact.SHA256, SHA512: artifact.SHA512,
		})
	}
	return projectpin.Pin{
		SchemaVersion: 1,
		Standard: projectpin.Standard{
			Profile: releaseartifactbuild.Profile, Version: report.Version,
			ReleaseTag: report.ReleaseTag, SourceRepository: report.SourceRepository,
			SourceCommit: report.SourceCommit,
		},
		Artifacts: artifacts,
	}
}

func artifactResults(records []releaseartifactbuild.ArtifactRecord) []ArtifactResult {
	values := make([]ArtifactResult, 0, len(records))
	for _, record := range records {
		values = append(values, ArtifactResult{
			Kind: record.Kind, Name: record.Name, Size: record.Size,
			SHA256: record.SHA256, SHA512: record.SHA512,
			UploadStatus: StatusNotPerformed, DownloadStatus: StatusNotPerformed,
		})
	}
	return values
}

func updateArtifactResult(result *Result, remote githubAsset, uploadStatus, downloadStatus string) {
	for index := range result.Artifacts {
		if result.Artifacts[index].Name != remote.Name {
			continue
		}
		result.Artifacts[index].RemoteAssetID = remote.ID
		result.Artifacts[index].RemoteSize = remote.Size
		result.Artifacts[index].RemoteDigest = remote.Digest
		if uploadStatus != "" {
			result.Artifacts[index].UploadStatus = uploadStatus
		}
		if downloadStatus != "" {
			result.Artifacts[index].DownloadStatus = downloadStatus
		}
	}
}

func prepareEvidence(result *Result, root, tag string, started time.Time) error {
	directory := filepath.Join(root, ".local", "validation", "releases", tag, "publication")
	if err := ensurePrivateDirectory(root, directory); err != nil {
		return errors.New("prepare private publication evidence directory")
	}
	base := "release-publication-" + started.UTC().Format("20060102T150405.000000000Z")
	result.EvidenceJSON = filepath.Join(directory, base+".json")
	result.EvidenceText = filepath.Join(directory, base+".txt")
	for _, path := range []string{result.EvidenceJSON, result.EvidenceText} {
		if _, err := os.Lstat(path); err == nil {
			return errors.New("publication evidence path already exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return errors.New("inspect publication evidence path")
		}
	}
	return nil
}

func writeEvidence(result Result) error {
	if result.EvidenceJSON == "" || result.EvidenceText == "" {
		return errors.New("publication evidence paths are unavailable")
	}
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return errors.New("encode publication JSON evidence")
	}
	jsonBytes = append(jsonBytes, '\n')
	if err := atomicWrite(result.EvidenceJSON, jsonBytes, 0o600); err != nil {
		return errors.New("write publication JSON evidence")
	}
	text := renderTextEvidence(result)
	if err := atomicWrite(result.EvidenceText, []byte(text), 0o600); err != nil {
		return errors.New("write publication text evidence")
	}
	return nil
}

func renderTextEvidence(result Result) string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "ISRAS RELEASE PUBLICATION EVIDENCE")
	fmt.Fprintln(&builder, "==================================")
	fmt.Fprintf(&builder, "Action: %s\n", result.Action)
	fmt.Fprintf(&builder, "Version: %s\n", result.Version)
	fmt.Fprintf(&builder, "Release tag: %s\n", result.ReleaseTag)
	fmt.Fprintf(&builder, "Source commit: %s\n", result.SourceCommit)
	fmt.Fprintf(&builder, "GitHub repository: %s\n", result.GitHubRepository)
	fmt.Fprintf(&builder, "Local verification: %s\n", result.LocalVerification)
	fmt.Fprintf(&builder, "Remote tag: %s\n", result.RemoteTag)
	fmt.Fprintf(&builder, "Release absence: %s\n", result.ReleaseAbsence)
	fmt.Fprintf(&builder, "Draft creation: %s\n", result.DraftCreation)
	fmt.Fprintf(&builder, "Asset upload: %s\n", result.AssetUpload)
	fmt.Fprintf(&builder, "Draft verification: %s\n", result.DraftVerification)
	fmt.Fprintf(&builder, "Publication: %s\n", result.Publication)
	fmt.Fprintf(&builder, "Final verification: %s\n", result.FinalVerification)
	fmt.Fprintf(&builder, "Cleanup: %s\n", result.Cleanup)
	if result.ReleaseURL != "" {
		fmt.Fprintf(&builder, "Release URL: %s\n", result.ReleaseURL)
	}
	if result.Failure != "" {
		fmt.Fprintf(&builder, "Failure: %s\n", result.Failure)
	}
	fmt.Fprintln(&builder, "Artifacts:")
	for _, artifact := range result.Artifacts {
		fmt.Fprintf(&builder, "- %s size=%d sha256=%s sha512=%s upload=%s download=%s\n", artifact.Name, artifact.Size, artifact.SHA256, artifact.SHA512, artifact.UploadStatus, artifact.DownloadStatus)
	}
	return redact.Sanitize(builder.String())
}

func readBoundedRegular(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maximum {
		return nil, errors.New("path is not a bounded regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > maximum {
		return nil, errors.New("file changed during read or exceeded its size boundary")
	}
	finalInfo, statErr := os.Lstat(path)
	if statErr != nil || !finalInfo.Mode().IsRegular() || finalInfo.Mode()&os.ModeSymlink != 0 || finalInfo.Size() != info.Size() || !finalInfo.ModTime().Equal(info.ModTime()) {
		return nil, errors.New("file changed during bounded read")
	}
	return data, nil
}

func secureDirectoryPath(root, requested string) (string, error) {
	path := requested
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("resolve directory path")
	}
	path = filepath.Clean(path)
	if err := rejectSymlinkComponents(path); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("path is not a regular directory")
	}
	return path, nil
}

func secureRegularPath(root, requested string, maximum int64) (string, error) {
	path := requested
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("resolve file path")
	}
	path = filepath.Clean(path)
	if err := rejectSymlinkComponents(filepath.Dir(path)); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maximum {
		return "", errors.New("path is not a bounded regular file")
	}
	return path, nil
}

func rejectSymlinkComponents(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return errors.New("resolve path for symbolic-link inspection")
	}
	absolute = filepath.Clean(absolute)
	volume := filepath.VolumeName(absolute)
	current := volume + string(filepath.Separator)
	remainder := strings.TrimPrefix(absolute, current)
	if remainder == absolute {
		current = string(filepath.Separator)
		remainder = strings.TrimPrefix(absolute, current)
	}
	for _, component := range strings.Split(remainder, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New("inspect path component")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("path contains a symbolic-link component")
		}
	}
	return nil
}

func ensurePrivateDirectory(root, path string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("private evidence directory is outside the repository")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		switch {
		case errors.Is(statErr, os.ErrNotExist):
			if err := os.Mkdir(current, 0o700); err != nil {
				return err
			}
		case statErr != nil:
			return statErr
		case !info.IsDir() || info.Mode()&os.ModeSymlink != 0:
			return errors.New("private evidence path contains a non-directory or symbolic link")
		default:
			if err := os.Chmod(current, 0o700); err != nil {
				return err
			}
		}
	}
	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".isras-publication-evidence-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if err := writeAll(temporary, data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func writeTemporaryJSON(root string, data []byte) (string, error) {
	directory := filepath.Join(root, ".local", "validation", "releases", ".publication-requests")
	if err := ensurePrivateDirectory(root, directory); err != nil {
		return "", errors.New("prepare private publication request directory")
	}
	file, err := os.CreateTemp(directory, "request-*.json")
	if err != nil {
		return "", errors.New("create publication request file")
	}
	path := file.Name()
	committed := false
	defer func() {
		_ = file.Close()
		if !committed {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return "", errors.New("secure publication request file")
	}
	if err := writeAll(file, data); err != nil {
		return "", errors.New("write publication request file")
	}
	if err := file.Close(); err != nil {
		return "", errors.New("close publication request file")
	}
	committed = true
	return path, nil
}

func decodeStrictJSON(data []byte, target any) error {
	if len(data) == 0 || len(data) > maxCommandOutput {
		return errors.New("JSON response is empty or oversized")
	}
	if err := rejectDuplicateJSON(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("JSON response contains multiple values")
		}
		return err
	}
	return nil
}

func decodeGitHubJSON(data []byte, target any) error {
	if len(data) == 0 || len(data) > maxCommandOutput {
		return errors.New("GitHub JSON response is empty or oversized")
	}
	if err := rejectDuplicateJSON(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("GitHub JSON response contains multiple values")
		}
		return err
	}
	return nil
}

func rejectDuplicateJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSON(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("JSON contains trailing data")
		}
		return err
	}
	return nil
}

func scanJSON(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok || seen[key] {
				return errors.New("JSON contains a duplicate or invalid object key")
			}
			seen[key] = true
			if err := scanJSON(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := scanJSON(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return errors.New("JSON contains an invalid delimiter")
	}
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func safeFailure(err error) string {
	if err == nil {
		return ""
	}
	value := redact.Sanitize(err.Error())
	if len(value) > 4096 {
		value = value[:4096] + " [truncated]"
	}
	return value
}
