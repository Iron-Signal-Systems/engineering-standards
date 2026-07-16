package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/dashboard"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/failurelog"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/secrets"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
)

const profile = "ISRAS-SD 0.1.0-development"

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	args, mode, err := parseMode(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}
	identity, err := repository.Discover(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}
	command := canonicalCommand(identity.Root)
	runner, err := validation.New(ctx, mode, command)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}
	if len(args) == 0 {
		args = []string{"all"}
	}

	switch args[0] {
	case "all":
		return render(runner.All(ctx))
	case "system":
		return render(model.Summary{Checks: runner.System(ctx)})
	case "repo", "repository":
		return render(model.Summary{Checks: runner.Repository(ctx)})
	case "go":
		return runGo(ctx, runner, args[1:])
	case "secrets":
		return runSecrets(ctx, runner, args[1:])
	case "fix":
		return runFix(ctx, runner, args[1:])
	case "help", "-h", "--help":
		usage(command)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "FAIL: unknown command %q\n\n", args[0])
		usage(command)
		return 2
	}
}

func parseMode(args []string) ([]string, string, error) {
	mode := "development"
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--mode" {
			if i+1 >= len(args) {
				return nil, "", errors.New("--mode requires development, commit, or release")
			}
			mode = args[i+1]
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out, mode, nil
}

func render(summary model.Summary) int {
	printer := dashboard.New(os.Stdout)
	printer.Header(profile)
	printer.Checks(summary.Checks)
	printer.Footer(summary)
	if summary.Failed() {
		return 1
	}
	return 0
}

func runGo(ctx context.Context, runner *validation.Runner, args []string) int {
	checks := runner.Go(ctx)
	if len(args) == 0 {
		return render(model.Summary{Checks: checks})
	}
	name := ""
	switch args[0] {
	case "formatting":
		if len(args) > 1 && args[1] == "--diff" {
			return formattingDiff(ctx, runner)
		}
		name = "Canonical formatting"
	case "vet":
		name = "Static analysis"
	case "tests":
		name = "Package tests"
	case "build":
		name = "Build"
	case "modules":
		return render(model.Summary{Checks: filterChecks(checks, "Module consistency", "Module integrity")})
	case "vulnerabilities":
		name = "Known vulnerabilities"
	default:
		fmt.Fprintf(os.Stderr, "FAIL: unknown Go validation command %q\n", args[0])
		return 2
	}
	return render(model.Summary{Checks: filterChecks(checks, name)})
}

func formattingDiff(ctx context.Context, runner *validation.Runner) int {
	files, err := goFiles(ctx, runner.Root)
	if err != nil {
		return actionFailure(runner, "formatting diff", err)
	}
	if len(files) == 0 {
		fmt.Println("No Go files found.")
		return 0
	}
	result := executil.Run(ctx, runner.Root, "gofmt", append([]string{"-d"}, files...)...)
	if result.Stdout != "" {
		fmt.Print(result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}
	if result.Err != nil {
		return actionCommandFailure(runner, "formatting diff", result)
	}
	if strings.TrimSpace(result.Stdout) == "" {
		fmt.Println("All Go files are canonically formatted.")
	}
	return 0
}

func runFix(ctx context.Context, runner *validation.Runner, args []string) int {
	if len(args) != 1 || args[0] != "formatting" {
		fmt.Fprintln(os.Stderr, "FAIL: supported repair command is: fix formatting")
		return 2
	}
	files, err := goFiles(ctx, runner.Root)
	if err != nil {
		return actionFailure(runner, "apply formatting", err)
	}
	if len(files) == 0 {
		fmt.Println("No Go files found.")
		return 0
	}
	result := executil.Run(ctx, runner.Root, "gofmt", append([]string{"-w"}, files...)...)
	if result.Err != nil {
		return actionCommandFailure(runner, "apply formatting", result)
	}
	fmt.Printf("Canonical formatting applied to %d Go file(s).\n\n", len(files))
	fmt.Println("Review the working-tree changes:")
	fmt.Println("  git diff -- '*.go'")
	fmt.Println()
	fmt.Println("Rerun formatting validation:")
	fmt.Println("  " + runner.Command + " go formatting")
	return 0
}

func runSecrets(ctx context.Context, runner *validation.Runner, args []string) int {
	if len(args) == 0 {
		return render(model.Summary{Checks: runner.Secrets(ctx)})
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "FAIL: secret action requires a finding ID")
		return 2
	}
	action, id := args[0], args[1]
	switch action {
	case "inspect":
		finding, err := secrets.Find(ctx, runner.Root, id)
		if err != nil {
			return actionFailure(runner, "inspect secret finding", err)
		}
		fmt.Println("SECRET FINDING")
		fmt.Println("==============")
		fmt.Printf("Finding ID:   %s\n", finding.ID)
		fmt.Printf("Rule:         %s\n", finding.Rule)
		fmt.Printf("Severity:     %s\n", finding.Severity)
		fmt.Printf("Location:     %s:%d:%d\n", finding.Path, finding.Line, finding.Column)
		fmt.Printf("Redactable:   %t\n", finding.Redactable)
		fmt.Printf("Allowable:    %t\n", finding.Allowable)
		fmt.Println("Detected value: [REDACTED]")
		return 0
	case "prepare-redaction":
		path, finding, err := secrets.PrepareRedaction(ctx, runner.Root, id)
		if err != nil {
			return actionFailure(runner, "prepare secret redaction", err)
		}
		fmt.Println("REDACTION PLAN PREPARED")
		fmt.Println("=======================")
		fmt.Printf("Finding:         %s\n", finding.ID)
		fmt.Printf("Affected source: %s:%d\n", finding.Path, finding.Line)
		fmt.Printf("Source modified: no\n")
		fmt.Printf("Local plan:      %s\n\n", relative(runner.Root, path))
		fmt.Println("[READ ONLY] Review the non-secret plan metadata:")
		fmt.Printf("  cat '%s'\n\n", relative(runner.Root, path))
		fmt.Println("[MODIFIES WORKING TREE] Apply the prepared plan:")
		fmt.Printf("  %s secrets apply-redaction %s\n", runner.Command, id)
		return 0
	case "apply-redaction":
		finding, err := secrets.ApplyRedaction(ctx, runner.Root, id)
		if err != nil {
			return actionFailure(runner, "apply secret redaction", err)
		}
		fmt.Println("REDACTION APPLIED")
		fmt.Println("=================")
		fmt.Printf("Finding:      %s\n", finding.ID)
		fmt.Printf("Affected file: %s\n", finding.Path)
		fmt.Println("Detected value: [REDACTED]")
		fmt.Println()
		fmt.Println("[READ ONLY] Review the source change:")
		fmt.Printf("  git diff -- '%s'\n\n", shellDisplay(finding.Path))
		fmt.Println("Rerun secret validation:")
		fmt.Printf("  %s secrets\n", runner.Command)
		return 0
	case "prepare-allow":
		reason, err := optionValue(args[2:], "--reason")
		if err != nil {
			return actionFailure(runner, "prepare secret allowlist proposal", err)
		}
		path, finding, err := secrets.PrepareAllow(ctx, runner.Root, id, reason)
		if err != nil {
			return actionFailure(runner, "prepare secret allowlist proposal", err)
		}
		fmt.Println("ALLOWLIST PROPOSAL PREPARED")
		fmt.Println("===========================")
		fmt.Printf("Finding:          %s\n", finding.ID)
		fmt.Printf("Tracked modified: no\n")
		fmt.Printf("Local proposal:   %s\n\n", relative(runner.Root, path))
		fmt.Println("[READ ONLY] Review the proposal:")
		fmt.Printf("  cat '%s'\n\n", relative(runner.Root, path))
		fmt.Println("[MODIFIES TRACKED ALLOWLIST] Apply the reviewed proposal:")
		fmt.Printf("  %s secrets apply-allow %s\n", runner.Command, id)
		return 0
	case "apply-allow":
		path, err := secrets.ApplyAllow(runner.Root, id)
		if err != nil {
			return actionFailure(runner, "apply secret allowlist proposal", err)
		}
		fmt.Println("ALLOWLIST UPDATED")
		fmt.Println("=================")
		fmt.Printf("Tracked file: %s\n\n", relative(runner.Root, path))
		fmt.Println("[READ ONLY] Review the tracked exception:")
		fmt.Printf("  git diff -- '%s'\n\n", relative(runner.Root, path))
		fmt.Println("Rerun secret validation:")
		fmt.Printf("  %s secrets\n", runner.Command)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "FAIL: unknown secret action %q\n", action)
		return 2
	}
}

func actionFailure(runner *validation.Runner, check string, err error) int {
	path, logErr := failurelog.Write(failurelog.Context{
		Root: runner.Root, Mode: runner.Mode, Check: check,
		Expected: "the requested action should complete safely", Observed: err.Error(),
		Commit: runner.Identity.Commit, Branch: runner.Identity.Branch,
		Started: time.Now(), Finished: time.Now(),
	})
	fmt.Fprintln(os.Stderr, "FAIL:", err)
	if logErr == nil {
		fmt.Fprintln(os.Stderr, "Failure log:", relative(runner.Root, path))
	}
	return 1
}

func actionCommandFailure(runner *validation.Runner, check string, result executil.Result) int {
	path, logErr := failurelog.Write(failurelog.Context{
		Root: runner.Root, Mode: runner.Mode, Check: check,
		Expected: "the requested command should complete", Observed: "command failed",
		Command: &result, Commit: runner.Identity.Commit, Branch: runner.Identity.Branch,
		Started: result.Started, Finished: result.Finished,
	})
	fmt.Fprintln(os.Stderr, "FAIL: command failed")
	if result.Stderr != "" {
		fmt.Fprintln(os.Stderr, strings.TrimSpace(result.Stderr))
	}
	if logErr == nil {
		fmt.Fprintln(os.Stderr, "Failure log:", relative(runner.Root, path))
	}
	return 1
}

func goFiles(ctx context.Context, root string) ([]string, error) {
	result := executil.Run(ctx, root, "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.go")
	if result.Err != nil {
		return nil, result.Err
	}
	var files []string
	for _, path := range strings.Split(result.Stdout, "\x00") {
		if path != "" {
			files = append(files, filepath.FromSlash(path))
		}
	}
	return files, nil
}

func filterChecks(checks []model.Check, names ...string) []model.Check {
	wanted := make(map[string]bool)
	for _, name := range names {
		wanted[name] = true
	}
	var out []model.Check
	for _, check := range checks {
		if wanted[check.Name] {
			out = append(out, check)
		}
	}
	return out
}

func optionValue(args []string, option string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == option {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", option)
			}
			return args[i+1], nil
		}
	}
	return "", fmt.Errorf("%s is required", option)
}

func canonicalCommand(root string) string {
	local := filepath.Join(root, ".local", "bin", "isras-validate")
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		return "./.local/bin/isras-validate"
	}
	if path, err := exec.LookPath(os.Args[0]); err == nil && !strings.Contains(path, "go-build") {
		return os.Args[0]
	}
	return "go run ./cmd/isras-validate"
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func shellDisplay(value string) string {
	return strings.ReplaceAll(value, "'", "'\\''")
}

func usage(command string) {
	fmt.Printf(`%s — Iron Signal Repository Assurance validation

Usage:
  %s all [--mode development|commit|release]
  %s system
  %s repo
  %s go [formatting|vet|tests|build|modules|vulnerabilities]
  %s go formatting --diff
  %s fix formatting
  %s secrets
  %s secrets inspect FINDING-ID
  %s secrets prepare-redaction FINDING-ID
  %s secrets apply-redaction FINDING-ID
  %s secrets prepare-allow FINDING-ID --reason 'bounded reason'
  %s secrets apply-allow FINDING-ID
`, profile, command, command, command, command, command, command, command, command, command, command, command, command)
}
