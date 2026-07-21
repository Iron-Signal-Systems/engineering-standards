package projectcommand

import (
	"strings"
	"testing"
)

func TestParseGovulncheckProtocolRetainsExactReachableFindings(
	t *testing.T,
) {
	stream := strings.Join([]string{
		`{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck","scanner_version":"v1.6.0","scan_level":"symbol","scan_mode":"source"}}`,
		`{"finding":{"osv":"GO-2026-0002","fixed_version":"v1.2.4","trace":[{"module":"example.com/dep","package":"example.com/dep/service","function":"Handle","receiver":"*Service"}]}}`,
		`{"finding":{"osv":"GO-2026-0001","trace":[{"module":"example.com/dep","package":"example.com/dep/service","function":"Open"}]}}`,
	}, "\n")

	summary, err := parseGovulncheckProtocol([]byte(stream))
	if err != nil {
		t.Fatal(err)
	}
	if summary.SymbolLevelFindings != 2 ||
		len(summary.ReachableFindings) != 2 {
		t.Fatalf("reachable summary = %#v", summary)
	}
	first := summary.ReachableFindings[0]
	second := summary.ReachableFindings[1]
	if first.AdvisoryID != "GO-2026-0001" ||
		first.Symbol != "Open" {
		t.Fatalf("first reachable finding = %#v", first)
	}
	if second.AdvisoryID != "GO-2026-0002" ||
		second.ModulePath != "example.com/dep" ||
		second.PackagePath != "example.com/dep/service" ||
		second.Symbol != "(*Service).Handle" ||
		second.FixedVersion != "v1.2.4" {
		t.Fatalf("second reachable finding = %#v", second)
	}
}

func TestCanonicalGovulncheckSymbol(t *testing.T) {
	tests := []struct {
		name     string
		receiver string
		function string
		want     string
		wantErr  bool
	}{
		{name: "function", function: "Open", want: "Open"},
		{name: "value receiver", receiver: "Service", function: "Handle", want: "Service.Handle"},
		{name: "pointer receiver", receiver: "*Service", function: "Handle", want: "(*Service).Handle"},
		{name: "canonical pointer receiver", receiver: "(*Service)", function: "Handle", want: "(*Service).Handle"},
		{name: "empty function", receiver: "Service", wantErr: true},
		{name: "wildcard receiver", receiver: "*", function: "Handle", wantErr: true},
		{name: "whitespace", receiver: "Bad Receiver", function: "Handle", wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := canonicalGovulncheckSymbol(
				testCase.receiver,
				testCase.function,
			)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("symbol = %q, want error", got)
				}
				return
			}
			if err != nil || got != testCase.want {
				t.Fatalf(
					"symbol = %q, error = %v, want %q",
					got,
					err,
					testCase.want,
				)
			}
		})
	}
}
