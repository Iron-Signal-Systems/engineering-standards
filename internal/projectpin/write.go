package projectpin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// CanonicalJSON returns the deterministic, newline-terminated representation
// used when committing a project pin. The resulting bytes are parsed again so
// generation can never bypass the normal project-pin validation boundary.
func CanonicalJSON(pin Pin) ([]byte, error) {
	data, err := json.MarshalIndent(pin, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode project pin: %w", err)
	}
	data = append(data, '\n')
	parsed, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("validate generated project pin: %w", err)
	}
	normalized, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("normalize project pin: %w", err)
	}
	normalized = append(normalized, '\n')
	if !bytes.Equal(data, normalized) {
		return nil, errors.New("generated project pin is not canonical")
	}
	return data, nil
}

// DefaultGoCommands returns the explicit first-adoption command set for the Go
// reference profile. Projects remain free to replace these arrays through a
// reviewed pin change after initialization.
const DefaultGoFormatCheckPath = "./.isras/check-go-format"

func DefaultGoCommands() map[string][]string {
	return map[string][]string{
		"build":                 {"go", "build", "./..."},
		"format_check":          {DefaultGoFormatCheckPath},
		"known_vulnerabilities": {"go", "run", "golang.org/x/vuln/cmd/govulncheck@v1.6.0", "./..."},
		"module_consistency":    {"go", "mod", "tidy", "-diff"},
		"module_integrity":      {"go", "mod", "verify"},
		"static_analysis":       {"go", "vet", "./..."},
		"test":                  {"go", "test", "./..."},
	}
}
