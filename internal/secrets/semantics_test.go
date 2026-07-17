package secrets

import (
	"strconv"
	"testing"
)

func TestApprovedExternalSecretReferencesAreNotAssignments(t *testing.T) {
	schemes := []string{"sec" + "ret", "vault", "keyring", "credential"}
	for _, scheme := range schemes {
		data := []byte(`{"credential_ref":"` + scheme + `://iron-atlas/cisco-readonly"}`)
		if findings := scanFile("configs/device.example.json", data); len(findings) != 0 {
			t.Fatalf("approved scheme %q produced findings: %#v", scheme, findings)
		}
	}
}

func TestApprovedReferenceStillDetectsEmbeddedURLPassword(t *testing.T) {
	value := "Embedded" + "ReferenceBoundary987"
	data := []byte(("sec" + "ret") + "://operator:" + value + "@vault.example.invalid/item")
	requireSemanticRule(t, scanFile("configs/device.example.json", data), "embedded-url-password")
}

func TestUnknownSchemeInSensitiveFieldIsNotTrusted(t *testing.T) {
	data := scannerAssignment("client_"+"secret", "custom://iron-atlas/cisco-readonly")
	requireSemanticRule(t, scanFile("configs/device.example.json", data), "sensitive-assignment")
}

func TestGoIdentifierAndSelectorExpressionsAreNotLiterals(t *testing.T) {
	field := "Client" + "Secret"
	source := "package fixture\n\n" +
		"type configuration struct { " + field + " string }\n" +
		"func use(config configuration, testClientSecret string) {\n" +
		"\t_ = configuration{" + field + ": config." + field + "}\n" +
		"\t_ = configuration{" + field + ": testClientSecret}\n" +
		"}\n"

	if findings := scanFile("internal/fixture/fixture.go", []byte(source)); len(findings) != 0 {
		t.Fatalf("Go references produced findings: %#v", findings)
	}
}

func TestGoStringLiteralRemainsDetectable(t *testing.T) {
	field := "Client" + "Secret"
	value := "Hard" + "CodedBoundary987"
	source := "package fixture\n\n" +
		"type configuration struct { " + field + " string }\n" +
		"var _ = configuration{" + field + ": " + strconv.Quote(value) + "}\n"

	requireSemanticRule(t, scanFile("internal/fixture/fixture.go", []byte(source)), "sensitive-assignment")
}

func TestMalformedGoLikeAssignmentRemainsDetectable(t *testing.T) {
	data := scannerAssignment("pass"+"word", "ThisMustBeCorrected123")
	requireSemanticRule(t, scanFile("internal/fixture/malformed.go", data), "sensitive-assignment")
}

func TestGoIdentifierInsideCommentIsNotLiteral(t *testing.T) {
	field := "Client" + "Secret"
	source := "package fixture\n\n// " + field + ": config." + field + "\n"
	if findings := scanFile("internal/fixture/comment.go", []byte(source)); len(findings) != 0 {
		t.Fatalf("Go comment reference produced findings: %#v", findings)
	}
}

func TestHardcodedValueInsideGoCommentRemainsDetectable(t *testing.T) {
	field := "Client" + "Secret"
	value := "Documented" + "LiteralBoundary987"
	source := "package fixture\n\n// " + field + ": " + value + "\n"
	requireSemanticRule(t, scanFile("internal/fixture/comment.go", []byte(source)), "sensitive-assignment")
}

func TestSensitiveTextInsideGoStringIsScannedAsStringContent(t *testing.T) {
	field := "Client" + "Secret"
	value := "String" + "ContentBoundary987"
	payload := field + ": " + value
	source := "package fixture\n\nvar _ = " + strconv.Quote(payload) + "\n"
	requireSemanticRule(t, scanFile("internal/fixture/string.go", []byte(source)), "sensitive-assignment")
}

func TestShellDynamicAssignmentsAreNotCommittedLiterals(t *testing.T) {
	name := "pass" + "word"
	cases := map[string]string{
		"command substitution":  name + `="$(read-secret)"`,
		"environment reference": name + `="${PASSWORD}"`,
		"runtime concatenation": name + `="$(printf '%s%s' 'sec' 'ret')"`,
	}
	for testName, source := range cases {
		t.Run(testName, func(t *testing.T) {
			if findings := scanFile("test-framework/portable.sh", []byte(source)); len(findings) != 0 {
				t.Fatalf("dynamic shell assignment produced findings: %#v", findings)
			}
		})
	}
}

func TestShellLiteralAssignmentRemainsDetectable(t *testing.T) {
	data := scannerAssignment("pass"+"word", "Shell"+"LiteralBoundary987")
	requireSemanticRule(t, scanFile("test-framework/portable.sh", data), "sensitive-assignment")
}

func TestHardcodedAssignmentInsideShellCommandBodyRemainsDetectable(t *testing.T) {
	outer := "pass" + "word"
	inner := ("to" + "ken") + "=" + ("Command" + "BodyBoundary987")
	source := outer + `="$(printf '%s' '` + inner + `')"`
	requireSemanticRule(t, scanFile("test-framework/portable.sh", []byte(source)), "sensitive-assignment")
}

func requireSemanticRule(t *testing.T, findings []Finding, ruleName string) Finding {
	t.Helper()
	for _, finding := range findings {
		if finding.Rule == ruleName {
			return finding
		}
	}
	t.Fatalf("finding not present: rule=%s findings=%#v", ruleName, findings)
	return Finding{}
}
