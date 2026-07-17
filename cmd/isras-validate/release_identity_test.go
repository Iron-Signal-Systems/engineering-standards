package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity"
)

func TestRenderIdentityReportsReleaseArtifactTag(t *testing.T) {
	identity := validatoridentity.Identity{
		Metadata: validatoridentity.Metadata{
			SchemaVersion:    1,
			Profile:          validatoridentity.Profile,
			StandardVersion:  "0.1.1",
			Ownership:        validatoridentity.OwnershipReleaseArtifact,
			SourceRepository: validatoridentity.SourceRepository,
			SourceCommit:     "89abcdef0123456789abcdef0123456789abcdef",
		},
		ReleaseTag:       "isras-v0.1.1",
		RepositoryCommit: "0123456789abcdef0123456789abcdef01234567",
	}
	var output bytes.Buffer
	renderIdentity(&output, identity)
	for _, expected := range []string{
		"Standard version:  0.1.1",
		"Ownership:         release-artifact",
		"Release tag:       isras-v0.1.1",
		"Source commit:     89abcdef0123456789abcdef0123456789abcdef",
		"Repository commit: 0123456789abcdef0123456789abcdef01234567",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("identity output missing %q:\n%s", expected, output.String())
		}
	}
}
