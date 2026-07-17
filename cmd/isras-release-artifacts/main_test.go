package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUsageRequiresBuildAction(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithIO([]string{"isras-release-artifacts"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage: isras-release-artifacts build") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUnknownActionFailsWithoutProduction(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithIO([]string{"isras-release-artifacts", "publish"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}
