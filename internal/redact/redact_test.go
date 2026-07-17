package redact

import (
	"bytes"
	"strings"
	"testing"
)

func TestSanitizeCredentialShapes(t *testing.T) {
	privateValue := "Private" + "Boundary987"
	githubValue := "github_pat_" + strings.Repeat("A", 24)
	classicValue := "ghp_" + strings.Repeat("B", 24)
	awsValue := "AKIA" + strings.Repeat("C", 16)
	slackValue := "xoxb-" + strings.Repeat("D", 20)

	input := strings.Join([]string{
		authorizationPrefix() + privateValue,
		sensitiveAssignment("pass"+"word", `"`+privateValue+` with spaces"`),
		"--client-" + "secret=" + privateValue,
		"https://operator:" + privateValue + "@github.com/org/repo.git",
		githubValue,
		classicValue,
		awsValue,
		slackValue,
		privateKeyBegin() + "\n" + privateValue + "\n" + privateKeyEnd(),
	}, "\n")

	output := Sanitize(input)
	for _, sensitive := range []string{
		privateValue,
		githubValue,
		classicValue,
		awsValue,
		slackValue,
		"operator",
	} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("sanitizer left sensitive value visible: %q\n%s", sensitive, output)
		}
	}
	if count := strings.Count(output, "[REDACTED"); count < 9 {
		t.Fatalf("expected every credential shape to be marked, got %d:\n%s", count, output)
	}
}

func TestWriterCensorsValueSplitAcrossWrites(t *testing.T) {
	value := "Split" + "Boundary987"
	var output bytes.Buffer
	writer := NewWriter(&output)

	parts := []string{
		authorizationPrefix() + "Spl",
		"itBound",
		"ary987\nnext line\n",
	}
	for _, part := range parts {
		if _, err := writer.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(output.String(), value) {
		t.Fatalf("split sensitive value reached output: %s", output.String())
	}
	if !strings.Contains(output.String(), authorizationPrefix()+"[REDACTED]") {
		t.Fatalf("redaction marker missing: %s", output.String())
	}
	if !strings.Contains(output.String(), "next line") {
		t.Fatalf("safe output was lost: %s", output.String())
	}
}

func TestWriterCensorsMultilinePrivateKeyBlock(t *testing.T) {
	value := "Private" + "KeyBoundary987"
	var output bytes.Buffer
	writer := NewWriter(&output)

	parts := []string{
		"before\n" + privateKeyBegin() + "\n",
		value + "\nmore-key-material\n",
		privateKeyEnd() + "\nafter\n",
	}
	for _, part := range parts {
		if _, err := writer.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}

	got := output.String()
	if strings.Contains(got, value) || strings.Contains(got, "more-key-material") || strings.Contains(got, "BEGIN OPENSSH") {
		t.Fatalf("private-key material reached output: %s", got)
	}
	if !strings.Contains(got, "[REDACTED PRIVATE KEY]") {
		t.Fatalf("private-key marker missing: %s", got)
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Fatalf("safe surrounding output was lost: %s", got)
	}
}

func TestWriterDropsOversizedUnterminatedLine(t *testing.T) {
	value := "Oversized" + "Boundary987"
	var output bytes.Buffer
	writer := NewWriter(&output)

	line := strings.Repeat("x", maxPendingLine) + value
	if _, err := writer.Write([]byte(line)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(output.String(), value) || strings.Contains(output.String(), strings.Repeat("x", 128)) {
		t.Fatalf("oversized line content reached output: %q", output.String())
	}
	if !strings.Contains(output.String(), "exceeded safe censoring limit") {
		t.Fatalf("oversized-line marker missing: %q", output.String())
	}
}

func TestSanitizePreservesNonSensitiveReleaseEvidence(t *testing.T) {
	input := "tag=isras-v0.1.1 commit=0123456789abcdef path=docs/releases/0.1.1.md"
	if output := Sanitize(input); output != input {
		t.Fatalf("non-sensitive release evidence changed: %q", output)
	}
}

func authorizationPrefix() string {
	return "Authorization: " + "Bear" + "er "
}

func privateKeyBegin() string {
	return "-----BEGIN OPENSSH " + "PRIVATE KEY-----"
}

func privateKeyEnd() string {
	return "-----END OPENSSH " + "PRIVATE KEY-----"
}

func sensitiveAssignment(name, value string) string {
	return name + "=" + value
}
