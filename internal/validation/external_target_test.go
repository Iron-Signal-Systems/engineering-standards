package validation

import (
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/repository"
)

func TestNewForIdentityUsesProvidedTargetWithoutRediscovery(t *testing.T) {
	identity := repository.Identity{
		Root:   "/tmp/explicit-target",
		Branch: "dev",
		Commit: "0123456789abcdef0123456789abcdef01234567",
		Origin: "git@github.com:Iron-Signal-Systems/example.git",
	}
	runner, err := NewForIdentity("commit", "isras --repo /tmp/explicit-target", identity)
	if err != nil {
		t.Fatal(err)
	}
	if runner.Root != identity.Root || runner.Identity.Commit != identity.Commit {
		t.Fatalf("runner target drifted: %#v", runner)
	}
	if runner.Mode != "commit" {
		t.Fatalf("mode = %q", runner.Mode)
	}
}

func TestNewForIdentityRejectsIncompleteIdentity(t *testing.T) {
	_, err := NewForIdentity("development", "isras", repository.Identity{})
	if err == nil {
		t.Fatal("incomplete target identity was accepted")
	}
}
