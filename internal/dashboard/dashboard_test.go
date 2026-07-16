package dashboard

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
)

func TestNonTerminalOutputHasNoANSIAndIncludesActions(t *testing.T) {
	var out bytes.Buffer
	printer := New(&out)
	printer.Header("test")
	printer.Checks([]model.Check{{
		Section: "GO SOURCE", Name: "Canonical formatting", Status: model.Fail,
		Detail: "one file", LogPath: ".local/validation/logs/example.log",
		Actions: []model.Action{{Label: "READ ONLY", Description: "Review:", Command: "gofmt -d file.go"}},
	}})
	printer.Footer(model.Summary{Checks: []model.Check{{Status: model.Fail}}})
	value := out.String()
	if strings.Contains(value, "\x1b[") {
		t.Fatal("non-terminal output contained ANSI escapes")
	}
	for _, expected := range []string{"AVAILABLE ACTIONS", "READ ONLY", "gofmt -d file.go", "Failure log", "Not ready."} {
		if !strings.Contains(value, expected) {
			t.Fatalf("output missing %q\n%s", expected, value)
		}
	}
}
