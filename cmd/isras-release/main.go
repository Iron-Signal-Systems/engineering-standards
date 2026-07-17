package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseworkflow"
)

func main() {
	os.Exit(run())
}

func run() int {
	return runWithIO(os.Args, os.Stdin, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdin io.Reader, stdout, stderr io.Writer) (code int) {
	safeOut := redact.NewWriter(stdout)
	safeErr := redact.NewWriter(stderr)
	defer func() {
		_ = safeOut.Flush()
		_ = safeErr.Flush()
	}()

	if len(args) < 2 {
		usage(safeErr)
		return 2
	}

	action := releaseworkflow.Action(args[1])
	switch action {
	case releaseworkflow.ActionCheck, releaseworkflow.ActionTag, releaseworkflow.ActionPublish:
	default:
		fmt.Fprintf(safeErr, "FAIL: unsupported action %q\n\n", args[1])
		usage(safeErr)
		return 2
	}

	set := flag.NewFlagSet("isras-release "+string(action), flag.ContinueOnError)
	set.SetOutput(safeErr)
	var (
		repo             string
		version          string
		branch           string
		remote           string
		githubRepository string
		title            string
		confirm          bool
		timeout          time.Duration
	)
	set.StringVar(&repo, "repo", "", "repository path; defaults to the current repository")
	set.StringVar(&version, "version", "", "expected release version; defaults to VERSION")
	set.StringVar(&branch, "branch", "dev", "authoritative release branch")
	set.StringVar(&remote, "remote", "origin", "authoritative Git remote")
	set.StringVar(&githubRepository, "github-repo", "", "GitHub owner/name; normally derived from origin")
	set.StringVar(&title, "title", "", "GitHub Release and tag title; defaults to the ISRAS baseline title")
	set.BoolVar(&confirm, "confirm", false, "confirm tag creation or remote publication")
	set.DurationVar(&timeout, "timeout", 90*time.Minute, "complete workflow timeout")
	set.Usage = func() {
		fmt.Fprintf(set.Output(), "Usage: isras-release %s [options]\n", action)
		set.PrintDefaults()
	}
	if err := set.Parse(args[2:]); err != nil {
		return 2
	}
	if set.NArg() != 0 {
		fmt.Fprintln(safeErr, "FAIL: unexpected positional arguments")
		set.Usage()
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := releaseworkflow.Run(ctx, releaseworkflow.Options{
		Root:             repo,
		Action:           action,
		ExpectedVersion:  version,
		Branch:           branch,
		Remote:           remote,
		GitHubRepository: githubRepository,
		Title:            title,
		Confirm:          confirm,
		Stdin:            stdin,
		Stdout:           safeOut,
		Stderr:           safeErr,
	})
	if err != nil {
		fmt.Fprintln(safeOut)
		fmt.Fprintln(safeErr, "RESULT")
		fmt.Fprintln(safeErr, "Overall status           ● FAIL  release workflow stopped safely")
		fmt.Fprintln(safeErr, "Reason                   ◆ INFO ", err)
		if result.LogPath != "" {
			fmt.Fprintln(safeErr, "Workflow log             ◆ INFO ", relative(result.RepositoryRoot, result.LogPath))
		}
		return 1
	}

	fmt.Fprintln(safeOut)
	fmt.Fprintln(safeOut, "RESULT")
	fmt.Fprintln(safeOut, "Overall status           ● PASS  release workflow stage completed")
	fmt.Fprintf(safeOut, "Action                   ● PASS  %s\n", action)
	fmt.Fprintf(safeOut, "Version                  ● PASS  %s\n", result.Version)
	fmt.Fprintf(safeOut, "Tag                      ● PASS  %s\n", result.Tag)
	fmt.Fprintf(safeOut, "Commit                   ● PASS  %s\n", result.Commit)
	if result.MainCommit != "" {
		fmt.Fprintf(safeOut, "Main                     ● PASS  %s\n", result.MainCommit)
	}
	if result.ReleaseURL != "" {
		fmt.Fprintf(safeOut, "GitHub Release           ● PASS  %s\n", result.ReleaseURL)
	}
	fmt.Fprintf(safeOut, "Workflow log             ◆ INFO  %s\n", relative(result.RepositoryRoot, result.LogPath))
	return 0
}

func usage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage: isras-release <check|tag|publish> [options]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "  check     validate the exact pushed release candidate without changing refs")
	fmt.Fprintln(writer, "  tag       run checks, then create or verify the signed local tag (--confirm)")
	fmt.Fprintln(writer, "  publish   run checks, push/verify the tag, fast-forward main, and publish GitHub Release (--confirm)")
}

func relative(root, path string) string {
	if root == "" || path == "" {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." {
		return path
	}
	return filepath.ToSlash(rel)
}
