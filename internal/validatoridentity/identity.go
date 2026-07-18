package validatoridentity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	MetadataPath                = "validation/isras-validator-identity.json"
	Profile                     = "ISRAS-SD"
	SourceRepository            = "github.com/Iron-Signal-Systems/engineering-standards"
	OwnershipReference          = "reference-repository"
	OwnershipProjectOwnedExport = "project-owned-export"
	OwnershipReleaseArtifact    = "release-artifact"
)

var (
	versionPattern       = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)
	stableVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	commitPattern        = regexp.MustCompile(`^[0-9a-f]{40}$`)

	releaseVersion      string
	releaseTag          string
	releaseSourceCommit string
)

type Metadata struct {
	SchemaVersion    int    `json:"schema_version"`
	Profile          string `json:"profile"`
	StandardVersion  string `json:"standard_version"`
	Ownership        string `json:"ownership"`
	SourceRepository string `json:"source_repository"`
	SourceCommit     string `json:"source_commit,omitempty"`
	TargetModule     string `json:"target_module,omitempty"`
}

type Identity struct {
	Metadata
	ReleaseTag       string
	RepositoryCommit string
}

func Embedded() (Identity, bool, error) {
	return linkedReleaseIdentity()
}

func Load(root, repositoryCommit string) (Identity, error) {
	if identity, configured, err := Embedded(); configured || err != nil {
		return identity, err
	}
	if !commitPattern.MatchString(repositoryCommit) {
		return Identity{}, fmt.Errorf("invalid repository commit identity %q", repositoryCommit)
	}

	path := filepath.Join(root, filepath.FromSlash(MetadataPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return Identity{}, fmt.Errorf("read validator identity metadata: %w", err)
	}

	var metadata Metadata
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return Identity{}, fmt.Errorf("parse validator identity metadata: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Identity{}, err
	}
	if err := validateMetadata(root, metadata); err != nil {
		return Identity{}, err
	}

	identity := Identity{Metadata: metadata, RepositoryCommit: repositoryCommit}
	if identity.Ownership == OwnershipReference {
		identity.SourceCommit = repositoryCommit
	}
	return identity, nil
}

func linkedReleaseIdentity() (Identity, bool, error) {
	configured := releaseVersion != "" || releaseTag != "" || releaseSourceCommit != ""
	if !configured {
		return Identity{}, false, nil
	}
	if !stableVersionPattern.MatchString(releaseVersion) {
		return Identity{}, true, errors.New("embedded release validator version is invalid")
	}
	if releaseTag != "isras-v"+releaseVersion {
		return Identity{}, true, errors.New("embedded release validator tag does not match its version")
	}
	if !commitPattern.MatchString(releaseSourceCommit) || strings.Trim(releaseSourceCommit, "0") == "" {
		return Identity{}, true, errors.New("embedded release validator source commit is invalid")
	}
	return Identity{
		Metadata: Metadata{
			SchemaVersion:    1,
			Profile:          Profile,
			StandardVersion:  releaseVersion,
			Ownership:        OwnershipReleaseArtifact,
			SourceRepository: SourceRepository,
			SourceCommit:     releaseSourceCommit,
		},
		ReleaseTag:       releaseTag,
		RepositoryCommit: releaseSourceCommit,
	}, true, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("parse trailing validator identity metadata: %w", err)
	}
	return errors.New("validator identity metadata contains multiple JSON values")
}

func validateMetadata(root string, metadata Metadata) error {
	if metadata.SchemaVersion != 1 {
		return fmt.Errorf("unsupported validator identity schema version %d", metadata.SchemaVersion)
	}
	if metadata.Profile != Profile {
		return fmt.Errorf("unexpected validator profile %q", metadata.Profile)
	}
	if !versionPattern.MatchString(metadata.StandardVersion) {
		return fmt.Errorf("invalid validator standard version %q", metadata.StandardVersion)
	}
	if metadata.SourceRepository != SourceRepository {
		return fmt.Errorf("unexpected validator source repository %q", metadata.SourceRepository)
	}

	switch metadata.Ownership {
	case OwnershipReference:
		if metadata.SourceCommit != "" {
			return errors.New("reference validator identity must not pin a separate source commit")
		}
		if metadata.TargetModule != "" {
			return errors.New("reference validator identity must not declare a target module")
		}
		versionData, err := os.ReadFile(filepath.Join(root, "VERSION"))
		if err != nil {
			return fmt.Errorf("read reference VERSION: %w", err)
		}
		version := strings.TrimSpace(string(versionData))
		if version != metadata.StandardVersion {
			return fmt.Errorf("validator identity version %q does not match VERSION %q", metadata.StandardVersion, version)
		}
	case OwnershipProjectOwnedExport:
		if !commitPattern.MatchString(metadata.SourceCommit) {
			return fmt.Errorf("invalid exported validator source commit %q", metadata.SourceCommit)
		}
		if strings.TrimSpace(metadata.TargetModule) == "" || strings.ContainsAny(metadata.TargetModule, "\r\n\t ") {
			return fmt.Errorf("invalid exported validator target module %q", metadata.TargetModule)
		}
	default:
		return fmt.Errorf("unsupported validator ownership %q", metadata.Ownership)
	}
	return nil
}

func (identity Identity) Header() string {
	label := "reference"
	switch identity.Ownership {
	case OwnershipProjectOwnedExport:
		label = "project-owned export"
	case OwnershipReleaseArtifact:
		label = "release artifact"
	}
	return fmt.Sprintf("%s %s [%s]", identity.Profile, identity.StandardVersion, label)
}
