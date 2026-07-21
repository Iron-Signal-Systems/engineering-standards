package projectcommand

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDiscoverGoModulesFindsEveryModuleInStableOrder(t *testing.T) {
	root := t.TempDir()
	writeModuleFile(
		t,
		root,
		"go.mod",
		"github.com/Iron-Signal-Systems/root",
		"1.25.12",
		"default",
	)
	writeModuleFile(
		t,
		root,
		"modules/alpha/go.mod",
		"github.com/Iron-Signal-Systems/root/alpha",
		"1.26.0",
		"",
	)
	writeModuleFile(
		t,
		root,
		"modules/zulu/go.mod",
		"github.com/Iron-Signal-Systems/root/zulu",
		"1.25.13",
		"go1.26.0",
	)

	modules, err := discoverGoModules(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 3 {
		t.Fatalf("module count = %d, want 3", len(modules))
	}

	expected := []struct {
		goMod     string
		directory string
		module    string
		minimum   string
		toolchain string
	}{
		{
			"go.mod",
			".",
			"github.com/Iron-Signal-Systems/root",
			"go1.25.12",
			"default",
		},
		{
			"modules/alpha/go.mod",
			"modules/alpha",
			"github.com/Iron-Signal-Systems/root/alpha",
			"go1.26.0",
			"",
		},
		{
			"modules/zulu/go.mod",
			"modules/zulu",
			"github.com/Iron-Signal-Systems/root/zulu",
			"go1.25.13",
			"go1.26.0",
		},
	}

	for index, want := range expected {
		got := modules[index]
		if got.GoModPath != want.goMod ||
			got.Directory != want.directory ||
			got.ModulePath != want.module ||
			got.Minimum != want.minimum ||
			got.Toolchain != want.toolchain {
			t.Fatalf("module %d = %+v, want %+v", index, got, want)
		}
	}
}

func TestDiscoverGoModulesRejectsDuplicateModuleIdentity(t *testing.T) {
	root := t.TempDir()
	writeModuleFile(
		t,
		root,
		"go.mod",
		"github.com/Iron-Signal-Systems/duplicate",
		"1.25.12",
		"",
	)
	writeModuleFile(
		t,
		root,
		"nested/go.mod",
		"github.com/Iron-Signal-Systems/duplicate",
		"1.25.12",
		"",
	)

	_, err := discoverGoModules(root)
	if err == nil || !strings.Contains(err.Error(), "declared by both") {
		t.Fatalf("error = %v", err)
	}
}

func TestDiscoverGoModulesRejectsHostileModulePaths(t *testing.T) {
	t.Run("missing root module", func(t *testing.T) {
		root := t.TempDir()
		writeModuleFile(
			t,
			root,
			"nested/go.mod",
			"github.com/Iron-Signal-Systems/nested",
			"1.25.12",
			"",
		)
		_, err := discoverGoModules(root)
		if err == nil || !strings.Contains(
			err.Error(),
			"root does not contain",
		) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("symlinked go.mod", func(t *testing.T) {
		root := t.TempDir()
		ensureModuleRepository(t, root)
		external := filepath.Join(t.TempDir(), "go.mod")
		if err := os.WriteFile(
			external,
			[]byte(
				"module github.com/Iron-Signal-Systems/external\n\n"+
					"go 1.25.12\n",
			),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(
			external,
			filepath.Join(root, "go.mod"),
		); err != nil {
			t.Fatal(err)
		}
		_, err := discoverGoModules(root)
		if err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("directory named go.mod", func(t *testing.T) {
		root := t.TempDir()
		ensureModuleRepository(t, root)
		if err := os.Mkdir(
			filepath.Join(root, "go.mod"),
			0o700,
		); err != nil {
			t.Fatal(err)
		}
		_, err := discoverGoModules(root)
		if err == nil || !strings.Contains(err.Error(), "non-regular") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestDiscoverGoModulesUsesRepositoryOwnedSourceBoundary(
	t *testing.T,
) {
	root := t.TempDir()
	writeModuleFile(
		t,
		root,
		"go.mod",
		"github.com/Iron-Signal-Systems/source-boundary",
		"1.25.12",
		"",
	)
	writeModuleFile(
		t,
		root,
		"modules/tracked/go.mod",
		"github.com/Iron-Signal-Systems/source-boundary/tracked",
		"1.25.12",
		"",
	)
	writeModuleFile(
		t,
		root,
		"modules/working/go.mod",
		"github.com/Iron-Signal-Systems/source-boundary/working",
		"1.25.12",
		"",
	)

	runModuleGit(
		t,
		root,
		"add",
		"go.mod",
		"modules/tracked/go.mod",
	)

	writeModuleFile(
		t,
		root,
		".local/validation/releases/run/repository/go.mod",
		"github.com/Iron-Signal-Systems/generated-validation-copy",
		"1.99.0",
		"",
	)
	writeModuleFile(
		t,
		root,
		".local/isras/project-commands/run/go.mod",
		"github.com/Iron-Signal-Systems/generated-command-copy",
		"1.99.0",
		"",
	)

	modules, err := discoverGoModules(root)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"go.mod",
		"modules/tracked/go.mod",
		"modules/working/go.mod",
	}
	if len(modules) != len(expected) {
		t.Fatalf("module count = %d, modules = %+v", len(modules), modules)
	}
	for index, path := range expected {
		if modules[index].GoModPath != path {
			t.Fatalf(
				"module %d path = %q, want %q",
				index,
				modules[index].GoModPath,
				path,
			)
		}
		if strings.HasPrefix(modules[index].GoModPath, ".local/") {
			t.Fatalf("local runtime module escaped: %+v", modules[index])
		}
	}
}

func TestRepositoryGoModulePathsIgnoreCallerPATH(t *testing.T) {
	root := t.TempDir()
	writeModuleFile(
		t,
		root,
		"go.mod",
		"github.com/Iron-Signal-Systems/bounded-git",
		"1.25.12",
		"",
	)

	t.Setenv("PATH", t.TempDir())

	paths, err := repositoryGoModulePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "go.mod" {
		t.Fatalf("module paths = %#v", paths)
	}
}

func TestRepositoryModuleInventoryExcludesLocalRuntimeEvidence(
	t *testing.T,
) {
	root := projectCommandRepositoryRoot(t)

	modules, err := discoverGoModules(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, module := range modules {
		if strings.HasPrefix(module.GoModPath, ".local/") {
			t.Fatalf(
				"local runtime evidence entered source inventory: %+v",
				module,
			)
		}
	}
}

func TestRepositoryGoModuleGitCommandTrustsExactRepositoryRoot(
	t *testing.T,
) {
	root := t.TempDir()
	writeModuleFile(
		t,
		root,
		"go.mod",
		"github.com/Iron-Signal-Systems/safe-directory",
		"1.25.12",
		"",
	)
	runModuleGit(t, root, "add", "go.mod")

	gitExecutable, err := boundedSystemExecutable("git")
	if err != nil {
		t.Fatal(err)
	}

	command := repositoryGoModuleGitCommand(root, gitExecutable)
	command.Env = append(
		command.Env,
		"GIT_TEST_ASSUME_DIFFERENT_OWNER=1",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"enumerate with forced different-owner protection: %v\n%s",
			err,
			output,
		)
	}
	if string(output) != "go.mod\x00" {
		t.Fatalf("module inventory output = %q", output)
	}

	expectedSafeDirectory := "safe.directory=" + root
	foundSafeDirectory := false
	for _, argument := range command.Args {
		if argument == expectedSafeDirectory {
			foundSafeDirectory = true
			break
		}
	}
	if !foundSafeDirectory {
		t.Fatalf(
			"Git arguments do not trust the exact repository root: %#v",
			command.Args,
		)
	}

	requiredEnvironment := map[string]bool{
		"GIT_CONFIG_GLOBAL=" + os.DevNull: false,
		"GIT_CONFIG_NOSYSTEM=1":           false,
	}
	for _, entry := range command.Env {
		if _, required := requiredEnvironment[entry]; required {
			requiredEnvironment[entry] = true
		}
	}
	for entry, found := range requiredEnvironment {
		if !found {
			t.Fatalf(
				"bounded Git environment is missing %q: %#v",
				entry,
				command.Env,
			)
		}
	}
}

func TestSelectGoToolchainUsesHighestModuleMinimum(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX fake Go executable")
	}

	root := t.TempDir()
	writeModuleFile(
		t,
		root,
		"go.mod",
		"github.com/Iron-Signal-Systems/root",
		"1.25.12",
		"default",
	)
	writeModuleFile(
		t,
		root,
		"modules/worker/go.mod",
		"github.com/Iron-Signal-Systems/root/worker",
		"1.26.0",
		"",
	)

	selectedBin := filepath.Join(root, "selected-go", "bin")
	if err := os.MkdirAll(selectedBin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeGo(
		t,
		filepath.Join(selectedBin, "go"),
		"go1.26.5-X:nodwarf5",
	)
	t.Setenv("PATH", selectedBin)

	selection, err := selectGoToolchain(root)
	if err != nil {
		t.Fatal(err)
	}
	if selection.Minimum != "go1.26.0" {
		t.Fatalf("project minimum = %q", selection.Minimum)
	}
	if len(selection.Modules) != 2 {
		t.Fatalf("module count = %d", len(selection.Modules))
	}
	for _, module := range selection.Modules {
		if !module.MinimumSatisfied {
			t.Fatalf("module not satisfied: %+v", module)
		}
	}
}

func TestSelectGoToolchainRejectsUnsatisfiedNestedModule(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX fake Go executable")
	}

	root := t.TempDir()
	writeModuleFile(
		t,
		root,
		"go.mod",
		"github.com/Iron-Signal-Systems/root",
		"1.25.12",
		"",
	)
	writeModuleFile(
		t,
		root,
		"modules/worker/go.mod",
		"github.com/Iron-Signal-Systems/root/worker",
		"1.27.0",
		"",
	)

	selectedBin := filepath.Join(root, "selected-go", "bin")
	if err := os.MkdirAll(selectedBin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeGo(
		t,
		filepath.Join(selectedBin, "go"),
		"go1.26.0",
	)
	t.Setenv("PATH", selectedBin)

	selection, err := selectGoToolchain(root)
	if err == nil ||
		!strings.Contains(err.Error(), "modules/worker/go.mod") {
		t.Fatalf("selection=%+v error=%v", selection, err)
	}
	if selection.MinimumSatisfied {
		t.Fatal("unsatisfied inventory recorded overall success")
	}
	if len(selection.Modules) != 2 {
		t.Fatalf("module count = %d", len(selection.Modules))
	}
	if selection.Modules[0].GoModPath != "go.mod" ||
		!selection.Modules[0].MinimumSatisfied {
		t.Fatalf("root module evidence = %+v", selection.Modules[0])
	}
	if selection.Modules[1].GoModPath != "modules/worker/go.mod" ||
		selection.Modules[1].MinimumSatisfied {
		t.Fatalf("nested module evidence = %+v", selection.Modules[1])
	}
}

func TestGoProfileEvidenceV2RecordsEveryDiscoveredModule(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX executable paths")
	}

	fixture := newFixture(
		t,
		"module-inventory-evidence",
		"#!/bin/sh\nexit 0\n",
	)
	writeModuleFile(
		t,
		fixture.root,
		"modules/worker/go.mod",
		"github.com/Iron-Signal-Systems/module-inventory-evidence/worker",
		"1.26.0",
		"default",
	)
	fixture.commit(t, "declare multi-module evidence fixture")

	selectedBin := filepath.Join(t.TempDir(), "selected-go", "bin")
	if err := os.MkdirAll(selectedBin, 0o700); err != nil {
		t.Fatal(err)
	}
	selectedGo := filepath.Join(selectedBin, "go")
	writeFakeGo(t, selectedGo, "go1.26.5-X:nodwarf5")

	originalPath := os.Getenv("PATH")
	if originalPath == "" {
		t.Fatal("test requires a caller PATH")
	}
	t.Setenv(
		"PATH",
		selectedBin+string(os.PathListSeparator)+originalPath,
	)

	result, err := Execute(
		context.Background(),
		fixture.request(t, "test"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.GoToolchain == nil {
		t.Fatal("Go toolchain evidence is missing")
	}
	if len(result.GoToolchain.Modules) != 2 {
		t.Fatalf(
			"module evidence count = %d",
			len(result.GoToolchain.Modules),
		)
	}

	rootModule := result.GoToolchain.Modules[0]
	workerModule := result.GoToolchain.Modules[1]
	if rootModule.GoModPath != "go.mod" ||
		rootModule.Directory != "." ||
		!rootModule.GoMinimumSatisfied {
		t.Fatalf("root module evidence = %+v", rootModule)
	}
	if workerModule.GoModPath != "modules/worker/go.mod" ||
		workerModule.Directory != "modules/worker" ||
		workerModule.ModulePath !=
			"github.com/Iron-Signal-Systems/module-inventory-evidence/worker" ||
		workerModule.GoMinimum != "go1.26.0" ||
		workerModule.ToolchainDirective != "default" ||
		!workerModule.GoMinimumSatisfied {
		t.Fatalf("worker module evidence = %+v", workerModule)
	}

	text, err := os.ReadFile(result.EvidenceText)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Go module count: 2",
		"Go module 1 go.mod: go.mod",
		"Go module 2 go.mod: modules/worker/go.mod",
		"Go module 2 path: " +
			"github.com/Iron-Signal-Systems/" +
			"module-inventory-evidence/worker",
		"Go module 2 minimum: go1.26.0",
		"Go module 2 toolchain directive: default",
		"Go module 2 minimum satisfied: true",
	} {
		if !strings.Contains(string(text), expected) {
			t.Fatalf("text evidence missing %q:\n%s", expected, text)
		}
	}
	assertEvidence(t, result)
}

func TestGoProfileEvidenceV2SynchronizesModuleRemoval(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX executable paths")
	}

	fixture := newFixture(
		t,
		"module-removal-evidence",
		"#!/bin/sh\nexit 0\n",
	)
	nestedGoMod := filepath.Join(
		fixture.root,
		"modules",
		"worker",
		"go.mod",
	)
	writeModuleFile(
		t,
		fixture.root,
		"modules/worker/go.mod",
		"github.com/Iron-Signal-Systems/module-removal-evidence/worker",
		"1.25.12",
		"",
	)
	fixture.commit(t, "declare nested module")

	selectedBin := filepath.Join(t.TempDir(), "selected-go", "bin")
	if err := os.MkdirAll(selectedBin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeGo(
		t,
		filepath.Join(selectedBin, "go"),
		"go1.26.0",
	)

	originalPath := os.Getenv("PATH")
	if originalPath == "" {
		t.Fatal("test requires a caller PATH")
	}
	t.Setenv(
		"PATH",
		selectedBin+string(os.PathListSeparator)+originalPath,
	)

	before, err := Execute(
		context.Background(),
		fixture.request(t, "test"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if before.GoToolchain == nil ||
		len(before.GoToolchain.Modules) != 2 {
		t.Fatalf("before removal = %+v", before.GoToolchain)
	}

	if err := os.Remove(nestedGoMod); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Dir(nestedGoMod)); err != nil {
		t.Fatal(err)
	}
	fixture.commit(t, "remove nested module")

	after, err := Execute(
		context.Background(),
		fixture.request(t, "test"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if after.GoToolchain == nil ||
		len(after.GoToolchain.Modules) != 1 ||
		after.GoToolchain.Modules[0].GoModPath != "go.mod" {
		t.Fatalf("after removal = %+v", after.GoToolchain)
	}
}

func ensureModuleRepository(t *testing.T, root string) {
	t.Helper()

	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		return
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}

	runModuleGit(t, root, "init", "-q")
	gitignore := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		if err := os.WriteFile(
			gitignore,
			[]byte(".local/\n"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
}

func runModuleGit(
	t *testing.T,
	root string,
	arguments ...string,
) {
	t.Helper()

	command := exec.Command("git", arguments...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"git %v failed: %v\n%s",
			arguments,
			err,
			output,
		)
	}
}

func writeModuleFile(
	t *testing.T,
	root string,
	relative string,
	module string,
	goVersion string,
	toolchain string,
) {
	t.Helper()
	ensureModuleRepository(t, root)

	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	content := "module " + module + "\n\n" +
		"go " + goVersion + "\n"
	if toolchain != "" {
		content += "toolchain " + toolchain + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
