package exportvalidator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestExporterAcceptsOrdinaryCloneAndStagesValidatedExport(t *testing.T) {
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{})

	output, err := runExporter(source, target, false)
	if err != nil {
		t.Fatalf("export failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "PROJECT VALIDATOR EXPORTED TRANSACTIONALLY") {
		t.Fatalf("success boundary missing:\n%s", output)
	}
	requireStagedPaths(t, target,
		".gitignore",
		"cmd/isras-validate/main.go",
		"internal/isras/dashboard/dashboard.go",
		"internal/isras/executil/executil.go",
		"internal/isras/failurelog/failurelog.go",
		"internal/isras/model/model.go",
		"internal/isras/redact/redact.go",
		"internal/isras/repository/repository.go",
		"internal/isras/secrets/secrets.go",
		"internal/isras/validation/validation.go",
		"internal/isras/validation/validation_test.go",
		"internal/isras/validatoridentity/identity.go",
		"internal/isras/validatoridentity/identity_test.go",
		"tools/isras/build-validator.sh",
		"validation/isras-validator-identity.json",
		"validation/secret-allowlist.json",
		"validation/tool-versions.json",
	)
}

func TestExporterPinsImmutableProjectOwnedIdentity(t *testing.T) {
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{})

	output, err := runExporter(source, target, false)
	if err != nil {
		t.Fatalf("export failed: %v\n%s", err, output)
	}

	data := readFile(t, filepath.Join(target, "validation", "isras-validator-identity.json"))
	var identity struct {
		SchemaVersion    int    `json:"schema_version"`
		Profile          string `json:"profile"`
		StandardVersion  string `json:"standard_version"`
		Ownership        string `json:"ownership"`
		SourceRepository string `json:"source_repository"`
		SourceCommit     string `json:"source_commit"`
		TargetModule     string `json:"target_module"`
	}
	if err := json.Unmarshal([]byte(data), &identity); err != nil {
		t.Fatal(err)
	}

	sourceHead := strings.TrimSpace(git(t, source, "rev-parse", "HEAD"))
	if identity.SchemaVersion != 1 || identity.Profile != "ISRAS-SD" {
		t.Fatalf("unexpected identity schema/profile: %#v", identity)
	}
	if identity.StandardVersion != "0.1.1-development" {
		t.Fatalf("unexpected standard version: %s", identity.StandardVersion)
	}
	if identity.Ownership != "project-owned-export" {
		t.Fatalf("unexpected ownership: %s", identity.Ownership)
	}
	if identity.SourceRepository != "github.com/Iron-Signal-Systems/engineering-standards" {
		t.Fatalf("unexpected source repository: %s", identity.SourceRepository)
	}
	if identity.SourceCommit != sourceHead {
		t.Fatalf("source commit mismatch: got %s want %s", identity.SourceCommit, sourceHead)
	}
	if identity.TargetModule != "example.com/target" {
		t.Fatalf("target module mismatch: %s", identity.TargetModule)
	}
	if !strings.Contains(output, "ISRAS source:    "+identity.SourceRepository+"@"+sourceHead) {
		t.Fatalf("identity evidence missing from output:\n%s", output)
	}
}

func TestExporterRejectsSourceIdentityVersionDrift(t *testing.T) {
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{})
	path := filepath.Join(source, "validation", "isras-validator-identity.json")
	data := strings.Replace(readFile(t, path), "0.1.1-development", "0.1.0-development", 1)
	writeFile(t, path, data, 0o644)
	git(t, source, "add", path)
	git(t, source, "commit", "-q", "-m", "drift identity")

	output, err := runExporter(source, target, false)
	if err == nil || !strings.Contains(output, "identity version does not match VERSION") {
		t.Fatalf("expected source identity drift rejection: err=%v\n%s", err, output)
	}
	requireClean(t, target)
}

func TestExporterAcceptsLinkedWorktree(t *testing.T) {
	source := createSourceFixture(t, false)
	base := createTargetFixture(t, targetOptions{})
	worktree := filepath.Join(t.TempDir(), "linked")
	git(t, base, "worktree", "add", "-q", "-b", "adoption-test", worktree)

	if info, err := os.Stat(filepath.Join(worktree, ".git")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("expected linked worktree .git file: info=%v err=%v", info, err)
	}

	output, err := runExporter(source, worktree, false)
	if err != nil {
		t.Fatalf("linked-worktree export failed: %v\n%s", err, output)
	}
	requireStagedPath(t, worktree, "cmd/isras-validate/main.go")
}

func TestExporterDryRunLeavesTargetClean(t *testing.T) {
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{})

	output, err := runExporter(source, target, true)
	if err != nil {
		t.Fatalf("dry run failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "PROJECT VALIDATOR EXPORT DRY RUN PASSED") {
		t.Fatalf("dry-run boundary missing:\n%s", output)
	}
	requireClean(t, target)
}

func TestExporterRejectsNonGitDirectory(t *testing.T) {
	source := createSourceFixture(t, false)
	target := t.TempDir()
	writeFile(t, filepath.Join(target, "go.mod"), "module example.com/target\n\ngo 1.23.0\n", 0o644)

	output, err := runExporter(source, target, false)
	if err == nil || !strings.Contains(output, "not a Git working tree") {
		t.Fatalf("expected non-Git rejection: err=%v\n%s", err, output)
	}
}

func TestExporterRejectsBareRepository(t *testing.T) {
	source := createSourceFixture(t, false)
	bare := filepath.Join(t.TempDir(), "bare.git")
	run(t, "", "git", "init", "--bare", "-q", bare)

	output, err := runExporter(source, bare, false)
	if err == nil || !strings.Contains(output, "must not be a bare Git repository") {
		t.Fatalf("expected bare-repository rejection: err=%v\n%s", err, output)
	}
}

func TestExporterRejectsDirtyTarget(t *testing.T) {
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{})
	writeFile(t, filepath.Join(target, "dirty.txt"), "dirty\n", 0o644)

	output, err := runExporter(source, target, false)
	if err == nil || !strings.Contains(output, "must be clean") {
		t.Fatalf("expected dirty-target rejection: err=%v\n%s", err, output)
	}
}

func TestExporterRejectsExistingExportPath(t *testing.T) {
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{existingExportPath: true})

	output, err := runExporter(source, target, false)
	if err == nil || !strings.Contains(output, "target path already exists") {
		t.Fatalf("expected existing-path rejection: err=%v\n%s", err, output)
	}
	requireClean(t, target)
}

func TestExporterRollsBackWhenAppliedTargetValidationFails(t *testing.T) {
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{})
	binDir := t.TempDir()
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	wrapper := fmt.Sprintf(`#!/usr/bin/env bash
set -Eeuo pipefail
if [[ "$PWD" == %q && "${1:-}" == "test" ]]; then
  echo "forced target validation failure" >&2
  exit 91
fi
exec %q "$@"
`, target, realGo)
	writeFile(t, filepath.Join(binDir, "go"), wrapper, 0o755)

	output, err := runExporterWithPath(source, target, false, binDir)
	if err == nil || !strings.Contains(output, "forced target validation failure") {
		t.Fatalf("expected forced target validation failure: err=%v\n%s", err, output)
	}
	requireClean(t, target)
	if _, err := os.Stat(filepath.Join(target, "cmd", "isras-validate")); !os.IsNotExist(err) {
		t.Fatalf("partial export remained after rollback: %v", err)
	}
}

func TestExporterPromotesExistingIndirectRequirement(t *testing.T) {
	dep := createDependencyModule(t, "example.com/exportdep")
	source := createSourceFixtureWithDependency(t, "example.com/exportdep")
	target := createTargetFixture(t, targetOptions{
		requireModule:   "example.com/exportdep v0.0.0",
		requireIndirect: true,
		replaceModule:   "example.com/exportdep",
		replacePath:     dep,
	})

	output, err := runExporter(source, target, false)
	if err != nil {
		t.Fatalf("promotion export failed: %v\n%s", err, output)
	}
	goMod := readFile(t, filepath.Join(target, "go.mod"))
	if strings.Contains(goMod, "example.com/exportdep v0.0.0 // indirect") {
		t.Fatalf("requirement was not promoted:\n%s", goMod)
	}
	if !strings.Contains(goMod, "example.com/exportdep v0.0.0") {
		t.Fatalf("promoted requirement missing:\n%s", goMod)
	}
	requireStagedPath(t, target, "go.mod")
}

func TestExporterAddsRequiredModuleDeterministically(t *testing.T) {
	dep := createDependencyModule(t, "example.com/exportdep")
	source := createSourceFixtureWithDependency(t, "example.com/exportdep")
	target := createTargetFixture(t, targetOptions{
		replaceModule: "example.com/exportdep",
		replacePath:   dep,
	})

	output, err := runExporter(source, target, false)
	if err != nil {
		t.Fatalf("module-addition export failed: %v\n%s", err, output)
	}
	goMod := readFile(t, filepath.Join(target, "go.mod"))
	if !strings.Contains(goMod, "example.com/exportdep v") {
		t.Fatalf("new requirement missing:\n%s", goMod)
	}
	requireStagedPath(t, target, "go.mod")
}

func TestExporterRejectsUnexpectedRequirementRemovalWithoutTargetMutation(t *testing.T) {
	unused := createDependencyModule(t, "example.com/unused")
	source := createSourceFixture(t, false)
	target := createTargetFixture(t, targetOptions{
		requireModule:   "example.com/unused v0.0.0",
		requireIndirect: true,
		replaceModule:   "example.com/unused",
		replacePath:     unused,
	})

	output, err := runExporter(source, target, false)
	if err == nil || !strings.Contains(output, "removed existing requirement") {
		t.Fatalf("expected requirement-removal rejection: err=%v\n%s", err, output)
	}
	requireClean(t, target)
}

type targetOptions struct {
	existingExportPath bool
	requireModule      string
	requireIndirect    bool
	replaceModule      string
	replacePath        string
}

func createSourceFixture(t *testing.T, dependency bool) string {
	t.Helper()
	if dependency {
		return createSourceFixtureWithDependency(t, "example.com/exportdep")
	}
	return createSourceFixtureWithDependency(t, "")
}

func createSourceFixtureWithDependency(t *testing.T, dependency string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "source")
	mustMkdir(t, filepath.Join(root, "tools"))

	repositoryRoot := testRepositoryRoot(t)
	copyFile(t,
		filepath.Join(repositoryRoot, "tools", "export-project-validator.sh"),
		filepath.Join(root, "tools", "export-project-validator.sh"),
		0o755,
	)

	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/Iron-Signal-Systems/engineering-standards\n\ngo 1.23.0\n", 0o644)
	writeFile(t, filepath.Join(root, "VERSION"), "0.1.1-development\n", 0o644)
	writeFile(t, filepath.Join(root, "cmd", "isras-validate", "main.go"), `package main

import (
	"fmt"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
)

func main() { fmt.Println(validation.Message()) }
`, 0o644)

	packages := []string{"dashboard", "executil", "failurelog", "model", "redact", "repository", "secrets"}
	for _, name := range packages {
		writeFile(t, filepath.Join(root, "internal", name, name+".go"), "package "+name+"\n", 0o644)
	}
	copyFile(t,
		filepath.Join(repositoryRoot, "internal", "validatoridentity", "identity.go"),
		filepath.Join(root, "internal", "validatoridentity", "identity.go"),
		0o644,
	)
	copyFile(t,
		filepath.Join(repositoryRoot, "internal", "validatoridentity", "identity_test.go"),
		filepath.Join(root, "internal", "validatoridentity", "identity_test.go"),
		0o644,
	)

	validationSource := "package validation\n\nfunc Message() string { return \"ok\" }\n"
	if dependency != "" {
		validationSource = fmt.Sprintf(`package validation

import dependency %q

func Message() string { return dependency.Value() }
`, dependency)
	}
	writeFile(t, filepath.Join(root, "internal", "validation", "validation.go"), validationSource, 0o644)
	writeFile(t, filepath.Join(root, "internal", "validation", "validation_test.go"), `package validation

import "testing"

func TestMessage(t *testing.T) {
	if Message() == "" { t.Fatal("empty") }
}
`, 0o644)
	writeFile(t, filepath.Join(root, "validation", "isras-validator-identity.json"), `{
  "schema_version": 1,
  "profile": "ISRAS-SD",
  "standard_version": "0.1.1-development",
  "ownership": "reference-repository",
  "source_repository": "github.com/Iron-Signal-Systems/engineering-standards"
}
`, 0o644)
	writeFile(t, filepath.Join(root, "validation", "secret-allowlist.json"), "{\"version\":1,\"entries\":[]}\n", 0o644)
	writeFile(t, filepath.Join(root, "validation", "tool-versions.json"), "{\"version\":1}\n", 0o644)
	writeFile(t, filepath.Join(root, "tools", "build-validator.sh"), "#!/usr/bin/env bash\nset -Eeuo pipefail\ngo build -o .local/bin/isras-validate ./cmd/isras-validate\n", 0o755)

	initAndCommit(t, root)
	return root
}

func createTargetFixture(t *testing.T, options targetOptions) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "target")
	mustMkdir(t, root)
	goMod := "module example.com/target\n\ngo 1.23.0\n"
	if options.requireModule != "" {
		goMod += "\nrequire " + options.requireModule
		if options.requireIndirect {
			goMod += " // indirect"
		}
		goMod += "\n"
	}
	if options.replaceModule != "" {
		goMod += "\nreplace " + options.replaceModule + " => " + filepath.ToSlash(options.replacePath) + "\n"
	}
	writeFile(t, filepath.Join(root, "go.mod"), goMod, 0o644)
	writeFile(t, filepath.Join(root, "main.go"), "package target\n\nfunc Ready() bool { return true }\n", 0o644)
	writeFile(t, filepath.Join(root, "main_test.go"), "package target\n\nimport \"testing\"\n\nfunc TestReady(t *testing.T) { if !Ready() { t.Fatal(\"not ready\") } }\n", 0o644)
	if options.existingExportPath {
		writeFile(t, filepath.Join(root, "cmd", "isras-validate", "main.go"), "package main\nfunc main() {}\n", 0o644)
	}
	initAndCommit(t, root)
	return root
}

func createDependencyModule(t *testing.T, module string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), strings.ReplaceAll(module, "/", "-"))
	writeFile(t, filepath.Join(root, "go.mod"), "module "+module+"\n\ngo 1.23.0\n", 0o644)
	writeFile(t, filepath.Join(root, "dependency.go"), "package exportdep\n\nfunc Value() string { return \"dependency\" }\n", 0o644)
	return root
}

func runExporter(source, target string, dryRun bool) (string, error) {
	return runExporterWithPath(source, target, dryRun, "")
}

func runExporterWithPath(source, target string, dryRun bool, pathPrefix string) (string, error) {
	args := []string{}
	if dryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, target)
	cmd := exec.Command(filepath.Join(source, "tools", "export-project-validator.sh"), args...)
	cmd.Dir = source
	cmd.Env = append(os.Environ(), "ISRAS_EXPORT_GO_TIMEOUT_SECONDS=60")
	if pathPrefix != "" {
		cmd.Env = append(cmd.Env, "PATH="+pathPrefix+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func initAndCommit(t *testing.T, root string) {
	t.Helper()
	git(t, root, "init", "-q")
	git(t, root, "config", "user.name", "ISRAS Test")
	git(t, root, "config", "user.email", "isras-test@example.invalid")
	// Disposable fixture commits are test scaffolding, not project evidence.
	// Override any developer-wide signing policy locally so automated tests do
	// not invoke encrypted signing keys or require an interactive agent.
	git(t, root, "config", "commit.gpgsign", "false")
	git(t, root, "add", ".")
	git(t, root, "commit", "-q", "-m", "fixture")
}

func requireClean(t *testing.T, root string) {
	t.Helper()
	output := run(t, root, "git", "status", "--porcelain=v1", "--untracked-files=all")
	if strings.TrimSpace(output) != "" {
		t.Fatalf("repository is not clean:\n%s", output)
	}
}

func requireStagedPath(t *testing.T, root, expected string) {
	t.Helper()
	paths := stagedPaths(t, root)
	for _, path := range paths {
		if path == expected {
			return
		}
	}
	t.Fatalf("staged path missing: %s\nactual: %v", expected, paths)
}

func requireStagedPaths(t *testing.T, root string, expected ...string) {
	t.Helper()
	actual := stagedPaths(t, root)
	sort.Strings(expected)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("unexpected staged paths:\nactual:\n%s\nexpected:\n%s", strings.Join(actual, "\n"), strings.Join(expected, "\n"))
	}
}

func stagedPaths(t *testing.T, root string) []string {
	t.Helper()
	output := run(t, root, "git", "diff", "--cached", "--name-only")
	var paths []string
	for _, path := range strings.Split(strings.TrimSpace(output), "\n") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	return run(t, root, "git", args...)
}

func run(t *testing.T, root, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	if root != "" {
		cmd.Dir = root
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func copyFile(t *testing.T, source, target string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, target, string(data), mode)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
