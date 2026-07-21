package projectcommand

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunGovulncheckModulesUsesEveryModuleAndSelectedGo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX shell scripts")
	}
	root := newGovulncheckRunnerRepository(t, []goModuleSelection{
		{GoModPath: "z/go.mod", Directory: "z", ModulePath: "example.com/z"},
		{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"},
		{GoModPath: "a/go.mod", Directory: "a", ModulePath: "example.com/a"},
	})
	outside := t.TempDir()
	logPath := filepath.Join(outside, "scanner.log")
	selectedGo := writeRunnerFakeGo(t, filepath.Join(outside, "go-bin"))
	tool := writeRunnerFakeScanner(t, filepath.Join(outside, "tool-bin"), fmt.Sprintf(`
printf '%s|%s|%s|%s|%s|%s\n' "$PWD" "$1" "$2" "$GOTOOLCHAIN" "$GOENV" "$(command -v go)" >> %q
printf '%%s' '{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck","scanner_version":"v1.6.0","go_version":"go1.26.5-X:nodwarf5","scan_level":"symbol","scan_mode":"source"}}'
printf '%%s' '{"SBOM":{"roots":["example.com/root/..."],"modules":[{"path":"example.com/root","version":""}]}}'
`, "%s", "%s", "%s", "%s", "%s", "%s", logPath))

	hostile := filepath.Join(outside, "hostile")
	if err := os.MkdirAll(hostile, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", hostile+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GOTOOLCHAIN", "auto")
	t.Setenv("GOENV", filepath.Join(outside, "caller-goenv"))

	selection := goToolchainSelection{
		Executable: selectedGo,
		Directory:  filepath.Dir(selectedGo),
		Actual:     "go1.26.5-X:nodwarf5",
		Modules: []goModuleSelection{
			{GoModPath: "z/go.mod", Directory: "z", ModulePath: "example.com/z"},
			{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"},
			{GoModPath: "a/go.mod", Directory: "a", ModulePath: "example.com/a"},
		},
	}
	identity := runnerToolIdentity(tool)

	run, err := runGovulncheckModules(context.Background(), root, selection, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Modules) != 3 {
		t.Fatalf("module count = %d", len(run.Modules))
	}
	wantOrder := []string{"a/go.mod", "go.mod", "z/go.mod"}
	for index, want := range wantOrder {
		if run.Modules[index].GoModPath != want {
			t.Fatalf("module %d = %q", index, run.Modules[index].GoModPath)
		}
		if run.Modules[index].PackageScope != "./..." {
			t.Fatalf("scope = %q", run.Modules[index].PackageScope)
		}
		if run.Modules[index].ExitCode != 0 || run.Modules[index].Protocol.ConfigMessages != 1 {
			t.Fatalf("result = %#v", run.Modules[index])
		}
		if run.Modules[index].Protocol.Config.GoVersion != selection.Actual {
			t.Fatalf("Go version = %q", run.Modules[index].Protocol.Config.GoVersion)
		}
	}

	lines := strings.Split(strings.TrimSpace(readRunnerFile(t, logPath)), "\n")
	if len(lines) != 3 {
		t.Fatalf("log lines = %d: %q", len(lines), lines)
	}
	for _, line := range lines {
		fields := strings.Split(line, "|")
		if len(fields) != 6 {
			t.Fatalf("log fields = %#v", fields)
		}
		if fields[1] != "-json" || fields[2] != "./..." {
			t.Fatalf("arguments = %#v", fields[1:3])
		}
		if fields[3] != "local" || fields[4] != "off" {
			t.Fatalf("Go environment = %#v", fields[3:5])
		}
		if fields[5] != selectedGo {
			t.Fatalf("selected go = %q, want %q", fields[5], selectedGo)
		}
		if strings.Contains(fields[5], hostile) {
			t.Fatalf("hostile caller PATH reached scanner: %q", line)
		}
	}
}

func TestRunGovulncheckModulesRejectsInvocationFailures(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX shell scripts")
	}
	tests := []struct {
		name      string
		body      string
		configure func(t *testing.T)
		want      string
	}{
		{name: "nonzero", body: "exit 7\n", want: "exit code 7"},
		{name: "malformed protocol", body: "printf '%s' '{\"config\":'\n", want: "parse govulncheck"},
		{name: "repository mutation", body: "touch MUTATED\nprintf '%s' '{\"config\":{\"protocol_version\":\"v1.0.0\"}}'\n", want: "changed Git-visible"},
		{name: "output limit", body: "printf '%s' '{\"config\":{\"protocol_version\":\"v1.0.0\"}}'\nprintf '%02048d' 0\n", configure: func(t *testing.T) {
			old := govulncheckModuleOutputLimit
			govulncheckModuleOutputLimit = 128
			t.Cleanup(func() { govulncheckModuleOutputLimit = old })
		}, want: "output exceeded"},
		{name: "timeout", body: "sleep 5\n", configure: func(t *testing.T) {
			old := govulncheckModuleTimeout
			govulncheckModuleTimeout = 100 * time.Millisecond
			t.Cleanup(func() { govulncheckModuleTimeout = old })
		}, want: "configured timeout"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newGovulncheckRunnerRepository(t, []goModuleSelection{{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"}})
			outside := t.TempDir()
			selectedGo := writeRunnerFakeGo(t, filepath.Join(outside, "go-bin"))
			tool := writeRunnerFakeScanner(t, filepath.Join(outside, "tool-bin"), test.body)
			if test.configure != nil {
				test.configure(t)
			}
			selection := goToolchainSelection{Executable: selectedGo, Directory: filepath.Dir(selectedGo), Actual: "go1.26.5-X:nodwarf5", Modules: []goModuleSelection{{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"}}}
			_, err := runGovulncheckModules(context.Background(), root, selection, runnerToolIdentity(tool))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRunGovulncheckModulesRejectsUnsafeInventory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX shell scripts")
	}
	root := newGovulncheckRunnerRepository(t, []goModuleSelection{{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"}})
	outside := t.TempDir()
	selectedGo := writeRunnerFakeGo(t, filepath.Join(outside, "go-bin"))
	tool := writeRunnerFakeScanner(t, filepath.Join(outside, "tool-bin"), "exit 0\n")
	tests := []struct {
		name    string
		modules []goModuleSelection
		want    string
	}{
		{name: "duplicate", modules: []goModuleSelection{{GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"}, {GoModPath: "go.mod", Directory: ".", ModulePath: "example.com/root"}}, want: "duplicate"},
		{name: "escape", modules: []goModuleSelection{{GoModPath: "../go.mod", Directory: "..", ModulePath: "example.com/root"}}, want: "unsafe"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selection := goToolchainSelection{Executable: selectedGo, Directory: filepath.Dir(selectedGo), Actual: "go1.26.5-X:nodwarf5", Modules: test.modules}
			_, err := runGovulncheckModules(context.Background(), root, selection, runnerToolIdentity(tool))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRunGovulncheckModulesLiveCandidate(t *testing.T) {
	if os.Getenv("ISRAS_RUN_LIVE_GOVULNCHECK") != "1" {
		t.Skip("live candidate scan is disabled")
	}
	root := os.Getenv("ISRAS_LIVE_REPO")
	toolPath := os.Getenv("ISRAS_LIVE_GOVULNCHECK")
	if root == "" || toolPath == "" {
		t.Fatal("live repository and tool paths are required")
	}
	selection, err := selectGoToolchain(root)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := verifyGovulncheckTool(context.Background(), selection.Executable, toolPath, filepath.Join(root, "validation", "tool-versions.json"))
	if err != nil {
		t.Fatal(err)
	}
	run, err := runGovulncheckModules(context.Background(), root, selection, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Modules) != len(selection.Modules) {
		t.Fatalf("scanned modules = %d, inventory = %d", len(run.Modules), len(selection.Modules))
	}
	for _, module := range run.Modules {
		if module.Protocol.UnknownLevelFindings != 0 {
			t.Fatalf("module %s unknown findings = %d", module.GoModPath, module.Protocol.UnknownLevelFindings)
		}
		t.Logf("module=%s messages=%d osv=%d findings=%d symbol=%d package=%d module_level=%d", module.GoModPath, module.Protocol.MessageCount, len(module.Protocol.OSVAdvisoryIDs), module.Protocol.FindingMessages, module.Protocol.SymbolLevelFindings, module.Protocol.PackageLevelFindings, module.Protocol.ModuleLevelFindings)
	}
}

func runnerToolIdentity(path string) govulncheckToolIdentity {
	digest := sha256.Sum256([]byte(path))
	return govulncheckToolIdentity{Executable: path, Directory: filepath.Dir(path), CommandPackage: govulncheckCommandPackage, Module: govulncheckModuleRoot, Version: "v1.6.0", BuildGoVersion: "go1.26.5-X:nodwarf5", SHA256: hex.EncodeToString(digest[:])}
}

func newGovulncheckRunnerRepository(t *testing.T, modules []goModuleSelection) string {
	t.Helper()
	root := t.TempDir()
	runRunnerGit(t, root, "init", "-q")
	runRunnerGit(t, root, "config", "user.name", "ISRAS Test")
	runRunnerGit(t, root, "config", "user.email", "isras-test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, module := range modules {
		path := filepath.Join(root, filepath.FromSlash(module.GoModPath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "module " + module.ModulePath + "\n\ngo 1.25.0\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runRunnerGit(t, root, "add", "-A")
	runRunnerGit(t, root, "commit", "-q", "-m", "fixture")
	return root
}

func writeRunnerFakeGo(t *testing.T, directory string) string {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "go")
	body := "#!/bin/sh\nif [ \"$1\" = env ] && [ \"$2\" = GOVERSION ]; then printf '%s\\n' 'go1.26.5-X:nodwarf5'; exit 0; fi\nexit 0\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeRunnerFakeScanner(t *testing.T, directory, body string) string {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "govulncheck")
	content := "#!/bin/sh\nset -eu\n" + body
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func runRunnerGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgSign=false"}, args...)...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
func readRunnerFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
