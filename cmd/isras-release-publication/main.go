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
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releasepublication"
)

func main() {
	os.Exit(run())
}

func run() int {
	return runWithIO(os.Args, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdout, stderr io.Writer) int {
	safeOut := redact.NewWriter(stdout)
	safeErr := redact.NewWriter(stderr)
	defer safeOut.Flush()
	defer safeErr.Flush()

	if len(args) < 2 {
		usage(safeErr)
		return 2
	}
	action := releasepublication.Action(args[1])
	if action != releasepublication.ActionCheck && action != releasepublication.ActionPublish {
		fmt.Fprintf(safeErr, "FAIL: unsupported publication action %q\n\n", args[1])
		usage(safeErr)
		return 2
	}

	set := flag.NewFlagSet("isras-release-publication "+string(action), flag.ContinueOnError)
	set.SetOutput(safeErr)
	var options releasepublication.Options
	var timeout time.Duration
	options.Action = action
	set.StringVar(&options.Root, "repo", "", "repository path; defaults to the current repository")
	set.StringVar(&options.ExpectedVersion, "version", "", "expected stable version; defaults to VERSION")
	set.StringVar(&options.Branch, "branch", "dev", "authoritative release branch")
	set.StringVar(&options.Remote, "remote", "origin", "authoritative Git remote")
	set.StringVar(&options.GitHubRepository, "github-repo", "", "GitHub owner/name; normally derived from origin")
	set.StringVar(&options.ArtifactDirectory, "artifacts", "", "deterministic six-file artifact directory")
	set.StringVar(&options.BuildEvidence, "build-evidence", "", "private artifact-build JSON evidence")
	set.StringVar(&options.NotesFile, "notes", "", "release notes path; defaults to docs/releases/VERSION.md")
	set.StringVar(&options.Title, "title", "", "release title; defaults to the ISRAS baseline title")
	set.BoolVar(&options.Confirm, "confirm", false, "confirm creation and publication of a GitHub Release")
	set.DurationVar(&timeout, "timeout", 45*time.Minute, "complete publication timeout")
	set.Usage = func() {
		fmt.Fprintf(set.Output(), "Usage: isras-release-publication %s [options]\n", action)
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
	result, err := releasepublication.Run(ctx, options)
	if err != nil {
		fmt.Fprintln(safeErr, "ISRAS RELEASE PUBLICATION")
		fmt.Fprintln(safeErr, "=========================")
		fmt.Fprintln(safeErr, "Status: FAIL")
		fmt.Fprintln(safeErr, "Reason:", err)
		if result.EvidenceJSON != "" {
			fmt.Fprintln(safeErr, "JSON evidence:", displayPath(result.RepositoryRoot, result.EvidenceJSON))
			fmt.Fprintln(safeErr, "Text evidence:", displayPath(result.RepositoryRoot, result.EvidenceText))
		}
		return 1
	}

	fmt.Fprintln(safeOut, "ISRAS RELEASE PUBLICATION")
	fmt.Fprintln(safeOut, "=========================")
	fmt.Fprintln(safeOut, "Status: PASS")
	fmt.Fprintf(safeOut, "Action: %s\n", result.Action)
	fmt.Fprintf(safeOut, "Version: %s\n", result.Version)
	fmt.Fprintf(safeOut, "Release tag: %s\n", result.ReleaseTag)
	fmt.Fprintf(safeOut, "Source commit: %s\n", result.SourceCommit)
	fmt.Fprintf(safeOut, "Artifacts verified: %d\n", len(result.Artifacts))
	if result.Action == string(releasepublication.ActionCheck) {
		fmt.Fprintln(safeOut, "Remote mutation: NOT PERFORMED")
	} else {
		fmt.Fprintln(safeOut, "Remote publication: PASS")
		fmt.Fprintf(safeOut, "Release URL: %s\n", result.ReleaseURL)
	}
	fmt.Fprintln(safeOut, "Tag creation: NOT PERFORMED")
	fmt.Fprintln(safeOut, "Tag push: NOT PERFORMED")
	fmt.Fprintln(safeOut, "Main branch update: NOT PERFORMED")
	fmt.Fprintln(safeOut, "JSON evidence:", displayPath(result.RepositoryRoot, result.EvidenceJSON))
	fmt.Fprintln(safeOut, "Text evidence:", displayPath(result.RepositoryRoot, result.EvidenceText))
	return 0
}

func usage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage: isras-release-publication <check|publish> [options]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "  check    verify exact local artifacts, remote branch, signed tag, and release absence")
	fmt.Fprintln(writer, "  publish  create a draft, upload and reverify exact assets, then publish (--confirm)")
}

func displayPath(root, path string) string {
	if root == "" || path == "" {
		return filepath.ToSlash(path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || len(relative) > 3 && relative[:3] == "../" {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
