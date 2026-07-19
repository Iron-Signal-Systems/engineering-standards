package projectadoption

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectorigin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

const (
	CallerWorkflowPath       = ".github/workflows/isras-validation.yml"
	AdoptionEvidencePath     = ".isras/adoption-verification.json"
	FormatCheckPath          = ".isras/check-go-format"
	DefaultEvidenceDirectory = projectpin.RuntimeEvidenceDirectory
)

type Request struct {
	Root       string
	ReleaseTag string
	GoDefaults bool
	Validator  validatoridentity.Identity
}

type Result struct {
	ProjectRepository string
	ReleaseTag        string
	SourceCommit      string
	PinPath           string
	WorkflowPath      string
	EvidencePath      string
	FormatCheckPath   string
	Changed           bool
	Report            releaseartifact.Report
}

type bootstrapRelease func(context.Context, string) (releaseartifact.Bootstrap, error)

func Initialize(ctx context.Context, request Request) (Result, error) {
	return initializeWithBootstrap(ctx, request, releaseartifact.BootstrapGitHub)
}

func initializeWithBootstrap(ctx context.Context, request Request, bootstrapRelease bootstrapRelease) (Result, error) {
	if !request.GoDefaults {
		return Result{}, errors.New("initialization requires the explicit --go-defaults authorization")
	}
	if err := authorizeRequestedReleaseValidator(request.Validator, request.ReleaseTag); err != nil {
		return Result{}, err
	}

	identity, err := repository.DiscoverFrom(ctx, request.Root)
	if err != nil {
		return Result{}, err
	}
	projectRepository, err := projectorigin.Canonical(identity.Origin)
	if err != nil {
		return Result{}, err
	}
	if err := validateEvidenceBoundary(ctx, identity.Root); err != nil {
		return Result{}, err
	}

	bootstrap, err := bootstrapRelease(ctx, request.ReleaseTag)
	if err != nil {
		return Result{ProjectRepository: projectRepository, ReleaseTag: request.ReleaseTag, Report: bootstrap.Report}, err
	}
	if err := authorizeVerifiedReleaseValidator(request.Validator, bootstrap.Standard); err != nil {
		return Result{ProjectRepository: projectRepository, ReleaseTag: request.ReleaseTag, SourceCommit: bootstrap.Standard.SourceCommit, Report: bootstrap.Report}, err
	}

	pin := projectpin.Pin{
		SchemaVersion: projectpin.SchemaVersion,
		Project:       projectpin.Project{Repository: projectRepository},
		Standard:      bootstrap.Standard,
		Artifacts:     bootstrap.Artifacts,
		Workflow: projectpin.Workflow{
			Repository: projectpin.SourceRepository,
			Path:       projectpin.ReusableWorkflowPath,
			Commit:     bootstrap.Standard.SourceCommit,
		},
		Profiles: []string{"go"},
		Commands: projectpin.DefaultGoCommands(),
		Evidence: projectpin.Evidence{Directory: DefaultEvidenceDirectory},
	}
	pinData, err := projectpin.CanonicalJSON(pin)
	if err != nil {
		return Result{ProjectRepository: projectRepository, ReleaseTag: request.ReleaseTag, Report: bootstrap.Report}, err
	}
	workflowData := callerWorkflow(bootstrap.Standard.SourceCommit)
	evidenceData, err := canonicalAdoptionEvidence(bootstrap)
	if err != nil {
		return Result{ProjectRepository: projectRepository, ReleaseTag: request.ReleaseTag, SourceCommit: bootstrap.Standard.SourceCommit, Report: bootstrap.Report}, err
	}

	files := []installFile{
		{Path: projectpin.MetadataPath, Data: pinData, Mode: 0o644},
		{Path: CallerWorkflowPath, Data: workflowData, Mode: 0o644},
		{Path: AdoptionEvidencePath, Data: evidenceData, Mode: 0o644},
		{Path: FormatCheckPath, Data: goFormatCheck(), Mode: 0o755},
	}
	changed, err := install(ctx, identity, files)
	if err != nil {
		return Result{ProjectRepository: projectRepository, ReleaseTag: request.ReleaseTag, SourceCommit: bootstrap.Standard.SourceCommit, Report: bootstrap.Report}, err
	}
	return Result{
		ProjectRepository: projectRepository,
		ReleaseTag:        request.ReleaseTag,
		SourceCommit:      bootstrap.Standard.SourceCommit,
		PinPath:           projectpin.MetadataPath,
		WorkflowPath:      CallerWorkflowPath,
		EvidencePath:      AdoptionEvidencePath,
		FormatCheckPath:   FormatCheckPath,
		Changed:           changed,
		Report:            bootstrap.Report,
	}, nil
}

func authorizeRequestedReleaseValidator(identity validatoridentity.Identity, releaseTag string) error {
	if identity.Ownership != validatoridentity.OwnershipReleaseArtifact {
		return errors.New("project initialization requires a linker-bound release validator")
	}
	if identity.Profile != projectpin.Profile || identity.SourceRepository != projectpin.SourceRepository {
		return errors.New("release validator profile or source repository is invalid")
	}
	if identity.ReleaseTag != releaseTag || releaseTag != "isras-v"+identity.StandardVersion {
		return errors.New("running release validator does not match the explicitly requested release")
	}
	if identity.SourceCommit == "" || identity.RepositoryCommit != identity.SourceCommit {
		return errors.New("running release validator has an inconsistent source identity")
	}
	return nil
}

func authorizeVerifiedReleaseValidator(identity validatoridentity.Identity, standard projectpin.Standard) error {
	if identity.Profile != standard.Profile ||
		identity.StandardVersion != standard.Version ||
		identity.ReleaseTag != standard.ReleaseTag ||
		identity.SourceRepository != standard.SourceRepository ||
		identity.SourceCommit != standard.SourceCommit ||
		identity.RepositoryCommit != standard.SourceCommit {
		return errors.New("running release validator identity does not match the verified release")
	}
	return nil
}

func validateEvidenceBoundary(ctx context.Context, root string) error {
	evidencePath := filepath.Join(root, filepath.FromSlash(DefaultEvidenceDirectory))
	exists, err := inspectExistingSafeDirectory(root, evidencePath)
	if err != nil {
		return errors.New("fixed project evidence directory is unsafe")
	}
	if exists {
		info, err := os.Lstat(evidencePath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("fixed project evidence path must be an untracked regular directory")
		}
	}
	tracked := executil.Run(ctx, root, "git", "ls-files", "--", DefaultEvidenceDirectory, DefaultEvidenceDirectory+"/**")
	if tracked.Err != nil {
		return errors.New("inspect fixed project evidence tracking state")
	}
	if strings.TrimSpace(tracked.Stdout) != "" {
		return errors.New("fixed project evidence directory must not contain tracked paths")
	}
	return nil
}

type adoptionEvidence struct {
	SchemaVersion    int                        `json:"schema_version"`
	Profile          string                     `json:"profile"`
	Version          string                     `json:"version"`
	ReleaseTag       string                     `json:"release_tag"`
	SourceRepository string                     `json:"source_repository"`
	SourceCommit     string                     `json:"source_commit"`
	Verification     adoptionVerification       `json:"verification"`
	Artifacts        []adoptionArtifactEvidence `json:"artifacts"`
}

type adoptionVerification struct {
	ReleaseRecord          string `json:"release_record"`
	SignedTag              string `json:"signed_tag"`
	AssetAcquisition       string `json:"asset_acquisition"`
	AssetInventory         string `json:"asset_inventory"`
	PinDigests             string `json:"pin_digests"`
	SHA256Manifest         string `json:"sha256_manifest"`
	SHA512Manifest         string `json:"sha512_manifest"`
	Provenance             string `json:"provenance"`
	ExecutionAuthorization string `json:"execution_authorization"`
}

type adoptionArtifactEvidence struct {
	Kind              string `json:"kind"`
	Name              string `json:"name"`
	OS                string `json:"os,omitempty"`
	Arch              string `json:"arch,omitempty"`
	Size              int64  `json:"size"`
	RemoteSize        int64  `json:"remote_size"`
	SHA256            string `json:"sha256"`
	SHA512            string `json:"sha512"`
	SHA256Manifest    string `json:"sha256_manifest"`
	SHA512Manifest    string `json:"sha512_manifest"`
	ProvenanceBinding string `json:"provenance_binding"`
}

func canonicalAdoptionEvidence(bootstrap releaseartifact.Bootstrap) ([]byte, error) {
	report := bootstrap.Report
	for name, status := range map[string]string{
		"release record":    report.ReleaseRecord,
		"signed tag":        report.SignedTag,
		"asset acquisition": report.AssetAcquisition,
		"asset inventory":   report.AssetInventory,
		"pin digests":       report.PinDigests,
		"SHA-256 manifest":  report.SHA256Manifest,
		"SHA-512 manifest":  report.SHA512Manifest,
		"provenance":        report.Provenance,
	} {
		if status != releaseartifact.StatusPass {
			return nil, fmt.Errorf("adoption evidence requires PASS %s verification", name)
		}
	}
	if report.ExecutionAuthorization != releaseartifact.AuthorizationGranted {
		return nil, errors.New("adoption evidence requires granted execution authorization")
	}

	artifacts := make([]adoptionArtifactEvidence, 0, len(report.Artifacts))
	for _, artifact := range report.Artifacts {
		if artifact.SHA256Status != releaseartifact.StatusPass || artifact.SHA512Status != releaseartifact.StatusPass {
			return nil, fmt.Errorf("adoption evidence artifact %q is not fully verified", artifact.Name)
		}
		artifacts = append(artifacts, adoptionArtifactEvidence{
			Kind: artifact.Kind, Name: artifact.Name, OS: artifact.OS, Arch: artifact.Arch,
			Size: artifact.Size, RemoteSize: artifact.RemoteSize,
			SHA256: artifact.ObservedSHA256, SHA512: artifact.ObservedSHA512,
			SHA256Manifest: artifact.SHA256Manifest, SHA512Manifest: artifact.SHA512Manifest,
			ProvenanceBinding: artifact.ProvenanceBinding,
		})
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	evidence := adoptionEvidence{
		SchemaVersion:    1,
		Profile:          bootstrap.Standard.Profile,
		Version:          bootstrap.Standard.Version,
		ReleaseTag:       bootstrap.Standard.ReleaseTag,
		SourceRepository: bootstrap.Standard.SourceRepository,
		SourceCommit:     bootstrap.Standard.SourceCommit,
		Verification: adoptionVerification{
			ReleaseRecord:          report.ReleaseRecord,
			SignedTag:              report.SignedTag,
			AssetAcquisition:       report.AssetAcquisition,
			AssetInventory:         report.AssetInventory,
			PinDigests:             report.PinDigests,
			SHA256Manifest:         report.SHA256Manifest,
			SHA512Manifest:         report.SHA512Manifest,
			Provenance:             report.Provenance,
			ExecutionAuthorization: report.ExecutionAuthorization,
		},
		Artifacts: artifacts,
	}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode canonical adoption evidence: %w", err)
	}
	return append(data, '\n'), nil
}

func callerWorkflow(sourceCommit string) []byte {
	workflow := fmt.Sprintf(`name: ISRAS Validation

on:
  pull_request:
  push:
  workflow_dispatch:

permissions:
  contents: read

jobs:
  validate:
    uses: Iron-Signal-Systems/engineering-standards/.github/workflows/validate-project.yml@%s
    permissions:
      contents: read
`, sourceCommit)
	return []byte(workflow)
}

func goFormatCheck() []byte {
	return []byte(`#!/usr/bin/env bash
set -Eeuo pipefail

list_file="$(mktemp)"
trap 'rm -f -- "$list_file"' EXIT
git ls-files -z -- '*.go' >"$list_file"
mapfile -d '' files <"$list_file"
if ((${#files[@]} == 0)); then
  exit 0
fi

output="$(gofmt -l -- "${files[@]}")"
if [[ -n "$output" ]]; then
  printf '%s\n' "$output" >&2
  exit 1
fi
`)
}

type installFile struct {
	Path string
	Data []byte
	Mode os.FileMode
}

func install(ctx context.Context, identity repository.Identity, files []installFile) (bool, error) {
	allExact := true
	anyExists := false
	for _, file := range files {
		exact, exists, err := inspectTarget(identity.Root, file)
		if err != nil {
			return false, err
		}
		allExact = allExact && exact
		anyExists = anyExists || exists
	}
	if allExact {
		return false, nil
	}
	if anyExists {
		return false, errors.New("refusing partial or conflicting ISRAS adoption state")
	}
	if err := requireCleanRepository(ctx, identity); err != nil {
		return false, err
	}

	createdFiles := make([]string, 0, len(files))
	createdDirectories := make([]string, 0, 3)
	rollback := func(cause error) error {
		return errors.Join(cause, cleanupCreated(createdFiles, createdDirectories))
	}

	for _, file := range files {
		absolute, err := safeTargetPath(identity.Root, file.Path)
		if err != nil {
			return false, rollback(err)
		}
		parent := filepath.Dir(absolute)
		if err := ensureSafeDirectory(identity.Root, parent, &createdDirectories); err != nil {
			return false, rollback(err)
		}
		temporary, err := os.CreateTemp(parent, ".isras-adoption-*")
		if err != nil {
			return false, rollback(errors.New("create temporary ISRAS adoption file"))
		}
		temporaryName := temporary.Name()
		discardTemporary := func(cause error) error {
			closeErr := temporary.Close()
			removeErr := os.Remove(temporaryName)
			if errors.Is(removeErr, os.ErrNotExist) {
				removeErr = nil
			}
			return errors.Join(cause, closeErr, removeErr, cleanupCreated(createdFiles, createdDirectories))
		}

		mode := file.Mode
		if mode == 0 {
			mode = 0o644
		}
		if _, err := temporary.Write(file.Data); err != nil {
			return false, discardTemporary(errors.New("write ISRAS adoption file"))
		}
		if err := temporary.Sync(); err != nil {
			return false, discardTemporary(errors.New("synchronize private ISRAS adoption file"))
		}
		if err := temporary.Chmod(mode); err != nil {
			return false, discardTemporary(errors.New("set ISRAS adoption file permissions"))
		}
		if err := temporary.Sync(); err != nil {
			return false, discardTemporary(errors.New("synchronize final ISRAS adoption file metadata"))
		}
		if err := temporary.Close(); err != nil {
			removeErr := os.Remove(temporaryName)
			return false, errors.Join(errors.New("close ISRAS adoption file"), removeErr, cleanupCreated(createdFiles, createdDirectories))
		}
		if err := os.Link(temporaryName, absolute); err != nil {
			removeErr := os.Remove(temporaryName)
			return false, errors.Join(errors.New("publish ISRAS adoption file without replacement"), removeErr, cleanupCreated(createdFiles, createdDirectories))
		}
		createdFiles = append(createdFiles, absolute)
		if err := os.Remove(temporaryName); err != nil {
			return false, rollback(errors.New("remove temporary ISRAS adoption file"))
		}
		if err := syncDirectory(parent); err != nil {
			return false, rollback(err)
		}
	}

	if err := requireRepositoryIdentity(ctx, identity); err != nil {
		return false, rollback(err)
	}
	for _, file := range files {
		exact, exists, err := inspectTarget(identity.Root, file)
		if err != nil || !exists || !exact {
			if err == nil {
				err = errors.New("published ISRAS adoption file failed final exact verification")
			}
			return false, rollback(err)
		}
	}
	return true, nil
}

func cleanupCreated(createdFiles, createdDirectories []string) error {
	var cleanupErrors []error
	for index := len(createdFiles) - 1; index >= 0; index-- {
		parent := filepath.Dir(createdFiles[index])
		if err := os.Remove(createdFiles[index]); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove published adoption file during rollback: %w", err))
		}
		if err := syncDirectory(parent); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	for index := len(createdDirectories) - 1; index >= 0; index-- {
		parent := filepath.Dir(createdDirectories[index])
		if err := os.Remove(createdDirectories[index]); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove adoption directory during rollback: %w", err))
		}
		if err := syncDirectory(parent); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}
func ensureSafeDirectory(root, target string, created *[]string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("ISRAS adoption directory escapes the target repository")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return errors.New("target repository root is not a regular directory")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		next := filepath.Join(current, component)
		info, statErr := os.Lstat(next)
		if errors.Is(statErr, os.ErrNotExist) {
			if mkdirErr := os.Mkdir(next, 0o755); mkdirErr != nil {
				if !errors.Is(mkdirErr, os.ErrExist) {
					return errors.New("create ISRAS adoption directory")
				}
				info, statErr = os.Lstat(next)
			} else {
				*created = append(*created, next)
				if err := syncDirectory(current); err != nil {
					return err
				}
				info, statErr = os.Lstat(next)
			}
		}
		if statErr != nil {
			return errors.New("inspect ISRAS adoption directory")
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("ISRAS adoption directory contains a symbolic link or non-directory component")
		}
		current = next
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return errors.New("open ISRAS adoption directory for synchronization")
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return errors.New("synchronize ISRAS adoption directory")
	}
	return nil
}

func inspectTarget(root string, file installFile) (bool, bool, error) {
	absolute, err := safeTargetPath(root, file.Path)
	if err != nil {
		return false, false, err
	}
	parentExists, err := inspectExistingSafeDirectory(root, filepath.Dir(absolute))
	if err != nil {
		return false, false, err
	}
	if !parentExists {
		return false, false, nil
	}
	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, false, errors.New("inspect existing ISRAS adoption file")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, true, errors.New("existing ISRAS adoption path is not a regular file")
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return false, true, errors.New("read existing ISRAS adoption file")
	}
	expectedMode := file.Mode
	if expectedMode == 0 {
		expectedMode = 0o644
	}
	exact := bytes.Equal(data, file.Data) && info.Mode().Perm() == expectedMode.Perm()
	return exact, true, nil
}

func inspectExistingSafeDirectory(root, target string) (bool, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false, errors.New("ISRAS adoption directory escapes the target repository")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return false, errors.New("target repository root is not a regular directory")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, errors.New("inspect ISRAS adoption directory")
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false, errors.New("ISRAS adoption directory contains a symbolic link or non-directory component")
		}
	}
	return true, nil
}

func safeTargetPath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "\\") {
		return "", errors.New("invalid ISRAS adoption path")
	}
	cleaned := filepath.Clean(relative)
	if cleaned != relative || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", errors.New("ISRAS adoption path escapes the target repository")
	}
	absolute := filepath.Join(root, cleaned)
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel != cleaned {
		return "", errors.New("resolve ISRAS adoption path")
	}
	return absolute, nil
}

func requireRepositoryIdentity(ctx context.Context, identity repository.Identity) error {
	current, err := repository.DiscoverFrom(ctx, identity.Root)
	if err != nil {
		return errors.New("rediscover target repository during adoption")
	}
	if current.Root != identity.Root || current.Commit != identity.Commit || current.Origin != identity.Origin {
		return errors.New("target repository identity changed during adoption")
	}
	return nil
}

func requireCleanRepository(ctx context.Context, identity repository.Identity) error {
	if err := requireRepositoryIdentity(ctx, identity); err != nil {
		return err
	}
	result := executil.Run(ctx, identity.Root, "git", "status", "--porcelain=v1", "--untracked-files=all")
	if result.Err != nil {
		return errors.New("inspect target repository state before adoption")
	}
	if strings.TrimSpace(result.Stdout) != "" {
		return errors.New("target repository must be clean before first ISRAS adoption")
	}
	commit := executil.Run(ctx, identity.Root, "git", "rev-parse", "--verify", "HEAD^{commit}")
	if commit.Err != nil || strings.TrimSpace(commit.Stdout) != identity.Commit {
		return errors.New("target repository HEAD changed during adoption preparation")
	}
	return nil
}
