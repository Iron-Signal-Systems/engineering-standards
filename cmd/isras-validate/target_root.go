package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

type globalOptions struct {
	Mode       string
	Repository string
}

func runTargetAware(rawArgs []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	args, options, err := parseGlobalOptions(rawArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}
	if len(args) == 0 {
		args = []string{"all"}
	}

	validatorIdentity, err := discoverValidatorIdentity(ctx, options.Repository)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}
	standaloneCommand := canonicalCommandForTarget("", false)

	switch args[0] {
	case "version", "--version":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "FAIL: version accepts no positional arguments")
			return 2
		}
		renderIdentity(os.Stdout, validatorIdentity)
		return 0
	case "help", "-h", "--help":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "FAIL: help accepts no positional arguments")
			return 2
		}
		usage(os.Stdout, standaloneCommand, validatorIdentity.Header())
		return 0
	}

	if !knownTargetCommand(args[0]) {
		fmt.Fprintf(os.Stderr, "FAIL: unknown command %q\n\n", args[0])
		usage(os.Stderr, standaloneCommand, validatorIdentity.Header())
		return 2
	}

	targetIdentity, err := repository.DiscoverFrom(ctx, options.Repository)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}
	command := canonicalCommandForTarget(targetIdentity.Root, options.Repository != "")
	runner, err := validation.NewForIdentity(options.Mode, command, targetIdentity)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}

	switch args[0] {
	case "all":
		return render(validatorIdentity.Header(), runner.All(ctx))
	case "system":
		return render(validatorIdentity.Header(), model.Summary{Checks: runner.System(ctx)})
	case "repo", "repository":
		return render(validatorIdentity.Header(), model.Summary{Checks: runner.Repository(ctx)})
	case "project-pin":
		return runProjectPin(ctx, runner, args[1:])
	case "go":
		return runGo(ctx, runner, validatorIdentity.Header(), args[1:])
	case "secrets":
		return runSecrets(ctx, runner, validatorIdentity.Header(), args[1:])
	case "fix":
		return runFix(ctx, runner, args[1:])
	default:
		panic("unreachable target command")
	}
}

func parseGlobalOptions(args []string) ([]string, globalOptions, error) {
	options := globalOptions{Mode: "development"}
	out := make([]string, 0, len(args))
	modeSeen := false
	repositorySeen := false

	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--mode":
			if modeSeen {
				return nil, globalOptions{}, errors.New("--mode may be declared only once")
			}
			if index+1 >= len(args) {
				return nil, globalOptions{}, errors.New("--mode requires development, commit, or release")
			}
			index++
			options.Mode = args[index]
			modeSeen = true
		case strings.HasPrefix(argument, "--mode="):
			if modeSeen {
				return nil, globalOptions{}, errors.New("--mode may be declared only once")
			}
			options.Mode = strings.TrimPrefix(argument, "--mode=")
			modeSeen = true
		case argument == "--repo":
			if repositorySeen {
				return nil, globalOptions{}, errors.New("--repo may be declared only once")
			}
			if index+1 >= len(args) {
				return nil, globalOptions{}, errors.New("--repo requires a repository path")
			}
			index++
			options.Repository = args[index]
			repositorySeen = true
		case strings.HasPrefix(argument, "--repo="):
			if repositorySeen {
				return nil, globalOptions{}, errors.New("--repo may be declared only once")
			}
			options.Repository = strings.TrimPrefix(argument, "--repo=")
			repositorySeen = true
		default:
			out = append(out, argument)
		}
	}

	if options.Mode != "development" && options.Mode != "commit" && options.Mode != "release" {
		return nil, globalOptions{}, fmt.Errorf("unsupported validation mode %q", options.Mode)
	}
	if repositorySeen {
		if options.Repository == "" || len(options.Repository) > 4096 || strings.ContainsAny(options.Repository, "\x00\r\n") {
			return nil, globalOptions{}, errors.New("--repo requires a valid bounded repository path")
		}
	}
	return out, options, nil
}

func knownTargetCommand(command string) bool {
	switch command {
	case "all", "system", "repo", "repository", "project-pin", "go", "secrets", "fix":
		return true
	default:
		return false
	}
}

func discoverValidatorIdentity(ctx context.Context, requestedTarget string) (validatoridentity.Identity, error) {
	if identity, configured, err := validatoridentity.Embedded(); configured || err != nil {
		return identity, err
	}

	candidates := make([]string, 0, 3)
	if executable, err := os.Executable(); err == nil && executable != "" {
		candidates = append(candidates, filepath.Dir(executable))
	}
	if workingDirectory, err := os.Getwd(); err == nil && workingDirectory != "" {
		candidates = append(candidates, workingDirectory)
	}
	if requestedTarget != "" {
		candidates = append(candidates, requestedTarget)
	}

	seen := make(map[string]bool)
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		absolute = filepath.Clean(absolute)
		if seen[absolute] {
			continue
		}
		seen[absolute] = true
		identityRepository, err := repository.DiscoverFrom(ctx, absolute)
		if err != nil {
			continue
		}
		identity, err := validatoridentity.Load(identityRepository.Root, identityRepository.Commit)
		if err == nil {
			return identity, nil
		}
	}
	return validatoridentity.Identity{}, errors.New("validator identity is unavailable outside its source repository")
}
