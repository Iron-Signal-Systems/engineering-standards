package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectcommand"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

func TestReleaseValidatorExecutesCommittedProjectCommandAgainstExplicitTarget(t *testing.T) {
	binary := buildEmbeddedValidator(t)
	target := createPinnedTarget(t, "github.com/Iron-Signal-Systems/command-target")
	runTargetGit(t, target, "remote", "add", "origin", "git@github.com:Iron-Signal-Systems/command-target.git")

	if err := os.WriteFile(filepath.Join(target, ".gitignore"), []byte(".local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(target, "tools", "project-command.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf 'argument=%s\\n' \"$1\"\nprintf 'outside=%s\\n' \"$PWD\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	pinPath := filepath.Join(target, projectpin.MetadataPath)
	data, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := projectpin.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	pin.Commands["test"] = []string{"./tools/project-command.sh", "$HOME;not-a-shell"}
	data, err = json.MarshalIndent(pin, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(pinPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	runTargetGit(t, target, "add", ".gitignore", ".isras/project.json", "tools/project-command.sh")
	runTargetGit(t, target, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "declare project command")

	outside := t.TempDir()
	output := runValidator(t, binary, outside, "--repo", target, "project-command", "run", "test")
	for _, expected := range []string{
		"Authorization:          GRANTED",
		"Command:                test",
		"Status:                 PASS",
		"Repository state drift: false",
		"JSON evidence:",
		"Text evidence:",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("command output missing %q:\n%s", expected, output)
		}
	}

	matches, err := filepath.Glob(filepath.Join(target, ".local", "isras", "project-commands", "*", "execution.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("execution evidence matches=%v err=%v", matches, err)
	}
	var result projectcommand.Result
	evidence, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(evidence, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "PASS" || result.Target.Root != target {
		t.Fatalf("unexpected evidence: %+v", result)
	}
	if !strings.Contains(result.Stdout.Sanitized, "argument=$HOME;not-a-shell") {
		t.Fatalf("exact argument was not preserved: %q", result.Stdout.Sanitized)
	}
	if !strings.Contains(result.Stdout.Sanitized, "outside="+target) {
		t.Fatalf("command did not run at target root: %q", result.Stdout.Sanitized)
	}
}
