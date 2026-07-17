package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/releasevalidation"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		repo    string
		ref     string
		timeout time.Duration
	)
	flag.StringVar(&repo, "repo", "", "repository path; defaults to the current repository")
	flag.StringVar(&ref, "ref", "", "remote branch ref; defaults to the current branch")
	flag.DurationVar(&timeout, "timeout", 45*time.Minute, "complete validation timeout")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "FAIL: unexpected positional arguments")
		flag.Usage()
		return 2
	}

	command := canonicalCommand()
	fmt.Println("IRON SIGNAL · CLEAN-CLONE RELEASE VALIDATION")
	fmt.Println("Exact pushed source · committed tests · release-mode validation")
	fmt.Println("────────────────────────────────────────────────────────────")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := releasevalidation.Run(ctx, releasevalidation.Options{
		Root:    repo,
		Ref:     ref,
		Command: command,
	})
	if err != nil {
		fmt.Println()
		fmt.Fprintln(os.Stderr, "RESULT")
		fmt.Fprintln(os.Stderr, "Overall status           ● FAIL  clean-clone release validation failed")
		fmt.Fprintln(os.Stderr, "Reason                   ◆ INFO ", err)
		if result.LogPath != "" {
			fmt.Fprintln(os.Stderr, "Failure log              ◆ INFO ", relative(result.RepositoryRoot, result.LogPath))
			fmt.Println()
			fmt.Println("[READ ONLY] Review the complete local failure log:")
			fmt.Printf("  sed -n '1,320p' '%s'\n\n", relative(result.RepositoryRoot, result.LogPath))
			fmt.Println("[READ ONLY] Review retained clean-clone evidence:")
			fmt.Printf("  find '%s' -maxdepth 2 -type f -print\n\n", relative(result.RepositoryRoot, result.RunDirectory))
		}
		fmt.Println("[NETWORK AND LOCAL FILE WRITES] Rerun the exact clean-clone campaign after correction:")
		fmt.Printf("  %s", command)
		if ref != "" {
			fmt.Printf(" --ref %s", shellQuote(ref))
		}
		fmt.Println()
		return 1
	}

	fmt.Println()
	fmt.Println("RESULT")
	fmt.Println("Overall status           ● PASS  exact pushed commit passed clean-clone release validation")
	fmt.Printf("Commit                   ● PASS  %s\n", result.Commit)
	fmt.Printf("Release summary          ◆ INFO  %s\n", relative(result.RepositoryRoot, result.SummaryPath))
	fmt.Printf("Validation log           ◆ INFO  %s\n", relative(result.RepositoryRoot, result.LogPath))
	fmt.Printf("Retained clone           ◆ INFO  %s\n", relative(result.RepositoryRoot, result.CloneDirectory))
	fmt.Println()
	fmt.Println("Ready for release-candidate preparation. No tag was created.")
	return 0
}

func canonicalCommand() string {
	executable, err := os.Executable()
	if err != nil {
		return "./.local/bin/isras-release-validate"
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "./.local/bin/isras-release-validate"
	}
	return shellQuote(executable)
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

func shellQuote(value string) string {
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') &&
			!(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') &&
			r != '/' && r != '.' && r != '_' && r != '-' && r != ':' && r != '@' {
			return "'" + value + "'"
		}
	}
	return value
}
