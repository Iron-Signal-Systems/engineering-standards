package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

func TestRenderIdentityDistinguishesProjectOwnedExport(t *testing.T) {
	identity := validatoridentity.Identity{
		Metadata: validatoridentity.Metadata{
			SchemaVersion:    1,
			Profile:          validatoridentity.Profile,
			StandardVersion:  "0.1.1-development",
			Ownership:        validatoridentity.OwnershipProjectOwnedExport,
			SourceRepository: validatoridentity.SourceRepository,
			SourceCommit:     "89abcdef0123456789abcdef0123456789abcdef",
			TargetModule:     "github.com/Iron-Signal-Systems/iron-atlas",
		},
		RepositoryCommit: "0123456789abcdef0123456789abcdef01234567",
	}

	var output bytes.Buffer
	renderIdentity(&output, identity)

	for _, expected := range []string{
		"ISRAS VALIDATOR IDENTITY",
		"Standard version:  0.1.1-development",
		"Ownership:         project-owned-export",
		"Source repository: github.com/Iron-Signal-Systems/engineering-standards",
		"Source commit:     89abcdef0123456789abcdef0123456789abcdef",
		"Target module:     github.com/Iron-Signal-Systems/iron-atlas",
		"Repository commit: 0123456789abcdef0123456789abcdef01234567",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("identity output missing %q:\n%s", expected, output.String())
		}
	}
}
