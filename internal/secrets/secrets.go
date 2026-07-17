package secrets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
)

const (
	maxFileSize          = 2 * 1024 * 1024
	sourceWorkingTree    = "working-tree"
	sourceStagedIndex    = "staged-index"
	sourceRepositoryPath = "repository-path"
)

type Finding struct {
	ID          string `json:"id"`
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Fingerprint string `json:"fingerprint"`
	Redactable  bool   `json:"redactable"`
	Allowable   bool   `json:"allowable"`
	Start       int    `json:"-"`
	End         int    `json:"-"`
}

type AllowEntry struct {
	ID          string `json:"id"`
	Rule        string `json:"rule"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
	Reason      string `json:"reason"`
	Assurance   string `json:"assurance"`
	Added       string `json:"added"`
	Expires     string `json:"expires,omitempty"`
}

type Allowlist struct {
	Version int          `json:"version"`
	Entries []AllowEntry `json:"entries"`
}

type Result struct {
	Findings []Finding
	Allowed  []Finding
	Scanned  int
	Skipped  int
}

type rule struct {
	name       string
	severity   string
	pattern    *regexp.Regexp
	group      int
	redactable bool
	allowable  bool
}

var rules = []rule{
	{
		name: "private-key-material", severity: "critical",
		pattern: regexp.MustCompile(`-----BEGIN (?:OPENSSH |RSA |EC |DSA )?PRIVATE KEY-----`),
		group:   0, redactable: false, allowable: false,
	},
	{
		name: "github-token", severity: "critical",
		pattern: regexp.MustCompile(`gh` + `[pousr]_[A-Za-z0-9]{20,}`),
		group:   0, redactable: true, allowable: false,
	},
	{
		name: "aws-access-key", severity: "critical",
		pattern: regexp.MustCompile(`AK` + `IA[0-9A-Z]{16}`),
		group:   0, redactable: true, allowable: false,
	},
	{
		name: "slack-token", severity: "critical",
		pattern: regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
		group:   0, redactable: true, allowable: false,
	},
	{
		name: "bearer-authorization", severity: "critical",
		pattern: regexp.MustCompile(`(?i)authorization\s*:\s*bearer\s+([^\s"']{8,})`),
		group:   1, redactable: true, allowable: false,
	},
	{
		name: "embedded-url-password", severity: "high",
		pattern: regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://[^\s/:@]+:([^\s/@]{4,})@`),
		group:   1, redactable: true, allowable: false,
	},
	{
		name: "sensitive-assignment", severity: "high",
		pattern: regexp.MustCompile(`(?i)(?:password|passwd|api[_-]?key|client[_-]?secret|access[_-]?token|refresh[_-]?token|secret|token)\s*[:=]\s*["']?([^\s,"';#]{8,})`),
		group:   1, redactable: true, allowable: true,
	},
}

var dangerousBaseNames = map[string]bool{
	".env":             true,
	".env.production":  true,
	"credentials":      true,
	"credentials.json": true,
	"secrets.yaml":     true,
	"secrets.yml":      true,
	"id_rsa":           true,
	"id_ed25519":       true,
	"kubeconfig":       true,
}

type indexEntry struct {
	Mode   string
	Object string
	Path   string
}

func ScanRepo(ctx context.Context, root string) (Result, error) {
	allowlist, err := loadAllowlist(filepath.Join(root, "validation", "secret-allowlist.json"))
	if err != nil {
		return Result{}, err
	}
	indexEntries, err := listIndexEntries(ctx, root)
	if err != nil {
		return Result{}, err
	}
	untracked, err := listUntrackedPaths(ctx, root)
	if err != nil {
		return Result{}, err
	}

	indexByPath := make(map[string]indexEntry, len(indexEntries))
	pathSet := make(map[string]struct{}, len(indexEntries)+len(untracked))
	for _, entry := range indexEntries {
		indexByPath[entry.Path] = entry
		pathSet[entry.Path] = struct{}{}
	}
	for _, path := range untracked {
		pathSet[path] = struct{}{}
	}

	paths := make([]string, 0, len(pathSet))
	for path := range pathSet {
		if path == "" || strings.HasPrefix(path, ".local/") {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	result := Result{}
	for _, rel := range paths {
		pathScanned := false

		if finding, ok := dangerousFilenameFinding(rel); ok {
			appendFinding(&result, finding, allowlist)
		}

		workData, workPresent, workScannable, err := readWorkingTreeFile(root, rel)
		if err != nil {
			return Result{}, err
		}
		if workPresent && workScannable {
			pathScanned = true
			appendFindings(&result, scanContent(rel, workData, sourceWorkingTree), allowlist)
		}

		if entry, ok := indexByPath[rel]; ok {
			indexData, indexPresent, indexScannable, err := readIndexBlob(ctx, root, entry)
			if err != nil {
				return Result{}, err
			}
			if indexPresent && indexScannable && (!workScannable || !bytes.Equal(indexData, workData)) {
				pathScanned = true
				appendFindings(&result, scanContent(rel, indexData, sourceStagedIndex), allowlist)
			}
		}

		if pathScanned {
			result.Scanned++
		} else {
			result.Skipped++
		}
	}

	result.Findings = deduplicate(result.Findings)
	result.Allowed = deduplicate(result.Allowed)
	sortFindings(result.Findings)
	sortFindings(result.Allowed)
	return result, nil
}

func listIndexEntries(ctx context.Context, root string) ([]indexEntry, error) {
	listed := executil.Run(ctx, root, "git", "ls-files", "--stage", "-z")
	if listed.Err != nil {
		return nil, fmt.Errorf("list staged repository files: %w", listed.Err)
	}
	var entries []indexEntry
	for _, record := range strings.Split(listed.Stdout, "\x00") {
		if record == "" {
			continue
		}
		tab := strings.IndexByte(record, '\t')
		if tab < 0 {
			return nil, errors.New("parse staged repository entry")
		}
		fields := strings.Fields(record[:tab])
		if len(fields) != 3 {
			return nil, errors.New("parse staged repository entry")
		}
		path := filepath.ToSlash(record[tab+1:])
		if fields[2] != "0" {
			return nil, fmt.Errorf("unmerged staged repository entry: %s", path)
		}
		entries = append(entries, indexEntry{
			Mode: fields[0], Object: fields[1], Path: path,
		})
	}
	return entries, nil
}

func listUntrackedPaths(ctx context.Context, root string) ([]string, error) {
	listed := executil.Run(ctx, root, "git", "ls-files", "-z", "--others", "--exclude-standard")
	if listed.Err != nil {
		return nil, fmt.Errorf("list untracked repository files: %w", listed.Err)
	}
	var paths []string
	for _, path := range strings.Split(listed.Stdout, "\x00") {
		if path != "" {
			paths = append(paths, filepath.ToSlash(path))
		}
	}
	return paths, nil
}

func readIndexBlob(ctx context.Context, root string, entry indexEntry) ([]byte, bool, bool, error) {
	if !strings.HasPrefix(entry.Mode, "100") && entry.Mode != "120000" {
		return nil, true, false, nil
	}
	sizeResult := executil.Run(ctx, root, "git", "cat-file", "-s", entry.Object)
	if sizeResult.Err != nil {
		return nil, false, false, fmt.Errorf("inspect staged %s: %w", entry.Path, sizeResult.Err)
	}
	size, err := strconv.ParseInt(strings.TrimSpace(sizeResult.Stdout), 10, 64)
	if err != nil || size < 0 {
		return nil, false, false, fmt.Errorf("parse staged size for %s", entry.Path)
	}
	if size > maxFileSize {
		return nil, true, false, nil
	}
	blobResult := executil.Run(ctx, root, "git", "cat-file", "blob", entry.Object)
	if blobResult.Err != nil {
		return nil, false, false, fmt.Errorf("read staged %s: %w", entry.Path, blobResult.Err)
	}
	data := []byte(blobResult.Stdout)
	if !utf8.Valid(data) || containsNUL(data) {
		return data, true, false, nil
	}
	return data, true, true, nil
}

func readWorkingTreeFile(root, rel string) ([]byte, bool, bool, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, false, nil
		}
		return nil, false, false, fmt.Errorf("inspect %s: %w", rel, err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxFileSize {
		return nil, true, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, true, false, fmt.Errorf("read %s: %w", rel, err)
	}
	if !utf8.Valid(data) || containsNUL(data) {
		return data, true, false, nil
	}
	return data, true, true, nil
}

func appendFindings(result *Result, findings []Finding, allowlist Allowlist) {
	for _, finding := range findings {
		appendFinding(result, finding, allowlist)
	}
}

func appendFinding(result *Result, finding Finding, allowlist Allowlist) {
	if allowed(finding, allowlist) {
		result.Allowed = append(result.Allowed, finding)
	} else {
		result.Findings = append(result.Findings, finding)
	}
}

func dangerousFilenameFinding(path string) (Finding, bool) {
	base := strings.ToLower(filepath.Base(path))
	if !dangerousBaseNames[base] &&
		!strings.HasSuffix(base, ".p12") &&
		!strings.HasSuffix(base, ".pfx") &&
		!strings.HasSuffix(base, ".key") {
		return Finding{}, false
	}
	return Finding{
		ID: findingID("dangerous-filename", path, 0), Rule: "dangerous-filename", Severity: "high",
		Path: path, Source: sourceRepositoryPath, Line: 1, Column: 1,
		Fingerprint: fingerprint("dangerous-filename", path, []byte(path)),
		Redactable:  false, Allowable: strings.Contains(path, "testdata/"),
	}, true
}

func Find(ctx context.Context, root, id string) (Finding, error) {
	result, err := ScanRepo(ctx, root)
	if err != nil {
		return Finding{}, err
	}
	for _, finding := range append(result.Findings, result.Allowed...) {
		if finding.ID == id {
			return finding, nil
		}
	}
	return Finding{}, fmt.Errorf("finding %s is not present in the current repository state", id)
}

func PrepareRedaction(ctx context.Context, root, id string) (string, Finding, error) {
	finding, err := Find(ctx, root, id)
	if err != nil {
		return "", Finding{}, err
	}
	if !finding.Redactable {
		return "", finding, errors.New("this finding class cannot be redacted automatically")
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(finding.Path)))
	if err != nil {
		return "", finding, err
	}
	plan := struct {
		Version     int    `json:"version"`
		FindingID   string `json:"finding_id"`
		Rule        string `json:"rule"`
		Path        string `json:"path"`
		Fingerprint string `json:"fingerprint"`
		FileSHA256  string `json:"file_sha256"`
		Replacement string `json:"replacement"`
		Created     string `json:"created"`
	}{
		Version: 1, FindingID: finding.ID, Rule: finding.Rule, Path: finding.Path,
		Fingerprint: finding.Fingerprint, FileSHA256: digest(data),
		Replacement: "REDACTED", Created: time.Now().UTC().Format(time.RFC3339),
	}
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", finding, err
	}
	dir := filepath.Join(root, ".local", "validation", "redactions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", finding, err
	}
	path := filepath.Join(dir, id+".json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return "", finding, err
	}
	return filepath.ToSlash(path), finding, nil
}

func ApplyRedaction(ctx context.Context, root, id string) (Finding, error) {
	planPath := filepath.Join(root, ".local", "validation", "redactions", id+".json")
	planData, err := os.ReadFile(planPath)
	if err != nil {
		return Finding{}, fmt.Errorf("read redaction plan: %w", err)
	}
	var plan struct {
		FindingID   string `json:"finding_id"`
		Path        string `json:"path"`
		Fingerprint string `json:"fingerprint"`
		FileSHA256  string `json:"file_sha256"`
		Replacement string `json:"replacement"`
	}
	if err := json.Unmarshal(planData, &plan); err != nil {
		return Finding{}, fmt.Errorf("parse redaction plan: %w", err)
	}
	finding, err := Find(ctx, root, id)
	if err != nil {
		return Finding{}, err
	}
	if finding.Path != plan.Path || finding.Fingerprint != plan.Fingerprint {
		return Finding{}, errors.New("current finding does not match the prepared plan")
	}
	path := filepath.Join(root, filepath.FromSlash(finding.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return Finding{}, err
	}
	if digest(data) != plan.FileSHA256 {
		return Finding{}, errors.New("source file changed after the redaction plan was prepared")
	}
	if finding.Start < 0 || finding.End > len(data) || finding.Start >= finding.End {
		return Finding{}, errors.New("finding byte range is invalid")
	}
	updated := make([]byte, 0, len(data))
	updated = append(updated, data[:finding.Start]...)
	updated = append(updated, plan.Replacement...)
	updated = append(updated, data[finding.End:]...)
	info, err := os.Stat(path)
	if err != nil {
		return Finding{}, err
	}
	if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil {
		return Finding{}, err
	}
	return finding, nil
}

func PrepareAllow(ctx context.Context, root, id, reason string) (string, Finding, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", Finding{}, errors.New("a non-empty reason is required")
	}
	finding, err := Find(ctx, root, id)
	if err != nil {
		return "", Finding{}, err
	}
	if !finding.Allowable {
		return "", finding, errors.New("this finding class cannot be allowlisted")
	}
	proposal := AllowEntry{
		ID:          "ALLOW-" + finding.ID,
		Rule:        finding.Rule,
		Path:        finding.Path,
		Fingerprint: finding.Fingerprint,
		Reason:      reason,
		Assurance:   "self-approved-exception",
		Added:       time.Now().UTC().Format("2006-01-02"),
	}
	encoded, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		return "", finding, err
	}
	dir := filepath.Join(root, ".local", "validation", "proposals")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", finding, err
	}
	path := filepath.Join(dir, id+"-allow.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return "", finding, err
	}
	return filepath.ToSlash(path), finding, nil
}

func ApplyAllow(root, id string) (string, error) {
	proposalPath := filepath.Join(root, ".local", "validation", "proposals", id+"-allow.json")
	data, err := os.ReadFile(proposalPath)
	if err != nil {
		return "", fmt.Errorf("read allowlist proposal: %w", err)
	}
	var entry AllowEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", fmt.Errorf("parse allowlist proposal: %w", err)
	}
	allowPath := filepath.Join(root, "validation", "secret-allowlist.json")
	allowlist, err := loadAllowlist(allowPath)
	if err != nil {
		return "", err
	}
	for _, existing := range allowlist.Entries {
		if existing.Rule == entry.Rule && existing.Path == entry.Path && existing.Fingerprint == entry.Fingerprint {
			return filepath.ToSlash(allowPath), nil
		}
	}
	allowlist.Entries = append(allowlist.Entries, entry)
	sort.Slice(allowlist.Entries, func(i, j int) bool { return allowlist.Entries[i].ID < allowlist.Entries[j].ID })
	encoded, err := json.MarshalIndent(allowlist, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(allowPath, append(encoded, '\n'), 0o644); err != nil {
		return "", err
	}
	return filepath.ToSlash(allowPath), nil
}

func scanFile(path string, data []byte) []Finding {
	var findings []Finding
	if finding, ok := dangerousFilenameFinding(path); ok {
		findings = append(findings, finding)
	}
	findings = append(findings, scanContent(path, data, sourceWorkingTree)...)
	return deduplicate(findings)
}

func scanContent(path string, data []byte, source string) []Finding {
	var findings []Finding
	semantics := semanticContextFor(path, data)
	for _, current := range rules {
		matches := current.pattern.FindAllSubmatchIndex(data, -1)
		for _, match := range matches {
			idx := current.group * 2
			if idx+1 >= len(match) || match[idx] < 0 || match[idx+1] < 0 {
				continue
			}
			start, end := match[idx], match[idx+1]
			if current.name == "sensitive-assignment" &&
				ignoreSensitiveAssignment(path, data, match[0], match[1], start, end, semantics) {
				continue
			}
			value := data[start:end]
			if placeholder(value) {
				continue
			}
			line, column := lineColumn(data, start)
			fp := fingerprint(current.name, path, value)
			allowable := current.allowable && allowableContext(path)
			if current.name == "private-key-material" {
				allowable = false
			}
			findings = append(findings, Finding{
				ID: findingIDForSource(current.name, path, start, source), Rule: current.name, Severity: current.severity,
				Path: path, Source: source, Line: line, Column: column, Fingerprint: fp,
				Redactable: current.redactable && source != sourceStagedIndex, Allowable: allowable,
				Start: start, End: end,
			})
		}
	}
	return deduplicate(findings)
}

func loadAllowlist(path string) (Allowlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Allowlist{Version: 1}, nil
		}
		return Allowlist{}, err
	}
	var allowlist Allowlist
	if err := json.Unmarshal(data, &allowlist); err != nil {
		return Allowlist{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if allowlist.Version != 1 {
		return Allowlist{}, fmt.Errorf("unsupported secret allowlist version %d", allowlist.Version)
	}
	return allowlist, nil
}

func allowed(finding Finding, allowlist Allowlist) bool {
	today := time.Now().UTC().Format("2006-01-02")
	for _, entry := range allowlist.Entries {
		if entry.Rule != finding.Rule || entry.Path != finding.Path || entry.Fingerprint != finding.Fingerprint {
			continue
		}
		if entry.Expires != "" && entry.Expires < today {
			continue
		}
		return true
	}
	return false
}

func placeholder(value []byte) bool {
	v := strings.ToLower(strings.Trim(strings.TrimSpace(string(value)), `"'`))
	if strings.HasPrefix(v, "${") || strings.HasPrefix(v, "<") || strings.HasPrefix(v, "{{") {
		return true
	}
	placeholders := []string{
		"redacted", "replace-me", "replace_me", "changeme", "change-me",
		"example", "example-only", "not-a-secret", "not-a-real-secret",
		"test-only", "placeholder", "your-token-here", "your-password-here",
	}
	for _, candidate := range placeholders {
		if v == candidate {
			return true
		}
	}
	return false
}

func fingerprint(ruleName, path string, value []byte) string {
	h := sha256.New()
	h.Write([]byte(ruleName))
	h.Write([]byte{0})
	h.Write([]byte(filepath.ToSlash(path)))
	h.Write([]byte{0})
	h.Write(value)
	return hex.EncodeToString(h.Sum(nil))
}

func findingID(ruleName, path string, offset int) string {
	return findingIDForSource(ruleName, path, offset, sourceWorkingTree)
}

func findingIDForSource(ruleName, path string, offset int, source string) string {
	h := sha256.New()
	h.Write([]byte(ruleName))
	h.Write([]byte{0})
	h.Write([]byte(filepath.ToSlash(path)))
	h.Write([]byte{0})
	h.Write([]byte(fmt.Sprintf("%d", offset)))
	if source != "" && source != sourceWorkingTree {
		h.Write([]byte{0})
		h.Write([]byte(source))
	}
	value := hex.EncodeToString(h.Sum(nil))
	return "ISSEC-" + strings.ToUpper(value[:16])
}

func allowableContext(path string) bool {
	path = "/" + strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(path, "/testdata/") ||
		strings.Contains(path, "/examples/") ||
		strings.Contains(path, "/docs/") ||
		strings.HasSuffix(path, "_test.go")
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func lineColumn(data []byte, offset int) (int, int) {
	line, column := 1, 1
	for i, b := range data {
		if i >= offset {
			break
		}
		if b == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return line, column
}

func containsNUL(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].ID < findings[j].ID
	})
}

func deduplicate(findings []Finding) []Finding {
	seen := make(map[string]bool)
	out := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		key := finding.ID + "\x00" + finding.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, finding)
	}
	return out
}
