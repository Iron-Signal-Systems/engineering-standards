package releasevalidation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Options struct {
	Root    string
	Ref     string
	Command string
}

type Result struct {
	RepositoryRoot string
	Origin         string
	Branch         string
	Ref            string
	Commit         string
	RunDirectory   string
	CloneDirectory string
	LogPath        string
	SummaryPath    string
	Started        time.Time
	Finished       time.Time
}

type toolVersions struct {
	Tools map[string]struct {
		Version string `json:"version"`
	} `json:"tools"`
}

var safeRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

func Run(ctx context.Context, opts Options) (result Result, runErr error) {
	result.Started = time.Now().UTC()

	root, err := resolveRoot(ctx, opts.Root)
	if err != nil {
		return result, err
	}
	result.RepositoryRoot = root

	branch, err := gitOutput(ctx, root, "branch", "--show-current")
	if err != nil {
		return result, fmt.Errorf("determine current branch: %w", err)
	}
	if branch == "" {
		return result, errors.New("clean-clone release validation requires a named branch")
	}
	result.Branch = branch

	ref := strings.TrimSpace(opts.Ref)
	if ref == "" {
		ref = branch
	}
	if !safeRefPattern.MatchString(ref) || strings.Contains(ref, "..") || strings.HasSuffix(ref, "/") {
		return result, fmt.Errorf("unsafe branch ref %q", ref)
	}
	result.Ref = ref

	commit, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("resolve HEAD: %w", err)
	}
	if len(commit) != 40 && len(commit) != 64 {
		return result, fmt.Errorf("unexpected commit identity %q", commit)
	}
	result.Commit = commit

	origin, err := gitOutput(ctx, root, "remote", "get-url", "origin")
	if err != nil {
		return result, fmt.Errorf("read origin URL: %w", err)
	}
	if origin == "" {
		return result, errors.New("origin URL is empty")
	}
	result.Origin = sanitizeOrigin(origin)

	status, err := gitOutput(ctx, root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return result, fmt.Errorf("inspect working tree: %w", err)
	}
	if status != "" {
		return result, errors.New("repository must be completely clean before clean-clone release validation")
	}

	verify := runCommand(ctx, root, nil, nil, "git", "verify-commit", "HEAD")
	if verify.Err != nil {
		return result, fmt.Errorf("current commit signature verification failed: %s", conciseOutput(verify))
	}

	remoteResult := runCommand(ctx, root, nil, nil, "git", "ls-remote", "--heads", "origin", "refs/heads/"+ref)
	if remoteResult.Err != nil {
		return result, fmt.Errorf("read remote branch tip: %s", conciseOutput(remoteResult))
	}
	remoteCommit, err := parseRemoteTip(remoteResult.Stdout, ref)
	if err != nil {
		return result, err
	}
	if remoteCommit != commit {
		return result, fmt.Errorf("remote branch refs/heads/%s is %s, but local HEAD is %s; push the exact commit before release validation", ref, short(remoteCommit), short(commit))
	}

	runID := buildRunID(result.Started, commit)
	runDir := filepath.Join(root, ".local", "validation", "releases", runID)
	cloneDir := filepath.Join(runDir, "repository")
	logPath := filepath.Join(runDir, "release-validation.log")
	summaryPath := filepath.Join(runDir, "release-summary.txt")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return result, fmt.Errorf("create release evidence directory: %w", err)
	}
	result.RunDirectory = runDir
	result.CloneDirectory = cloneDir
	result.LogPath = logPath
	result.SummaryPath = summaryPath

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return result, fmt.Errorf("create release validation log: %w", err)
	}
	defer logFile.Close()

	writeLogHeader(logFile, result)
	defer func() {
		result.Finished = time.Now().UTC()
		status := "PASS"
		if runErr != nil {
			status = "FAIL"
		}
		if err := writeSummary(result, status, runErr); err != nil && runErr == nil {
			runErr = fmt.Errorf("write release summary: %w", err)
		}
	}()

	stage("Remote source", "confirmed exact pushed commit", result.Commit)
	stage("Clean clone", "cloning canonical origin", result.Origin)
	clone := runCommand(ctx, root, logFile, os.Stdin,
		"git", "clone", "--no-checkout", "--origin", "origin", "--no-tags", origin, cloneDir)
	if clone.Err != nil {
		return result, fmt.Errorf("clone canonical origin: %s", conciseOutput(clone))
	}

	stage("Exact checkout", "checking out detached commit", short(commit))
	checkout := runCommand(ctx, root, logFile, os.Stdin,
		"git", "-C", cloneDir, "checkout", "--detach", commit)
	if checkout.Err != nil {
		return result, fmt.Errorf("check out exact commit: %s", conciseOutput(checkout))
	}

	clonedCommit, err := gitOutput(ctx, cloneDir, "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("resolve clean-clone HEAD: %w", err)
	}
	if clonedCommit != commit {
		return result, fmt.Errorf("clean clone resolved %s, expected %s", clonedCommit, commit)
	}
	clonedStatus, err := gitOutput(ctx, cloneDir, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return result, fmt.Errorf("inspect clean-clone working tree: %w", err)
	}
	if clonedStatus != "" {
		return result, errors.New("clean clone is unexpectedly modified before validation")
	}

	version, err := readToolVersion(filepath.Join(cloneDir, "validation", "tool-versions.json"), "govulncheck")
	if err != nil {
		return result, err
	}
	toolBin := filepath.Join(cloneDir, ".local", "tools", "bin")
	validatorBin := filepath.Join(cloneDir, ".local", "bin", "isras-validate")
	if err := os.MkdirAll(toolBin, 0o700); err != nil {
		return result, fmt.Errorf("create clean-clone tool directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(validatorBin), 0o700); err != nil {
		return result, fmt.Errorf("create clean-clone validator directory: %w", err)
	}

	stage("Pinned tooling", "installing govulncheck", version)
	env := append(os.Environ(), "GOBIN="+toolBin)
	install := runCommandWithEnv(ctx, cloneDir, logFile, os.Stdin, env,
		"go", "install", "golang.org/x/vuln/cmd/govulncheck@"+version)
	if install.Err != nil {
		return result, fmt.Errorf("install pinned govulncheck: %s", conciseOutput(install))
	}

	stage("Validator build", "building exact committed validator", short(commit))
	build := runCommand(ctx, cloneDir, logFile, os.Stdin,
		"go", "build", "-trimpath", "-o", validatorBin, "./cmd/isras-validate")
	if build.Err != nil {
		return result, fmt.Errorf("build clean-clone validator: %s", conciseOutput(build))
	}
	if err := os.Chmod(validatorBin, 0o755); err != nil {
		return result, fmt.Errorf("mark validator executable: %w", err)
	}

	stage("Release validation", "running complete release-mode checks", short(commit))
	validate := runCommand(ctx, cloneDir, logFile, os.Stdin,
		validatorBin, "all", "--mode", "release")
	if validate.Err != nil {
		return result, fmt.Errorf("clean-clone release validation failed: %s", conciseOutput(validate))
	}

	finalStatus, err := gitOutput(ctx, cloneDir, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return result, fmt.Errorf("inspect clean clone after validation: %w", err)
	}
	if finalStatus != "" {
		return result, errors.New("clean clone acquired tracked or unignored source changes during validation")
	}

	stage("Result", "clean-clone release validation passed", short(commit))
	return result, nil
}

type commandResult struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

func runCommand(ctx context.Context, dir string, log io.Writer, stdin io.Reader, name string, args ...string) commandResult {
	return runCommandWithEnv(ctx, dir, log, stdin, nil, name, args...)
}

func runCommandWithEnv(ctx context.Context, dir string, log io.Writer, stdin io.Reader, env []string, name string, args ...string) commandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	if env != nil {
		cmd.Env = env
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if log != nil {
		fmt.Fprintf(log, "\nCOMMAND: %s\n", safeCommand(name, args))
	}

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	if log != nil {
		fmt.Fprintf(log, "EXIT CODE: %d\n", exitCode)
		if stdout.Len() > 0 {
			fmt.Fprintln(log, "STANDARD OUTPUT:")
			fmt.Fprint(log, stdout.String())
			if !strings.HasSuffix(stdout.String(), "\n") {
				fmt.Fprintln(log)
			}
		}
		if stderr.Len() > 0 {
			fmt.Fprintln(log, "STANDARD ERROR:")
			fmt.Fprint(log, stderr.String())
			if !strings.HasSuffix(stderr.String(), "\n") {
				fmt.Fprintln(log)
			}
		}
	}

	return commandResult{
		Command:  safeCommand(name, args),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

func resolveRoot(ctx context.Context, requested string) (string, error) {
	dir := strings.TrimSpace(requested)
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	root, err := gitOutput(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("current directory is not inside a Git repository")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return absolute, nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	result := runCommand(ctx, dir, nil, nil, "git", args...)
	if result.Err != nil {
		return "", fmt.Errorf("%s: %s", result.Command, conciseOutput(result))
	}
	return strings.TrimSpace(result.Stdout), nil
}

func parseRemoteTip(output, ref string) (string, error) {
	lines := nonEmptyLines(output)
	if len(lines) == 0 {
		return "", fmt.Errorf("remote branch refs/heads/%s does not exist", ref)
	}
	if len(lines) != 1 {
		return "", fmt.Errorf("remote branch lookup returned %d entries, expected exactly one", len(lines))
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 2 {
		return "", errors.New("remote branch lookup returned malformed output")
	}
	if fields[1] != "refs/heads/"+ref {
		return "", fmt.Errorf("remote lookup returned unexpected ref %q", fields[1])
	}
	if len(fields[0]) != 40 && len(fields[0]) != 64 {
		return "", errors.New("remote branch lookup returned an invalid commit identity")
	}
	return fields[0], nil
}

func readToolVersion(path, name string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read tool version declaration: %w", err)
	}
	var versions toolVersions
	if err := json.Unmarshal(data, &versions); err != nil {
		return "", fmt.Errorf("parse tool version declaration: %w", err)
	}
	version := strings.TrimSpace(versions.Tools[name].Version)
	if version == "" {
		return "", fmt.Errorf("tool version %q is not declared", name)
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+([-.][A-Za-z0-9.]+)?$`).MatchString(version) {
		return "", fmt.Errorf("tool version %q is not an exact semantic version", version)
	}
	return version, nil
}

func buildRunID(started time.Time, commit string) string {
	return started.UTC().Format("20060102T150405Z") + "-" + short(commit)
}

func writeLogHeader(w io.Writer, result Result) {
	fmt.Fprintln(w, "IRON SIGNAL CLEAN-CLONE RELEASE VALIDATION")
	fmt.Fprintln(w, "==========================================")
	fmt.Fprintf(w, "Started:    %s\n", result.Started.Format(time.RFC3339))
	fmt.Fprintf(w, "Repository: %s\n", result.RepositoryRoot)
	fmt.Fprintf(w, "Origin:     %s\n", result.Origin)
	fmt.Fprintf(w, "Branch:     %s\n", result.Branch)
	fmt.Fprintf(w, "Remote ref: refs/heads/%s\n", result.Ref)
	fmt.Fprintf(w, "Commit:     %s\n", result.Commit)
}

func writeSummary(result Result, status string, runErr error) error {
	if result.SummaryPath == "" {
		return nil
	}
	var b strings.Builder
	b.WriteString("Iron Signal Clean-Clone Release Validation\n")
	b.WriteString("==========================================\n\n")
	fmt.Fprintf(&b, "Status:       %s\n", status)
	fmt.Fprintf(&b, "Repository:   %s\n", result.RepositoryRoot)
	fmt.Fprintf(&b, "Origin:       %s\n", result.Origin)
	fmt.Fprintf(&b, "Branch:       %s\n", result.Branch)
	fmt.Fprintf(&b, "Remote ref:   refs/heads/%s\n", result.Ref)
	fmt.Fprintf(&b, "Commit:       %s\n", result.Commit)
	fmt.Fprintf(&b, "Started:      %s\n", result.Started.Format(time.RFC3339))
	fmt.Fprintf(&b, "Finished:     %s\n", result.Finished.Format(time.RFC3339))
	fmt.Fprintf(&b, "Clone:        %s\n", result.CloneDirectory)
	fmt.Fprintf(&b, "Log:          %s\n", result.LogPath)
	if runErr != nil {
		fmt.Fprintf(&b, "Failure:      %s\n", cleanText(runErr.Error()))
	}
	return os.WriteFile(result.SummaryPath, []byte(b.String()), 0o600)
}

func sanitizeOrigin(raw string) string {
	if strings.HasPrefix(raw, "git@") || strings.HasPrefix(raw, "ssh://") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return raw
	}
	parsed.User = url.User("REDACTED")
	return parsed.String()
}

func safeCommand(name string, args []string) string {
	parts := []string{name}
	for _, arg := range args {
		if strings.Contains(arg, "://") {
			parts = append(parts, shellQuote(sanitizeOrigin(arg)))
			continue
		}
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if regexp.MustCompile(`^[A-Za-z0-9_./:@=-]+$`).MatchString(value) {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func conciseOutput(result commandResult) string {
	text := strings.TrimSpace(result.Stderr)
	if text == "" {
		text = strings.TrimSpace(result.Stdout)
	}
	if text == "" {
		if result.Err != nil {
			return result.Err.Error()
		}
		return "no output"
	}
	scanner := bufio.NewScanner(strings.NewReader(text))
	if scanner.Scan() {
		return cleanText(scanner.Text())
	}
	return "command failed"
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func nonEmptyLines(value string) []string {
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func short(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func stage(name, detail, value string) {
	fmt.Printf("%-24s ● %-5s %s", name, "PASS", detail)
	if value != "" {
		fmt.Printf(" · %s", value)
	}
	fmt.Println()
}
