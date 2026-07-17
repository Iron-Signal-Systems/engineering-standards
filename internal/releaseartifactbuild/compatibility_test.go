package releaseartifactbuild

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifact"
)

func TestProducedArtifactsPassLocalVerifierWithoutExecutionAuthorization(t *testing.T) {
	runner := newFakeRunner(t)
	builder := Builder{
		Runner: runner,
		Now: func() time.Time {
			return time.Date(2026, 7, 17, 20, 0, 0, 0, time.UTC)
		},
	}
	result, err := builder.Build(context.Background(), Options{
		Root:               runner.root,
		OutputDirectory:    filepath.Join(runner.root, ".local", "releases", "compatibility"),
		PublishedAt:        "2026-07-17T20:00:00Z",
		ValidationCampaign: "isras-v0.1.1-release-acceptance",
		ReleaseAuthority:   "Iron Signal Systems release authority",
	})
	if err != nil {
		t.Fatal(err)
	}

	artifacts := make([]projectpin.Artifact, 0, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts = append(artifacts, projectpin.Artifact{
			Kind: artifact.Kind, OS: artifact.OS, Arch: artifact.Arch,
			Name: artifact.Name, SHA256: artifact.SHA256, SHA512: artifact.SHA512,
		})
	}
	pin := projectpin.Pin{
		SchemaVersion: projectpin.SchemaVersion,
		Project: projectpin.Project{
			Repository: "github.com/Iron-Signal-Systems/compatibility-fixture",
		},
		Standard: projectpin.Standard{
			Profile: projectpin.Profile, Version: result.Version,
			ReleaseTag: result.ReleaseTag, SourceRepository: projectpin.SourceRepository,
			SourceCommit: result.SourceCommit,
		},
		Artifacts: artifacts,
		Workflow: projectpin.Workflow{
			Repository: projectpin.SourceRepository,
			Path:       projectpin.ReusableWorkflowPath,
			Commit:     result.SourceCommit,
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
		Evidence: projectpin.Evidence{Directory: ".local/validation"},
	}
	if _, err := projectpin.Parse(mustPinJSON(t, pin)); err != nil {
		t.Fatalf("produced pin fixture is invalid: %v", err)
	}
	report, err := releaseartifact.VerifyDirectory(pin, result.OutputDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if report.AssetInventory != releaseartifact.StatusPass ||
		report.PinDigests != releaseartifact.StatusPass ||
		report.SHA256Manifest != releaseartifact.StatusPass ||
		report.SHA512Manifest != releaseartifact.StatusPass ||
		report.Provenance != releaseartifact.StatusPass {
		t.Fatalf("verification report did not pass: %#v", report)
	}
	if report.ExecutionAuthorization != releaseartifact.AuthorizationDenied {
		t.Fatalf("local verification authorized execution: %s", report.ExecutionAuthorization)
	}
}

func mustPinJSON(t *testing.T, pin projectpin.Pin) []byte {
	t.Helper()
	data, err := json.Marshal(pin)
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}
