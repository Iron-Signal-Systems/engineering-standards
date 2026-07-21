package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

const externalTargetSourceCommit = "89abcdef0123456789abcdef0123456789abcdef"

func TestParseGlobalOptionsExtractsExplicitTarget(t *testing.T) {
	args, options, err := parseGlobalOptions([]string{
		"project-pin", "--repo", "/src/example", "validate", "--mode=commit",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.Repository != "/src/example" || options.Mode != "commit" {
		t.Fatalf("options = %#v", options)
	}
	if strings.Join(args, " ") != "project-pin validate" {
		t.Fatalf("args = %#v", args)
	}
}

func TestParseGlobalOptionsRejectsDuplicateOrInvalidValues(t *testing.T) {
	for _, args := range [][]string{
		{"--repo", "a", "--repo", "b", "all"},
		{"--repo=", "all"},
		{"--mode", "commit", "--mode", "release", "all"},
		{"--mode=unknown", "all"},
	} {
		if _, _, err := parseGlobalOptions(args); err == nil {
			t.Fatalf("invalid arguments accepted: %#v", args)
		}
	}
}

func TestReleaseValidatorWorksOutsideTargetRepository(t *testing.T) {
	binary := buildEmbeddedValidator(t)
	outside := t.TempDir()
	first := createPinnedTarget(t, "github.com/Iron-Signal-Systems/fixture-one")
	second := createPinnedTarget(t, "github.com/Iron-Signal-Systems/fixture-two")

	version := runValidator(t, binary, outside, "version")
	for _, expected := range []string{
		"Standard version:  0.1.1",
		"Ownership:         release-artifact",
		"Release tag:       isras-v0.1.1",
		"Source commit:     " + externalTargetSourceCommit,
		"Repository commit: " + externalTargetSourceCommit,
	} {
		if !strings.Contains(version, expected) {
			t.Fatalf("version output missing %q:\n%s", expected, version)
		}
	}

	help := runValidator(t, binary, outside, "help")
	if !strings.Contains(help, "--repo PATH") {
		t.Fatalf("help did not declare external target option:\n%s", help)
	}

	firstOutput := runValidator(t, binary, outside, "--repo", first, "project-pin", "validate")
	if !strings.Contains(firstOutput, "fixture-one") || strings.Contains(firstOutput, "fixture-two") {
		t.Fatalf("first target output was contaminated:\n%s", firstOutput)
	}
	secondOutput := runValidator(t, binary, outside, "project-pin", "--repo", second, "validate")
	if !strings.Contains(secondOutput, "fixture-two") || strings.Contains(secondOutput, "fixture-one") {
		t.Fatalf("second target output was contaminated:\n%s", secondOutput)
	}
}

func buildEmbeddedValidator(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	binary := filepath.Join(t.TempDir(), "isras-validator-linux-amd64")
	linkerFlags := strings.Join([]string{
		"-s",
		"-w",
		"-buildid=",
		"-X=github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity.releaseVersion=0.1.1",
		"-X=github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity.releaseTag=isras-v0.1.1",
		"-X=github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity.releaseSourceCommit=" + externalTargetSourceCommit,
	}, " ")
	command := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags", linkerFlags, "-o", binary, "./cmd/isras-validate")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build embedded validator: %v\n%s", err, output)
	}
	return binary
}

func createPinnedTarget(t *testing.T, repositoryName string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".isras"), 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := "module " + repositoryName + "\n\ngo 1.25.12\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	pin := projectpin.Pin{
		SchemaVersion: projectpin.SchemaVersion,
		Project:       projectpin.Project{Repository: repositoryName},
		Standard: projectpin.Standard{
			Profile:          projectpin.Profile,
			Version:          "0.1.1",
			ReleaseTag:       "isras-v0.1.1",
			SourceRepository: projectpin.SourceRepository,
			SourceCommit:     externalTargetSourceCommit,
		},
		Artifacts: []projectpin.Artifact{
			{Kind: "validator", OS: "linux", Arch: "amd64", Name: "isras-validator-linux-amd64", SHA256: strings.Repeat("1", 64), SHA512: strings.Repeat("1", 128)},
			{Kind: "framework", Name: "isras-project-framework.tar.gz", SHA256: strings.Repeat("2", 64), SHA512: strings.Repeat("2", 128)},
			{Kind: "contracts", Name: "isras-contracts.tar.gz", SHA256: strings.Repeat("3", 64), SHA512: strings.Repeat("3", 128)},
			{Kind: "provenance", Name: "provenance.json", SHA256: strings.Repeat("4", 64), SHA512: strings.Repeat("4", 128)},
			{Kind: "sha256-manifest", Name: "SHA256SUMS", SHA256: strings.Repeat("5", 64), SHA512: strings.Repeat("5", 128)},
			{Kind: "sha512-manifest", Name: "SHA512SUMS", SHA256: strings.Repeat("6", 64), SHA512: strings.Repeat("6", 128)},
		},
		Workflow: projectpin.Workflow{
			Repository: projectpin.SourceRepository,
			Path:       projectpin.ReusableWorkflowPath,
			Commit:     externalTargetSourceCommit,
		},
		Profiles: []string{"go"},
		Commands: map[string][]string{
			"format_check":          {"gofmt", "-d", "."},
			"static_analysis":       {"go", "vet", "./..."},
			"test":                  {"go", "test", "./..."},
			"build":                 {"go", "build", "./..."},
			"module_consistency":    {"go", "mod", "tidy", "-diff"},
			"module_integrity":      {"go", "mod", "verify"},
			"known_vulnerabilities": {"govulncheck", "./..."},
		},
		Evidence: projectpin.Evidence{Directory: projectpin.RuntimeEvidenceDirectory},
	}
	data, err := json.MarshalIndent(pin, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if _, err := projectpin.Parse(data); err != nil {
		t.Fatalf("fixture pin is invalid: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".isras", "project.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	runTargetGit(t, root, "init", "-q")
	runTargetGit(t, root, "config", "user.name", "ISRAS Target Fixture")
	runTargetGit(t, root, "config", "user.email", "isras-target@example.invalid")
	runTargetGit(t, root, "add", ".isras/project.json", "go.mod")
	runTargetGit(t, root, "-c", "commit.gpgsign=false", "commit", "-q", "-m", repositoryName)
	return root
}

func runValidator(t *testing.T, binary, directory string, arguments ...string) string {
	t.Helper()
	command := exec.Command(binary, arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("validator %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func runTargetGit(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgSign=false"}, arguments...)...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
}
