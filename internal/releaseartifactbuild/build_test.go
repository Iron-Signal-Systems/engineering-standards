package releaseartifactbuild

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

type fakeRunner struct {
	root           string
	version        string
	goVersion      string
	origin         string
	sources        map[string]string
	modes          map[string]string
	failSourcePath string
}

func newFakeRunner(t *testing.T) *fakeRunner {
	t.Helper()
	root := t.TempDir()
	return &fakeRunner{
		root:      root,
		version:   "0.1.1",
		goVersion: "go1.25.12",
		origin:    "git@github.com:Iron-Signal-Systems/engineering-standards.git",
		sources: map[string]string{
			"VERSION":                                "0.1.1\n",
			"go.mod":                                 "module github.com/Iron-Signal-Systems/engineering-standards\n\ngo 1.25.12\n",
			FrameworkListPath:                        "integration-guides/PROJECT-ADOPTION.md\nrelease/framework-files.txt\nstandards/GO-REFERENCE-PROFILE.md\n",
			ContractsListPath:                        "release/contract-files.txt\nschemas/isras-project-v1.schema.json\nstandards/PROJECT-PIN-SCHEMA.md\n",
			"integration-guides/PROJECT-ADOPTION.md": "project adoption\n",
			"standards/GO-REFERENCE-PROFILE.md":      "go profile\n",
			"schemas/isras-project-v1.schema.json":   "{}\n",
			"standards/PROJECT-PIN-SCHEMA.md":        "pin schema\n",
		},
		modes: map[string]string{},
	}
}

func (runner *fakeRunner) Run(_ context.Context, directory string, _ []string, name string, arguments ...string) (string, string, error) {
	if name == "git" {
		return runner.runGit(directory, arguments...)
	}
	if name == "go" {
		if reflect.DeepEqual(arguments, []string{"env", "GOVERSION"}) {
			return runner.goVersion + "\n", "", nil
		}
		if len(arguments) > 0 && arguments[0] == "build" {
			for index := range arguments {
				if arguments[index] == "-o" && index+1 < len(arguments) {
					if err := os.WriteFile(arguments[index+1], []byte("deterministic-validator-bytes\n"), 0o755); err != nil {
						return "", "", err
					}
					return "", "", nil
				}
			}
			return "", "", errors.New("go build omitted output")
		}
	}
	if filepath.Base(name) == ValidatorName && reflect.DeepEqual(arguments, []string{"version"}) {
		return strings.Join([]string{
			"ISRAS VALIDATOR IDENTITY",
			"Standard version:  " + runner.version,
			"Ownership:         release-artifact",
			"Release tag:       isras-v" + runner.version,
			"Source repository: " + SourceRepository,
			"Source commit:     " + testCommit,
			"Repository commit: " + testCommit,
			"",
		}, "\n"), "", nil
	}
	return "", "", errors.New("unexpected command")
}

func (runner *fakeRunner) runGit(directory string, arguments ...string) (string, string, error) {
	if directory != runner.root {
		return "", "", errors.New("unexpected repository directory")
	}
	joined := strings.Join(arguments, " ")
	switch joined {
	case "rev-parse --show-toplevel":
		return runner.root + "\n", "", nil
	case "status --porcelain=v1 --untracked-files=all":
		return "", "", nil
	case "rev-parse HEAD":
		return testCommit + "\n", "", nil
	case "verify-commit " + testCommit:
		return "", "", nil
	case "show " + testCommit + ":VERSION":
		return runner.version + "\n", "", nil
	case "cat-file -t isras-v" + runner.version:
		return "tag\n", "", nil
	case "verify-tag isras-v" + runner.version:
		return "", "", nil
	case "rev-parse isras-v" + runner.version + "^{commit}":
		return testCommit + "\n", "", nil
	case "remote get-url origin":
		return runner.origin + "\n", "", nil
	case "show " + testCommit + ":go.mod":
		return runner.sources["go.mod"], "", nil
	}
	if len(arguments) == 2 && arguments[0] == "show" && strings.HasPrefix(arguments[1], testCommit+":") {
		path := strings.TrimPrefix(arguments[1], testCommit+":")
		if path == runner.failSourcePath {
			return "", "", errors.New("injected source failure")
		}
		value, ok := runner.sources[path]
		if !ok {
			return "", "", errors.New("missing source")
		}
		return value, "", nil
	}
	if len(arguments) == 3 && arguments[0] == "cat-file" && arguments[1] == "-t" && strings.HasPrefix(arguments[2], testCommit+":") {
		return "blob\n", "", nil
	}
	if len(arguments) == 4 && arguments[0] == "ls-tree" && arguments[1] == testCommit && arguments[2] == "--" {
		path := arguments[3]
		mode := runner.modes[path]
		if mode == "" {
			mode = "100644"
		}
		return mode + " blob deadbeef\t" + path + "\n", "", nil
	}
	return "", "", errors.New("unexpected git command: " + joined)
}

func TestBuildProducesDeterministicExactArtifactSet(t *testing.T) {
	runner := newFakeRunner(t)
	fixedNow := time.Date(2026, 7, 17, 20, 0, 0, 0, time.UTC)
	builder := Builder{Runner: runner, Now: func() time.Time { return fixedNow }}
	baseOptions := Options{
		Root:               runner.root,
		PublishedAt:        "2026-07-17T20:00:00Z",
		ValidationCampaign: "isras-v0.1.1-release-acceptance",
		ReleaseAuthority:   "Iron Signal Systems release authority",
	}
	firstOptions := baseOptions
	firstOptions.OutputDirectory = filepath.Join(runner.root, ".local", "releases", "out-one")
	first, err := builder.Build(context.Background(), firstOptions)
	if err != nil {
		t.Fatal(err)
	}
	secondOptions := baseOptions
	secondOptions.OutputDirectory = filepath.Join(runner.root, ".local", "releases", "out-two")
	second, err := builder.Build(context.Background(), secondOptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Artifacts) != 6 || len(second.Artifacts) != 6 {
		t.Fatalf("artifact counts = %d, %d", len(first.Artifacts), len(second.Artifacts))
	}
	firstDigests := digestMap(first.Artifacts)
	secondDigests := digestMap(second.Artifacts)
	if !reflect.DeepEqual(firstDigests, secondDigests) {
		t.Fatalf("artifact rebuild differs:\nfirst=%v\nsecond=%v", firstDigests, secondDigests)
	}
	entries, err := os.ReadDir(first.OutputDirectory)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	wantNames := []string{ContractsName, FrameworkName, ProvenanceName, SHA256Name, SHA512Name, ValidatorName}
	sort.Strings(wantNames)
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("artifact names = %v", names)
	}
	for _, evidence := range []string{first.EvidenceJSON, first.EvidenceText} {
		info, err := os.Stat(evidence)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("evidence mode = %o", info.Mode().Perm())
		}
	}
}

func TestBuildRejectsDevelopmentVersion(t *testing.T) {
	runner := newFakeRunner(t)
	runner.version = "0.1.1-development"
	builder := Builder{Runner: runner, Now: time.Now}
	_, err := builder.Build(context.Background(), Options{
		Root: runner.root, OutputDirectory: filepath.Join(runner.root, ".local", "releases", "out"),
		PublishedAt: "2026-07-17T20:00:00Z", ValidationCampaign: "campaign", ReleaseAuthority: "authority",
	})
	if err == nil || !strings.Contains(err.Error(), "stable MAJOR.MINOR.PATCH") {
		t.Fatalf("expected stable-version failure, got %v", err)
	}
}

func TestBuildDoesNotOverwriteExistingOutput(t *testing.T) {
	runner := newFakeRunner(t)
	output := filepath.Join(runner.root, ".local", "releases", "existing")
	if err := os.MkdirAll(output, 0o700); err != nil {
		t.Fatal(err)
	}
	builder := Builder{Runner: runner, Now: time.Now}
	_, err := builder.Build(context.Background(), Options{
		Root: runner.root, OutputDirectory: output,
		PublishedAt: "2026-07-17T20:00:00Z", ValidationCampaign: "campaign", ReleaseAuthority: "authority",
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing-output failure, got %v", err)
	}
}

func TestBuildRemovesPartialOutputAfterFailure(t *testing.T) {
	runner := newFakeRunner(t)
	runner.failSourcePath = "schemas/isras-project-v1.schema.json"
	output := filepath.Join(runner.root, ".local", "releases", "failed-output")
	builder := Builder{Runner: runner, Now: time.Now}
	_, err := builder.Build(context.Background(), Options{
		Root: runner.root, OutputDirectory: output,
		PublishedAt: "2026-07-17T20:00:00Z", ValidationCampaign: "campaign", ReleaseAuthority: "authority",
	})
	if err == nil {
		t.Fatal("expected injected build failure")
	}
	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("partial output remains: %v", statErr)
	}
}

func digestMap(records []ArtifactRecord) map[string][2]string {
	out := make(map[string][2]string, len(records))
	for _, record := range records {
		out[record.Name] = [2]string{record.SHA256, record.SHA512}
	}
	return out
}

func TestBuildRejectsOutputOutsideRepositoryLocalReleases(t *testing.T) {
	runner := newFakeRunner(t)
	builder := Builder{Runner: runner, Now: time.Now}
	_, err := builder.Build(context.Background(), Options{
		Root: runner.root, OutputDirectory: filepath.Join(runner.root, "outside"),
		PublishedAt: "2026-07-17T20:00:00Z", ValidationCampaign: "campaign", ReleaseAuthority: "authority",
	})
	if err == nil || !strings.Contains(err.Error(), ".local/releases") {
		t.Fatalf("expected bounded-output failure, got %v", err)
	}
}

func TestBuildRejectsSymlinkedPrivateDirectory(t *testing.T) {
	runner := newFakeRunner(t)
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(runner.root, ".local"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(runner.root, ".local", "releases")); err != nil {
		t.Fatal(err)
	}
	builder := Builder{Runner: runner, Now: time.Now}
	_, err := builder.Build(context.Background(), Options{
		Root:            runner.root,
		OutputDirectory: filepath.Join(runner.root, ".local", "releases", "symlinked"),
		PublishedAt:     "2026-07-17T20:00:00Z", ValidationCampaign: "campaign", ReleaseAuthority: "authority",
	})
	if err == nil || !strings.Contains(err.Error(), "secure release artifact output parent") {
		t.Fatalf("expected symlink-path failure, got %v", err)
	}
}
