package docimpact

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	PolicySchemaVersion = 1
	maxPolicyBytes      = 256 * 1024
	maxRules            = 128
	maxPatterns         = 128
)

type Policy struct {
	SchemaVersion int    `json:"schema_version"`
	Rules         []Rule `json:"rules"`
}

type Rule struct {
	ID           string        `json:"id"`
	Description  string        `json:"description"`
	Triggers     []Pattern     `json:"triggers"`
	Requirements []Requirement `json:"requirements"`
}

type Requirement struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	All         []Pattern `json:"all,omitempty"`
	Any         []Pattern `json:"any,omitempty"`
}

type Pattern struct {
	Path   string `json:"path,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Suffix string `json:"suffix,omitempty"`
}

type Report struct {
	SchemaVersion int          `json:"schema_version"`
	Status        string       `json:"status"`
	PolicyPath    string       `json:"policy_path"`
	ChangedPaths  []string     `json:"changed_paths"`
	Triggered     []RuleReport `json:"triggered_rules"`
}

type RuleReport struct {
	ID           string              `json:"id"`
	Description  string              `json:"description"`
	TriggerPaths []string            `json:"trigger_paths"`
	Requirements []RequirementReport `json:"requirements"`
	Status       string              `json:"status"`
}

type RequirementReport struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	MatchedPaths []string `json:"matched_paths"`
	Status       string   `json:"status"`
}

func LoadPolicy(root string, path string) (Policy, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Policy{}, errors.New("resolve documentation-impact repository root")
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	if !filepath.IsAbs(path) {
		return Policy{}, errors.New("documentation-impact policy path must be absolute")
	}
	absolutePath := filepath.Clean(path)
	if !pathWithin(absoluteRoot, absolutePath) {
		return Policy{}, errors.New("documentation-impact policy path escapes the repository")
	}
	if err := rejectSymlinkComponents(absoluteRoot, absolutePath); err != nil {
		return Policy{}, err
	}

	info, err := os.Lstat(absolutePath)
	if err != nil {
		return Policy{}, errors.New("inspect documentation-impact policy")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Policy{}, errors.New("documentation-impact policy must not be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return Policy{}, errors.New("documentation-impact policy must be a regular file")
	}
	if info.Size() > maxPolicyBytes {
		return Policy{}, fmt.Errorf(
			"documentation-impact policy exceeds %d bytes",
			maxPolicyBytes,
		)
	}

	file, err := os.Open(absolutePath)
	if err != nil {
		return Policy{}, errors.New("open documentation-impact policy")
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, maxPolicyBytes+1))
	decoder.DisallowUnknownFields()

	var policy Policy
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("decode documentation-impact policy: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Policy{}, errors.New(
				"documentation-impact policy contains multiple JSON values",
			)
		}
		return Policy{}, fmt.Errorf(
			"decode trailing documentation-impact policy data: %w",
			err,
		)
	}

	if err := ValidatePolicy(policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func ValidatePolicy(policy Policy) error {
	if policy.SchemaVersion != PolicySchemaVersion {
		return fmt.Errorf(
			"documentation-impact schema_version must be %d",
			PolicySchemaVersion,
		)
	}
	if len(policy.Rules) == 0 || len(policy.Rules) > maxRules {
		return fmt.Errorf(
			"documentation-impact rules must contain 1-%d entries",
			maxRules,
		)
	}

	ruleIDs := make(map[string]struct{}, len(policy.Rules))
	for index, rule := range policy.Rules {
		if !validID(rule.ID) {
			return fmt.Errorf(
				"documentation-impact rule %d has invalid id",
				index+1,
			)
		}
		if _, duplicate := ruleIDs[rule.ID]; duplicate {
			return fmt.Errorf(
				"documentation-impact rule id %q is duplicated",
				rule.ID,
			)
		}
		ruleIDs[rule.ID] = struct{}{}

		if !validText(rule.Description, 10, 1024) {
			return fmt.Errorf(
				"documentation-impact rule %q has invalid description",
				rule.ID,
			)
		}
		if len(rule.Triggers) == 0 || len(rule.Triggers) > maxPatterns {
			return fmt.Errorf(
				"documentation-impact rule %q triggers must contain 1-%d entries",
				rule.ID,
				maxPatterns,
			)
		}
		for patternIndex, pattern := range rule.Triggers {
			if err := validatePattern(pattern); err != nil {
				return fmt.Errorf(
					"documentation-impact rule %q trigger %d: %w",
					rule.ID,
					patternIndex+1,
					err,
				)
			}
		}
		if len(rule.Requirements) == 0 ||
			len(rule.Requirements) > maxPatterns {
			return fmt.Errorf(
				"documentation-impact rule %q requirements must contain 1-%d entries",
				rule.ID,
				maxPatterns,
			)
		}

		requirementIDs := make(
			map[string]struct{},
			len(rule.Requirements),
		)
		for requirementIndex, requirement := range rule.Requirements {
			if !validID(requirement.ID) {
				return fmt.Errorf(
					"documentation-impact rule %q requirement %d has invalid id",
					rule.ID,
					requirementIndex+1,
				)
			}
			if _, duplicate := requirementIDs[requirement.ID]; duplicate {
				return fmt.Errorf(
					"documentation-impact rule %q requirement id %q is duplicated",
					rule.ID,
					requirement.ID,
				)
			}
			requirementIDs[requirement.ID] = struct{}{}
			if !validText(requirement.Description, 10, 1024) {
				return fmt.Errorf(
					"documentation-impact rule %q requirement %q has invalid description",
					rule.ID,
					requirement.ID,
				)
			}
			if (len(requirement.All) == 0) == (len(requirement.Any) == 0) {
				return fmt.Errorf(
					"documentation-impact rule %q requirement %q must declare exactly one of all or any",
					rule.ID,
					requirement.ID,
				)
			}
			patterns := requirement.All
			if len(patterns) == 0 {
				patterns = requirement.Any
			}
			if len(patterns) > maxPatterns {
				return fmt.Errorf(
					"documentation-impact rule %q requirement %q exceeds %d patterns",
					rule.ID,
					requirement.ID,
					maxPatterns,
				)
			}
			for patternIndex, pattern := range patterns {
				if err := validatePattern(pattern); err != nil {
					return fmt.Errorf(
						"documentation-impact rule %q requirement %q pattern %d: %w",
						rule.ID,
						requirement.ID,
						patternIndex+1,
						err,
					)
				}
			}
		}
	}
	return nil
}

func Evaluate(
	policy Policy,
	policyPath string,
	changedPaths []string,
) (Report, error) {
	if err := ValidatePolicy(policy); err != nil {
		return Report{}, err
	}
	if !safeRepositoryPath(policyPath) {
		return Report{}, errors.New(
			"documentation-impact policy evidence path is unsafe",
		)
	}

	changedSet := make(map[string]struct{}, len(changedPaths))
	for _, path := range changedPaths {
		if !safeRepositoryPath(path) {
			return Report{}, fmt.Errorf(
				"documentation-impact changed path is unsafe: %q",
				path,
			)
		}
		changedSet[path] = struct{}{}
	}
	changed := sortedKeys(changedSet)

	report := Report{
		SchemaVersion: PolicySchemaVersion,
		Status:        "PASS",
		PolicyPath:    policyPath,
		ChangedPaths:  changed,
	}

	rules := append([]Rule(nil), policy.Rules...)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})

	for _, rule := range rules {
		triggerPaths := matchingPaths(changed, rule.Triggers)
		if len(triggerPaths) == 0 {
			continue
		}
		ruleReport := RuleReport{
			ID:           rule.ID,
			Description:  rule.Description,
			TriggerPaths: triggerPaths,
			Status:       "PASS",
		}

		requirements := append(
			[]Requirement(nil),
			rule.Requirements...,
		)
		sort.Slice(requirements, func(i, j int) bool {
			return requirements[i].ID < requirements[j].ID
		})
		for _, requirement := range requirements {
			requirementReport := RequirementReport{
				ID:          requirement.ID,
				Description: requirement.Description,
				Status:      "PASS",
			}
			if len(requirement.All) > 0 {
				var matched []string
				for _, pattern := range requirement.All {
					paths := matchingPaths(changed, []Pattern{pattern})
					if len(paths) == 0 {
						requirementReport.Status = "FAIL"
						ruleReport.Status = "FAIL"
						report.Status = "FAIL"
						continue
					}
					matched = append(matched, paths...)
				}
				requirementReport.MatchedPaths = uniqueSorted(matched)
			} else {
				requirementReport.MatchedPaths = matchingPaths(
					changed,
					requirement.Any,
				)
				if len(requirementReport.MatchedPaths) == 0 {
					requirementReport.Status = "FAIL"
					ruleReport.Status = "FAIL"
					report.Status = "FAIL"
				}
			}
			ruleReport.Requirements = append(
				ruleReport.Requirements,
				requirementReport,
			)
		}
		report.Triggered = append(report.Triggered, ruleReport)
	}
	return report, nil
}

func matchingPaths(paths []string, patterns []Pattern) []string {
	var matched []string
	for _, path := range paths {
		for _, pattern := range patterns {
			if patternMatches(pattern, path) {
				matched = append(matched, path)
				break
			}
		}
	}
	return uniqueSorted(matched)
}

func patternMatches(pattern Pattern, path string) bool {
	if pattern.Path != "" {
		return path == pattern.Path
	}
	if !strings.HasPrefix(path, pattern.Prefix) {
		return false
	}
	return pattern.Suffix == "" || strings.HasSuffix(path, pattern.Suffix)
}

func validatePattern(pattern Pattern) error {
	exact := pattern.Path != ""
	prefixed := pattern.Prefix != ""
	if exact == prefixed {
		return errors.New(
			"pattern must declare exactly one of path or prefix",
		)
	}
	if exact {
		if pattern.Suffix != "" {
			return errors.New(
				"exact path pattern must not declare suffix",
			)
		}
		if !safeRepositoryPath(pattern.Path) {
			return errors.New("exact path is unsafe")
		}
		return nil
	}
	if !safePrefix(pattern.Prefix) {
		return errors.New("prefix is unsafe")
	}
	if pattern.Suffix != "" && !safeSuffix(pattern.Suffix) {
		return errors.New("suffix is unsafe")
	}
	return nil
}

func safeRepositoryPath(path string) bool {
	if path == "" ||
		len(path) > 4096 ||
		strings.Contains(path, `\`) ||
		strings.ContainsAny(path, "\x00\r\n") ||
		strings.HasPrefix(path, "/") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path &&
		clean != "." &&
		clean != ".." &&
		!strings.HasPrefix(clean, "../") &&
		!strings.HasPrefix(clean, ".git/") &&
		!strings.HasPrefix(clean, ".local/")
}

func safePrefix(prefix string) bool {
	if prefix == "" ||
		len(prefix) > 4096 ||
		strings.Contains(prefix, `\`) ||
		strings.ContainsAny(prefix, "\x00\r\n") ||
		strings.HasPrefix(prefix, "/") ||
		!strings.HasSuffix(prefix, "/") {
		return false
	}
	trimmed := strings.TrimSuffix(prefix, "/")
	return safeRepositoryPath(trimmed)
}

func safeSuffix(suffix string) bool {
	return len(suffix) <= 256 &&
		suffix != "" &&
		!strings.ContainsAny(suffix, "\x00\r\n/\\") &&
		!unicode.IsSpace(rune(suffix[0]))
}

func validID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && index > 0:
		case r == '-' && index > 0:
		default:
			return false
		}
	}
	return !strings.HasSuffix(value, "-")
}

func validText(value string, minimum, maximum int) bool {
	if len(value) < minimum ||
		len(value) > maximum ||
		strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if r == '\x00' ||
			r == '\r' ||
			r == '\n' ||
			unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func rejectSymlinkComponents(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil ||
		relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New(
			"documentation-impact policy path escapes the repository",
		)
	}

	current := root
	for _, component := range strings.Split(
		relative,
		string(filepath.Separator),
	) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New(
				"inspect documentation-impact policy path",
			)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New(
				"documentation-impact policy path contains a symbolic link",
			)
		}
	}
	return nil
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." ||
		(relative != ".." &&
			!strings.HasPrefix(
				relative,
				".."+string(filepath.Separator),
			))
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return sortedKeys(set)
}
