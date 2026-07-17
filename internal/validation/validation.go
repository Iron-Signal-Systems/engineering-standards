package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/failurelog"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/secrets"
)

type Runner struct {
	Root     string
	Mode     string
	Identity repository.Identity
	Command  string
}

func New(ctx context.Context, mode, command string) (*Runner, error) {
	identity, err := repository.Discover(ctx)
	if err != nil {
		return nil, err
	}
	if mode == "" {
		mode = "development"
	}
	if mode != "development" && mode != "commit" && mode != "release" {
		return nil, fmt.Errorf("unsupported validation mode %q", mode)
	}
	return &Runner{Root: identity.Root, Mode: mode, Identity: identity, Command: command}, nil
}

func (r *Runner) All(ctx context.Context) model.Summary {
	checks := append([]model.Check{}, r.System(ctx)...)
	checks = append(checks, r.Repository(ctx)...)
	checks = append(checks, r.Go(ctx)...)
	checks = append(checks, r.Secrets(ctx)...)
	return finish(checks)
}

func (r *Runner) System(ctx context.Context) []model.Check {
	_ = ctx
	osName := runtime.GOOS
	detail := osName + "/" + runtime.GOARCH
	status := model.Warn
	if runtime.GOOS == "linux" {
		if detected := linuxName(); detected != "" {
			detail = detected + " · " + runtime.GOARCH
		}
		lower := strings.ToLower(detail)
		if strings.Contains(lower, "arch") || strings.Contains(lower, "ubuntu") || strings.Contains(lower, "fedora") {
			status = model.Pass
		}
	}
	check := model.Check{Section: "SYSTEM", Name: "Operating system", Status: status, Detail: detail}
	if status == model.Warn {
		check.Actions = []model.Action{{
			Label: "READ ONLY", Description: "Review the declared platform scope:",
			Command: "sed -n '1,220p' standards/PLATFORM-SUPPORT.md",
		}}
	}
	return []model.Check{
		check,
		{Section: "SYSTEM", Name: "Go runtime", Status: model.Info, Detail: runtime.Version()},
	}
}

func (r *Runner) Repository(ctx context.Context) []model.Check {
	checks := []model.Check{
		{Section: "REPOSITORY", Name: "Repository root", Status: model.Pass, Detail: r.Root},
		{Section: "REPOSITORY", Name: "Current branch", Status: model.Info, Detail: valueOr(r.Identity.Branch, "detached")},
		{Section: "REPOSITORY", Name: "Current commit", Status: model.Info, Detail: short(r.Identity.Commit)},
	}
	originStatus, originDetail := model.Pass, "SSH"
	if r.Identity.Origin == "" {
		originStatus, originDetail = model.Fail, "origin unavailable"
	} else if !strings.HasPrefix(r.Identity.Origin, "git@") && !strings.HasPrefix(r.Identity.Origin, "ssh://") {
		originStatus, originDetail = model.Warn, r.Identity.Origin
	}
	origin := model.Check{Section: "REPOSITORY", Name: "Origin transport", Status: originStatus, Detail: originDetail}
	if originStatus != model.Pass {
		origin.Actions = []model.Action{{Label: "READ ONLY", Description: "Review the configured origin:", Command: "git remote -v"}}
		if originStatus == model.Fail {
			origin.LogPath = r.simpleFailure("origin transport", "an SSH origin URL", originDetail, origin.Actions)
		}
	}
	checks = append(checks, origin)

	statusResult := executil.Run(ctx, r.Root, "git", "status", "--porcelain=v1", "--untracked-files=all")
	if statusResult.Err != nil {
		checks = append(checks, r.commandFailure("Working tree", "REPOSITORY", statusResult,
			"Git status should complete", "Git status failed",
			[]model.Action{{Label: "READ ONLY", Description: "Run Git status directly:", Command: "git status --short --branch"}}))
	} else {
		dirty := strings.TrimSpace(statusResult.Stdout) != ""
		status, detail := model.Pass, "clean"
		if dirty && r.Mode == "development" {
			status, detail = model.Warn, "modified development tree"
		} else if dirty {
			status, detail = model.Fail, "changes present"
		}
		check := model.Check{Section: "REPOSITORY", Name: "Working tree", Status: status, Detail: detail}
		if dirty {
			check.Actions = []model.Action{{Label: "READ ONLY", Description: "Review every pending path:", Command: "git status --short\ngit diff\ngit diff --cached"}}
			if status == model.Fail {
				check.LogPath = r.simpleFailure("working tree", "a clean exact commit", detail, check.Actions)
			}
		}
		checks = append(checks, check)
	}

	checks = append(checks, r.commitSignatureCheck(ctx))
	return checks
}

func (r *Runner) Go(ctx context.Context) []model.Check {
	var checks []model.Check
	goFiles := trackedGoFiles(ctx, r.Root)
	if len(goFiles) == 0 {
		return []model.Check{{Section: "GO SOURCE", Name: "Go source", Status: model.Warn, Detail: "no Go files found"}}
	}
	format := executil.Run(ctx, r.Root, "gofmt", append([]string{"-l"}, goFiles...)...)
	if format.Err != nil {
		checks = append(checks, r.commandFailure("Canonical formatting", "GO SOURCE", format,
			"gofmt should inspect all Go source", "gofmt failed",
			[]model.Action{{Label: "READ ONLY", Description: "Show formatting differences:", Command: r.Command + " go formatting --diff"}}))
	} else if strings.TrimSpace(format.Stdout) != "" {
		actions := []model.Action{
			{Label: "READ ONLY", Description: "Show formatting differences:", Command: r.Command + " go formatting --diff"},
			{Label: "MODIFIES WORKING TREE", Description: "Apply canonical Go formatting:", Command: r.Command + " fix formatting"},
			{Label: "READ ONLY", Description: "Rerun formatting validation:", Command: r.Command + " go formatting"},
		}
		check := model.Check{Section: "GO SOURCE", Name: "Canonical formatting", Status: model.Fail, Detail: fmt.Sprintf("%d file(s) require formatting", nonEmptyLines(format.Stdout)), Actions: actions}
		check.LogPath = r.commandFailureLog("canonical formatting", "no files returned by gofmt -l", strings.TrimSpace(format.Stdout), format, actions)
		checks = append(checks, check)
	} else {
		checks = append(checks, model.Check{Section: "GO SOURCE", Name: "Canonical formatting", Status: model.Pass, Detail: "all Go files formatted"})
	}

	checks = append(checks, r.goCommand(ctx, "Static analysis", "go vet ./...", "go", []string{"vet", "./..."},
		[]model.Action{{Label: "READ ONLY", Description: "Rerun static analysis with direct output:", Command: "go vet ./..."}}))
	checks = append(checks, r.goCommand(ctx, "Package tests", "go test -count=1 ./...", "go", []string{"test", "-count=1", "./..."},
		[]model.Action{{Label: "READ ONLY", Description: "Rerun all tests with verbose output:", Command: "go test -count=1 -v ./..."}}))
	checks = append(checks, r.goCommand(ctx, "Build", "go build ./...", "go", []string{"build", "./..."},
		[]model.Action{{Label: "READ ONLY", Description: "Rerun the complete build:", Command: "go build ./..."}}))
	checks = append(checks, r.goCommand(ctx, "Module consistency", "go mod tidy -diff", "go", []string{"mod", "tidy", "-diff"},
		[]model.Action{
			{Label: "READ ONLY", Description: "Display the required module changes:", Command: "go mod tidy -diff"},
			{Label: "MODIFIES WORKING TREE", Description: "Apply reviewed module corrections:", Command: "go mod tidy\ngit diff -- go.mod go.sum"},
		}))
	checks = append(checks, r.goCommand(ctx, "Module integrity", "go mod verify", "go", []string{"mod", "verify"},
		[]model.Action{{Label: "READ ONLY", Description: "Rerun module-cache verification:", Command: "go mod verify"}}))
	checks = append(checks, r.vulnerabilityCheck(ctx))
	return checks
}

func (r *Runner) Secrets(ctx context.Context) []model.Check {
	result, err := secrets.ScanRepo(ctx, r.Root)
	if err != nil {
		actions := []model.Action{{Label: "READ ONLY", Description: "Rerun the secret scanner:", Command: r.Command + " secrets"}}
		check := model.Check{Section: "SECRET PROTECTION", Name: "Repository secret scan", Status: model.Fail, Detail: err.Error(), Actions: actions}
		check.LogPath = r.simpleFailure("repository secret scan", "tracked and untracked source should be scanned", err.Error(), actions)
		return []model.Check{check}
	}
	checks := []model.Check{
		{Section: "SECRET PROTECTION", Name: "Files scanned", Status: model.Info, Detail: fmt.Sprintf("%d text files · %d skipped", result.Scanned, result.Skipped)},
		{Section: "SECRET PROTECTION", Name: "Bounded exceptions", Status: model.Info, Detail: fmt.Sprintf("%d allowed finding(s)", len(result.Allowed))},
	}
	if len(result.Findings) == 0 {
		checks = append(checks, model.Check{Section: "SECRET PROTECTION", Name: "Unresolved findings", Status: model.Pass, Detail: "none detected"})
		return checks
	}
	for _, finding := range result.Findings {
		actions := secretActions(r.Command, finding)
		detail := fmt.Sprintf("%s · %s:%d · %s", finding.ID, finding.Path, finding.Line, finding.Rule)
		check := model.Check{Section: "SECRET PROTECTION", Name: "Possible sensitive value", Status: model.Fail, Detail: detail, Actions: actions}
		check.LogPath = r.simpleFailure("secret "+finding.ID, "no unresolved sensitive-value findings", detail+" · detected value [REDACTED]", actions)
		checks = append(checks, check)
	}
	return checks
}

func (r *Runner) goCommand(ctx context.Context, name, expected, command string, args []string, actions []model.Action) model.Check {
	result := executil.Run(ctx, r.Root, command, args...)
	if result.Err != nil {
		return r.commandFailure(name, "GO SOURCE", result, expected+" should pass", "command failed", actions)
	}
	detail := "completed"
	switch name {
	case "Static analysis":
		detail = "no diagnostics"
	case "Package tests":
		detail = "all package tests passed"
	case "Build":
		detail = "all packages compiled"
	case "Module consistency":
		detail = "no changes required"
	case "Module integrity":
		detail = "all modules verified"
	default:
		if output := strings.TrimSpace(result.Stdout); output != "" {
			detail = firstLine(output)
		}
	}
	return model.Check{Section: "GO SOURCE", Name: name, Status: model.Pass, Detail: detail}
}

func (r *Runner) vulnerabilityCheck(ctx context.Context) model.Check {
	tool, version, observed := govulncheckTool(ctx, r.Root)
	if tool == "" {
		actions := []model.Action{{
			Label:       "NETWORK ACCESS — INSTALLS PINNED TOOL",
			Description: "Install the repository-declared govulncheck version into local tooling:",
			Command:     fmt.Sprintf("mkdir -p .local/tools/bin\nGOBIN=\"$PWD/.local/tools/bin\" go install golang.org/x/vuln/cmd/govulncheck@%s", version),
		}, {
			Label: "READ ONLY", Description: "Verify the installed binary's embedded module version:", Command: "go version -m .local/tools/bin/govulncheck",
		}, {
			Label: "READ ONLY", Description: "Rerun vulnerability validation:", Command: r.Command + " go vulnerabilities",
		}}
		check := model.Check{Section: "GO SOURCE", Name: "Known vulnerabilities", Status: model.Fail, Detail: vulnerabilityToolDetail(version, observed), Actions: actions}
		check.LogPath = r.simpleFailure("known vulnerabilities", "the pinned govulncheck tool should run", vulnerabilityToolDetail(version, observed)+" · "+observed, actions)
		return check
	}
	result := executil.Run(ctx, r.Root, tool, "./...")
	if result.Err != nil {
		actions := []model.Action{
			{Label: "READ ONLY", Description: "Rerun the bounded vulnerability report:", Command: shellPath(tool) + " ./..."},
			{Label: "READ ONLY", Description: "Review available module updates before selecting a version:", Command: "go list -m -u all"},
			{Label: "READ ONLY", Description: "Rerun vulnerability validation after remediation:", Command: r.Command + " go vulnerabilities"},
		}
		return r.commandFailure("Known vulnerabilities", "GO SOURCE", result,
			"govulncheck should report no reachable known vulnerabilities", "vulnerability check failed", actions)
	}
	return model.Check{Section: "GO SOURCE", Name: "Known vulnerabilities", Status: model.Pass, Detail: "no reachable findings reported"}
}

func (r *Runner) commandFailure(name, section string, result executil.Result, expected, observed string, actions []model.Action) model.Check {
	check := model.Check{Section: section, Name: name, Status: model.Fail, Detail: observed, Actions: actions, Started: result.Started, Finished: result.Finished}
	check.LogPath = r.commandFailureLog(name, expected, observed, result, actions)
	return check
}

func (r *Runner) commandFailureLog(name, expected, observed string, result executil.Result, actions []model.Action) string {
	path, err := failurelog.Write(failurelog.Context{
		Root: r.Root, Mode: r.Mode, Check: name, Expected: expected, Observed: observed,
		Actions: actionStrings(actions), Command: &result, Commit: r.Identity.Commit, Branch: r.Identity.Branch,
		Started: result.Started, Finished: result.Finished,
	})
	if err != nil {
		return "log creation failed: " + err.Error()
	}
	return relative(r.Root, path)
}

func (r *Runner) simpleFailure(name, expected, observed string, actions []model.Action) string {
	path, err := failurelog.Write(failurelog.Context{
		Root: r.Root, Mode: r.Mode, Check: name, Expected: expected, Observed: observed,
		Actions: actionStrings(actions), Commit: r.Identity.Commit, Branch: r.Identity.Branch,
		Started: time.Now(), Finished: time.Now(),
	})
	if err != nil {
		return "log creation failed: " + err.Error()
	}
	return relative(r.Root, path)
}

func finish(checks []model.Check) model.Summary {
	summary := model.Summary{Checks: checks}
	status, detail := model.Pass, fmt.Sprintf("%d checks passed · %d warning(s)", summary.Count(model.Pass), summary.Count(model.Warn))
	if summary.Failed() {
		status, detail = model.Fail, fmt.Sprintf("%d required check(s) failed", summary.Count(model.Fail))
	}
	summary.Checks = append(summary.Checks, model.Check{Section: "RESULT", Name: "Overall status", Status: status, Detail: detail})
	return summary
}

func linuxName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
		}
	}
	return "Linux"
}

func trackedGoFiles(ctx context.Context, root string) []string {
	result := executil.Run(ctx, root, "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.go")
	if result.Err != nil {
		return nil
	}
	var files []string
	for _, path := range strings.Split(result.Stdout, "\x00") {
		if path != "" {
			files = append(files, filepath.FromSlash(path))
		}
	}
	sort.Strings(files)
	return files
}

func govulncheckTool(ctx context.Context, root string) (string, string, string) {
	version := "v1.6.0"
	data, err := os.ReadFile(filepath.Join(root, "validation", "tool-versions.json"))
	if err == nil {
		var config struct {
			Tools map[string]struct {
				Version string `json:"version"`
			} `json:"tools"`
		}
		if json.Unmarshal(data, &config) == nil && config.Tools["govulncheck"].Version != "" {
			version = config.Tools["govulncheck"].Version
		}
	}
	var candidates []string
	local := filepath.Join(root, ".local", "tools", "bin", "govulncheck")
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		candidates = append(candidates, local)
	}
	if path, err := exec.LookPath("govulncheck"); err == nil && path != local {
		candidates = append(candidates, path)
	}
	if len(candidates) == 0 {
		return "", version, "not installed"
	}
	var observed []string
	for _, candidate := range candidates {
		result := executil.Run(ctx, root, "go", "version", "-m", candidate)
		text := strings.TrimSpace(result.Stdout + "\n" + result.Stderr)
		if result.Err == nil && strings.Contains(text, "golang.org/x/vuln") && strings.Contains(text, version) {
			return candidate, version, version
		}
		if text == "" {
			text = "version metadata unavailable"
		}
		observed = append(observed, filepath.ToSlash(candidate)+": "+firstLine(text))
	}
	return "", version, strings.Join(observed, "; ")
}

func vulnerabilityToolDetail(required, observed string) string {
	if observed == "" || observed == "not installed" {
		return "govulncheck unavailable · required " + required
	}
	return "govulncheck version mismatch · required " + required
}

func secretActions(command string, finding secrets.Finding) []model.Action {
	actions := []model.Action{{
		Label: "READ ONLY", Description: "Inspect finding metadata without displaying the detected value:",
		Command: command + " secrets inspect " + finding.ID,
	}}
	if finding.Redactable {
		actions = append(actions, model.Action{
			Label: "CREATES LOCAL PLAN", Description: "Prepare a redaction plan without changing source:",
			Command: command + " secrets prepare-redaction " + finding.ID,
		}, model.Action{
			Label: "MODIFIES WORKING TREE", Description: "Apply the separately prepared redaction plan:",
			Command: command + " secrets apply-redaction " + finding.ID,
		})
	}
	if finding.Allowable {
		actions = append(actions, model.Action{
			Label: "CREATES EXCEPTION PROPOSAL", Description: "Prepare a bounded exception only after confirming a false positive or inert fixture:",
			Command: command + " secrets prepare-allow " + finding.ID + " --reason 'describe why this value is non-sensitive'",
		}, model.Action{
			Label: "MODIFIES TRACKED ALLOWLIST", Description: "Apply the reviewed exception proposal:",
			Command: command + " secrets apply-allow " + finding.ID,
		})
	}
	actions = append(actions, model.Action{
		Label: "READ ONLY", Description: "Rerun secret validation:", Command: command + " secrets",
	})
	return actions
}

func actionStrings(actions []model.Action) []string {
	out := make([]string, 0, len(actions))
	for _, action := range actions {
		out = append(out, "["+action.Label+"] "+action.Description+" "+action.Command)
	}
	return out
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func firstLine(value string) string {
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return value[:idx]
	}
	return value
}

func nonEmptyLines(value string) int {
	count := 0
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func short(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return valueOr(value, "unavailable")
}

func shellPath(value string) string {
	if strings.ContainsAny(value, " \t'\"") {
		return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
	}
	return value
}
