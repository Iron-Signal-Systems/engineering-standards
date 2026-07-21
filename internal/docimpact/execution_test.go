package docimpact

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesPassingEvidence(t *testing.T) {
	root := initializeGitRepository(t)
	writePolicy(
		t,
		filepath.Join(
			root,
			filepath.FromSlash(PolicyRelativePath),
		),
		testPolicy(),
	)
	writeText(t, filepath.Join(root, "README.md"), "base\n")
	base := commitAll(t, root, "base")

	writeText(
		t,
		filepath.Join(root, "internal", "example", "example.go"),
		"package example\n",
	)
	writeText(t, filepath.Join(root, "CHANGELOG.md"), "changed\n")
	writeText(
		t,
		filepath.Join(root, "standards", "EXAMPLE.md"),
		"# Example\n",
	)
	writeText(
		t,
		filepath.Join(root, "docs", "records", "EXAMPLE.md"),
		"# Record\n",
	)
	head := commitAll(t, root, "head")

	result, err := Run(
		context.Background(),
		Request{
			Root:       root,
			BaseCommit: base,
			HeadCommit: head,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Evidence.Status != "PASS" ||
		result.Evidence.Report.Status != "PASS" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Evidence.Policy.SHA256) != 64 {
		t.Fatalf("policy digest = %q", result.Evidence.Policy.SHA256)
	}
	assertExecutionEvidenceFiles(t, result)
}

func TestRunWritesFailingEvidence(t *testing.T) {
	root := initializeGitRepository(t)
	writePolicy(
		t,
		filepath.Join(
			root,
			filepath.FromSlash(PolicyRelativePath),
		),
		testPolicy(),
	)
	writeText(t, filepath.Join(root, "README.md"), "base\n")
	base := commitAll(t, root, "base")

	writeText(
		t,
		filepath.Join(root, "internal", "example", "example.go"),
		"package example\n",
	)
	head := commitAll(t, root, "head")

	result, err := Run(
		context.Background(),
		Request{
			Root:       root,
			BaseCommit: base,
			HeadCommit: head,
		},
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"requirements are not satisfied",
	) {
		t.Fatalf("result=%#v error=%v", result, err)
	}
	if result.Evidence.Status != "FAIL" ||
		result.Evidence.Report.Status != "FAIL" {
		t.Fatalf("result = %#v", result)
	}
	assertExecutionEvidenceFiles(t, result)
}

func TestRunWritesPolicyFailureEvidence(t *testing.T) {
	root := initializeGitRepository(t)
	writeText(
		t,
		filepath.Join(
			root,
			filepath.FromSlash(PolicyRelativePath),
		),
		`{"schema_version":2,"rules":[]}`,
	)
	writeText(t, filepath.Join(root, "README.md"), "base\n")
	base := commitAll(t, root, "base")
	writeText(t, filepath.Join(root, "README.md"), "head\n")
	head := commitAll(t, root, "head")

	result, err := Run(
		context.Background(),
		Request{
			Root:       root,
			BaseCommit: base,
			HeadCommit: head,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("result=%#v error=%v", result, err)
	}
	if result.Evidence.Status != "FAIL" ||
		!strings.Contains(result.Evidence.Failure, "schema_version") {
		t.Fatalf("failure evidence = %#v", result.Evidence)
	}
	assertExecutionEvidenceFiles(t, result)
}

func TestRunRejectsSymlinkedEvidenceDirectory(t *testing.T) {
	root := initializeGitRepository(t)
	writePolicy(
		t,
		filepath.Join(
			root,
			filepath.FromSlash(PolicyRelativePath),
		),
		testPolicy(),
	)
	writeText(t, filepath.Join(root, "README.md"), "base\n")
	commit := commitAll(t, root, "base")

	external := t.TempDir()
	local := filepath.Join(root, ".local")
	if err := os.Symlink(external, local); err != nil {
		t.Fatal(err)
	}
	_, err := Run(
		context.Background(),
		Request{
			Root:       root,
			BaseCommit: commit,
			HeadCommit: commit,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v", err)
	}
}

func assertExecutionEvidenceFiles(t *testing.T, result Result) {
	t.Helper()
	for _, path := range []string{
		result.EvidenceJSON,
		result.EvidenceText,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o", path, info.Mode().Perm())
		}
	}
	data, err := os.ReadFile(result.EvidenceJSON)
	if err != nil {
		t.Fatal(err)
	}
	var evidence ExecutionEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.SchemaVersion != ExecutionSchemaVersion {
		t.Fatalf("schema version = %d", evidence.SchemaVersion)
	}
	text, err := os.ReadFile(result.EvidenceText)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(
		string(text),
		"DOCUMENTATION IMPACT EVIDENCE",
	) {
		t.Fatalf("text evidence = %s", text)
	}
}
