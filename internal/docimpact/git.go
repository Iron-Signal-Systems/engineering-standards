package docimpact

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const maxGitOutputBytes = 16 * 1024 * 1024

type Comparison struct {
	RequestedBase string   `json:"requested_base"`
	RequestedHead string   `json:"requested_head"`
	BaseCommit    string   `json:"base_commit"`
	HeadCommit    string   `json:"head_commit"`
	MergeBase     string   `json:"merge_base"`
	ChangedPaths  []string `json:"changed_paths"`
}

func CollectComparison(
	ctx context.Context,
	root string,
	baseCommit string,
	headCommit string,
) (Comparison, error) {
	if ctx == nil {
		return Comparison{}, errors.New(
			"documentation-impact context is required",
		)
	}
	if !validCommitID(baseCommit) {
		return Comparison{}, errors.New(
			"documentation-impact base must be an exact 40-character commit ID",
		)
	}
	if !validCommitID(headCommit) {
		return Comparison{}, errors.New(
			"documentation-impact head must be an exact 40-character commit ID",
		)
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Comparison{}, errors.New(
			"resolve documentation-impact repository root",
		)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return Comparison{}, errors.New(
			"documentation-impact Git executable is unavailable",
		)
	}
	gitPath, err = filepath.Abs(gitPath)
	if err != nil {
		return Comparison{}, errors.New(
			"resolve documentation-impact Git executable",
		)
	}
	gitPath = filepath.Clean(gitPath)

	topLevel, err := runGitText(
		ctx,
		gitPath,
		absoluteRoot,
		"rev-parse",
		"--show-toplevel",
	)
	if err != nil {
		return Comparison{}, err
	}
	resolvedTop, err := filepath.Abs(strings.TrimSpace(topLevel))
	if err != nil || filepath.Clean(resolvedTop) != absoluteRoot {
		return Comparison{}, errors.New(
			"documentation-impact target is not the exact Git repository root",
		)
	}

	resolvedBase, err := resolveCommit(
		ctx,
		gitPath,
		absoluteRoot,
		baseCommit,
	)
	if err != nil {
		return Comparison{}, fmt.Errorf(
			"resolve documentation-impact base commit: %w",
			err,
		)
	}
	resolvedHead, err := resolveCommit(
		ctx,
		gitPath,
		absoluteRoot,
		headCommit,
	)
	if err != nil {
		return Comparison{}, fmt.Errorf(
			"resolve documentation-impact head commit: %w",
			err,
		)
	}

	mergeBaseText, err := runGitText(
		ctx,
		gitPath,
		absoluteRoot,
		"merge-base",
		resolvedBase,
		resolvedHead,
	)
	if err != nil {
		return Comparison{}, fmt.Errorf(
			"resolve documentation-impact merge base: %w",
			err,
		)
	}
	mergeBase := strings.TrimSpace(mergeBaseText)
	if !validCommitID(mergeBase) {
		return Comparison{}, errors.New(
			"documentation-impact merge base is not an exact commit ID",
		)
	}

	output, err := runGitBytes(
		ctx,
		gitPath,
		absoluteRoot,
		"diff",
		"--name-only",
		"-z",
		"--no-renames",
		"--diff-filter=ACDMRTUXB",
		mergeBase,
		resolvedHead,
		"--",
	)
	if err != nil {
		return Comparison{}, fmt.Errorf(
			"collect documentation-impact changed paths: %w",
			err,
		)
	}

	paths, err := parseNULPaths(output)
	if err != nil {
		return Comparison{}, err
	}
	return Comparison{
		RequestedBase: baseCommit,
		RequestedHead: headCommit,
		BaseCommit:    resolvedBase,
		HeadCommit:    resolvedHead,
		MergeBase:     mergeBase,
		ChangedPaths:  paths,
	}, nil
}

func resolveCommit(
	ctx context.Context,
	gitPath string,
	root string,
	commit string,
) (string, error) {
	text, err := runGitText(
		ctx,
		gitPath,
		root,
		"rev-parse",
		"--verify",
		commit+"^{commit}",
	)
	if err != nil {
		return "", err
	}
	resolved := strings.TrimSpace(text)
	if !validCommitID(resolved) {
		return "", errors.New(
			"Git did not return an exact commit ID",
		)
	}
	return resolved, nil
}

func runGitText(
	ctx context.Context,
	gitPath string,
	root string,
	args ...string,
) (string, error) {
	output, err := runGitBytes(ctx, gitPath, root, args...)
	return string(output), err
}

func runGitBytes(
	ctx context.Context,
	gitPath string,
	root string,
	args ...string,
) ([]byte, error) {
	commandArgs := []string{
		"-c",
		"core.quotepath=false",
		"-c",
		"safe.directory=" + root,
		"-C",
		root,
	}
	commandArgs = append(commandArgs, args...)

	command := exec.CommandContext(ctx, gitPath, commandArgs...)
	command.Env = []string{
		"PATH=" + filepath.Dir(gitPath) + ":/usr/bin:/bin",
		"LC_ALL=C",
		"LANG=C",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
	}

	var stdout limitedBuffer
	var stderr limitedBuffer
	stdout.limit = maxGitOutputBytes
	stderr.limit = 1024 * 1024
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) ||
			errors.Is(ctx.Err(), context.Canceled) {
			return nil, ctx.Err()
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, errors.New(detail)
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, errors.New(
			"documentation-impact Git output exceeded its bounded limit",
		)
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func parseNULPaths(output []byte) ([]string, error) {
	if len(output) == 0 {
		return []string{}, nil
	}
	if output[len(output)-1] != 0 {
		return nil, errors.New(
			"documentation-impact Git path output is not NUL terminated",
		)
	}

	set := make(map[string]struct{})
	for _, value := range bytes.Split(output[:len(output)-1], []byte{0}) {
		path := string(value)
		if !safeRepositoryPath(path) {
			return nil, fmt.Errorf(
				"documentation-impact Git returned unsafe path %q",
				path,
			)
		}
		set[path] = struct{}{}
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func validCommitID(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (buffer *limitedBuffer) Write(value []byte) (int, error) {
	if buffer.exceeded {
		return len(value), nil
	}
	remaining := buffer.limit - buffer.Len()
	if remaining <= 0 {
		buffer.exceeded = true
		return len(value), nil
	}
	if len(value) > remaining {
		_, _ = buffer.Buffer.Write(value[:remaining])
		buffer.exceeded = true
		return len(value), nil
	}
	return buffer.Buffer.Write(value)
}
