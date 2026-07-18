package projectadoption

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
)

var projectRepositoryNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

const (
	CallerWorkflowPath       = ".github/workflows/isras-validation.yml"
	AdoptionEvidencePath     = ".isras/adoption-verification.json"
	FormatCheckPath          = ".isras/check-go-format"
	DefaultEvidenceDirectory = ".local/isras"
)

type Request struct {
	Root              string
	ReleaseTag        string
	EvidenceDirectory string
	GoDefaults        bool
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

func Initialize(ctx context.Context, request Request) (Result, error) {
	if !request.GoDefaults {
		return Result{}, errors.New("initialization requires the explicit --go-defaults authorization")
	}
	if request.EvidenceDirectory == "" {
		request.EvidenceDirectory = DefaultEvidenceDirectory
	}
	if err := validateEvidenceDirectory(request.EvidenceDirectory); err != nil {
		return Result{}, err
	}

	identity, err := repository.DiscoverFrom(ctx, request.Root)
	if err != nil {
		return Result{}, err
	}
	projectRepository, err := canonicalProjectOrigin(identity.Origin)
	if err != nil {
		return Result{}, err
	}

	bootstrap, err := releaseartifact.BootstrapGitHub(ctx, request.ReleaseTag)
	if err != nil {
		return Result{ProjectRepository: projectRepository, ReleaseTag: request.ReleaseTag, Report: bootstrap.Report}, err
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
		Evidence: projectpin.Evidence{Directory: request.EvidenceDirectory},
	}
	pinData, err := projectpin.CanonicalJSON(pin)
	if err != nil {
		return Result{ProjectRepository: projectRepository, ReleaseTag: request.ReleaseTag, Report: bootstrap.Report}, err
	}
	workflowData := callerWorkflow(bootstrap.Standard.SourceCommit)
	evidenceData, err := json.MarshalIndent(bootstrap.Report, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode adoption verification evidence: %w", err)
	}
	evidenceData = append(evidenceData, '\n')

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

func canonicalProjectOrigin(origin string) (string, error) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", errors.New("target repository requires an origin remote")
	}
	var repositoryPath string
	if strings.HasPrefix(origin, "git@github.com:") {
		repositoryPath = strings.TrimPrefix(origin, "git@github.com:")
	} else {
		parsed, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
			return "", errors.New("target origin must identify github.com")
		}
		scheme := strings.ToLower(parsed.Scheme)
		switch scheme {
		case "https", "ssh", "git":
		default:
			return "", errors.New("target origin uses an unsupported scheme")
		}
		if parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", errors.New("target origin must not contain a query or fragment")
		}
		if parsed.User != nil {
			_, hasPassword := parsed.User.Password()
			if scheme != "ssh" || parsed.User.Username() != "git" || hasPassword {
				return "", errors.New("target origin contains unsupported credentials")
			}
		}
		repositoryPath = strings.TrimPrefix(parsed.Path, "/")
	}
	repositoryPath = strings.TrimSuffix(repositoryPath, ".git")
	parts := strings.Split(repositoryPath, "/")
	if len(parts) != 2 || parts[0] != "Iron-Signal-Systems" || !projectRepositoryNamePattern.MatchString(parts[1]) {
		return "", errors.New("target origin must identify one Iron-Signal-Systems repository")
	}
	return "github.com/" + repositoryPath, nil
}

func validateEvidenceDirectory(value string) error {
	if value == "" || len(value) > 255 || strings.Contains(value, "\\") || strings.ContainsAny(value, "\x00\r\n\t") {
		return errors.New("evidence directory must be a bounded relative slash-separated path")
	}
	if strings.HasPrefix(value, "/") || value == "." || value == ".." || path.Clean(value) != value || strings.HasPrefix(value, "../") {
		return errors.New("evidence directory must remain inside the target repository")
	}
	if value == ".git" || strings.HasPrefix(value, ".git/") {
		return errors.New("evidence directory must not be inside .git")
	}
	return nil
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

mapfile -d '' files < <(git ls-files -z -- '*.go')
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
	cleanup := func() {
		for index := len(createdFiles) - 1; index >= 0; index-- {
			parent := filepath.Dir(createdFiles[index])
			_ = os.Remove(createdFiles[index])
			_ = syncDirectory(parent)
		}
		for index := len(createdDirectories) - 1; index >= 0; index-- {
			parent := filepath.Dir(createdDirectories[index])
			_ = os.Remove(createdDirectories[index])
			_ = syncDirectory(parent)
		}
	}

	for _, file := range files {
		absolute, err := safeTargetPath(identity.Root, file.Path)
		if err != nil {
			cleanup()
			return false, err
		}
		parent := filepath.Dir(absolute)
		if err := ensureSafeDirectory(identity.Root, parent, &createdDirectories); err != nil {
			cleanup()
			return false, err
		}
		temporary, err := os.CreateTemp(parent, ".isras-adoption-*")
		if err != nil {
			cleanup()
			return false, errors.New("create temporary ISRAS adoption file")
		}
		temporaryName := temporary.Name()
		if file.Mode == 0 {
			file.Mode = 0o644
		}
		if err := temporary.Chmod(file.Mode); err != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryName)
			cleanup()
			return false, errors.New("set ISRAS adoption file permissions")
		}
		if _, err := temporary.Write(file.Data); err != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryName)
			cleanup()
			return false, errors.New("write ISRAS adoption file")
		}
		if err := temporary.Sync(); err != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryName)
			cleanup()
			return false, errors.New("synchronize ISRAS adoption file")
		}
		if err := temporary.Close(); err != nil {
			_ = os.Remove(temporaryName)
			cleanup()
			return false, errors.New("close ISRAS adoption file")
		}
		if err := os.Link(temporaryName, absolute); err != nil {
			_ = os.Remove(temporaryName)
			cleanup()
			return false, errors.New("publish ISRAS adoption file without replacement")
		}
		createdFiles = append(createdFiles, absolute)
		if err := os.Remove(temporaryName); err != nil {
			_ = os.Remove(absolute)
			cleanup()
			return false, errors.New("remove temporary ISRAS adoption file")
		}
		if err := syncDirectory(parent); err != nil {
			cleanup()
			return false, err
		}
	}
	return true, nil
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

func requireCleanRepository(ctx context.Context, identity repository.Identity) error {
	current, err := repository.DiscoverFrom(ctx, identity.Root)
	if err != nil {
		return errors.New("rediscover target repository before adoption")
	}
	if current.Root != identity.Root || current.Commit != identity.Commit || current.Origin != identity.Origin {
		return errors.New("target repository identity changed during adoption preparation")
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
