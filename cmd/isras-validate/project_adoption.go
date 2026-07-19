package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectadoption"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

func runProjectPinInitialize(ctx context.Context, runner *validation.Runner, args []string) int {
	options, err := parseProjectInitializationArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL: initialize project pin:", err)
		return 2
	}
	validator, configured, identityErr := validatoridentity.Embedded()
	if identityErr != nil {
		fmt.Fprintln(os.Stderr, "FAIL: initialize project pin: load linker-bound release validator identity:", identityErr)
		return 1
	}
	if !configured {
		fmt.Fprintln(os.Stderr, "FAIL: initialize project pin: project initialization requires a linker-bound release validator")
		return 1
	}
	result, initializeErr := projectadoption.Initialize(ctx, projectadoption.Request{
		Root:       runner.Root,
		ReleaseTag: options.ReleaseTag,
		GoDefaults: options.GoDefaults,
		Validator:  validator,
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
	fmt.Printf("Runtime evidence:      %s\n", projectadoption.DefaultEvidenceDirectory)
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
	ReleaseTag string
	GoDefaults bool
}

func parseProjectInitializationArgs(args []string) (projectInitializationOptions, error) {
	var options projectInitializationOptions
	releaseSeen := false
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
	return options, nil
}
