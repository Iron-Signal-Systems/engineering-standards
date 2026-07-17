package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
)

func TestAssessCommitSignatureUnsignedDevelopmentWarnsWithoutAmend(t *testing.T) {
	assessment := assessCommitSignature(
		"development",
		"git@github.com:Iron-Signal-Systems/engineering-standards.git",
		"tree 0123456789abcdef\ncommitter Developer <developer@example.invalid> 0 +0000\n",
		executil.Result{Err: errors.New("exit status 1"), Stderr: "error: commit is unsigned"},
		"./.local/bin/isras-validate",
	)

	if assessment.Status != model.Warn {
		t.Fatalf("status = %s, want WARN", assessment.Status)
	}
	if !actionContains(assessment.Actions, "git commit -S") {
		t.Fatal("development remediation does not offer a new signed commit")
	}
	assertNoAmendAction(t, assessment.Actions)
}

func TestAssessCommitSignatureUnsignedCommitModeFailsWithoutRewriteCommand(t *testing.T) {
	assessment := assessCommitSignature(
		"commit",
		"git@github.com:Iron-Signal-Systems/engineering-standards.git",
		"tree 0123456789abcdef\ncommitter Developer <developer@example.invalid> 0 +0000\n",
		executil.Result{Err: errors.New("exit status 1"), Stderr: "error: commit is unsigned"},
		"./.local/bin/isras-validate",
	)

	if assessment.Status != model.Fail {
		t.Fatalf("status = %s, want FAIL", assessment.Status)
	}
	if !actionContains(assessment.Actions, "git branch -r --contains HEAD") {
		t.Fatal("commit-mode remediation does not check whether the commit is published")
	}
	assertNoAmendAction(t, assessment.Actions)
}

func TestAssessCommitSignatureRecognizesGitHubWebFlowMissingKey(t *testing.T) {
	raw := "tree 0123456789abcdef\n" +
		"committer GitHub <noreply@github.com> 0 +0000\n" +
		"gpgsig -----BEGIN PGP SIGNATURE-----\n signed-data\n -----END PGP SIGNATURE-----\n"
	assessment := assessCommitSignature(
		"commit",
		"git@github.com:Iron-Signal-Systems/engineering-standards.git",
		raw,
		executil.Result{Err: errors.New("exit status 1"), Stderr: "gpg: Can't check signature: No public key"},
		"./.local/bin/isras-validate",
	)

	if assessment.Status != model.Fail {
		t.Fatalf("status = %s, want FAIL", assessment.Status)
	}
	if !strings.Contains(assessment.Detail, "GitHub web-flow") {
		t.Fatalf("detail = %q, want GitHub web-flow diagnosis", assessment.Detail)
	}
	if !actionContains(assessment.Actions, "https://github.com/web-flow.gpg") {
		t.Fatal("GitHub web-flow remediation does not provide the published key source")
	}
	if !actionContains(assessment.Actions, "gpg --import") {
		t.Fatal("GitHub web-flow remediation does not provide a separate import action")
	}
	assertNoAmendAction(t, assessment.Actions)
}

func TestAssessCommitSignatureDoesNotAssumeUnknownOpenPGPKeyIsGitHub(t *testing.T) {
	raw := "tree 0123456789abcdef\n" +
		"committer Other <other@example.invalid> 0 +0000\n" +
		"gpgsig -----BEGIN PGP SIGNATURE-----\n signed-data\n -----END PGP SIGNATURE-----\n"
	assessment := assessCommitSignature(
		"commit",
		"git@example.invalid:organization/repository.git",
		raw,
		executil.Result{Err: errors.New("exit status 1"), Stderr: "gpg: Can't check signature: No public key"},
		"./.local/bin/isras-validate",
	)

	if actionContains(assessment.Actions, "https://github.com/web-flow.gpg") {
		t.Fatal("unknown OpenPGP signer incorrectly received GitHub-specific remediation")
	}
	if !actionContains(assessment.Actions, "gpg --list-keys") {
		t.Fatal("unknown OpenPGP signer does not receive generic trusted-key guidance")
	}
}

func TestAssessCommitSignatureSSHFailureShowsAllowedSignersReview(t *testing.T) {
	raw := "tree 0123456789abcdef\n" +
		"committer Developer <developer@example.invalid> 0 +0000\n" +
		"gpgsig -----BEGIN SSH SIGNATURE-----\n signed-data\n -----END SSH SIGNATURE-----\n"
	assessment := assessCommitSignature(
		"commit",
		"git@github.com:Iron-Signal-Systems/engineering-standards.git",
		raw,
		executil.Result{Err: errors.New("exit status 1"), Stderr: "Good signature with ED25519 key but no principal matched"},
		"./.local/bin/isras-validate",
	)

	if assessment.Status != model.Fail {
		t.Fatalf("status = %s, want FAIL", assessment.Status)
	}
	if !actionContains(assessment.Actions, "gpg.ssh.allowedSignersFile") {
		t.Fatal("SSH verification failure does not show allowed-signers review")
	}
	assertNoAmendAction(t, assessment.Actions)
}

func actionContains(actions []model.Action, fragment string) bool {
	for _, action := range actions {
		if strings.Contains(action.Command, fragment) {
			return true
		}
	}
	return false
}

func assertNoAmendAction(t *testing.T, actions []model.Action) {
	t.Helper()
	if actionContains(actions, "commit --amend") {
		t.Fatal("signature remediation must never recommend automatic commit amendment")
	}
}
