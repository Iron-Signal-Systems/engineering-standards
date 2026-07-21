package projectcommand

import (
	"strings"
	"testing"
)

func TestParseGovulncheckProtocolAcceptsConcatenatedMessages(t *testing.T) {
	stream := strings.Join([]string{
		`{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck","scanner_version":"v1.6.0","db":"https://vuln.go.dev","go_version":"go1.26.5","scan_level":"symbol","scan_mode":"source"}}`,
		`{"progress":{"message":"Scanning"}}`,
		`{"SBOM":{"go_version":"go1.26.5","roots":["example.com/root/...","example.com/root/..."],"modules":[{"path":"example.com/root","version":""},{"path":"example.com/dep","version":"v1.2.3"}]}}`,
		`{"osv":{"id":"GO-2026-0002"}}`,
		`{"osv":{"id":"GO-2026-0001"}}`,
		`{"finding":{"osv":"GO-2026-0001","trace":[{"module":"example.com/dep"}]}}`,
		`{"finding":{"osv":"GO-2026-0001","trace":[{"module":"example.com/dep","package":"example.com/dep/pkg"}]}}`,
		`{"finding":{"osv":"GO-2026-0002","trace":[{"module":"example.com/dep","package":"example.com/dep/pkg","function":"Vulnerable"}]}}`,
	}, "\n")

	summary, err := parseGovulncheckProtocol([]byte(stream))
	if err != nil {
		t.Fatal(err)
	}
	if summary.MessageCount != 8 {
		t.Fatalf("message count = %d", summary.MessageCount)
	}
	if summary.Config.ProtocolVersion != "v1.0.0" ||
		summary.Config.ScannerVersion != "v1.6.0" ||
		summary.Config.ScanLevel != "symbol" ||
		summary.Config.ScanMode != "source" {
		t.Fatalf("config = %#v", summary.Config)
	}
	if summary.ModuleLevelFindings != 1 ||
		summary.PackageLevelFindings != 1 ||
		summary.SymbolLevelFindings != 1 ||
		summary.UnknownLevelFindings != 0 {
		t.Fatalf("finding counts = %#v", summary)
	}
	if strings.Join(summary.OSVAdvisoryIDs, ",") !=
		"GO-2026-0001,GO-2026-0002" {
		t.Fatalf("OSV ids = %#v", summary.OSVAdvisoryIDs)
	}
	if strings.Join(summary.FindingAdvisoryIDs, ",") !=
		"GO-2026-0001,GO-2026-0002" {
		t.Fatalf("finding ids = %#v", summary.FindingAdvisoryIDs)
	}
	if len(summary.SBOMRoots) != 1 ||
		summary.SBOMRoots[0] != "example.com/root/..." {
		t.Fatalf("SBOM roots = %#v", summary.SBOMRoots)
	}
	if len(summary.SBOMModules) != 2 ||
		summary.SBOMModules[0].Path != "example.com/dep" ||
		summary.SBOMModules[1].Path != "example.com/root" {
		t.Fatalf("SBOM modules = %#v", summary.SBOMModules)
	}
}

func TestParseGovulncheckProtocolRejectsInvalidBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   string
	}{
		{name: "empty", stream: "", want: "stream is empty"},
		{name: "malformed", stream: `{"config":`, want: "decode govulncheck protocol"},
		{name: "non_object", stream: `"config"`, want: "must be an object"},
		{name: "null", stream: `null`, want: "must be an object"},
		{name: "zero_fields", stream: `{}`, want: "exactly one field"},
		{name: "multiple_fields", stream: `{"config":{"protocol_version":"v1.0.0"},"progress":{}}`, want: "exactly one field"},
		{name: "unknown_field", stream: `{"config":{"protocol_version":"v1.0.0"}}{"future":{}}`, want: "unsupported field"},
		{name: "config_not_first", stream: `{"progress":{}}`, want: "first message must be config"},
		{name: "duplicate_config", stream: `{"config":{"protocol_version":"v1.0.0"}}{"config":{"protocol_version":"v1.0.0"}}`, want: "duplicate config"},
		{name: "missing_protocol_version", stream: `{"config":{"scanner_name":"govulncheck"}}`, want: "protocol_version is required"},
		{name: "missing_osv_id", stream: `{"config":{"protocol_version":"v1.0.0"}}{"osv":{}}`, want: "OSV id is required"},
		{name: "missing_finding_osv", stream: `{"config":{"protocol_version":"v1.0.0"}}{"finding":{"trace":[{"module":"example.com/module"}]}}`, want: "finding OSV id is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseGovulncheckProtocol([]byte(test.stream))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestParseGovulncheckProtocolRecordsUnknownFindingLevel(t *testing.T) {
	stream := `{"config":{"protocol_version":"v1.0.0"}}` +
		`{"finding":{"osv":"GO-2026-0001","trace":[{}]}}`

	summary, err := parseGovulncheckProtocol([]byte(stream))
	if err != nil {
		t.Fatal(err)
	}
	if summary.UnknownLevelFindings != 1 {
		t.Fatalf("unknown findings = %d", summary.UnknownLevelFindings)
	}
}

func TestGovulncheckFindingClassification(t *testing.T) {
	tests := []struct {
		name    string
		finding govulncheckProtocolFinding
		want    govulncheckFindingLevel
	}{
		{name: "missing_trace", want: govulncheckFindingLevelUnknown},
		{name: "empty_frame", finding: govulncheckProtocolFinding{Trace: []govulncheckProtocolFrame{{}}}, want: govulncheckFindingLevelUnknown},
		{name: "module", finding: govulncheckProtocolFinding{Trace: []govulncheckProtocolFrame{{Module: "example.com/module"}}}, want: govulncheckFindingLevelModule},
		{name: "package", finding: govulncheckProtocolFinding{Trace: []govulncheckProtocolFrame{{Module: "example.com/module", Package: "example.com/module/pkg"}}}, want: govulncheckFindingLevelPackage},
		{name: "symbol", finding: govulncheckProtocolFinding{Trace: []govulncheckProtocolFrame{{Module: "example.com/module", Package: "example.com/module/pkg", Function: "Vulnerable"}}}, want: govulncheckFindingLevelSymbol},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := govulncheckFindingClassification(test.finding); got != test.want {
				t.Fatalf("classification = %q, want %q", got, test.want)
			}
		})
	}
}
