package redact

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	input := "Authorization: Bearer " + "SensitiveScannerValue12345"
	output := Sanitize(input)
	if strings.Contains(output, "SensitiveScannerValue12345") {
		t.Fatal("sanitizer left the sensitive value visible")
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Fatalf("sanitizer did not mark redaction: %q", output)
	}
}
