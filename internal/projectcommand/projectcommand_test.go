package projectcommand

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

const testSourceCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestExecuteUsesExactArgumentsAndSanitizedEnvironment(t *testing.T) {
	fixture := newFixture(t, "example-project", `#!/bin/sh
printf 'first=%s\n' "$1"
printf 'second=%s\n' "$2"
printf 'inherited=%s\n' "${DANGEROUS_VALUE-unset}"
printf 'home=%s\n' "$HOME"
token_prefix='gh'
token_suffix='p_AAAAAAAAAAAAAAAAAAAAAAAA'
printf 'token=%s%s\n' "$token_prefix" "$token_suffix"
`)
	fixture.setCommand(t, "test", []string{"./tools/project-check.sh", "literal;touch-not-run", "$HOME"})
	fixture.commit(t, "declare exact command")
	t.Setenv("DANGEROUS_VALUE", "must-not-be-inherited")

	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "PASS" || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(result.Stdout.Sanitized, "first=literal;touch-not-run") {
		t.Fatalf("first argument changed: %q", result.Stdout.Sanitized)
	}
	if !strings.Contains(result.Stdout.Sanitized, "second=$HOME") {
		t.Fatalf("second argument was expanded: %q", result.Stdout.Sanitized)
	}
	if !strings.Contains(result.Stdout.Sanitized, "inherited=unset") {
		t.Fatalf("unapproved environment variable reached command: %q", result.Stdout.Sanitized)
	}
	if strings.Contains(result.Stdout.Sanitized, "ghp_A") || !strings.Contains(result.Stdout.Sanitized, "[REDACTED]") {
		t.Fatalf("credential-shaped output was not redacted: %q", result.Stdout.Sanitized)
	}
	assertEvidence(t, result)
}

func TestExecuteRejectsWorkingTreePinDriftBeforeExecution(t *testing.T) {
	fixture := newFixture(t, "pin-drift", `#!/bin/sh
printf 'ran\n' > command-ran
`)
	fixture.commit(t, "baseline")
	path := filepath.Join(fixture.root, projectpin.MetadataPath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Execute(context.Background(), fixture.requestFromCommitted(t, "test"))
	if err == nil || !strings.Contains(err.Error(), "working-tree project pin differs") {
		t.Fatalf("expected pin-drift denial, got result=%+v err=%v", result, err)
	}
	if result.Authorization != "DENIED" || result.Status != "DENIED" {
		t.Fatalf("unexpected denial result: %+v", result)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.root, "command-ran")); !os.IsNotExist(statErr) {
		t.Fatal("project command executed despite pin drift")
	}
}

func TestExecuteRejectsValidatorIdentityMismatch(t *testing.T) {
	fixture := newFixture(t, "identity-mismatch", "#!/bin/sh\nexit 0\n")
	fixture.commit(t, "baseline")
	request := fixture.request(t, "test")
	request.Validator.SourceCommit = strings.Repeat("b", 40)
	result, err := Execute(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "validator release identity") {
		t.Fatalf("expected identity denial, got result=%+v err=%v", result, err)
	}
	if result.Authorization != "DENIED" {
		t.Fatalf("authorization = %q", result.Authorization)
	}
}

func TestExecuteRejectsOriginMismatch(t *testing.T) {
	fixture := newFixture(t, "origin-mismatch", "#!/bin/sh\nexit 0\n")
	fixture.commit(t, "baseline")
	request := fixture.request(t, "test")
	request.Target.Origin = "git@github.com:Iron-Signal-Systems/another-project.git"
	_, err := Execute(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "origin does not match") {
		t.Fatalf("expected origin denial, got %v", err)
	}
}

func TestExecuteRejectsTrackedEvidenceDirectory(t *testing.T) {
	fixture := newFixture(t, "tracked-evidence", "#!/bin/sh\nexit 0\n")
	fixture.commit(t, "baseline")
	path := filepath.Join(fixture.root, ".local", "isras", "tracked.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, fixture.root, "add", "-f", ".local/isras/tracked.txt")
	runGit(t, fixture.root, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "track runtime evidence")

	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || !strings.Contains(err.Error(), "must not contain tracked paths") {
		t.Fatalf("expected tracked-evidence denial, got result=%+v err=%v", result, err)
	}
	if result.Authorization != "DENIED" || result.Status != "DENIED" {
		t.Fatalf("unexpected denial result: %+v", result)
	}
}

func TestExecuteRejectsOpaqueShellCommandString(t *testing.T) {
	fixture := newFixture(t, "shell-string", "#!/bin/sh\nexit 0\n")
	fixture.setCommand(t, "test", []string{"sh", "-c", "printf unsafe"})
	fixture.commit(t, "declare shell string")
	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || !strings.Contains(err.Error(), "opaque shell command string") {
		t.Fatalf("expected shell denial, got result=%+v err=%v", result, err)
	}
	assertEvidence(t, result)
}

func TestExecuteRejectsSymlinkedProjectExecutable(t *testing.T) {
	fixture := newFixture(t, "symlink-executable", "#!/bin/sh\nexit 0\n")
	fixture.commit(t, "baseline")
	path := filepath.Join(fixture.root, "tools", "project-check.sh")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/bin/true", path); err != nil {
		t.Fatal(err)
	}
	result, err := Execute(context.Background(), fixture.requestFromCommitted(t, "test"))
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symlink denial, got result=%+v err=%v", result, err)
	}
	assertEvidence(t, result)
}

func TestExecuteBoundsTimeout(t *testing.T) {
	fixture := newFixture(t, "timeout", "#!/bin/sh\nsleep 10\n")
	fixture.commit(t, "baseline")
	previous := executionTimeout
	executionTimeout = 150 * time.Millisecond
	t.Cleanup(func() { executionTimeout = previous })

	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || !result.TimedOut || result.Status != "FAIL" {
		t.Fatalf("expected timeout, got result=%+v err=%v", result, err)
	}
	assertEvidence(t, result)
}

func TestExecuteBoundsOutput(t *testing.T) {
	fixture := newFixture(t, "output-limit", `#!/bin/sh
while :; do
  printf '0123456789abcdef0123456789abcdef\n'
done
`)
	fixture.commit(t, "baseline")
	previous := maxOutputBytes
	maxOutputBytes = 4096
	t.Cleanup(func() { maxOutputBytes = previous })

	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || !result.OutputLimitExceeded || result.Stdout.Bytes != maxOutputBytes {
		t.Fatalf("expected output-limit failure, got result=%+v err=%v", result, err)
	}
	assertEvidence(t, result)
}

func TestExecuteRejectsRepositoryMutation(t *testing.T) {
	fixture := newFixture(t, "mutation", "#!/bin/sh\nprintf 'changed\\n' >> README.md\n")
	fixture.commit(t, "baseline")
	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || !result.RepositoryStateChanged || result.Status != "FAIL" {
		t.Fatalf("expected repository-state failure, got result=%+v err=%v", result, err)
	}
	assertEvidence(t, result)
}

func TestExecuteRetainsFailureEvidence(t *testing.T) {
	fixture := newFixture(t, "failure", "#!/bin/sh\nprintf 'failure detail\\n' >&2\nexit 7\n")
	fixture.commit(t, "baseline")
	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || result.ExitCode != 7 || result.Status != "FAIL" {
		t.Fatalf("expected exit failure, got result=%+v err=%v", result, err)
	}
	if !strings.Contains(result.Stderr.Sanitized, "failure detail") {
		t.Fatalf("stderr evidence missing: %q", result.Stderr.Sanitized)
	}
	assertEvidence(t, result)
}

func TestCanonicalRepositoryRejectsCredentialBearingOrigin(t *testing.T) {
	for _, origin := range []string{
		strings.Join([]string{"ssh://", "git:password@", "github.com/Iron-Signal-Systems/example-project.git"}, ""),
		strings.Join([]string{"https://", "operator:password@", "github.com/Iron-Signal-Systems/example-project.git"}, ""),
	} {
		if _, err := canonicalRepository(origin); err == nil || !strings.Contains(err.Error(), "credentials") {
			t.Fatalf("credential-bearing origin accepted: %q err=%v", origin, err)
		}
	}
}

func TestExecuteTerminatesBackgroundDescendants(t *testing.T) {
	fixture := newFixture(t, "background", "#!/bin/sh\nsleep 30 >/dev/null 2>&1 &\nprintf '%s\\n' \"$!\"\n")
	fixture.commit(t, "baseline")
	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(result.Stdout.Sanitized))
	if err != nil {
		t.Fatalf("parse child pid from %q: %v", result.Stdout.Sanitized, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		terminated, detail := processTerminated(pid)
		if terminated {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background process %d survived project command completion: %s", pid, detail)
		}
		time.Sleep(20 * time.Millisecond)
	}
	assertEvidence(t, result)
}

func processTerminated(pid int) (bool, string) {
	err := syscall.Kill(pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return true, "process no longer exists"
	}
	if err != nil {
		return false, "process probe failed: " + err.Error()
	}

	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if errors.Is(err, os.ErrNotExist) {
		return true, "process no longer exists"
	}
	if err != nil {
		return false, "process exists; state unavailable: " + err.Error()
	}

	closingParen := strings.LastIndexByte(string(stat), ')')
	if closingParen < 0 || closingParen+2 >= len(stat) {
		return false, "process exists; malformed /proc state"
	}
	state := stat[closingParen+2]
	if state == 'Z' {
		return true, "process is a terminated zombie awaiting init reaping"
	}
	return false, "process state=" + string(state)
}

func TestProcessTerminatedAcceptsZombie(t *testing.T) {
	command := exec.Command("/bin/sh", "-c", "exit 0")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Wait() }()

	pid := command.Process.Pid
	deadline := time.Now().Add(2 * time.Second)
	for {
		stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
		if err != nil {
			t.Fatalf("read child state: %v", err)
		}
		closingParen := strings.LastIndexByte(string(stat), ')')
		if closingParen >= 0 && closingParen+2 < len(stat) && stat[closingParen+2] == 'Z' {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("child process %d did not enter zombie state", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}

	terminated, detail := processTerminated(pid)
	if !terminated || !strings.Contains(detail, "zombie") {
		t.Fatalf("zombie process was not recognized as terminated: terminated=%v detail=%q", terminated, detail)
	}
}

func TestExecuteRejectsSymlinkedEvidenceDirectory(t *testing.T) {
	fixture := newFixture(t, "evidence-symlink", "#!/bin/sh\nexit 0\n")
	fixture.commit(t, "baseline")
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(fixture.root, ".local")); err != nil {
		t.Fatal(err)
	}
	result, err := Execute(context.Background(), fixture.request(t, "test"))
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected evidence-path denial, got result=%+v err=%v", result, err)
	}
}

type testFixture struct {
	root string
	pin  projectpin.Pin
}

func newFixture(t *testing.T, repositoryName, script string) *testFixture {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.name", "ISRAS Test")
	runGit(t, root, "config", "user.email", "isras-test@example.invalid")
	runGit(t, root, "remote", "add", "origin", "git@github.com:Iron-Signal-Systems/"+repositoryName+".git")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	goMod := "module github.com/Iron-Signal-Systems/" + repositoryName + "\n\ngo 1.25.12\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(root, "tools", "project-check.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := &testFixture{root: root, pin: validPin(repositoryName)}
	fixture.writePin(t)
	return fixture
}

func (fixture *testFixture) setCommand(t *testing.T, name string, arguments []string) {
	t.Helper()
	fixture.pin.Commands[name] = append([]string(nil), arguments...)
	fixture.writePin(t)
}

func (fixture *testFixture) writePin(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(fixture.root, ".isras"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(fixture.pin, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(fixture.root, projectpin.MetadataPath), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (fixture *testFixture) commit(t *testing.T, message string) {
	t.Helper()
	runGit(t, fixture.root, "add", "-A")
	runGit(t, fixture.root, "commit", "-q", "-m", message)
}

func (fixture *testFixture) request(t *testing.T, name string) Request {
	t.Helper()
	pin, err := projectpin.LoadCommitted(context.Background(), fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	return fixture.requestWithPin(t, name, pin)
}

func (fixture *testFixture) requestFromCommitted(t *testing.T, name string) Request {
	t.Helper()
	data := gitOutput(t, fixture.root, "show", "HEAD:"+projectpin.MetadataPath)
	pin, err := projectpin.Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return fixture.requestWithPin(t, name, pin)
}

func (fixture *testFixture) requestWithPin(t *testing.T, name string, pin projectpin.Pin) Request {
	t.Helper()
	identity, err := repository.DiscoverFrom(context.Background(), fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	return Request{
		Root:   fixture.root,
		Mode:   "development",
		Target: identity,
		Validator: validatoridentity.Identity{
			Metadata: validatoridentity.Metadata{
				SchemaVersion:    1,
				Profile:          projectpin.Profile,
				StandardVersion:  "1.2.3",
				Ownership:        validatoridentity.OwnershipReleaseArtifact,
				SourceRepository: projectpin.SourceRepository,
				SourceCommit:     testSourceCommit,
			},
			ReleaseTag:       "isras-v1.2.3",
			RepositoryCommit: testSourceCommit,
		},
		Pin:  pin,
		Name: name,
	}
}

func validPin(repositoryName string) projectpin.Pin {
	commands := map[string][]string{
		"format_check":          {"go", "version"},
		"static_analysis":       {"go", "version"},
		"test":                  {"./tools/project-check.sh"},
		"build":                 {"go", "version"},
		"module_consistency":    {"go", "version"},
		"module_integrity":      {"go", "version"},
		"known_vulnerabilities": {"go", "version"},
	}
	return projectpin.Pin{
		SchemaVersion: 1,
		Project:       projectpin.Project{Repository: "github.com/Iron-Signal-Systems/" + repositoryName},
		Standard: projectpin.Standard{
			Profile:          projectpin.Profile,
			Version:          "1.2.3",
			ReleaseTag:       "isras-v1.2.3",
			SourceRepository: projectpin.SourceRepository,
			SourceCommit:     testSourceCommit,
		},
		Artifacts: []projectpin.Artifact{
			{Kind: "validator", OS: "linux", Arch: "amd64", Name: "validator", SHA256: strings.Repeat("1", 64), SHA512: strings.Repeat("1", 128)},
			{Kind: "framework", Name: "framework.tar.gz", SHA256: strings.Repeat("2", 64), SHA512: strings.Repeat("2", 128)},
			{Kind: "contracts", Name: "contracts.tar.gz", SHA256: strings.Repeat("3", 64), SHA512: strings.Repeat("3", 128)},
			{Kind: "provenance", Name: "provenance.json", SHA256: strings.Repeat("4", 64), SHA512: strings.Repeat("4", 128)},
			{Kind: "sha256-manifest", Name: "SHA256SUMS", SHA256: strings.Repeat("5", 64), SHA512: strings.Repeat("5", 128)},
			{Kind: "sha512-manifest", Name: "SHA512SUMS", SHA256: strings.Repeat("6", 64), SHA512: strings.Repeat("6", 128)},
		},
		Workflow: projectpin.Workflow{Repository: projectpin.SourceRepository, Path: projectpin.ReusableWorkflowPath, Commit: testSourceCommit},
		Profiles: []string{"go"},
		Commands: commands,
		Evidence: projectpin.Evidence{Directory: projectpin.RuntimeEvidenceDirectory},
	}
}

func assertEvidence(t *testing.T, result Result) {
	t.Helper()
	for _, path := range []string{result.EvidenceJSON, result.EvidenceText} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("evidence mode for %s = %o", path, info.Mode().Perm())
		}
	}
	info, err := os.Stat(filepath.Dir(result.EvidenceJSON))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("run directory mode = %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(result.EvidenceJSON)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "ghp_A") || strings.Contains(string(data), "must-not-be-inherited") {
		t.Fatalf("sensitive value reached evidence: %s", data)
	}
}

func runGit(t *testing.T, root string, arguments ...string) {
	t.Helper()
	_ = gitOutput(t, root, arguments...)
}

func gitOutput(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgSign=false"}, arguments...)...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", arguments, err, output)
	}
	return string(output)
}
