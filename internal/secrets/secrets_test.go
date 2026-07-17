package secrets

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSensitiveAssignmentDetectedWithoutValueExposure(t *testing.T) {
	data := []byte("pass" + "word=ThisIsOnlyScannerData123\n")
	findings := scanFile("docs/example.txt", data)
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	finding := findings[0]
	if finding.Rule != "sensitive-assignment" {
		t.Fatalf("unexpected rule %q", finding.Rule)
	}
	if strings.Contains(finding.ID, "ThisIsOnlyScannerData123") || strings.Contains(finding.Fingerprint, "ThisIsOnlyScannerData123") {
		t.Fatal("finding metadata exposed the detected value")
	}
	if !finding.Redactable || !finding.Allowable {
		t.Fatal("generic assignment should support bounded redaction and allowlist review")
	}
}

func TestOrdinarySourceCannotBeAllowlisted(t *testing.T) {
	data := []byte("pass" + "word=ThisMustBeCorrected123\n")
	findings := scanFile("internal/config/config.go", data)
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	if findings[0].Allowable {
		t.Fatal("ordinary application source must be corrected rather than allowlisted")
	}
}

func TestPlaceholdersAreIgnored(t *testing.T) {
	data := []byte("api_" + "key=replace-me\n")
	if findings := scanFile("config/example.txt", data); len(findings) != 0 {
		t.Fatalf("expected placeholder to be ignored, got %#v", findings)
	}
}

func TestPrivateKeyCannotBeAllowedOrAutomaticallyRedacted(t *testing.T) {
	data := []byte("-----BEGIN OPENSSH " + "PRIVATE KEY-----\n")
	findings := scanFile("config/key.txt", data)
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	if findings[0].Allowable || findings[0].Redactable {
		t.Fatal("private key material must not be auto-allowed or partially redacted")
	}
}

func TestPrepareAndApplyRedaction(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, "config.txt")
	if err := os.WriteFile(path, []byte("client_"+"secret"+"=OnlyForScannerRegression987\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "config.txt")
	result, err := ScanRepo(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(result.Findings))
	}
	id := result.Findings[0].ID
	if _, _, err := PrepareRedaction(context.Background(), root, id); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRedaction(context.Background(), root, id); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updated), "OnlyForScannerRegression987") {
		t.Fatal("detected value remained after redaction")
	}
	if !strings.Contains(string(updated), "REDACTED") {
		t.Fatalf("replacement missing: %q", string(updated))
	}
}

func TestAllowlistProposalContainsFingerprintNotValue(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, "testdata", "fixture.txt")
	value := "DeliberatelyInertFixtureValue987"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("access_"+"token"+"="+value+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "testdata/fixture.txt")
	result, err := ScanRepo(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(result.Findings))
	}
	proposal, _, err := PrepareAllow(context.Background(), root, result.Findings[0].ID, "inert regression fixture")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), value) {
		t.Fatal("allowlist proposal exposed the detected value")
	}
}

func TestStagedSecretCannotBeHiddenByCleanWorkingTree(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, "config.txt")
	value := "Stage" + "OnlyBoundary987"
	if err := os.WriteFile(path, scannerAssignment("client_secret", value), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "config.txt")
	if err := os.WriteFile(path, scannerAssignment("client_secret", "${CLIENT_SECRET}"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := ScanRepo(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	finding := requireFinding(t, result, "sensitive-assignment", sourceStagedIndex)
	if finding.Redactable {
		t.Fatal("staged-index finding must not offer working-tree redaction")
	}
	if strings.Contains(finding.ID, value) || strings.Contains(finding.Fingerprint, value) {
		t.Fatal("finding metadata exposed the staged value")
	}
}

func TestWorkingTreeSecretCannotBeHiddenByCleanIndex(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, "config.txt")
	if err := os.WriteFile(path, scannerAssignment("client_secret", "${CLIENT_SECRET}"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "config.txt")
	if err := os.WriteFile(path, scannerAssignment("client_secret", "Working"+"TreeBoundary987"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := ScanRepo(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	finding := requireFinding(t, result, "sensitive-assignment", sourceWorkingTree)
	if !finding.Redactable {
		t.Fatal("working-tree finding should retain redaction support")
	}
}

func TestIdenticalIndexAndWorkingTreeContentIsNotDuplicated(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, "config.txt")
	if err := os.WriteFile(path, scannerAssignment("client_secret", "Same"+"Boundary987"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "config.txt")
	result, err := ScanRepo(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, finding := range result.Findings {
		if finding.Rule == "sensitive-assignment" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one finding for identical content, got %d", count)
	}
}

func TestDangerousBinaryFilenameFailsBeforeBinarySkip(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, "identity.p12")
	if err := os.WriteFile(path, []byte{0, 1, 2, 3}, 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "identity.p12")
	result, err := ScanRepo(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	requireFinding(t, result, "dangerous-filename", sourceRepositoryPath)
}

func TestDangerousOversizedFilenameFailsBeforeSizeSkip(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, "credentials.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxFileSize+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "credentials.json")
	result, err := ScanRepo(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	requireFinding(t, result, "dangerous-filename", sourceRepositoryPath)
}

func scannerAssignment(name, value string) []byte {
	return []byte(name + "=" + value + "\n")
}

func requireFinding(t *testing.T, result Result, ruleName, source string) Finding {
	t.Helper()
	for _, finding := range result.Findings {
		if finding.Rule == ruleName && finding.Source == source {
			return finding
		}
	}
	t.Fatalf("finding not present: rule=%s source=%s findings=%#v", ruleName, source, result.Findings)
	return Finding{}
}

func initRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "config", "user.name", "ISRAS Test")
	git(t, root, "config", "user.email", "isras-test@example.invalid")
	if err := os.MkdirAll(filepath.Join(root, "validation"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "validation", "secret-allowlist.json"), []byte("{\"version\":1,\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "validation/secret-allowlist.json")
	return root
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
