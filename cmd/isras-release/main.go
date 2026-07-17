package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseworkflow"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		usage()
		return 2
	}

	action := releaseworkflow.Action(os.Args[1])
	switch action {
	case releaseworkflow.ActionCheck, releaseworkflow.ActionTag, releaseworkflow.ActionPublish:
	default:
		fmt.Fprintf(os.Stderr, "FAIL: unsupported action %q\n\n", os.Args[1])
		usage()
		return 2
	}

	set := flag.NewFlagSet("isras-release "+string(action), flag.ContinueOnError)
	set.SetOutput(os.Stderr)
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
	if err := set.Parse(os.Args[2:]); err != nil {
		return 2
	}
	if set.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "FAIL: unexpected positional arguments")
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
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
	})
	if err != nil {
		fmt.Println()
		fmt.Fprintln(os.Stderr, "RESULT")
		fmt.Fprintln(os.Stderr, "Overall status           ● FAIL  release workflow stopped safely")
		fmt.Fprintln(os.Stderr, "Reason                   ◆ INFO ", err)
		if result.LogPath != "" {
			fmt.Fprintln(os.Stderr, "Workflow log             ◆ INFO ", relative(result.RepositoryRoot, result.LogPath))
		}
		return 1
	}

	fmt.Println()
	fmt.Println("RESULT")
	fmt.Println("Overall status           ● PASS  release workflow stage completed")
	fmt.Printf("Action                   ● PASS  %s\n", action)
	fmt.Printf("Version                  ● PASS  %s\n", result.Version)
	fmt.Printf("Tag                      ● PASS  %s\n", result.Tag)
	fmt.Printf("Commit                   ● PASS  %s\n", result.Commit)
	if result.MainCommit != "" {
		fmt.Printf("Main                     ● PASS  %s\n", result.MainCommit)
	}
	if result.ReleaseURL != "" {
		fmt.Printf("GitHub Release           ● PASS  %s\n", result.ReleaseURL)
	}
	fmt.Printf("Workflow log             ◆ INFO  %s\n", relative(result.RepositoryRoot, result.LogPath))
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: isras-release <check|tag|publish> [options]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  check     validate the exact pushed release candidate without changing refs")
	fmt.Fprintln(os.Stderr, "  tag       run checks, then create or verify the signed local tag (--confirm)")
	fmt.Fprintln(os.Stderr, "  publish   run checks, push/verify the tag, fast-forward main, and publish GitHub Release (--confirm)")
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
