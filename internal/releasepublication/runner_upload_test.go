package releasepublication

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestOSRunnerUsesReleaseUploadTransportAndWaitsForAuthoritativeState(t *testing.T) {
	root := t.TempDir()
	assetName := "SHA256SUMS"
	assetPath := filepath.Join(root, assetName)
	if err := os.WriteFile(assetPath, []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "gh.log")
	markerPath := filepath.Join(root, "uploaded")
	countPath := filepath.Join(root, "asset-read-count")
	ghPath := filepath.Join(root, "gh")
	script := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$GH_TEST_LOG"
if [ "$1" = "api" ] && [ "$2" = "--method" ] && [ "$3" = "GET" ]; then
  case "$4" in
    */releases/77)
      printf '%s\n' '{"id":77,"tag_name":"isras-v0.1.1","draft":true,"prerelease":false}'
      ;;
    */releases/77/assets\?per_page=100)
      if [ ! -f "$GH_TEST_MARKER" ]; then
        printf '%s\n' '[]'
        exit 0
      fi
      count=0
      if [ -f "$GH_TEST_COUNT" ]; then
        count=$(cat "$GH_TEST_COUNT")
      fi
      count=$((count + 1))
      printf '%s' "$count" > "$GH_TEST_COUNT"
      if [ "$count" -lt 3 ]; then
        printf '%s\n' '[]'
      else
        printf '%s\n' '[{"id":101,"name":"SHA256SUMS","state":"uploaded","size":8,"digest":"sha256:fixture"}]'
      fi
      ;;
    *)
      exit 93
      ;;
  esac
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
		"GH_TEST_COUNT="+countPath,
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
	countData, err := os.ReadFile(countPath)
	if err != nil {
		t.Fatal(err)
	}
	count, err := strconv.Atoi(string(countData))
	if err != nil || count != 3 {
		t.Fatalf("expected three post-upload asset observations, got %q", countData)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if strings.Count(logText, "release upload isras-v0.1.1 "+assetPath+" --repo Iron-Signal-Systems/engineering-standards") != 1 {
		t.Fatalf("upload command was not issued exactly once: %s", logText)
	}
	if !strings.Contains(logText, "/releases/77/assets?per_page=100") {
		t.Fatalf("dedicated release-assets endpoint was not used: %s", logText)
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
	logPath := filepath.Join(root, "gh.log")
	ghPath := filepath.Join(root, "gh")
	script := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$GH_TEST_LOG"
if [ "$1" = "api" ] && [ "$2" = "--method" ] && [ "$3" = "GET" ]; then
  case "$4" in
    */releases/77)
      printf '%s\n' '{"id":77,"tag_name":"isras-v0.1.1","draft":true,"prerelease":false}'
      ;;
    */releases/77/assets\?per_page=100)
      if [ -f "$GH_TEST_MARKER" ]; then
        printf '%s\n' '[{"id":101,"name":"SHA256SUMS","state":"uploaded","size":8,"digest":"sha256:fixture"}]'
      else
        printf '%s\n' '[]'
      fi
      ;;
    *)
      exit 93
      ;;
  esac
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
		append(os.Environ(),
			"GH_TEST_LOG="+logPath,
			"GH_TEST_MARKER="+markerPath,
		),
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
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(logData), "release upload ") != 1 {
		t.Fatalf("uncertain upload was retried: %s", logData)
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
