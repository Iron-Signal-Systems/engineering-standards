package projectcommand

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	govulncheckExceptionSchemaVersion = 1
	maxGovulncheckExceptionBytes      = 1024 * 1024
	maxGovulncheckExceptions          = 256
)

type govulncheckExceptionDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	Exceptions    []govulncheckException `json:"exceptions"`
}

type govulncheckException struct {
	AdvisoryID           string                          `json:"advisory_id"`
	Scope                govulncheckExceptionScope       `json:"scope"`
	Justification        string                          `json:"justification"`
	CompensatingControls []string                        `json:"compensating_controls"`
	Owner                string                          `json:"owner"`
	Approval             govulncheckExceptionApproval    `json:"approval"`
	ExpiresAt            string                          `json:"expires_at"`
	Remediation          govulncheckExceptionRemediation `json:"remediation"`
}

type govulncheckExceptionScope struct {
	GoModPath   string `json:"go_mod_path"`
	ModulePath  string `json:"module_path"`
	PackagePath string `json:"package_path"`
	Symbol      string `json:"symbol"`
}

type govulncheckExceptionApproval struct {
	ApprovedBy string `json:"approved_by"`
	ApprovedAt string `json:"approved_at"`
	Record     string `json:"record"`
}

type govulncheckExceptionRemediation struct {
	Owner      string `json:"owner"`
	TargetDate string `json:"target_date"`
	Plan       string `json:"plan"`
}

func loadGovulncheckExceptions(
	root string,
	path string,
	now time.Time,
) (govulncheckExceptionDocument, error) {
	if now.IsZero() {
		return govulncheckExceptionDocument{}, errors.New(
			"govulncheck exception evaluation time is required",
		)
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return govulncheckExceptionDocument{}, errors.New(
			"resolve govulncheck exception repository root",
		)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	if !filepath.IsAbs(path) {
		return govulncheckExceptionDocument{}, errors.New(
			"govulncheck exception path must be absolute",
		)
	}
	absolutePath := filepath.Clean(path)
	if !govulncheckExceptionPathWithin(absoluteRoot, absolutePath) {
		return govulncheckExceptionDocument{}, errors.New(
			"govulncheck exception path escapes the repository",
		)
	}
	if err := rejectGovulncheckExceptionSymlinks(
		absoluteRoot,
		absolutePath,
	); err != nil {
		return govulncheckExceptionDocument{}, err
	}

	info, err := os.Lstat(absolutePath)
	if err != nil {
		return govulncheckExceptionDocument{}, errors.New(
			"inspect govulncheck exception document",
		)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return govulncheckExceptionDocument{}, errors.New(
			"govulncheck exception document must not be a symbolic link",
		)
	}
	if !info.Mode().IsRegular() {
		return govulncheckExceptionDocument{}, errors.New(
			"govulncheck exception document must be a regular file",
		)
	}
	if info.Size() > maxGovulncheckExceptionBytes {
		return govulncheckExceptionDocument{}, fmt.Errorf(
			"govulncheck exception document exceeds %d bytes",
			maxGovulncheckExceptionBytes,
		)
	}

	file, err := os.Open(absolutePath)
	if err != nil {
		return govulncheckExceptionDocument{}, errors.New(
			"open govulncheck exception document",
		)
	}
	defer file.Close()

	decoder := json.NewDecoder(
		io.LimitReader(
			file,
			maxGovulncheckExceptionBytes+1,
		),
	)
	decoder.DisallowUnknownFields()

	var document govulncheckExceptionDocument
	if err := decoder.Decode(&document); err != nil {
		return govulncheckExceptionDocument{}, fmt.Errorf(
			"decode govulncheck exception document: %w",
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return govulncheckExceptionDocument{}, errors.New(
				"govulncheck exception document contains multiple JSON values",
			)
		}
		return govulncheckExceptionDocument{}, fmt.Errorf(
			"decode trailing govulncheck exception data: %w",
			err,
		)
	}

	if err := validateGovulncheckExceptionDocument(
		document,
		now.UTC(),
	); err != nil {
		return govulncheckExceptionDocument{}, err
	}

	sort.Slice(document.Exceptions, func(i, j int) bool {
		return govulncheckExceptionKey(
			document.Exceptions[i],
		) < govulncheckExceptionKey(
			document.Exceptions[j],
		)
	})
	return document, nil
}

func validateGovulncheckExceptionDocument(
	document govulncheckExceptionDocument,
	now time.Time,
) error {
	if document.SchemaVersion != govulncheckExceptionSchemaVersion {
		return fmt.Errorf(
			"govulncheck exception schema_version must be %d",
			govulncheckExceptionSchemaVersion,
		)
	}
	if len(document.Exceptions) > maxGovulncheckExceptions {
		return fmt.Errorf(
			"govulncheck exception count exceeds %d",
			maxGovulncheckExceptions,
		)
	}

	seen := make(map[string]struct{}, len(document.Exceptions))
	for index, exception := range document.Exceptions {
		if err := validateGovulncheckException(
			exception,
			now,
		); err != nil {
			return fmt.Errorf(
				"govulncheck exception %d: %w",
				index+1,
				err,
			)
		}
		key := govulncheckExceptionKey(exception)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf(
				"govulncheck exception %d duplicates advisory and scope %q",
				index+1,
				key,
			)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateGovulncheckException(
	exception govulncheckException,
	now time.Time,
) error {
	if !validGovulncheckAdvisoryID(exception.AdvisoryID) {
		return errors.New("advisory_id is invalid")
	}
	if !validGovulncheckGoModPath(exception.Scope.GoModPath) {
		return errors.New("scope.go_mod_path is invalid")
	}
	if !validGovulncheckScopeToken(
		exception.Scope.ModulePath,
		4096,
	) {
		return errors.New("scope.module_path is invalid")
	}
	if !validGovulncheckScopeToken(
		exception.Scope.PackagePath,
		4096,
	) {
		return errors.New("scope.package_path is invalid")
	}
	if !validGovulncheckSymbol(exception.Scope.Symbol) {
		return errors.New("scope.symbol is invalid")
	}
	if !validGovernanceText(
		exception.Justification,
		20,
		8192,
	) {
		return errors.New(
			"justification must contain 20-8192 single-line characters",
		)
	}
	if len(exception.CompensatingControls) == 0 ||
		len(exception.CompensatingControls) > 32 {
		return errors.New(
			"compensating_controls must contain 1-32 entries",
		)
	}
	controlSet := make(map[string]struct{}, len(exception.CompensatingControls))
	for _, control := range exception.CompensatingControls {
		if !validGovernanceText(control, 10, 2048) {
			return errors.New(
				"compensating_controls entries must contain 10-2048 single-line characters",
			)
		}
		if _, duplicate := controlSet[control]; duplicate {
			return errors.New(
				"compensating_controls contains a duplicate entry",
			)
		}
		controlSet[control] = struct{}{}
	}
	if !validGovernanceIdentity(exception.Owner) {
		return errors.New("owner is invalid")
	}
	if !validGovernanceIdentity(exception.Approval.ApprovedBy) {
		return errors.New("approval.approved_by is invalid")
	}
	if exception.Owner == exception.Approval.ApprovedBy {
		return errors.New(
			"approval.approved_by must be independent from owner",
		)
	}
	if !validGovernanceText(
		exception.Approval.Record,
		3,
		4096,
	) {
		return errors.New("approval.record is invalid")
	}

	approvedAt, err := parseCanonicalUTCTimestamp(
		exception.Approval.ApprovedAt,
	)
	if err != nil {
		return fmt.Errorf("approval.approved_at: %w", err)
	}
	expiresAt, err := parseCanonicalUTCTimestamp(
		exception.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("expires_at: %w", err)
	}
	if approvedAt.After(now) {
		return errors.New("approval.approved_at is in the future")
	}
	if !expiresAt.After(now) {
		return errors.New("exception is expired")
	}
	if !expiresAt.After(approvedAt) {
		return errors.New(
			"expires_at must be after approval.approved_at",
		)
	}

	if !validGovernanceIdentity(exception.Remediation.Owner) {
		return errors.New("remediation.owner is invalid")
	}
	targetDate, err := time.Parse(
		"2006-01-02",
		exception.Remediation.TargetDate,
	)
	if err != nil ||
		targetDate.Format("2006-01-02") !=
			exception.Remediation.TargetDate {
		return errors.New(
			"remediation.target_date must be canonical YYYY-MM-DD",
		)
	}
	expirationDate, err := time.Parse(
		"2006-01-02",
		expiresAt.UTC().Format("2006-01-02"),
	)
	if err != nil {
		return errors.New("derive exception expiration date")
	}
	if targetDate.After(expirationDate) {
		return errors.New(
			"remediation.target_date must not be after expires_at",
		)
	}
	if !validGovernanceText(
		exception.Remediation.Plan,
		20,
		8192,
	) {
		return errors.New(
			"remediation.plan must contain 20-8192 single-line characters",
		)
	}
	return nil
}

func govulncheckExceptionKey(
	exception govulncheckException,
) string {
	return strings.Join(
		[]string{
			exception.AdvisoryID,
			exception.Scope.GoModPath,
			exception.Scope.ModulePath,
			exception.Scope.PackagePath,
			exception.Scope.Symbol,
		},
		"\x00",
	)
}

func validGovulncheckAdvisoryID(value string) bool {
	if len(value) == 0 || len(value) > 256 {
		return false
	}
	for index, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case index > 0 && strings.ContainsRune("._:-", r):
		default:
			return false
		}
	}
	return true
}

func validGovulncheckGoModPath(value string) bool {
	if value == "" ||
		len(value) > 4096 ||
		strings.Contains(value, `\`) ||
		strings.ContainsAny(value, "*?[") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean != value ||
		clean == "." ||
		clean == ".." ||
		strings.HasPrefix(clean, "../") ||
		strings.HasPrefix(clean, "/") ||
		strings.HasPrefix(clean, ".local/") {
		return false
	}
	if clean == "go.mod" {
		return true
	}
	return strings.HasSuffix(clean, "/go.mod")
}

func validGovulncheckScopeToken(
	value string,
	maximum int,
) bool {
	if value == "" ||
		len(value) > maximum ||
		strings.Contains(value, `\`) ||
		strings.ContainsAny(value, "*?[") {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validGovulncheckSymbol(value string) bool {
	if value == "" ||
		len(value) > 1024 ||
		strings.Contains(value, `\`) ||
		strings.ContainsAny(value, "?[") {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	for index, r := range value {
		if r != '*' {
			continue
		}
		if index == 0 || value[index-1] != '(' {
			return false
		}
		if index+1 >= len(value) {
			return false
		}
	}
	return value != "*"
}

func validGovernanceIdentity(value string) bool {
	return validGovernanceText(value, 3, 512)
}

func validGovernanceText(
	value string,
	minimum int,
	maximum int,
) bool {
	if len(value) < minimum ||
		len(value) > maximum ||
		strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if r == '\r' ||
			r == '\n' ||
			r == '\x00' ||
			unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func parseCanonicalUTCTimestamp(
	value string,
) (time.Time, error) {
	if !strings.HasSuffix(value, "Z") {
		return time.Time{}, errors.New(
			"must use a canonical UTC timestamp ending in Z",
		)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, errors.New(
			"must be a valid RFC3339 timestamp",
		)
	}
	parsed = parsed.UTC()
	if parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, errors.New(
			"must use canonical RFC3339Nano UTC formatting",
		)
	}
	return parsed, nil
}

func rejectGovulncheckExceptionSymlinks(
	root string,
	path string,
) error {
	relative, err := filepath.Rel(root, path)
	if err != nil ||
		relative == ".." ||
		strings.HasPrefix(
			relative,
			".."+string(filepath.Separator),
		) {
		return errors.New(
			"govulncheck exception path escapes the repository",
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
				"inspect govulncheck exception path",
			)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New(
				"govulncheck exception path contains a symbolic link",
			)
		}
	}
	return nil
}

func govulncheckExceptionPathWithin(
	root string,
	path string,
) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." ||
		relative != ".." &&
			!strings.HasPrefix(
				relative,
				".."+string(filepath.Separator),
			)
}
