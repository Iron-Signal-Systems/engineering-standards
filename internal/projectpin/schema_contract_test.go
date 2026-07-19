package projectpin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONSchemaRequiresFixedRuntimeEvidenceDirectory(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "schemas", "isras-project-v1.schema.json"))
	if err != nil {
		t.Fatalf("read project-pin JSON Schema: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse project-pin JSON Schema: %v", err)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("project-pin JSON Schema has no properties object")
	}
	evidence, ok := properties["evidence"].(map[string]any)
	if !ok {
		t.Fatal("project-pin JSON Schema has no evidence object")
	}
	evidenceProperties, ok := evidence["properties"].(map[string]any)
	if !ok {
		t.Fatal("project-pin JSON Schema evidence has no properties object")
	}
	directory, ok := evidenceProperties["directory"].(map[string]any)
	if !ok {
		t.Fatal("project-pin JSON Schema has no evidence directory rule")
	}
	if got, ok := directory["const"].(string); !ok || got != RuntimeEvidenceDirectory {
		t.Fatalf("project-pin JSON Schema evidence directory const = %#v, want %q", directory["const"], RuntimeEvidenceDirectory)
	}
	if len(directory) != 1 {
		t.Fatalf("project-pin JSON Schema evidence directory has unexpected alternative rules: %#v", directory)
	}
}
