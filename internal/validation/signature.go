package validation

import (
	"context"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
)

type signatureAssessment struct {
	Status   model.Status
	Detail   string
	Expected string
	Actions  []model.Action
}

func (r *Runner) commitSignatureCheck(ctx context.Context) model.Check {
	verify := executil.Run(ctx, r.Root, "git", "verify-commit", "HEAD")
	if verify.Err == nil {
		return model.Check{
			Section:  "REPOSITORY",
			Name:     "Commit signature",
			Status:   model.Pass,
			Detail:   "verified",
			Started:  verify.Started,
			Finished: verify.Finished,
		}
	}

	raw := executil.Run(ctx, r.Root, "git", "cat-file", "commit", "HEAD")
	assessment := assessCommitSignature(r.Mode, r.Identity.Origin, raw.Stdout, verify, r.Command)
	check := model.Check{
		Section:  "REPOSITORY",
		Name:     "Commit signature",
		Status:   assessment.Status,
		Detail:   assessment.Detail,
		Actions:  assessment.Actions,
		Started:  verify.Started,
		Finished: verify.Finished,
	}
	if assessment.Status == model.Fail {
		check.LogPath = r.commandFailureLog(
			"commit signature",
			assessment.Expected,
			assessment.Detail,
			verify,
			assessment.Actions,
		)
	}
	return check
}

func assessCommitSignature(mode, origin, rawCommit string, verify executil.Result, command string) signatureAssessment {
	const expected = "HEAD should contain a cryptographically verifiable Git signature"

	review := model.Action{
		Label:       "READ ONLY",
		Description: "Review the current commit signature:",
		Command:     "git show --show-signature --no-patch HEAD",
	}
	rerun := model.Action{
		Label:       "READ ONLY",
		Description: "Rerun repository validation after correcting verification:",
		Command:     signatureRerunCommand(command, mode),
	}
	kind := signatureKind(rawCommit)
	observed := strings.ToLower(verify.Stdout + "\n" + verify.Stderr)

	if kind == "unsigned" {
		actions := []model.Action{
			review,
			{
				Label:       "READ ONLY",
				Description: "Determine whether the current commit is already present on a remote branch:",
				Command:     "git branch -r --contains HEAD",
			},
		}
		status := model.Fail
		detail := "unsigned exact commit"
		if mode == "development" {
			status = model.Warn
			detail = "current HEAD is unsigned; the next commit must be signed"
			actions = append(actions, model.Action{
				Label:       "MODIFIES GIT HISTORY",
				Description: "Create the next commit with the configured signing identity:",
				Command:     "git commit -S",
			})
		} else {
			actions = append(actions, model.Action{
				Label:       "READ ONLY",
				Description: "Review the signing and history-rewrite policy before correcting an exact commit:",
				Command:     "sed -n '1,240p' standards/RELEASES-AND-SIGNING.md",
			})
		}
		actions = append(actions, rerun)
		return signatureAssessment{Status: status, Detail: detail, Expected: expected, Actions: actions}
	}

	if kind == "OpenPGP" && strings.Contains(observed, "no public key") {
		actions := []model.Action{review}
		detail := "OpenPGP signature present; signer public key is unavailable locally"
		if githubWebCommit(origin, rawCommit) {
			detail = "GitHub web-flow signature present; GitHub public key is unavailable locally"
			actions = append(actions,
				model.Action{
					Label:       "NETWORK ACCESS — CREATES LOCAL KEY FILE",
					Description: "Download GitHub's published web-flow public key and inspect its fingerprint before import:",
					Command: "mkdir -p .local/validation/keys\n" +
						"curl --proto '=https' --tlsv1.2 -fsSL https://github.com/web-flow.gpg " +
						"-o .local/validation/keys/github-web-flow.gpg\n" +
						"gpg --show-keys --with-fingerprint .local/validation/keys/github-web-flow.gpg",
				},
				model.Action{
					Label:       "MODIFIES LOCAL GPG KEYRING",
					Description: "After reviewing the downloaded key, import it into the local GPG keyring:",
					Command:     "gpg --import .local/validation/keys/github-web-flow.gpg",
				},
			)
		} else {
			actions = append(actions, model.Action{
				Label:       "READ ONLY",
				Description: "Review locally available OpenPGP public keys before obtaining the signer key through a trusted channel:",
				Command:     "gpg --list-keys --keyid-format=long",
			})
		}
		actions = append(actions, rerun)
		return signatureAssessment{Status: model.Fail, Detail: detail, Expected: expected, Actions: actions}
	}

	if kind == "SSH" {
		actions := []model.Action{
			review,
			{
				Label:       "READ ONLY",
				Description: "Review the configured SSH allowed-signers file and its source:",
				Command: "git config --show-origin --get gpg.ssh.allowedSignersFile\n" +
					"allowed_signers=\"$(git config --get gpg.ssh.allowedSignersFile)\"\n" +
					"test -n \"$allowed_signers\" && sed -n '1,200p' \"$allowed_signers\"",
			},
			rerun,
		}
		return signatureAssessment{
			Status:   model.Fail,
			Detail:   "SSH signature present; signer trust or allowed-signers verification failed",
			Expected: expected,
			Actions:  actions,
		}
	}

	actions := []model.Action{
		review,
		{
			Label:       "READ ONLY",
			Description: "Review the commit object's signature header without changing history:",
			Command:     "git cat-file commit HEAD | sed -n '1,45p'",
		},
		rerun,
	}
	return signatureAssessment{
		Status:   model.Fail,
		Detail:   "signed commit could not be verified",
		Expected: expected,
		Actions:  actions,
	}
}

func signatureKind(rawCommit string) string {
	switch {
	case strings.Contains(rawCommit, "-----BEGIN SSH SIGNATURE-----"):
		return "SSH"
	case strings.Contains(rawCommit, "-----BEGIN PGP SIGNATURE-----"):
		return "OpenPGP"
	case strings.Contains(rawCommit, "\ngpgsig ") || strings.Contains(rawCommit, "\ngpgsig-sha256 "):
		return "signed"
	default:
		return "unsigned"
	}
}

func githubWebCommit(origin, rawCommit string) bool {
	if !githubOrigin(origin) {
		return false
	}
	commit := strings.ToLower(rawCommit)
	return strings.Contains(commit, "committer github <noreply@github.com>") ||
		strings.Contains(commit, "committer github <web-flow@users.noreply.github.com>")
}

func githubOrigin(origin string) bool {
	origin = strings.ToLower(strings.TrimSpace(origin))
	return strings.HasPrefix(origin, "git@github.com:") ||
		strings.HasPrefix(origin, "ssh://git@github.com/") ||
		strings.HasPrefix(origin, "https://github.com/")
}

func signatureRerunCommand(command, mode string) string {
	result := command + " repository"
	if mode != "" && mode != "development" {
		result += " --mode " + mode
	}
	return result
}
