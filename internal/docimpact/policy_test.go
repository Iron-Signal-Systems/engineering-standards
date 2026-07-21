package docimpact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateDocumentationImpact(t *testing.T) {
	policy := testPolicy()

	tests := []struct {
		name       string
		changed    []string
		wantStatus string
		wantRules  []string
	}{
		{
			name:       "documentation only",
			changed:    []string{"README.md", "standards/ISRAS-VISION.md"},
			wantStatus: "PASS",
		},
		{
			name: "implementation synchronized",
			changed: []string{
				"internal/example/example.go",
				"CHANGELOG.md",
				"standards/EXAMPLE.md",
				"docs/records/EXAMPLE.md",
			},
			wantStatus: "PASS",
			wantRules:  []string{"implementation"},
		},
		{
			name: "implementation missing record",
			changed: []string{
				"cmd/tool/main.go",
				"CHANGELOG.md",
				"standards/EXAMPLE.md",
			},
			wantStatus: "FAIL",
			wantRules:  []string{"implementation"},
		},
		{
			name: "schema synchronized",
			changed: []string{
				"schemas/example.schema.json",
				"schemas/examples/example.json",
				"CHANGELOG.md",
				"standards/EXAMPLE.md",
				"docs/records/EXAMPLE.md",
			},
			wantStatus: "PASS",
			wantRules:  []string{"schema"},
		},
		{
			name: "schema missing example",
			changed: []string{
				"schemas/example.schema.json",
				"CHANGELOG.md",
				"standards/EXAMPLE.md",
				"docs/records/EXAMPLE.md",
			},
			wantStatus: "FAIL",
			wantRules:  []string{"schema"},
		},
		{
			name: "workflow missing testing standard",
			changed: []string{
				".github/workflows/validate.yml",
				"CHANGELOG.md",
				"docs/records/WORKFLOW.md",
			},
			wantStatus: "FAIL",
			wantRules:  []string{"workflow"},
		},
		{
			name: "overlapping rules all pass",
			changed: []string{
				"validation/documentation-impact-policy.json",
				"schemas/isras-documentation-impact-policy-v1.schema.json",
				"schemas/examples/isras-documentation-impact-policy-v1.example.json",
				"CHANGELOG.md",
				"standards/DOCUMENTATION-IMPACT.md",
				"docs/records/DOCUMENTATION-IMPACT.md",
			},
			wantStatus: "PASS",
			wantRules:  []string{"policy", "schema"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			report, err := Evaluate(
				policy,
				"validation/documentation-impact-policy.json",
				testCase.changed,
			)
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != testCase.wantStatus {
				t.Fatalf(
					"status = %q, want %q: %#v",
					report.Status,
					testCase.wantStatus,
					report,
				)
			}
			var ruleIDs []string
			for _, rule := range report.Triggered {
				ruleIDs = append(ruleIDs, rule.ID)
			}
			if strings.Join(ruleIDs, ",") !=
				strings.Join(testCase.wantRules, ",") {
				t.Fatalf(
					"rules = %v, want %v",
					ruleIDs,
					testCase.wantRules,
				)
			}
		})
	}
}

func TestEvaluateDocumentationImpactIsDeterministic(t *testing.T) {
	report, err := Evaluate(
		testPolicy(),
		"validation/documentation-impact-policy.json",
		[]string{
			"standards/Z.md",
			"internal/z/z.go",
			"CHANGELOG.md",
			"docs/records/Z.md",
			"internal/a/a.go",
			"standards/A.md",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "CHANGELOG.md,docs/records/Z.md,internal/a/a.go,internal/z/z.go,standards/A.md,standards/Z.md"
	if strings.Join(report.ChangedPaths, ",") != want {
		t.Fatalf(
			"changed paths = %q, want %q",
			strings.Join(report.ChangedPaths, ","),
			want,
		)
	}
	if len(report.Triggered) != 1 ||
		report.Triggered[0].ID != "implementation" {
		t.Fatalf("triggered rules = %#v", report.Triggered)
	}
}

func TestEvaluateDocumentationImpactRejectsUnsafePaths(t *testing.T) {
	for _, path := range []string{
		"../escape",
		"/absolute",
		`bad\path`,
		".git/config",
		".local/evidence",
		"line\nbreak",
	} {
		_, err := Evaluate(
			testPolicy(),
			"validation/documentation-impact-policy.json",
			[]string{path},
		)
		if err == nil || !strings.Contains(err.Error(), "unsafe") {
			t.Fatalf("path %q error = %v", path, err)
		}
	}
}

func TestValidatePolicyRejectsHostilePolicy(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Policy)
		contains string
	}{
		{
			name: "schema",
			mutate: func(policy *Policy) {
				policy.SchemaVersion = 2
			},
			contains: "schema_version",
		},
		{
			name: "duplicate rule",
			mutate: func(policy *Policy) {
				policy.Rules = append(
					policy.Rules,
					policy.Rules[0],
				)
			},
			contains: "duplicated",
		},
		{
			name: "bad id",
			mutate: func(policy *Policy) {
				policy.Rules[0].ID = "Bad_ID"
			},
			contains: "invalid id",
		},
		{
			name: "empty triggers",
			mutate: func(policy *Policy) {
				policy.Rules[0].Triggers = nil
			},
			contains: "triggers",
		},
		{
			name: "path and prefix",
			mutate: func(policy *Policy) {
				policy.Rules[0].Triggers[0] = Pattern{
					Path:   "go.mod",
					Prefix: "internal/",
				}
			},
			contains: "exactly one",
		},
		{
			name: "unsafe prefix",
			mutate: func(policy *Policy) {
				policy.Rules[0].Triggers[0] = Pattern{
					Prefix: "../",
				}
			},
			contains: "prefix is unsafe",
		},
		{
			name: "both all and any",
			mutate: func(policy *Policy) {
				policy.Rules[0].Requirements[0].Any =
					[]Pattern{{Path: "README.md"}}
			},
			contains: "exactly one of all or any",
		},
		{
			name: "duplicate requirement",
			mutate: func(policy *Policy) {
				policy.Rules[0].Requirements = append(
					policy.Rules[0].Requirements,
					policy.Rules[0].Requirements[0],
				)
			},
			contains: "requirement id",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			policy := testPolicy()
			testCase.mutate(&policy)
			err := ValidatePolicy(policy)
			if err == nil ||
				!strings.Contains(err.Error(), testCase.contains) {
				t.Fatalf(
					"error = %v, want substring %q",
					err,
					testCase.contains,
				)
			}
		})
	}
}

func TestLoadPolicyRejectsJSONAndFilesystemAttacks(t *testing.T) {
	root := t.TempDir()

	t.Run("valid", func(t *testing.T) {
		path := filepath.Join(root, "validation", "policy.json")
		writePolicy(t, path, testPolicy())
		loaded, err := LoadPolicy(root, path)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.SchemaVersion != 1 {
			t.Fatalf("schema version = %d", loaded.SchemaVersion)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		path := filepath.Join(root, "unknown.json")
		content := `{"schema_version":1,"rules":[],"unknown":true}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadPolicy(root, path)
		if err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("multiple values", func(t *testing.T) {
		path := filepath.Join(root, "multiple.json")
		content := `{"schema_version":1,"rules":[]} {}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := LoadPolicy(root, path)
		if err == nil ||
			!strings.Contains(err.Error(), "multiple JSON values") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("relative", func(t *testing.T) {
		_, err := LoadPolicy(root, "policy.json")
		if err == nil || !strings.Contains(err.Error(), "absolute") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("escape", func(t *testing.T) {
		outside := filepath.Join(t.TempDir(), "policy.json")
		writePolicy(t, outside, testPolicy())
		_, err := LoadPolicy(root, outside)
		if err == nil || !strings.Contains(err.Error(), "escapes") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		external := t.TempDir()
		externalPath := filepath.Join(external, "policy.json")
		writePolicy(t, externalPath, testPolicy())
		link := filepath.Join(root, "linked")
		if err := os.Symlink(external, link); err != nil {
			t.Fatal(err)
		}
		_, err := LoadPolicy(root, filepath.Join(link, "policy.json"))
		if err == nil ||
			!strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("error = %v", err)
		}
	})
}

func testPolicy() Policy {
	common := []Requirement{
		{
			ID:          "changelog",
			Description: "The same change set updates the unreleased changelog.",
			All:         []Pattern{{Path: "CHANGELOG.md"}},
		},
		{
			ID:          "standard",
			Description: "The same change set updates at least one governing standard.",
			Any:         []Pattern{{Prefix: "standards/", Suffix: ".md"}},
		},
		{
			ID:          "record",
			Description: "The same change set updates at least one implementation or acceptance record.",
			Any:         []Pattern{{Prefix: "docs/records/", Suffix: ".md"}},
		},
	}

	return Policy{
		SchemaVersion: 1,
		Rules: []Rule{
			{
				ID:          "implementation",
				Description: "Repository implementation or validator source changes require synchronized governance documentation.",
				Triggers: []Pattern{
					{Prefix: "internal/", Suffix: ".go"},
					{Prefix: "cmd/", Suffix: ".go"},
					{Path: "go.mod"},
					{Path: "go.sum"},
				},
				Requirements: common,
			},
			{
				ID:          "policy",
				Description: "Documentation-impact policy changes require synchronized governance documentation.",
				Triggers: []Pattern{
					{Path: "validation/documentation-impact-policy.json"},
				},
				Requirements: common,
			},
			{
				ID:          "schema",
				Description: "Schema changes require a synchronized example and governance documentation.",
				Triggers: []Pattern{
					{Prefix: "schemas/", Suffix: ".schema.json"},
				},
				Requirements: append(
					append([]Requirement(nil), common...),
					Requirement{
						ID:          "example",
						Description: "The same change set updates at least one governed schema example.",
						Any: []Pattern{{
							Prefix: "schemas/examples/",
							Suffix: ".json",
						}},
					},
				),
			},
			{
				ID:          "workflow",
				Description: "Hosted validation workflow changes require testing-governance synchronization.",
				Triggers: []Pattern{{
					Prefix: ".github/workflows/",
					Suffix: ".yml",
				}},
				Requirements: []Requirement{
					common[0],
					{
						ID:          "testing-standard",
						Description: "The same change set updates the testing or execution standard.",
						Any: []Pattern{
							{Path: "standards/TESTING-AND-VALIDATION.md"},
							{Path: "standards/PROJECT-COMMAND-EXECUTION.md"},
						},
					},
					common[2],
				},
			},
		},
	}
}

func writePolicy(t *testing.T, path string, policy Policy) {
	t.Helper()
	data, err := json.Marshal(policy)
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

func TestRepositoryDocumentationImpactPolicyLoadsAndCoversReleaseBoundaries(
	t *testing.T,
) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(
		filepath.Join(workingDirectory, "..", ".."),
	)
	policyPath := filepath.Join(
		root,
		filepath.FromSlash(
			"validation/documentation-impact-policy.json",
		),
	)

	policy, err := LoadPolicy(root, policyPath)
	if err != nil {
		t.Fatalf("load governed repository policy: %v", err)
	}

	var releaseRule *Rule
	for index := range policy.Rules {
		if policy.Rules[index].ID == "release-and-adoption" {
			releaseRule = &policy.Rules[index]
			break
		}
	}
	if releaseRule == nil {
		t.Fatal("release-and-adoption rule is missing")
	}

	prefixes := make(map[string]struct{})
	for _, trigger := range releaseRule.Triggers {
		if trigger.Prefix != "" {
			prefixes[trigger.Prefix] = struct{}{}
		}
	}

	var releasePaths []string
	for _, boundary := range []struct {
		directory string
		prefix    string
	}{
		{
			directory: filepath.Join(root, "internal"),
			prefix:    "release",
		},
		{
			directory: filepath.Join(root, "cmd"),
			prefix:    "isras-release",
		},
	} {
		entries, err := os.ReadDir(boundary.directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !entry.IsDir() ||
				!strings.HasPrefix(
					entry.Name(),
					boundary.prefix,
				) {
				continue
			}
			relative, err := filepath.Rel(
				root,
				filepath.Join(
					boundary.directory,
					entry.Name(),
				),
			)
			if err != nil {
				t.Fatal(err)
			}
			prefix := filepath.ToSlash(relative) + "/"
			if _, covered := prefixes[prefix]; !covered {
				t.Fatalf(
					"release boundary %q is not covered by an exact directory prefix",
					prefix,
				)
			}
			releasePaths = append(
				releasePaths,
				prefix+"acceptance_fixture.go",
			)
		}
	}

	if len(releasePaths) == 0 {
		t.Fatal("no release implementation directories were discovered")
	}

	changed := append(
		releasePaths,
		"CHANGELOG.md",
		"standards/RELEASE-PROCESS.md",
		"docs/records/DOCUMENTATION-IMPACT-GATE.md",
	)
	report, err := Evaluate(
		policy,
		PolicyRelativePathForRepositoryTest,
		changed,
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "PASS" {
		t.Fatalf("release policy report = %#v", report)
	}

	found := false
	for _, rule := range report.Triggered {
		if rule.ID != "release-and-adoption" {
			continue
		}
		found = true
		if rule.Status != "PASS" {
			t.Fatalf("release rule = %#v", rule)
		}
	}
	if !found {
		t.Fatal("release-and-adoption rule did not trigger")
	}
}

const PolicyRelativePathForRepositoryTest = "validation/documentation-impact-policy.json"
