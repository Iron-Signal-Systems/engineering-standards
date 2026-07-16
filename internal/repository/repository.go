package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
)

type Identity struct {
	Root   string
	Branch string
	Commit string
	Origin string
}

func Discover(ctx context.Context) (Identity, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Identity{}, err
	}
	rootResult := executil.Run(ctx, cwd, "git", "rev-parse", "--show-toplevel")
	if rootResult.Err != nil {
		return Identity{}, errors.New("current directory is not inside a Git repository")
	}
	root := strings.TrimSpace(rootResult.Stdout)
	if root == "" {
		return Identity{}, errors.New("Git returned an empty repository root")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return Identity{}, err
	}
	branch := output(ctx, root, "git", "branch", "--show-current")
	commit := output(ctx, root, "git", "rev-parse", "HEAD")
	origin := output(ctx, root, "git", "remote", "get-url", "origin")
	return Identity{Root: root, Branch: branch, Commit: commit, Origin: origin}, nil
}

func output(ctx context.Context, root, name string, args ...string) string {
	result := executil.Run(ctx, root, name, args...)
	if result.Err != nil {
		return ""
	}
	return strings.TrimSpace(result.Stdout)
}
