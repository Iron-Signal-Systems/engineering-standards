package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifactbuild"
)

func main() {
	os.Exit(run())
}

func run() int {
	return runWithIO(os.Args, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || args[1] != "build" {
		usage(stderr)
		return 2
	}
	set := flag.NewFlagSet("isras-release-artifacts build", flag.ContinueOnError)
	set.SetOutput(stderr)
	var options releaseartifactbuild.Options
	var timeout time.Duration
	set.StringVar(&options.Root, "repo", "", "repository path; defaults to the current repository")
	set.StringVar(&options.OutputDirectory, "output", "", "artifact output directory; defaults under .local/releases")
	set.StringVar(&options.ExpectedVersion, "version", "", "expected stable release version; defaults to VERSION")
	set.StringVar(&options.PublishedAt, "published-at", "", "RFC3339 release publication timestamp")
	set.StringVar(&options.ValidationCampaign, "validation-campaign", "", "accepted release-validation campaign identity")
	set.StringVar(&options.ReleaseAuthority, "release-authority", "", "bounded release authority identity")
	set.DurationVar(&timeout, "timeout", 30*time.Minute, "complete artifact-production timeout")
	set.Usage = func() {
		fmt.Fprintln(set.Output(), "Usage: isras-release-artifacts build [options]")
		set.PrintDefaults()
	}
	if err := set.Parse(args[2:]); err != nil {
		return 2
	}
	if set.NArg() != 0 {
		fmt.Fprintln(stderr, "FAIL: unexpected positional arguments")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := releaseartifactbuild.Build(ctx, options)
	if err != nil {
		fmt.Fprintln(stderr, "FAIL:", err)
		return 1
	}
	fmt.Fprintln(stdout, "ISRAS RELEASE ARTIFACT PRODUCTION")
	fmt.Fprintln(stdout, "=================================")
	fmt.Fprintln(stdout, "Status:             PASS")
	fmt.Fprintf(stdout, "Version:            %s\n", result.Version)
	fmt.Fprintf(stdout, "Release tag:        %s\n", result.ReleaseTag)
	fmt.Fprintf(stdout, "Source commit:      %s\n", result.SourceCommit)
	fmt.Fprintf(stdout, "Go version:         %s\n", result.GoVersion)
	fmt.Fprintf(stdout, "Artifacts produced: %d\n", len(result.Artifacts))
	fmt.Fprintf(stdout, "Output directory:   %s\n", relative(result.OutputDirectory))
	fmt.Fprintf(stdout, "JSON evidence:      %s\n", relative(result.EvidenceJSON))
	fmt.Fprintf(stdout, "Text evidence:      %s\n", relative(result.EvidenceText))
	fmt.Fprintln(stdout, "Artifact execution: NOT PERFORMED")
	fmt.Fprintln(stdout, "Artifact publish:   NOT PERFORMED")
	return 0
}

func usage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage: isras-release-artifacts build [options]")
}

func relative(value string) string {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(value)
	}
	relativePath, err := filepath.Rel(workingDirectory, value)
	if err != nil || relativePath == ".." || len(relativePath) >= 3 && relativePath[:3] == "../" {
		return filepath.ToSlash(value)
	}
	return filepath.ToSlash(relativePath)
}
