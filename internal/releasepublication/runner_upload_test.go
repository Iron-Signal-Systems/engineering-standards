package releasepublication

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOSRunnerUsesReleaseUploadTransport(t *testing.T) {
	root := t.TempDir()
	assetName := "SHA256SUMS"
	assetPath := filepath.Join(root, assetName)
	if err := os.WriteFile(assetPath, []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "gh.log")
	markerPath := filepath.Join(root, "uploaded")
	ghPath := filepath.Join(root, "gh")
	script := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$GH_TEST_LOG"
if [ "$1" = "api" ] && [ "$2" = "--method" ] && [ "$3" = "GET" ]; then
  if [ -f "$GH_TEST_MARKER" ]; then
    printf '%s\n' '{"id":77,"tag_name":"isras-v0.1.1","draft":true,"prerelease":false,"assets":[{"id":101,"name":"SHA256SUMS","state":"uploaded","size":8,"digest":"sha256:fixture"}]}'
  else
    printf '%s\n' '{"id":77,"tag_name":"isras-v0.1.1","draft":true,"prerelease":false,"assets":[]}'
  fi
  exit 0
fi
if [ "$1" = "release" ] && [ "$2" = "upload" ]; then
  case " $* " in
    *" --clobber "*) exit 91 ;;
  esac
  : > "$GH_TEST_MARKER"
  exit 0
fi
exit 92
`
	if err := os.WriteFile(ghPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+":/usr/bin:/bin")
	environment := append(os.Environ(),
		"GH_TEST_LOG="+logPath,
		"GH_TEST_MARKER="+markerPath,
	)
	result := (OSRunner{}).Run(
		context.Background(),
		root,
		environment,
		"gh",
		"api",
		"--method",
		"POST",
		"-H",
		"Content-Type: application/octet-stream",
		"--input",
		assetPath,
		"repos/Iron-Signal-Systems/engineering-standards/releases/77/assets?name=SHA256SUMS",
	)
	if result.Err != nil {
		t.Fatalf("upload transport failed: %v: %s", result.Err, result.Stderr)
	}
	var asset githubAsset
	if err := json.Unmarshal(result.Stdout, &asset); err != nil {
		t.Fatal(err)
	}
	if asset.ID != 101 || asset.Name != assetName || asset.State != "uploaded" {
		t.Fatalf("unexpected authoritative asset: %#v", asset)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if !strings.Contains(logText, "release upload isras-v0.1.1 "+assetPath+" --repo Iron-Signal-Systems/engineering-standards") {
		t.Fatalf("gh release upload was not used: %s", logText)
	}
	if strings.Contains(logText, "api --method POST") || strings.Contains(logText, "--clobber") {
		t.Fatalf("unsafe upload transport was used: %s", logText)
	}
}

func TestOSRunnerAcceptsAuthoritativeAssetAfterUncertainUploadFailure(t *testing.T) {
	root := t.TempDir()
	assetName := "SHA256SUMS"
	assetPath := filepath.Join(root, assetName)
	if err := os.WriteFile(assetPath, []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	markerPath := filepath.Join(root, "uploaded")
	ghPath := filepath.Join(root, "gh")
	script := `#!/bin/sh
set -eu
if [ "$1" = "api" ] && [ "$2" = "--method" ] && [ "$3" = "GET" ]; then
  if [ -f "$GH_TEST_MARKER" ]; then
    printf '%s\n' '{"id":77,"tag_name":"isras-v0.1.1","draft":true,"prerelease":false,"assets":[{"id":101,"name":"SHA256SUMS","state":"uploaded","size":8,"digest":"sha256:fixture"}]}'
  else
    printf '%s\n' '{"id":77,"tag_name":"isras-v0.1.1","draft":true,"prerelease":false,"assets":[]}'
  fi
  exit 0
fi
if [ "$1" = "release" ] && [ "$2" = "upload" ]; then
  : > "$GH_TEST_MARKER"
  exit 1
fi
exit 92
`
	if err := os.WriteFile(ghPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+":/usr/bin:/bin")
	result := (OSRunner{}).Run(
		context.Background(),
		root,
		append(os.Environ(), "GH_TEST_MARKER="+markerPath),
		"gh",
		"api",
		"--method",
		"POST",
		"-H",
		"Content-Type: application/octet-stream",
		"--input",
		assetPath,
		"repos/Iron-Signal-Systems/engineering-standards/releases/77/assets?name=SHA256SUMS",
	)
	if result.Err != nil {
		t.Fatalf("authoritatively observed upload was rejected: %v", result.Err)
	}
}

func TestParseReleaseAssetUploadCommandRejectsPathNameDrift(t *testing.T) {
	_, matched, err := parseReleaseAssetUploadCommand(
		"gh",
		[]string{
			"api",
			"--method",
			"POST",
			"-H",
			"Content-Type: application/octet-stream",
			"--input",
			"/tmp/not-the-manifest",
			"repos/Iron-Signal-Systems/engineering-standards/releases/77/assets?name=SHA256SUMS",
		},
	)
	if !matched || err == nil {
		t.Fatalf("path/name drift was not rejected: matched=%v err=%v", matched, err)
	}
}
