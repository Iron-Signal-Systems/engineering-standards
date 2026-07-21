package projectcommand

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const govulncheckExceptionRelativePath = ".isras/govulncheck-exceptions.json"

type govulncheckExceptionSource struct {
	Present     bool
	Path        string
	SHA256      string
	EvaluatedAt time.Time
	Document    govulncheckExceptionDocument
}

type GovulncheckExceptionsEvidence struct {
	Present       bool                                   `json:"present"`
	Path          string                                 `json:"path"`
	SHA256        string                                 `json:"sha256"`
	SchemaVersion int                                    `json:"schema_version"`
	EvaluatedAt   string                                 `json:"evaluated_at"`
	Used          []GovulncheckUsedExceptionEvidence     `json:"used"`
	Unused        []GovulncheckExceptionRecordEvidence   `json:"unused"`
	Unexcepted    []GovulncheckFindingOccurrenceEvidence `json:"unexcepted"`
	Unknown       []GovulncheckUnknownFindingEvidence    `json:"unknown"`
}

type GovulncheckUsedExceptionEvidence struct {
	Exception GovulncheckExceptionRecordEvidence   `json:"exception"`
	Finding   GovulncheckFindingOccurrenceEvidence `json:"finding"`
}

type GovulncheckExceptionRecordEvidence struct {
	AdvisoryID           string                                  `json:"advisory_id"`
	Scope                GovulncheckExceptionScopeEvidence       `json:"scope"`
	Justification        string                                  `json:"justification"`
	CompensatingControls []string                                `json:"compensating_controls"`
	Owner                string                                  `json:"owner"`
	Approval             GovulncheckExceptionApprovalEvidence    `json:"approval"`
	ExpiresAt            string                                  `json:"expires_at"`
	Remediation          GovulncheckExceptionRemediationEvidence `json:"remediation"`
}

type GovulncheckExceptionScopeEvidence struct {
	GoModPath   string `json:"go_mod_path"`
	ModulePath  string `json:"module_path"`
	PackagePath string `json:"package_path"`
	Symbol      string `json:"symbol"`
}

type GovulncheckExceptionApprovalEvidence struct {
	ApprovedBy string `json:"approved_by"`
	ApprovedAt string `json:"approved_at"`
	Record     string `json:"record"`
}

type GovulncheckExceptionRemediationEvidence struct {
	Owner      string `json:"owner"`
	TargetDate string `json:"target_date"`
	Plan       string `json:"plan"`
}

type GovulncheckFindingOccurrenceEvidence struct {
	AdvisoryID    string   `json:"advisory_id"`
	GoModPath     string   `json:"go_mod_path"`
	ModulePath    string   `json:"module_path"`
	PackagePath   string   `json:"package_path"`
	Symbol        string   `json:"symbol"`
	FixedVersions []string `json:"fixed_versions"`
	Occurrences   int      `json:"occurrences"`
}

type GovulncheckUnknownFindingEvidence struct {
	GoModPath   string `json:"go_mod_path"`
	Occurrences int    `json:"occurrences"`
}

func loadOptionalGovulncheckExceptions(
	root string,
	now time.Time,
) (govulncheckExceptionSource, error) {
	var source govulncheckExceptionSource
	if now.IsZero() {
		return source, errors.New(
			"govulncheck exception evaluation time is required",
		)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return source, errors.New(
			"resolve govulncheck exception repository root",
		)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	absolutePath := filepath.Join(
		absoluteRoot,
		filepath.FromSlash(govulncheckExceptionRelativePath),
	)
	source.Path = govulncheckExceptionRelativePath
	source.EvaluatedAt = now.UTC()
	source.Document = govulncheckExceptionDocument{
		SchemaVersion: govulncheckExceptionSchemaVersion,
		Exceptions:    []govulncheckException{},
	}

	parent := filepath.Dir(absolutePath)
	if err := rejectGovulncheckExceptionSymlinks(
		absoluteRoot,
		parent,
	); err != nil {
		return source, err
	}

	info, err := os.Lstat(absolutePath)
	if errors.Is(err, os.ErrNotExist) {
		return source, nil
	}
	if err != nil {
		return source, errors.New(
			"inspect optional govulncheck exception document",
		)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return source, errors.New(
			"govulncheck exception document must not be a symbolic link",
		)
	}
	if !info.Mode().IsRegular() {
		return source, errors.New(
			"govulncheck exception document must be a regular file",
		)
	}
	if info.Size() > maxGovulncheckExceptionBytes {
		return source, fmt.Errorf(
			"govulncheck exception document exceeds %d bytes",
			maxGovulncheckExceptionBytes,
		)
	}

	before, err := readBoundedGovulncheckExceptionFile(absolutePath)
	if err != nil {
		return source, err
	}
	document, err := loadGovulncheckExceptions(
		absoluteRoot,
		absolutePath,
		now,
	)
	if err != nil {
		return source, err
	}
	after, err := readBoundedGovulncheckExceptionFile(absolutePath)
	if err != nil {
		return source, err
	}
	beforeDigest := sha256.Sum256(before)
	afterDigest := sha256.Sum256(after)
	if beforeDigest != afterDigest {
		return source, errors.New(
			"govulncheck exception document changed during evaluation",
		)
	}

	source.Present = true
	source.SHA256 = hex.EncodeToString(afterDigest[:])
	source.Document = document
	return source, nil
}

func readBoundedGovulncheckExceptionFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New(
			"open govulncheck exception document for evidence",
		)
	}
	defer file.Close()

	data, err := io.ReadAll(
		io.LimitReader(file, maxGovulncheckExceptionBytes+1),
	)
	if err != nil {
		return nil, errors.New(
			"read govulncheck exception document for evidence",
		)
	}
	if len(data) > maxGovulncheckExceptionBytes {
		return nil, fmt.Errorf(
			"govulncheck exception document exceeds %d bytes",
			maxGovulncheckExceptionBytes,
		)
	}
	return data, nil
}

func projectGovulncheckExceptionEvidence(
	source govulncheckExceptionSource,
	reconciliation govulncheckExceptionReconciliation,
) (GovulncheckExceptionsEvidence, error) {
	if source.EvaluatedAt.IsZero() {
		return GovulncheckExceptionsEvidence{}, errors.New(
			"govulncheck exception evidence requires an evaluation time",
		)
	}
	if source.Path != govulncheckExceptionRelativePath {
		return GovulncheckExceptionsEvidence{}, errors.New(
			"govulncheck exception evidence path is not governed",
		)
	}
	if source.Document.SchemaVersion != govulncheckExceptionSchemaVersion {
		return GovulncheckExceptionsEvidence{}, errors.New(
			"govulncheck exception evidence schema version is unsupported",
		)
	}
	if source.Present {
		if len(source.SHA256) != 64 {
			return GovulncheckExceptionsEvidence{}, errors.New(
				"present govulncheck exception evidence requires SHA-256",
			)
		}
	} else if source.SHA256 != "" ||
		len(source.Document.Exceptions) != 0 {
		return GovulncheckExceptionsEvidence{}, errors.New(
			"absent govulncheck exception evidence cannot contain a digest or records",
		)
	}

	evidence := GovulncheckExceptionsEvidence{
		Present:       source.Present,
		Path:          source.Path,
		SHA256:        source.SHA256,
		SchemaVersion: source.Document.SchemaVersion,
		EvaluatedAt:   source.EvaluatedAt.UTC().Format(time.RFC3339Nano),
		Used:          make([]GovulncheckUsedExceptionEvidence, 0, len(reconciliation.Used)),
		Unused:        make([]GovulncheckExceptionRecordEvidence, 0, len(reconciliation.Unused)),
		Unexcepted:    make([]GovulncheckFindingOccurrenceEvidence, 0, len(reconciliation.Unexcepted)),
		Unknown:       make([]GovulncheckUnknownFindingEvidence, 0, len(reconciliation.Unknown)),
	}

	for _, used := range reconciliation.Used {
		evidence.Used = append(
			evidence.Used,
			GovulncheckUsedExceptionEvidence{
				Exception: projectGovulncheckExceptionRecord(
					used.Exception,
				),
				Finding: projectGovulncheckFindingOccurrence(
					used.Finding,
				),
			},
		)
	}
	for _, unused := range reconciliation.Unused {
		evidence.Unused = append(
			evidence.Unused,
			projectGovulncheckExceptionRecord(unused),
		)
	}
	for _, finding := range reconciliation.Unexcepted {
		evidence.Unexcepted = append(
			evidence.Unexcepted,
			projectGovulncheckFindingOccurrence(finding),
		)
	}
	for _, unknown := range reconciliation.Unknown {
		evidence.Unknown = append(
			evidence.Unknown,
			GovulncheckUnknownFindingEvidence{
				GoModPath:   unknown.GoModPath,
				Occurrences: unknown.Occurrences,
			},
		)
	}
	return evidence, nil
}

func projectGovulncheckExceptionRecord(
	exception govulncheckException,
) GovulncheckExceptionRecordEvidence {
	return GovulncheckExceptionRecordEvidence{
		AdvisoryID: exception.AdvisoryID,
		Scope: GovulncheckExceptionScopeEvidence{
			GoModPath:   exception.Scope.GoModPath,
			ModulePath:  exception.Scope.ModulePath,
			PackagePath: exception.Scope.PackagePath,
			Symbol:      exception.Scope.Symbol,
		},
		Justification: exception.Justification,
		CompensatingControls: append(
			[]string(nil),
			exception.CompensatingControls...,
		),
		Owner: exception.Owner,
		Approval: GovulncheckExceptionApprovalEvidence{
			ApprovedBy: exception.Approval.ApprovedBy,
			ApprovedAt: exception.Approval.ApprovedAt,
			Record:     exception.Approval.Record,
		},
		ExpiresAt: exception.ExpiresAt,
		Remediation: GovulncheckExceptionRemediationEvidence{
			Owner:      exception.Remediation.Owner,
			TargetDate: exception.Remediation.TargetDate,
			Plan:       exception.Remediation.Plan,
		},
	}
}

func projectGovulncheckFindingOccurrence(
	finding govulncheckFindingOccurrence,
) GovulncheckFindingOccurrenceEvidence {
	return GovulncheckFindingOccurrenceEvidence{
		AdvisoryID:    finding.AdvisoryID,
		GoModPath:     finding.GoModPath,
		ModulePath:    finding.ModulePath,
		PackagePath:   finding.PackagePath,
		Symbol:        finding.Symbol,
		FixedVersions: append([]string(nil), finding.FixedVersions...),
		Occurrences:   finding.Occurrences,
	}
}

func evaluateGovulncheckExceptionReconciliation(
	reconciliation govulncheckExceptionReconciliation,
) error {
	var failures []error

	if len(reconciliation.Unknown) > 0 {
		values := make([]string, 0, len(reconciliation.Unknown))
		for _, unknown := range reconciliation.Unknown {
			values = append(
				values,
				fmt.Sprintf(
					"%s=%d",
					unknown.GoModPath,
					unknown.Occurrences,
				),
			)
		}
		sort.Strings(values)
		failures = append(
			failures,
			fmt.Errorf(
				"govulncheck produced unknown-level findings: %s",
				strings.Join(values, "; "),
			),
		)
	}

	if len(reconciliation.Unexcepted) > 0 {
		values := make([]string, 0, len(reconciliation.Unexcepted))
		for _, finding := range reconciliation.Unexcepted {
			values = append(
				values,
				fmt.Sprintf(
					"%s[%s %s %s %s x%d]",
					finding.GoModPath,
					finding.AdvisoryID,
					finding.ModulePath,
					finding.PackagePath,
					finding.Symbol,
					finding.Occurrences,
				),
			)
		}
		sort.Strings(values)
		failures = append(
			failures,
			fmt.Errorf(
				"govulncheck found reachable vulnerabilities without exact governed exceptions: %s",
				strings.Join(values, "; "),
			),
		)
	}

	if len(reconciliation.Unused) > 0 {
		values := make([]string, 0, len(reconciliation.Unused))
		for _, exception := range reconciliation.Unused {
			values = append(
				values,
				fmt.Sprintf(
					"%s[%s %s %s %s]",
					exception.AdvisoryID,
					exception.Scope.GoModPath,
					exception.Scope.ModulePath,
					exception.Scope.PackagePath,
					exception.Scope.Symbol,
				),
			)
		}
		sort.Strings(values)
		failures = append(
			failures,
			fmt.Errorf(
				"govulncheck exception document contains unused or unmatched records: %s",
				strings.Join(values, "; "),
			),
		)
	}

	return errors.Join(failures...)
}

func renderGovulncheckExceptionEvidence(
	builder *strings.Builder,
	evidence *GovulncheckExceptionsEvidence,
) {
	if evidence == nil {
		return
	}
	fmt.Fprintf(builder, "Govulncheck exception document present: %t\n", evidence.Present)
	fmt.Fprintf(builder, "Govulncheck exception document path: %s\n", evidence.Path)
	fmt.Fprintf(builder, "Govulncheck exception document SHA-256: %s\n", evidence.SHA256)
	fmt.Fprintf(builder, "Govulncheck exception schema version: %d\n", evidence.SchemaVersion)
	fmt.Fprintf(builder, "Govulncheck exception evaluated at: %s\n", evidence.EvaluatedAt)
	fmt.Fprintf(builder, "Govulncheck used exception count: %d\n", len(evidence.Used))
	fmt.Fprintf(builder, "Govulncheck unused exception count: %d\n", len(evidence.Unused))
	fmt.Fprintf(builder, "Govulncheck unexcepted reachable finding count: %d\n", len(evidence.Unexcepted))
	fmt.Fprintf(builder, "Govulncheck unknown finding module count: %d\n", len(evidence.Unknown))

	for index, used := range evidence.Used {
		number := index + 1
		fmt.Fprintf(builder, "Govulncheck used exception %d advisory: %s\n", number, used.Exception.AdvisoryID)
		fmt.Fprintf(builder, "Govulncheck used exception %d go.mod: %s\n", number, used.Exception.Scope.GoModPath)
		fmt.Fprintf(builder, "Govulncheck used exception %d module: %s\n", number, used.Exception.Scope.ModulePath)
		fmt.Fprintf(builder, "Govulncheck used exception %d package: %s\n", number, used.Exception.Scope.PackagePath)
		fmt.Fprintf(builder, "Govulncheck used exception %d symbol: %s\n", number, used.Exception.Scope.Symbol)
		fmt.Fprintf(builder, "Govulncheck used exception %d owner: %s\n", number, used.Exception.Owner)
		fmt.Fprintf(builder, "Govulncheck used exception %d approved by: %s\n", number, used.Exception.Approval.ApprovedBy)
		fmt.Fprintf(builder, "Govulncheck used exception %d approval record: %s\n", number, used.Exception.Approval.Record)
		fmt.Fprintf(builder, "Govulncheck used exception %d expires at: %s\n", number, used.Exception.ExpiresAt)
		fmt.Fprintf(builder, "Govulncheck used exception %d remediation owner: %s\n", number, used.Exception.Remediation.Owner)
		fmt.Fprintf(builder, "Govulncheck used exception %d remediation target: %s\n", number, used.Exception.Remediation.TargetDate)
		fmt.Fprintf(builder, "Govulncheck used exception %d occurrence count: %d\n", number, used.Finding.Occurrences)
		fmt.Fprintf(builder, "Govulncheck used exception %d fixed versions: %s\n", number, strings.Join(used.Finding.FixedVersions, ", "))
	}
	for index, unused := range evidence.Unused {
		number := index + 1
		fmt.Fprintf(builder, "Govulncheck unused exception %d advisory: %s\n", number, unused.AdvisoryID)
		fmt.Fprintf(builder, "Govulncheck unused exception %d go.mod: %s\n", number, unused.Scope.GoModPath)
		fmt.Fprintf(builder, "Govulncheck unused exception %d module: %s\n", number, unused.Scope.ModulePath)
		fmt.Fprintf(builder, "Govulncheck unused exception %d package: %s\n", number, unused.Scope.PackagePath)
		fmt.Fprintf(builder, "Govulncheck unused exception %d symbol: %s\n", number, unused.Scope.Symbol)
	}
	for index, finding := range evidence.Unexcepted {
		number := index + 1
		fmt.Fprintf(builder, "Govulncheck unexcepted finding %d advisory: %s\n", number, finding.AdvisoryID)
		fmt.Fprintf(builder, "Govulncheck unexcepted finding %d go.mod: %s\n", number, finding.GoModPath)
		fmt.Fprintf(builder, "Govulncheck unexcepted finding %d module: %s\n", number, finding.ModulePath)
		fmt.Fprintf(builder, "Govulncheck unexcepted finding %d package: %s\n", number, finding.PackagePath)
		fmt.Fprintf(builder, "Govulncheck unexcepted finding %d symbol: %s\n", number, finding.Symbol)
		fmt.Fprintf(builder, "Govulncheck unexcepted finding %d occurrences: %d\n", number, finding.Occurrences)
	}
	for index, unknown := range evidence.Unknown {
		number := index + 1
		fmt.Fprintf(builder, "Govulncheck unknown finding %d go.mod: %s\n", number, unknown.GoModPath)
		fmt.Fprintf(builder, "Govulncheck unknown finding %d occurrences: %d\n", number, unknown.Occurrences)
	}
}
