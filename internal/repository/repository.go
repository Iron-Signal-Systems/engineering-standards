package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
)

const maxRequestedPathLength = 4096

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

type Identity struct {
	Root   string
	Branch string
	Commit string
	Origin string
}

func Discover(ctx context.Context) (Identity, error) {
	return DiscoverFrom(ctx, "")
}

func DiscoverFrom(ctx context.Context, requested string) (Identity, error) {
	start, err := resolveStartDirectory(requested)
	if err != nil {
		return Identity{}, err
	}
	if err := rejectSymlinkComponents(start); err != nil {
		return Identity{}, err
	}

	rootResult := executil.Run(ctx, start, "git", "rev-parse", "--show-toplevel")
	if rootResult.Err != nil {
		return Identity{}, errors.New("target path is not inside a Git repository")
	}
	root := strings.TrimSpace(rootResult.Stdout)
	if root == "" || strings.ContainsAny(root, "\x00\r\n") {
		return Identity{}, errors.New("Git returned an invalid repository root")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return Identity{}, errors.New("resolve repository root")
	}
	root = filepath.Clean(root)
	if err := rejectSymlinkComponents(root); err != nil {
		return Identity{}, errors.New("repository root contains a symbolic-link component")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return Identity{}, errors.New("repository root is not a regular directory")
	}
	if !pathWithin(root, start) {
		return Identity{}, errors.New("requested target path is outside the discovered repository root")
	}

	commitResult := executil.Run(ctx, root, "git", "rev-parse", "--verify", "HEAD^{commit}")
	if commitResult.Err != nil {
		return Identity{}, errors.New("target repository does not have a resolvable HEAD commit")
	}
	commit := strings.TrimSpace(commitResult.Stdout)
	if !commitPattern.MatchString(commit) || strings.Trim(commit, "0") == "" {
		return Identity{}, errors.New("target repository returned an invalid HEAD commit")
	}

	branch := optionalOutput(ctx, root, "git", "branch", "--show-current")
	origin := optionalOutput(ctx, root, "git", "remote", "get-url", "origin")
	return Identity{Root: root, Branch: branch, Commit: commit, Origin: origin}, nil
}

func resolveStartDirectory(requested string) (string, error) {
	start := requested
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", errors.New("read current working directory")
		}
		start = cwd
	}
	if len(start) > maxRequestedPathLength || strings.ContainsAny(start, "\x00\r\n") {
		return "", errors.New("target repository path is invalid or oversized")
	}
	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", errors.New("resolve target repository path")
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Lstat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("target repository path does not exist")
		}
		return "", errors.New("inspect target repository path")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("target repository path must not be a symbolic link")
	}
	if !info.IsDir() {
		return "", errors.New("target repository path is not a directory")
	}
	return absolute, nil
}

func rejectSymlinkComponents(value string) error {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return errors.New("resolve path for symbolic-link inspection")
	}
	absolute = filepath.Clean(absolute)
	volume := filepath.VolumeName(absolute)
	root := volume + string(filepath.Separator)
	remainder := strings.TrimPrefix(absolute, root)
	current := root
	if remainder == absolute {
		current = string(filepath.Separator)
		remainder = strings.TrimPrefix(absolute, current)
	}
	if remainder == "" {
		return nil
	}
	for _, component := range strings.Split(remainder, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New("inspect target repository path component")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("target repository path contains a symbolic-link component")
		}
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func optionalOutput(ctx context.Context, root, name string, args ...string) string {
	result := executil.Run(ctx, root, name, args...)
	if result.Err != nil {
		return ""
	}
	return strings.TrimSpace(result.Stdout)
}
