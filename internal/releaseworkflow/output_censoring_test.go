package releaseworkflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
)

func TestReleaseOutputHelperProcess(t *testing.T) {
	if os.Getenv("ISRAS_RELEASE_OUTPUT_HELPER") != "1" {
		return
	}
	value := os.Getenv("ISRAS_RELEASE_OUTPUT_VALUE")
	fmt.Fprintf(os.Stdout, "Authorization: Bearer %s\n", value)
	fmt.Fprintf(os.Stderr, "password=\"%s with spaces\"\n", value)
	if os.Getenv("ISRAS_RELEASE_OUTPUT_SUCCESS") == "1" {
		return
	}
	os.Exit(41)
}

func TestStreamedCommandOutputAndErrorsAreCensored(t *testing.T) {
	value := "Release" + "OutputBoundary987"
	t.Setenv("ISRAS_RELEASE_OUTPUT_HELPER", "1")
	t.Setenv("ISRAS_RELEASE_OUTPUT_VALUE", value)

	engine, stdout, stderr, logOutput := newOutputTestEngine(t)
	output, err := engine.execute(false, true, os.Args[0], "-test.run=^TestReleaseOutputHelperProcess$")
	if err == nil {
		t.Fatal("helper failure was not returned")
	}
	if flushErr := engine.flushWriters(); flushErr != nil {
		t.Fatal(flushErr)
	}

	assertCensored(t, value, output, err.Error(), stdout.String(), stderr.String(), logOutput.String())
	if !strings.Contains(stdout.String()+stderr.String()+logOutput.String(), "[REDACTED]") {
		t.Fatal("streamed output did not contain a redaction marker")
	}
}

func TestCapturedOutputIsCensoredBeforeLogging(t *testing.T) {
	value := "Captured" + "Boundary987"
	t.Setenv("ISRAS_RELEASE_OUTPUT_HELPER", "1")
	t.Setenv("ISRAS_RELEASE_OUTPUT_VALUE", value)
	t.Setenv("ISRAS_RELEASE_OUTPUT_SUCCESS", "1")

	engine, _, _, logOutput := newOutputTestEngine(t)
	output, err := engine.capture(os.Args[0], "-test.run=^TestReleaseOutputHelperProcess$")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, value) {
		t.Fatal("structured capture did not retain internal command output for parsing")
	}
	if flushErr := engine.flushWriters(); flushErr != nil {
		t.Fatal(flushErr)
	}

	assertCensored(t, value, logOutput.String())
	if !strings.Contains(logOutput.String(), "[REDACTED]") {
		t.Fatal("captured command log did not contain a redaction marker")
	}
}

func TestSafeCommandCensorsSensitiveArguments(t *testing.T) {
	value := "Argument" + "Boundary987"
	command := safeCommand("gh", []string{"release", "create", "--token", value})
	assertCensored(t, value, command)
	if !strings.Contains(command, "--token [REDACTED]") {
		t.Fatalf("safe command did not preserve bounded context: %s", command)
	}
}

func TestBoundedBufferTruncatesCapturedOutput(t *testing.T) {
	var buffer boundedBuffer
	payload := bytes.Repeat([]byte("x"), maxCapturedCommandOutput+1024)
	written, err := buffer.Write(payload)
	if err != nil {
		t.Fatal(err)
	}
	if written != len(payload) {
		t.Fatalf("writer reported %d bytes, expected %d", written, len(payload))
	}
	output := buffer.String()
	if len(output) > maxCapturedCommandOutput+128 {
		t.Fatalf("bounded capture exceeded limit: %d", len(output))
	}
	if !strings.Contains(output, "OUTPUT TRUNCATED") {
		t.Fatal("truncation marker missing")
	}
}

func TestBoundedBufferLenMatchesCapturedBytes(t *testing.T) {
	var buffer boundedBuffer
	if _, err := buffer.Write([]byte("captured")); err != nil {
		t.Fatal(err)
	}
	if got, want := buffer.Len(), len("captured"); got != want {
		t.Fatalf("bounded buffer length = %d, want %d", got, want)
	}
}

func TestCensoredErrorPreservesErrorChain(t *testing.T) {
	value := "Error" + "Boundary987"
	underlying := fmt.Errorf("password=%s: %w", value, context.Canceled)
	censored := censorError(underlying)
	assertCensored(t, value, censored.Error())
	if !errors.Is(censored, context.Canceled) {
		t.Fatal("censored error did not preserve the original error chain")
	}
}

func newOutputTestEngine(t *testing.T) (*engine, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var logOutput bytes.Buffer

	logWriter := redact.NewWriter(&logOutput)
	current := &engine{
		ctx:    context.Background(),
		result: Result{RepositoryRoot: t.TempDir()},
		log:    logWriter,
		out:    redact.NewWriter(io.MultiWriter(&stdout, logWriter)),
		errOut: redact.NewWriter(io.MultiWriter(&stderr, logWriter)),
		in:     strings.NewReader(""),
	}
	return current, &stdout, &stderr, &logOutput
}

func assertCensored(t *testing.T, sensitive string, values ...string) {
	t.Helper()
	for index, value := range values {
		if strings.Contains(value, sensitive) {
			t.Fatalf("sensitive value reached output %d: %s", index, value)
		}
	}
}
