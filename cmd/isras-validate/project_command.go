package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectcommand"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

func runProjectCommand(ctx context.Context, runner *validation.Runner, validator validatoridentity.Identity, args []string) int {
	if len(args) != 2 || args[0] != "run" {
		fmt.Fprintln(os.Stderr, "FAIL: usage is project-command run NAME")
		return 2
	}
	pin, err := projectpin.LoadCommitted(ctx, runner.Root)
	if err != nil {
		return actionFailure(runner, "authorize project command", err)
	}
	result, executeErr := projectcommand.Execute(ctx, projectcommand.Request{
		Root:      runner.Root,
		Mode:      runner.Mode,
		Target:    runner.Identity,
		Validator: validator,
		Pin:       pin,
		Name:      args[1],
	})
	renderProjectCommandResult(result, runner.Root)
	if executeErr != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", redact.Sanitize(executeErr.Error()))
		return 1
	}
	return 0
}

func renderProjectCommandResult(result projectcommand.Result, root string) {
	fmt.Println("PROJECT COMMAND EXECUTION")
	fmt.Println("=========================")
	fmt.Printf("Authorization:          %s\n", result.Authorization)
	fmt.Printf("Command:                %s\n", result.CommandName)
	fmt.Printf("Status:                 %s\n", result.Status)
	fmt.Printf("Exit code:              %d\n", result.ExitCode)
	fmt.Printf("Timed out:              %t\n", result.TimedOut)
	fmt.Printf("Output limit exceeded:  %t\n", result.OutputLimitExceeded)
	fmt.Printf("Repository state drift: %t\n", result.RepositoryStateChanged)
	if result.Failure != "" {
		fmt.Printf("Failure:                %s\n", redact.Sanitize(result.Failure))
	}
	if result.EvidenceJSON != "" {
		fmt.Printf("JSON evidence:          %s\n", relativePath(root, result.EvidenceJSON))
	}
	if result.EvidenceText != "" {
		fmt.Printf("Text evidence:          %s\n", relativePath(root, result.EvidenceText))
	}
}

func relativePath(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(value)
}
