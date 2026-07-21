package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/dashboard"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/failurelog"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/secrets"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

func main() {
	os.Exit(run())
}

func run() int {
	return runTargetAware(os.Args[1:])
}

func parseMode(args []string) ([]string, string, error) {
	parsed, options, err := parseGlobalOptions(args)
	return parsed, options.Mode, err
}

func render(header string, summary model.Summary) int {
	printer := dashboard.New(os.Stdout)
	printer.Header(header)
	printer.Checks(summary.Checks)
	printer.Footer(summary)
	if summary.Failed() {
		return 1
	}
	return 0
}

func renderIdentity(writer io.Writer, identity validatoridentity.Identity) {
	fmt.Fprintln(writer, "ISRAS VALIDATOR IDENTITY")
	fmt.Fprintln(writer, "========================")
	fmt.Fprintf(writer, "Profile:           %s\n", identity.Profile)
	fmt.Fprintf(writer, "Standard version:  %s\n", identity.StandardVersion)
	fmt.Fprintf(writer, "Ownership:         %s\n", identity.Ownership)
	if identity.ReleaseTag != "" {
		fmt.Fprintf(writer, "Release tag:       %s\n", identity.ReleaseTag)
	}
	fmt.Fprintf(writer, "Source repository: %s\n", identity.SourceRepository)
	fmt.Fprintf(writer, "Source commit:     %s\n", identity.SourceCommit)
	if identity.TargetModule != "" {
		fmt.Fprintf(writer, "Target module:     %s\n", identity.TargetModule)
	}
	fmt.Fprintf(writer, "Repository commit: %s\n", identity.RepositoryCommit)
}

func runProjectPin(ctx context.Context, runner *validation.Runner, args []string) int {
	action := "validate"
	if len(args) > 0 {
		action = args[0]
	}

	switch action {
	case "initialize":
		return runProjectPinInitialize(ctx, runner, args[1:])
	case "validate", "inspect":
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "FAIL: project-pin %s accepts no options\n", action)
			return 2
		}
	case "verify-artifacts":
	default:
		fmt.Fprintf(os.Stderr, "FAIL: unknown project-pin action %q\n", action)
		return 2
	}

	pin, err := projectpin.Load(runner.Root)
	if err != nil {
		return actionFailure(runner, action+" project pin", err)
	}

	switch action {
	case "validate":
		renderProjectPinValidation(os.Stdout, pin)
		return 0
	case "inspect":
		renderProjectPin(os.Stdout, pin)
		return 0
	case "verify-artifacts":
		report, jsonPath, textPath, verifyErr := verifyProjectArtifacts(ctx, runner.Root, pin, args[1:])
		renderProjectArtifactVerification(os.Stdout, runner.Root, report, jsonPath, textPath)
		if verifyErr != nil {
			return actionFailure(runner, "verify project artifacts", verifyErr)
		}
		return 0
	default:
		panic("unreachable project-pin action")
	}
}

func renderProjectPinValidation(writer io.Writer, pin projectpin.Pin) {
	fmt.Fprintln(writer, "PROJECT PIN DECLARATION VALIDATION")
	fmt.Fprintln(writer, "==================================")
	fmt.Fprintln(writer, "Declaration status:    PASS")
	fmt.Fprintln(writer, "Validation scope:      metadata structure and identity only")
	fmt.Fprintln(writer, "Artifact verification: NOT PERFORMED")
	fmt.Fprintf(writer, "Path:                  %s\n", projectpin.MetadataPath)
	fmt.Fprintf(writer, "Project:               %s\n", pin.Project.Repository)
	fmt.Fprintf(writer, "ISRAS release:         %s\n", pin.Standard.ReleaseTag)
	fmt.Fprintf(writer, "Source commit:         %s\n", pin.Standard.SourceCommit)
}

func renderProjectPin(writer io.Writer, pin projectpin.Pin) {
	commandNames := make([]string, 0, len(pin.Commands))
	for name := range pin.Commands {
		commandNames = append(commandNames, name)
	}
	sort.Strings(commandNames)

	fmt.Fprintln(writer, "ISRAS PROJECT PIN DECLARATION")
	fmt.Fprintln(writer, "=============================")
	fmt.Fprintln(writer, "Declaration status:      PASS")
	fmt.Fprintln(writer, "Validation scope:        metadata structure and identity only")
	fmt.Fprintln(writer, "Artifact verification:   NOT PERFORMED")
	fmt.Fprintln(writer, "Verification reason:     artifact bytes were not acquired or hashed")
	fmt.Fprintf(writer, "Schema version:          %d\n", pin.SchemaVersion)
	fmt.Fprintf(writer, "Project:                 %s\n", pin.Project.Repository)
	fmt.Fprintf(writer, "ISRAS profile:           %s\n", pin.Standard.Profile)
	fmt.Fprintf(writer, "ISRAS version:           %s\n", pin.Standard.Version)
	fmt.Fprintf(writer, "Release tag:             %s\n", pin.Standard.ReleaseTag)
	fmt.Fprintf(writer, "Source repository:       %s\n", pin.Standard.SourceRepository)
	fmt.Fprintf(writer, "Source commit:           %s\n", pin.Standard.SourceCommit)
	fmt.Fprintf(writer, "Workflow path:           %s\n", pin.Workflow.Path)
	fmt.Fprintf(writer, "Workflow commit:         %s\n", pin.Workflow.Commit)
	fmt.Fprintf(writer, "Profiles:                %s\n", strings.Join(pin.Profiles, ", "))
	fmt.Fprintf(writer, "Evidence:                %s\n", pin.Evidence.Directory)
	fmt.Fprintf(writer, "Commands declared:       %s\n", strings.Join(commandNames, ", "))
	fmt.Fprintf(writer, "Artifacts declared:      %d\n", len(pin.Artifacts))
	for index, artifact := range pin.Artifacts {
		label := artifact.Kind
		if artifact.Kind == "validator" {
			label += " (" + artifact.OS + "/" + artifact.Arch + ")"
		}
		fmt.Fprintf(writer, "  %d. %s: %s\n", index+1, label, artifact.Name)
		fmt.Fprintf(writer, "     Declared SHA-256: %s\n", abbreviatedDigest(artifact.SHA256))
		fmt.Fprintf(writer, "     Declared SHA-512: %s\n", abbreviatedDigest(artifact.SHA512))
		fmt.Fprintln(writer, "     Byte verification: NOT PERFORMED")
	}
}

func abbreviatedDigest(value string) string {
	const visible = 12
	if len(value) <= visible*2 {
		return value
	}
	return value[:visible] + "..." + value[len(value)-visible:]
}

func runGo(ctx context.Context, runner *validation.Runner, header string, args []string) int {
	checks := runner.Go(ctx)
	if len(args) == 0 {
		return render(header, model.Summary{Checks: checks})
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
		return render(header, model.Summary{Checks: filterChecks(checks, "Module consistency", "Module integrity")})
	case "vulnerabilities":
		name = "Known vulnerabilities"
	default:
		fmt.Fprintf(os.Stderr, "FAIL: unknown Go validation command %q\n", args[0])
		return 2
	}
	return render(header, model.Summary{Checks: filterChecks(checks, name)})
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
	fmt.Println(rootedShellCommand(runner.Root, "git diff -- '*.go'"))
	fmt.Println()
	fmt.Println("Rerun formatting validation:")
	fmt.Println("  " + runner.Command + " go formatting")
	return 0
}

func runSecrets(ctx context.Context, runner *validation.Runner, header string, args []string) int {
	if len(args) == 0 {
		return render(header, model.Summary{Checks: runner.Secrets(ctx)})
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
		fmt.Printf("%s\n\n", rootedShellCommand(runner.Root, "cat "+shellQuote(relative(runner.Root, path))))
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
		fmt.Printf("%s\n\n", rootedShellCommand(runner.Root, "git diff -- "+shellQuote(finding.Path)))
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
		fmt.Printf("%s\n\n", rootedShellCommand(runner.Root, "cat "+shellQuote(relative(runner.Root, path))))
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
		fmt.Printf("%s\n\n", rootedShellCommand(runner.Root, "git diff -- "+shellQuote(relative(runner.Root, path))))
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
	return canonicalCommandForTarget(root, false)
}

func canonicalCommandForTarget(root string, includeRepository bool) string {
	command := ""
	if executable, err := os.Executable(); err == nil && executable != "" && !strings.Contains(filepath.ToSlash(executable), "/go-build") {
		if absolute, absErr := filepath.Abs(executable); absErr == nil {
			command = shellQuote(filepath.Clean(absolute))
		}
	}
	if command == "" {
		if path, err := exec.LookPath(os.Args[0]); err == nil && !strings.Contains(path, "go-build") {
			command = shellQuote(path)
		} else {
			command = "go run ./cmd/isras-validate"
		}
	}
	if includeRepository {
		command += " --repo " + shellQuote(root)
	}
	return command
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

func shellQuote(value string) string {
	return "'" + shellDisplay(value) + "'"
}

func rootedShellCommand(root, command string) string {
	return "cd -- " + shellQuote(root) + "\n" + command
}

func usage(writer io.Writer, command, header string) {
	fmt.Fprintf(writer, `%s — Iron Signal Repository Assurance validation

Global options:
  --repo PATH                         explicit target repository directory
  --mode development|commit|release  validation strictness

Usage:
  %s [--repo PATH] all [--mode development|commit|release]
  %s [--repo PATH] system
  %s [--repo PATH] repo
  %s [--repo PATH] go [formatting|vet|tests|build|modules|vulnerabilities]
  %s [--repo PATH] go formatting --diff
  %s [--repo PATH] fix formatting
  %s version
  %s help
  %s [--repo PATH] secrets
  %s [--repo PATH] secrets inspect FINDING-ID
  %s [--repo PATH] secrets prepare-redaction FINDING-ID
  %s [--repo PATH] secrets apply-redaction FINDING-ID
  %s [--repo PATH] secrets prepare-allow FINDING-ID --reason 'bounded reason'
  %s [--repo PATH] secrets apply-allow FINDING-ID
`, header, command, command, command, command, command, command, command, command, command, command, command, command, command, command)
	fmt.Fprintf(writer, "  %s [--repo PATH] project-pin initialize --release isras-vMAJOR.MINOR.PATCH --go-defaults\n", command)
	fmt.Fprintf(writer, "  %s [--repo PATH] project-pin validate\n", command)
	fmt.Fprintf(writer, "  %s [--repo PATH] project-pin inspect\n", command)
	fmt.Fprintf(writer, "  %s [--repo PATH] project-pin verify-artifacts [--source-directory PATH]\n", command)
	fmt.Fprintf(writer, "  %s [--repo PATH] project-command run NAME\n", command)
	fmt.Fprintf(writer, "  %s [--repo PATH] documentation-impact --base COMMIT --head COMMIT\n", command)
}
