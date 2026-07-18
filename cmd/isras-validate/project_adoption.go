package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectadoption"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
)

func runProjectPinInitialize(ctx context.Context, runner *validation.Runner, args []string) int {
	options, err := parseProjectInitializationArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL: initialize project pin:", err)
		return 2
	}
	result, initializeErr := projectadoption.Initialize(ctx, projectadoption.Request{
		Root:              runner.Root,
		ReleaseTag:        options.ReleaseTag,
		EvidenceDirectory: options.EvidenceDirectory,
		GoDefaults:        options.GoDefaults,
	})
	if result.Report.ReleaseTag != "" {
		renderProjectArtifactVerification(os.Stdout, runner.Root, result.Report, "", "")
		fmt.Println()
	}
	if initializeErr != nil {
		fmt.Fprintln(os.Stderr, "FAIL: initialize project pin:", initializeErr)
		return 1
	}

	fmt.Println("ISRAS PROJECT ADOPTION")
	fmt.Println("======================")
	if result.Changed {
		fmt.Println("Adoption status:       CREATED")
	} else {
		fmt.Println("Adoption status:       ALREADY PRESENT")
	}
	fmt.Printf("Project:               %s\n", result.ProjectRepository)
	fmt.Printf("Release tag:           %s\n", result.ReleaseTag)
	fmt.Printf("Source commit:         %s\n", result.SourceCommit)
	fmt.Printf("Project pin:           %s\n", result.PinPath)
	fmt.Printf("Caller workflow:       %s\n", result.WorkflowPath)
	fmt.Printf("Verification evidence: %s\n", result.EvidencePath)
	fmt.Printf("Go format checker:     %s\n", result.FormatCheckPath)
	fmt.Println()
	fmt.Println("Review the adoption change:")
	fmt.Println(rootedShellCommand(runner.Root, "git diff -- .isras .github/workflows/isras-validation.yml"))
	fmt.Println()
	fmt.Println("Validate the generated declaration:")
	fmt.Printf("  %s project-pin validate\n", runner.Command)
	fmt.Println()
	fmt.Println("Commit the pin, caller workflow, and adoption evidence together after review.")
	return 0
}

type projectInitializationOptions struct {
	ReleaseTag        string
	EvidenceDirectory string
	GoDefaults        bool
}

func parseProjectInitializationArgs(args []string) (projectInitializationOptions, error) {
	var options projectInitializationOptions
	releaseSeen := false
	evidenceSeen := false
	goDefaultsSeen := false
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--release":
			if releaseSeen || index+1 >= len(args) {
				return projectInitializationOptions{}, errors.New("--release must be declared exactly once with a value")
			}
			index++
			options.ReleaseTag = args[index]
			releaseSeen = true
		case "--evidence-directory":
			if evidenceSeen || index+1 >= len(args) {
				return projectInitializationOptions{}, errors.New("--evidence-directory may be declared once with a value")
			}
			index++
			options.EvidenceDirectory = args[index]
			evidenceSeen = true
		case "--go-defaults":
			if goDefaultsSeen {
				return projectInitializationOptions{}, errors.New("--go-defaults may be declared only once")
			}
			options.GoDefaults = true
			goDefaultsSeen = true
		default:
			return projectInitializationOptions{}, fmt.Errorf("unknown project initialization option %q", args[index])
		}
	}
	if !releaseSeen || strings.TrimSpace(options.ReleaseTag) == "" {
		return projectInitializationOptions{}, errors.New("project initialization requires --release isras-vMAJOR.MINOR.PATCH")
	}
	if !options.GoDefaults {
		return projectInitializationOptions{}, errors.New("project initialization requires explicit --go-defaults authorization")
	}
	if evidenceSeen && (strings.TrimSpace(options.EvidenceDirectory) == "" || strings.ContainsAny(options.EvidenceDirectory, "\x00\r\n")) {
		return projectInitializationOptions{}, errors.New("--evidence-directory requires a valid bounded relative path")
	}
	return options, nil
}
