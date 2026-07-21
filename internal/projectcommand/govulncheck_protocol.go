package projectcommand

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const maxGovulncheckProtocolBytes = 16 * 1024 * 1024

type govulncheckFindingLevel string

const (
	govulncheckFindingLevelModule  govulncheckFindingLevel = "module"
	govulncheckFindingLevelPackage govulncheckFindingLevel = "package"
	govulncheckFindingLevelSymbol  govulncheckFindingLevel = "symbol"
	govulncheckFindingLevelUnknown govulncheckFindingLevel = "unknown"
)

type govulncheckProtocolConfig struct {
	ProtocolVersion      string `json:"protocol_version"`
	ScannerName          string `json:"scanner_name,omitempty"`
	ScannerVersion       string `json:"scanner_version,omitempty"`
	Database             string `json:"db,omitempty"`
	DatabaseLastModified string `json:"db_last_modified,omitempty"`
	GoVersion            string `json:"go_version,omitempty"`
	ScanLevel            string `json:"scan_level,omitempty"`
	ScanMode             string `json:"scan_mode,omitempty"`
}

type govulncheckProtocolModule struct {
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
}

type govulncheckProtocolSBOM struct {
	GoVersion string                      `json:"go_version,omitempty"`
	Modules   []govulncheckProtocolModule `json:"modules,omitempty"`
	Roots     []string                    `json:"roots,omitempty"`
}

type govulncheckProtocolFrame struct {
	Module   string `json:"module"`
	Version  string `json:"version,omitempty"`
	Package  string `json:"package,omitempty"`
	Function string `json:"function,omitempty"`
	Receiver string `json:"receiver,omitempty"`
}

type govulncheckProtocolFinding struct {
	OSV          string                     `json:"osv,omitempty"`
	FixedVersion string                     `json:"fixed_version,omitempty"`
	Trace        []govulncheckProtocolFrame `json:"trace,omitempty"`
}

type govulncheckReachableFinding struct {
	AdvisoryID   string
	ModulePath   string
	PackagePath  string
	Symbol       string
	FixedVersion string
}

type govulncheckProtocolOSV struct {
	ID string `json:"id"`
}

type govulncheckProtocolSummary struct {
	Config               govulncheckProtocolConfig
	MessageCount         int
	ConfigMessages       int
	ProgressMessages     int
	SBOMMessages         int
	OSVMessages          int
	FindingMessages      int
	SBOMRoots            []string
	SBOMModules          []govulncheckProtocolModule
	OSVAdvisoryIDs       []string
	FindingAdvisoryIDs   []string
	ModuleLevelFindings  int
	PackageLevelFindings int
	SymbolLevelFindings  int
	UnknownLevelFindings int
	ReachableFindings    []govulncheckReachableFinding
}

func parseGovulncheckProtocol(data []byte) (govulncheckProtocolSummary, error) {
	var summary govulncheckProtocolSummary
	if len(data) == 0 {
		return summary, errors.New("govulncheck protocol stream is empty")
	}
	if len(data) > maxGovulncheckProtocolBytes {
		return summary, fmt.Errorf(
			"govulncheck protocol stream exceeds %d bytes",
			maxGovulncheckProtocolBytes,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	osvIDs := make(map[string]struct{})
	findingIDs := make(map[string]struct{})
	roots := make(map[string]struct{})
	modules := make(map[string]govulncheckProtocolModule)

	for {
		var raw json.RawMessage
		err := decoder.Decode(&raw)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return summary, fmt.Errorf(
				"decode govulncheck protocol message %d: %w",
				summary.MessageCount+1,
				err,
			)
		}

		var message map[string]json.RawMessage
		if err := json.Unmarshal(raw, &message); err != nil || message == nil {
			return summary, fmt.Errorf(
				"govulncheck protocol message %d must be an object",
				summary.MessageCount+1,
			)
		}
		if len(message) != 1 {
			return summary, fmt.Errorf(
				"govulncheck protocol message %d must contain exactly one field",
				summary.MessageCount+1,
			)
		}

		var field string
		var payload json.RawMessage
		for key, value := range message {
			field = key
			payload = value
		}

		if summary.MessageCount == 0 && field != "config" {
			return summary, errors.New(
				"govulncheck protocol first message must be config",
			)
		}

		switch field {
		case "config":
			if summary.ConfigMessages != 0 {
				return summary, errors.New(
					"govulncheck protocol contains duplicate config messages",
				)
			}
			if err := json.Unmarshal(payload, &summary.Config); err != nil {
				return summary, fmt.Errorf(
					"decode govulncheck config: %w",
					err,
				)
			}
			if summary.Config.ProtocolVersion == "" {
				return summary, errors.New(
					"govulncheck config protocol_version is required",
				)
			}
			summary.ConfigMessages++

		case "progress":
			if summary.ConfigMessages == 0 {
				return summary, errors.New(
					"govulncheck protocol config must occur first",
				)
			}
			var progress struct {
				Message string `json:"message,omitempty"`
			}
			if err := json.Unmarshal(payload, &progress); err != nil {
				return summary, fmt.Errorf(
					"decode govulncheck progress: %w",
					err,
				)
			}
			summary.ProgressMessages++

		case "SBOM":
			if summary.ConfigMessages == 0 {
				return summary, errors.New(
					"govulncheck protocol config must occur first",
				)
			}
			var sbom govulncheckProtocolSBOM
			if err := json.Unmarshal(payload, &sbom); err != nil {
				return summary, fmt.Errorf(
					"decode govulncheck SBOM: %w",
					err,
				)
			}
			for _, root := range sbom.Roots {
				if root != "" {
					roots[root] = struct{}{}
				}
			}
			for _, module := range sbom.Modules {
				if module.Path == "" {
					continue
				}
				modules[module.Path+"\x00"+module.Version] = module
			}
			summary.SBOMMessages++

		case "osv":
			if summary.ConfigMessages == 0 {
				return summary, errors.New(
					"govulncheck protocol config must occur first",
				)
			}
			var osv govulncheckProtocolOSV
			if err := json.Unmarshal(payload, &osv); err != nil {
				return summary, fmt.Errorf(
					"decode govulncheck OSV: %w",
					err,
				)
			}
			if osv.ID == "" {
				return summary, errors.New(
					"govulncheck OSV id is required",
				)
			}
			osvIDs[osv.ID] = struct{}{}
			summary.OSVMessages++

		case "finding":
			if summary.ConfigMessages == 0 {
				return summary, errors.New(
					"govulncheck protocol config must occur first",
				)
			}
			var finding govulncheckProtocolFinding
			if err := json.Unmarshal(payload, &finding); err != nil {
				return summary, fmt.Errorf(
					"decode govulncheck finding: %w",
					err,
				)
			}
			if finding.OSV == "" {
				return summary, errors.New(
					"govulncheck finding OSV id is required",
				)
			}
			findingIDs[finding.OSV] = struct{}{}
			switch govulncheckFindingClassification(finding) {
			case govulncheckFindingLevelModule:
				summary.ModuleLevelFindings++
			case govulncheckFindingLevelPackage:
				summary.PackageLevelFindings++
			case govulncheckFindingLevelSymbol:
				reachable, err := exactGovulncheckReachableFinding(finding)
				if err != nil {
					return summary, fmt.Errorf(
						"decode exact govulncheck reachable finding: %w",
						err,
					)
				}
				summary.SymbolLevelFindings++
				summary.ReachableFindings = append(
					summary.ReachableFindings,
					reachable,
				)
			default:
				summary.UnknownLevelFindings++
			}
			summary.FindingMessages++

		default:
			return summary, fmt.Errorf(
				"govulncheck protocol message %d contains unsupported field %q",
				summary.MessageCount+1,
				field,
			)
		}

		summary.MessageCount++
	}

	if summary.ConfigMessages == 0 {
		return summary, errors.New(
			"govulncheck protocol config message is required",
		)
	}

	summary.SBOMRoots = sortedStringSet(roots)
	summary.SBOMModules = sortedGovulncheckModules(modules)
	summary.OSVAdvisoryIDs = sortedStringSet(osvIDs)
	summary.FindingAdvisoryIDs = sortedStringSet(findingIDs)
	sortGovulncheckReachableFindings(summary.ReachableFindings)
	return summary, nil
}

func govulncheckFindingClassification(
	finding govulncheckProtocolFinding,
) govulncheckFindingLevel {
	if len(finding.Trace) == 0 {
		return govulncheckFindingLevelUnknown
	}
	frame := finding.Trace[0]
	if frame.Function != "" {
		return govulncheckFindingLevelSymbol
	}
	if frame.Package != "" {
		return govulncheckFindingLevelPackage
	}
	if frame.Module != "" {
		return govulncheckFindingLevelModule
	}
	return govulncheckFindingLevelUnknown
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedGovulncheckModules(
	values map[string]govulncheckProtocolModule,
) []govulncheckProtocolModule {
	result := make([]govulncheckProtocolModule, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Path == result[right].Path {
			return result[left].Version < result[right].Version
		}
		return result[left].Path < result[right].Path
	})
	return result
}
func exactGovulncheckReachableFinding(
	finding govulncheckProtocolFinding,
) (govulncheckReachableFinding, error) {
	if govulncheckFindingClassification(finding) !=
		govulncheckFindingLevelSymbol {
		return govulncheckReachableFinding{}, errors.New(
			"finding is not symbol-level",
		)
	}
	frame := finding.Trace[0]
	if finding.OSV == "" ||
		frame.Module == "" ||
		frame.Package == "" ||
		frame.Function == "" {
		return govulncheckReachableFinding{}, errors.New(
			"symbol-level finding lacks exact advisory, module, package, or function identity",
		)
	}
	symbol, err := canonicalGovulncheckSymbol(
		frame.Receiver,
		frame.Function,
	)
	if err != nil {
		return govulncheckReachableFinding{}, err
	}
	return govulncheckReachableFinding{
		AdvisoryID:   finding.OSV,
		ModulePath:   frame.Module,
		PackagePath:  frame.Package,
		Symbol:       symbol,
		FixedVersion: finding.FixedVersion,
	}, nil
}

func canonicalGovulncheckSymbol(
	receiver string,
	function string,
) (string, error) {
	if !validGovulncheckProtocolIdentifier(function) {
		return "", errors.New(
			"govulncheck function identity is invalid",
		)
	}
	if receiver == "" {
		return function, nil
	}
	if strings.ContainsAny(receiver, "\x00\r\n\t ") ||
		strings.ContainsAny(receiver, "?[\\") {
		return "", errors.New(
			"govulncheck receiver identity is invalid",
		)
	}
	switch {
	case strings.HasPrefix(receiver, "(*") &&
		strings.HasSuffix(receiver, ")"):
		return receiver + "." + function, nil
	case strings.HasPrefix(receiver, "*"):
		if len(receiver) == 1 {
			return "", errors.New(
				"govulncheck receiver identity is invalid",
			)
		}
		return "(" + receiver + ")." + function, nil
	default:
		return receiver + "." + function, nil
	}
}

func validGovulncheckProtocolIdentifier(value string) bool {
	if value == "" || len(value) > 1024 {
		return false
	}
	if strings.ContainsAny(value, "\x00\r\n\t *?[\\") {
		return false
	}
	return true
}

func govulncheckReachableFindingKey(
	finding govulncheckReachableFinding,
) string {
	return strings.Join(
		[]string{
			finding.AdvisoryID,
			finding.ModulePath,
			finding.PackagePath,
			finding.Symbol,
			finding.FixedVersion,
		},
		"\x00",
	)
}

func sortGovulncheckReachableFindings(
	findings []govulncheckReachableFinding,
) {
	sort.Slice(findings, func(i, j int) bool {
		return govulncheckReachableFindingKey(
			findings[i],
		) < govulncheckReachableFindingKey(
			findings[j],
		)
	})
}
