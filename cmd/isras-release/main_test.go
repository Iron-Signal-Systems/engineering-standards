package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnsupportedActionIsCensored(t *testing.T) {
	value := "Action" + "Boundary987"
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithIO(
		[]string{"isras-release", sensitiveAssignment("to"+"ken", value)},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, value) {
		t.Fatalf("unsupported action exposed sensitive text: %s", combined)
	}
	if !strings.Contains(combined, "to"+"ken=[REDACTED]") {
		t.Fatalf("redaction marker missing: %s", combined)
	}
}

func TestInvalidFlagValueIsCensored(t *testing.T) {
	value := "Flag" + "Boundary987"
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithIO(
		[]string{"isras-release", "check", "--timeout", sensitiveAssignment("to"+"ken", value)},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, value) {
		t.Fatalf("flag parser exposed sensitive text: %s", combined)
	}
	if !strings.Contains(combined, "[REDACTED]") {
		t.Fatalf("redaction marker missing: %s", combined)
	}
}

func sensitiveAssignment(name, value string) string {
	return name + "=" + value
}
