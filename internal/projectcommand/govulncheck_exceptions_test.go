package projectcommand

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadGovulncheckExceptionsAcceptsAndSortsExactRecords(
	t *testing.T,
) {
	root := t.TempDir()
	now := time.Date(2026, 7, 21, 10, 30, 0, 0, time.UTC)
	path := filepath.Join(root, ".isras", "govulncheck-exceptions.json")

	document := validGovulncheckExceptionDocument(now)
	second := document.Exceptions[0]
	second.AdvisoryID = "GO-2026-0001"
	second.Scope.Symbol = "Other"
	document.Exceptions = append(document.Exceptions, second)
	writeGovulncheckExceptionDocument(t, path, document)

	loaded, err := loadGovulncheckExceptions(root, path, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Exceptions) != 2 {
		t.Fatalf("exception count = %d", len(loaded.Exceptions))
	}
	if loaded.Exceptions[0].AdvisoryID != "GO-2026-0001" ||
		loaded.Exceptions[1].AdvisoryID != "GO-2026-9999" {
		t.Fatalf("exceptions not sorted: %#v", loaded.Exceptions)
	}
}

func TestLoadGovulncheckExceptionsRejectsHostileDocuments(
	t *testing.T,
) {
	now := time.Date(2026, 7, 21, 10, 30, 0, 0, time.UTC)

	cases := []struct {
		name     string
		mutate   func(*govulncheckExceptionDocument)
		contains string
	}{
		{
			name: "unsupported schema",
			mutate: func(document *govulncheckExceptionDocument) {
				document.SchemaVersion = 2
			},
			contains: "schema_version",
		},
		{
			name: "duplicate exact scope",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions = append(
					document.Exceptions,
					document.Exceptions[0],
				)
			},
			contains: "duplicates advisory and scope",
		},
		{
			name: "wildcard scope",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Scope.Symbol = "*"
			},
			contains: "scope.symbol",
		},
		{
			name: "question wildcard scope",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Scope.Symbol = "Serv?ce.Handle"
			},
			contains: "scope.symbol",
		},
		{
			name: "escaped go mod",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Scope.GoModPath = "../go.mod"
			},
			contains: "scope.go_mod_path",
		},
		{
			name: "reserved evidence scope",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Scope.GoModPath =
					".local/generated/go.mod"
			},
			contains: "scope.go_mod_path",
		},
		{
			name: "missing symbol",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Scope.Symbol = ""
			},
			contains: "scope.symbol",
		},
		{
			name: "short justification",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Justification = "too short"
			},
			contains: "justification",
		},
		{
			name: "no controls",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].CompensatingControls = nil
			},
			contains: "compensating_controls",
		},
		{
			name: "duplicate control",
			mutate: func(document *govulncheckExceptionDocument) {
				control := document.Exceptions[0].
					CompensatingControls[0]
				document.Exceptions[0].CompensatingControls =
					append(
						document.Exceptions[0].
							CompensatingControls,
						control,
					)
			},
			contains: "duplicate entry",
		},
		{
			name: "self approval",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Approval.ApprovedBy =
					document.Exceptions[0].Owner
			},
			contains: "independent",
		},
		{
			name: "future approval",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Approval.ApprovedAt =
					now.Add(time.Hour).Format(time.RFC3339Nano)
			},
			contains: "future",
		},
		{
			name: "non UTC approval",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Approval.ApprovedAt =
					"2026-07-21T06:00:00-04:00"
			},
			contains: "ending in Z",
		},
		{
			name: "expired",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].ExpiresAt =
					now.Add(-time.Minute).Format(time.RFC3339Nano)
			},
			contains: "expired",
		},
		{
			name: "target after expiration",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Remediation.TargetDate =
					"2026-08-01"
			},
			contains: "must not be after",
		},
		{
			name: "multiline remediation",
			mutate: func(document *govulncheckExceptionDocument) {
				document.Exceptions[0].Remediation.Plan =
					"replace vulnerable dependency\nlater"
			},
			contains: "remediation.plan",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(
				root,
				".isras",
				"govulncheck-exceptions.json",
			)
			document := validGovulncheckExceptionDocument(now)
			testCase.mutate(&document)
			writeGovulncheckExceptionDocument(
				t,
				path,
				document,
			)

			_, err := loadGovulncheckExceptions(
				root,
				path,
				now,
			)
			if err == nil ||
				!strings.Contains(
					err.Error(),
					testCase.contains,
				) {
				t.Fatalf(
					"error = %v, want substring %q",
					err,
					testCase.contains,
				)
			}
		})
	}
}

func TestLoadGovulncheckExceptionsRejectsJSONAndFilesystemAttacks(
	t *testing.T,
) {
	now := time.Date(2026, 7, 21, 10, 30, 0, 0, time.UTC)

	t.Run("unknown field", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "exceptions.json")
		content := `{"schema_version":1,"exceptions":[],"unknown":true}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := loadGovulncheckExceptions(root, path, now)
		if err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("multiple JSON values", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "exceptions.json")
		content := `{"schema_version":1,"exceptions":[]} {}`

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := loadGovulncheckExceptions(root, path, now)
		if err == nil ||
			!strings.Contains(err.Error(), "multiple JSON values") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("relative path", func(t *testing.T) {
		root := t.TempDir()
		_, err := loadGovulncheckExceptions(
			root,
			"exceptions.json",
			now,
		)
		if err == nil || !strings.Contains(err.Error(), "absolute") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("escaped path", func(t *testing.T) {
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "exceptions.json")
		if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := loadGovulncheckExceptions(
			root,
			outside,
			now,
		)
		if err == nil || !strings.Contains(err.Error(), "escapes") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("symlink component", func(t *testing.T) {
		root := t.TempDir()
		external := t.TempDir()
		if err := os.Symlink(
			external,
			filepath.Join(root, "linked"),
		); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "linked", "exceptions.json")
		if err := os.WriteFile(
			filepath.Join(external, "exceptions.json"),
			[]byte(`{"schema_version":1,"exceptions":[]}`),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		_, err := loadGovulncheckExceptions(root, path, now)
		if err == nil ||
			!strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("nonregular file", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "exceptions.json")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := loadGovulncheckExceptions(root, path, now)
		if err == nil ||
			!strings.Contains(err.Error(), "regular file") {
			t.Fatalf("error = %v", err)
		}
	})
}

func validGovulncheckExceptionDocument(
	now time.Time,
) govulncheckExceptionDocument {
	return govulncheckExceptionDocument{
		SchemaVersion: 1,
		Exceptions: []govulncheckException{{
			AdvisoryID: "GO-2026-9999",
			Scope: govulncheckExceptionScope{
				GoModPath:   "go.mod",
				ModulePath:  "example.com/project",
				PackagePath: "example.com/project/internal/service",
				Symbol:      "(*Service).Handle",
			},
			Justification: "The vulnerable path is temporarily required for compatibility during the bounded migration.",
			CompensatingControls: []string{
				"Untrusted input is rejected before the affected symbol is called.",
				"Runtime monitoring alerts on every invocation of the affected path.",
			},
			Owner: "security-owner@example.invalid",
			Approval: govulncheckExceptionApproval{
				ApprovedBy: "security-approver@example.invalid",
				ApprovedAt: now.Add(-time.Hour).Format(time.RFC3339Nano),
				Record:     "SEC-2026-0042",
			},
			ExpiresAt: now.Add(7 * 24 * time.Hour).Format(time.RFC3339Nano),
			Remediation: govulncheckExceptionRemediation{
				Owner:      "engineering-owner@example.invalid",
				TargetDate: "2026-07-28",
				Plan:       "Upgrade the affected dependency and remove this exception after complete regression validation.",
			},
		}},
	}
}

func writeGovulncheckExceptionDocument(
	t *testing.T,
	path string,
	document govulncheckExceptionDocument,
) {
	t.Helper()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
