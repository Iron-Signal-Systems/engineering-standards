package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnsupportedPublicationActionIsRejected(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithIO([]string{"isras-release-publication", "unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported publication action") {
		t.Fatalf("missing action error: %s", stderr.String())
	}
}

func TestInvalidTimeoutIsCensored(t *testing.T) {
	secret := "to" + "ken=PublicationBoundary987"
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithIO([]string{"isras-release-publication", "check", "--timeout", secret}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "PublicationBoundary987") {
		t.Fatalf("sensitive value was exposed: %s", combined)
	}
	if !strings.Contains(combined, "[REDACTED]") {
		t.Fatalf("redaction marker missing: %s", combined)
	}
}

func TestDisplayPathUsesRepositoryRelativePath(t *testing.T) {
	got := displayPath("/src/repository", "/src/repository/.local/evidence.json")
	if got != ".local/evidence.json" {
		t.Fatalf("displayPath = %q", got)
	}
}
