package releaseartifact

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

func TestGitHubVerifyPassesSignedReleaseAndExactAssets(t *testing.T) {
	fixture, pin := buildFixture(t)
	client := GitHubClient{Run: fakeGitHubRunner(t, fixture, pin, fakeGitHubOptions{})}
	report, err := client.Verify(context.Background(), pin)
	if err != nil {
		t.Fatal(err)
	}
	if report.ReleaseRecord != StatusPass || report.SignedTag != StatusPass || report.AssetAcquisition != StatusPass || report.ExecutionAuthorization != AuthorizationGranted {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestGitHubVerifyRejectsUnverifiedTag(t *testing.T) {
	fixture, pin := buildFixture(t)
	client := GitHubClient{Run: fakeGitHubRunner(t, fixture, pin, fakeGitHubOptions{unverifiedTag: true})}
	report, err := client.Verify(context.Background(), pin)
	if err == nil || !strings.Contains(err.Error(), "not verified") {
		t.Fatalf("expected signed-tag failure, got %v", err)
	}
	if report.ExecutionAuthorization != AuthorizationDenied {
		t.Fatalf("authorization = %s", report.ExecutionAuthorization)
	}
}

func TestGitHubVerifyRejectsReleaseAssetDigestDrift(t *testing.T) {
	fixture, pin := buildFixture(t)
	client := GitHubClient{Run: fakeGitHubRunner(t, fixture, pin, fakeGitHubOptions{digestDrift: true})}
	_, err := client.Verify(context.Background(), pin)
	if err == nil || !strings.Contains(err.Error(), "asset digest") {
		t.Fatalf("expected release asset digest failure, got %v", err)
	}
}

func TestGitHubVerifyRejectsDraftRelease(t *testing.T) {
	fixture, pin := buildFixture(t)
	client := GitHubClient{Run: fakeGitHubRunner(t, fixture, pin, fakeGitHubOptions{draft: true})}
	_, err := client.Verify(context.Background(), pin)
	if err == nil || !strings.Contains(err.Error(), "draft") {
		t.Fatalf("expected draft release failure, got %v", err)
	}
}

func TestGitHubVerifyRejectsLightweightTag(t *testing.T) {
	fixture, pin := buildFixture(t)
	client := GitHubClient{Run: fakeGitHubRunner(t, fixture, pin, fakeGitHubOptions{lightweightTag: true})}
	_, err := client.Verify(context.Background(), pin)
	if err == nil || !strings.Contains(err.Error(), "annotated") {
		t.Fatalf("expected annotated tag failure, got %v", err)
	}
}

func TestGitHubVerifyRejectsReleaseChangeDuringVerification(t *testing.T) {
	fixture, pin := buildFixture(t)
	client := GitHubClient{Run: fakeGitHubRunner(t, fixture, pin, fakeGitHubOptions{driftOnRecheck: true})}
	report, err := client.Verify(context.Background(), pin)
	if err == nil || !strings.Contains(err.Error(), "changed or failed final verification") {
		t.Fatalf("expected final release verification failure, got %v", err)
	}
	if report.ExecutionAuthorization != AuthorizationDenied {
		t.Fatalf("authorization = %s", report.ExecutionAuthorization)
	}
}

type fakeGitHubOptions struct {
	draft          bool
	unverifiedTag  bool
	digestDrift    bool
	lightweightTag bool
	driftOnRecheck bool
}

func fakeGitHubRunner(t *testing.T, fixture string, pin projectpin.Pin, options fakeGitHubOptions) CommandRunner {
	t.Helper()
	releaseCalls := 0
	return func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "/releases/tags/"):
			releaseCalls++
			assets := make([]releaseAsset, 0, len(pin.Artifacts))
			for index, artifact := range pin.Artifacts {
				info, err := os.Stat(filepath.Join(fixture, artifact.Name))
				if err != nil {
					t.Fatal(err)
				}
				digest := "sha256:" + artifact.SHA256
				if (options.digestDrift || (options.driftOnRecheck && releaseCalls > 1)) && index == 0 {
					digest = "sha256:" + strings.Repeat("a", 64)
				}
				assets = append(assets, releaseAsset{Name: artifact.Name, State: "uploaded", Size: info.Size(), Digest: digest})
			}
			return mustMarshal(t, releaseRecord{TagName: pin.Standard.ReleaseTag, Draft: options.draft, Assets: assets}), nil
		case strings.Contains(joined, "/git/ref/tags/"):
			var reference gitReference
			reference.Ref = "refs/tags/" + pin.Standard.ReleaseTag
			reference.Object.Type = "tag"
			if options.lightweightTag {
				reference.Object.Type = "commit"
			}
			reference.Object.SHA = strings.Repeat("b", 40)
			return mustMarshal(t, reference), nil
		case strings.Contains(joined, "/git/tags/"):
			var tag annotatedTag
			tag.Tag = pin.Standard.ReleaseTag
			tag.Object.Type = "commit"
			tag.Object.SHA = pin.Standard.SourceCommit
			tag.Verification.Verified = !options.unverifiedTag
			tag.Verification.Reason = "valid"
			tag.Verification.Signature = "signature"
			tag.Verification.Payload = "payload"
			tag.Verification.VerifiedAt = "2026-07-17T20:00:00Z"
			return mustMarshal(t, tag), nil
		case len(args) >= 2 && args[0] == "release" && args[1] == "download":
			directory := ""
			patterns := make(map[string]bool)
			for index := 0; index < len(args); index++ {
				switch args[index] {
				case "--dir":
					index++
					directory = args[index]
				case "--pattern":
					index++
					patterns[args[index]] = true
				}
			}
			if directory == "" || len(patterns) != len(pin.Artifacts) {
				return nil, errors.New("unexpected download arguments")
			}
			for _, artifact := range pin.Artifacts {
				if !patterns[artifact.Name] {
					return nil, errors.New("missing exact asset pattern")
				}
				data, err := os.ReadFile(filepath.Join(fixture, artifact.Name))
				if err != nil {
					return nil, err
				}
				if err := os.WriteFile(filepath.Join(directory, artifact.Name), data, 0o600); err != nil {
					return nil, err
				}
			}
			return nil, nil
		default:
			return nil, errors.New("unexpected GitHub command")
		}
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}
