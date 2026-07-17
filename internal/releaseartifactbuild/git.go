package releaseartifactbuild

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	stableVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	commitPattern        = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type sourceBoundary struct {
	Root        string
	Version     string
	Tag         string
	Commit      string
	GoVersion   string
	GoDirective string
}

func inspectSource(ctx context.Context, runner commandRunner, requestedRoot, expectedVersion string) (sourceBoundary, error) {
	root := requestedRoot
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return sourceBoundary{}, errors.New("resolve repository root")
	}
	rootOutput, _, err := runner.Run(ctx, absolute, nil, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return sourceBoundary{}, errors.New("discover repository root")
	}
	canonicalRoot := strings.TrimSpace(rootOutput)
	if canonicalRoot == "" {
		return sourceBoundary{}, errors.New("repository root is empty")
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return sourceBoundary{}, errors.New("resolve canonical repository root")
	}

	status, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return sourceBoundary{}, errors.New("inspect repository status")
	}
	if status != "" {
		return sourceBoundary{}, errors.New("release artifact production requires a clean repository")
	}

	commitOutput, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return sourceBoundary{}, errors.New("resolve release source commit")
	}
	commit := strings.TrimSpace(commitOutput)
	if !commitPattern.MatchString(commit) || strings.Trim(commit, "0") == "" {
		return sourceBoundary{}, errors.New("release source commit is invalid")
	}
	if _, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "verify-commit", commit); err != nil {
		return sourceBoundary{}, errors.New("release source commit signature verification failed")
	}

	versionData, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "show", commit+":VERSION")
	if err != nil {
		return sourceBoundary{}, errors.New("read VERSION from the release source commit")
	}
	version := strings.TrimSpace(versionData)
	if !stableVersionPattern.MatchString(version) {
		return sourceBoundary{}, errors.New("release artifact production requires a stable MAJOR.MINOR.PATCH VERSION")
	}
	if expectedVersion != "" && expectedVersion != version {
		return sourceBoundary{}, errors.New("expected version does not match the release source VERSION")
	}
	tag := "isras-v" + version

	typeOutput, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "cat-file", "-t", tag)
	if err != nil || strings.TrimSpace(typeOutput) != "tag" {
		return sourceBoundary{}, errors.New("release tag must exist locally as an annotated tag object")
	}
	if _, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "verify-tag", tag); err != nil {
		return sourceBoundary{}, errors.New("release tag signature verification failed")
	}
	tagCommitOutput, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "rev-parse", tag+"^{commit}")
	if err != nil || strings.TrimSpace(tagCommitOutput) != commit {
		return sourceBoundary{}, errors.New("release tag does not point to the exact release source commit")
	}

	originOutput, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "remote", "get-url", "origin")
	if err != nil || !canonicalOrigin(strings.TrimSpace(originOutput)) {
		return sourceBoundary{}, errors.New("origin is not the canonical Engineering Standards repository")
	}

	goMod, _, err := runner.Run(ctx, canonicalRoot, nil, "git", "show", commit+":go.mod")
	if err != nil {
		return sourceBoundary{}, errors.New("read go.mod from the release source commit")
	}
	goDirective, err := parseGoDirective(goMod)
	if err != nil {
		return sourceBoundary{}, err
	}
	goVersionOutput, _, err := runner.Run(ctx, canonicalRoot, sanitizedEnvironment(), "go", "env", "GOVERSION")
	if err != nil {
		return sourceBoundary{}, errors.New("read Go toolchain version")
	}
	goVersion := strings.TrimSpace(goVersionOutput)
	if goVersion != "go"+goDirective {
		return sourceBoundary{}, fmt.Errorf("Go toolchain %s does not match go.mod directive %s", goVersion, goDirective)
	}

	return sourceBoundary{
		Root: canonicalRoot, Version: version, Tag: tag, Commit: commit,
		GoVersion: goVersion, GoDirective: goDirective,
	}, nil
}

func parseGoDirective(goMod string) (string, error) {
	for _, line := range strings.Split(goMod, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "go" {
			if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(fields[1]) {
				return "", errors.New("go.mod must declare an exact patch-level Go version")
			}
			return fields[1], nil
		}
	}
	return "", errors.New("go.mod does not contain a Go version directive")
}

func canonicalOrigin(value string) bool {
	switch value {
	case "git@github.com:Iron-Signal-Systems/engineering-standards.git",
		"https://github.com/Iron-Signal-Systems/engineering-standards.git",
		"ssh://git@github.com/Iron-Signal-Systems/engineering-standards.git":
		return true
	default:
		return false
	}
}
