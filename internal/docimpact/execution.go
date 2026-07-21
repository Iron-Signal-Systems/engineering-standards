package docimpact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PolicyRelativePath     = "validation/documentation-impact-policy.json"
	EvidenceDirectory      = ".local/validation/documentation-impact"
	EvidenceJSONName       = "execution.json"
	EvidenceTextName       = "execution.txt"
	ExecutionSchemaVersion = 1
)

type Request struct {
	Root       string
	BaseCommit string
	HeadCommit string
}

type Result struct {
	Evidence     ExecutionEvidence
	EvidenceJSON string
	EvidenceText string
}

type ExecutionEvidence struct {
	SchemaVersion int            `json:"schema_version"`
	Status        string         `json:"status"`
	Failure       string         `json:"failure,omitempty"`
	Policy        PolicyEvidence `json:"policy"`
	Comparison    Comparison     `json:"comparison"`
	Report        Report         `json:"report"`
}

type PolicyEvidence struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	SchemaVersion int    `json:"schema_version"`
}

func Run(ctx context.Context, request Request) (Result, error) {
	var result Result
	if ctx == nil {
		return result, errors.New(
			"documentation-impact context is required",
		)
	}

	root, err := filepath.Abs(request.Root)
	if err != nil {
		return result, errors.New(
			"resolve documentation-impact repository root",
		)
	}
	root = filepath.Clean(root)

	evidenceDirectory, err := ensureEvidenceDirectory(root)
	if err != nil {
		return result, err
	}
	result.EvidenceJSON = filepath.Join(
		evidenceDirectory,
		EvidenceJSONName,
	)
	result.EvidenceText = filepath.Join(
		evidenceDirectory,
		EvidenceTextName,
	)

	execution := ExecutionEvidence{
		SchemaVersion: ExecutionSchemaVersion,
		Status:        "FAIL",
		Policy: PolicyEvidence{
			Path: PolicyRelativePath,
		},
		Comparison: Comparison{
			RequestedBase: request.BaseCommit,
			RequestedHead: request.HeadCommit,
		},
		Report: Report{
			SchemaVersion: PolicySchemaVersion,
			Status:        "FAIL",
			PolicyPath:    PolicyRelativePath,
		},
	}

	policyPath := filepath.Join(
		root,
		filepath.FromSlash(PolicyRelativePath),
	)
	beforeDigest, digestErr := digestRegularFile(
		root,
		policyPath,
		maxPolicyBytes,
	)
	if digestErr != nil {
		return finalizeFailure(result, execution, digestErr)
	}
	execution.Policy.SHA256 = beforeDigest

	policy, loadErr := LoadPolicy(root, policyPath)
	if loadErr != nil {
		return finalizeFailure(result, execution, loadErr)
	}
	afterDigest, digestErr := digestRegularFile(
		root,
		policyPath,
		maxPolicyBytes,
	)
	if digestErr != nil {
		return finalizeFailure(result, execution, digestErr)
	}
	if beforeDigest != afterDigest {
		return finalizeFailure(
			result,
			execution,
			errors.New(
				"documentation-impact policy changed during evaluation",
			),
		)
	}
	execution.Policy.SHA256 = beforeDigest
	execution.Policy.SchemaVersion = policy.SchemaVersion

	comparison, compareErr := CollectComparison(
		ctx,
		root,
		request.BaseCommit,
		request.HeadCommit,
	)
	if compareErr != nil {
		return finalizeFailure(result, execution, compareErr)
	}
	execution.Comparison = comparison

	report, evaluateErr := Evaluate(
		policy,
		PolicyRelativePath,
		comparison.ChangedPaths,
	)
	if evaluateErr != nil {
		return finalizeFailure(result, execution, evaluateErr)
	}
	execution.Report = report
	execution.Status = report.Status
	if report.Status != "PASS" {
		execution.Failure =
			"documentation-impact requirements are not satisfied"
	}

	result.Evidence = execution
	if err := writeExecutionEvidence(result); err != nil {
		return result, err
	}
	if report.Status != "PASS" {
		return result, errors.New(execution.Failure)
	}
	return result, nil
}

func finalizeFailure(
	result Result,
	execution ExecutionEvidence,
	failure error,
) (Result, error) {
	execution.Status = "FAIL"
	execution.Failure = failure.Error()
	result.Evidence = execution
	if err := writeExecutionEvidence(result); err != nil {
		return result, errors.Join(failure, err)
	}
	return result, failure
}

func writeExecutionEvidence(result Result) error {
	data, err := json.MarshalIndent(result.Evidence, "", "  ")
	if err != nil {
		return errors.New(
			"encode documentation-impact JSON evidence",
		)
	}
	data = append(data, '\n')
	if err := writeAtomicPrivate(result.EvidenceJSON, data); err != nil {
		return err
	}
	if err := writeAtomicPrivate(
		result.EvidenceText,
		[]byte(renderExecutionText(result.Evidence)),
	); err != nil {
		return err
	}
	return nil
}

func renderExecutionText(evidence ExecutionEvidence) string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "DOCUMENTATION IMPACT EVIDENCE")
	fmt.Fprintln(&builder, "=============================")
	fmt.Fprintf(&builder, "Schema version: %d\n", evidence.SchemaVersion)
	fmt.Fprintf(&builder, "Status: %s\n", evidence.Status)
	if evidence.Failure != "" {
		fmt.Fprintf(&builder, "Failure: %s\n", evidence.Failure)
	}
	fmt.Fprintf(&builder, "Policy path: %s\n", evidence.Policy.Path)
	fmt.Fprintf(&builder, "Policy SHA-256: %s\n", evidence.Policy.SHA256)
	fmt.Fprintf(&builder, "Policy schema version: %d\n", evidence.Policy.SchemaVersion)
	fmt.Fprintf(&builder, "Requested base: %s\n", evidence.Comparison.RequestedBase)
	fmt.Fprintf(&builder, "Requested head: %s\n", evidence.Comparison.RequestedHead)
	fmt.Fprintf(&builder, "Resolved base: %s\n", evidence.Comparison.BaseCommit)
	fmt.Fprintf(&builder, "Resolved head: %s\n", evidence.Comparison.HeadCommit)
	fmt.Fprintf(&builder, "Merge base: %s\n", evidence.Comparison.MergeBase)
	fmt.Fprintf(&builder, "Changed paths: %d\n", len(evidence.Comparison.ChangedPaths))
	for _, path := range evidence.Comparison.ChangedPaths {
		fmt.Fprintf(&builder, "  - %s\n", path)
	}
	fmt.Fprintf(&builder, "Triggered rules: %d\n", len(evidence.Report.Triggered))
	for _, rule := range evidence.Report.Triggered {
		fmt.Fprintf(&builder, "Rule %s: %s\n", rule.ID, rule.Status)
		fmt.Fprintf(&builder, "  Description: %s\n", rule.Description)
		fmt.Fprintln(&builder, "  Trigger paths:")
		for _, path := range rule.TriggerPaths {
			fmt.Fprintf(&builder, "    - %s\n", path)
		}
		for _, requirement := range rule.Requirements {
			fmt.Fprintf(
				&builder,
				"  Requirement %s: %s\n",
				requirement.ID,
				requirement.Status,
			)
			fmt.Fprintf(
				&builder,
				"    Description: %s\n",
				requirement.Description,
			)
			if len(requirement.MatchedPaths) == 0 {
				fmt.Fprintln(&builder, "    Matched paths: none")
				continue
			}
			fmt.Fprintln(&builder, "    Matched paths:")
			for _, path := range requirement.MatchedPaths {
				fmt.Fprintf(&builder, "      - %s\n", path)
			}
		}
	}
	return builder.String()
}

func digestRegularFile(
	root string,
	path string,
	maximum int64,
) (string, error) {
	if !pathWithin(root, path) {
		return "", errors.New(
			"documentation-impact policy path escapes the repository",
		)
	}
	if err := rejectSymlinkComponents(root, path); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", errors.New(
			"inspect documentation-impact policy for digest",
		)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New(
			"documentation-impact policy digest source must be a regular file",
		)
	}
	if info.Size() > maximum {
		return "", errors.New(
			"documentation-impact policy exceeds its bounded size",
		)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New(
			"read documentation-impact policy for digest",
		)
	}
	if int64(len(data)) > maximum {
		return "", errors.New(
			"documentation-impact policy exceeds its bounded size",
		)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func ensureEvidenceDirectory(root string) (string, error) {
	current := root
	for _, component := range strings.Split(
		filepath.FromSlash(EvidenceDirectory),
		string(filepath.Separator),
	) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		switch {
		case errors.Is(err, os.ErrNotExist):
			if err := os.Mkdir(current, 0o700); err != nil {
				return "", errors.New(
					"create documentation-impact evidence directory",
				)
			}
		case err != nil:
			return "", errors.New(
				"inspect documentation-impact evidence directory",
			)
		case info.Mode()&os.ModeSymlink != 0:
			return "", errors.New(
				"documentation-impact evidence path contains a symbolic link",
			)
		case !info.IsDir():
			return "", errors.New(
				"documentation-impact evidence path contains a non-directory",
			)
		}
	}
	return filepath.Join(
		root,
		filepath.FromSlash(EvidenceDirectory),
	), nil
}

func writeAtomicPrivate(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".isras-docimpact-*")
	if err != nil {
		return errors.New(
			"create documentation-impact evidence temporary file",
		)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return errors.New(
			"set documentation-impact evidence permissions",
		)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return errors.New(
			"write documentation-impact evidence",
		)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return errors.New(
			"synchronize documentation-impact evidence",
		)
	}
	if err := temporary.Close(); err != nil {
		return errors.New(
			"close documentation-impact evidence",
		)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return errors.New(
			"finalize documentation-impact evidence",
		)
	}
	return nil
}
