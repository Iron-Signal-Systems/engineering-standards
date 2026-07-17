package releaseartifactbuild

import "time"

const (
	Profile          = "ISRAS-SD"
	SourceRepository = "github.com/Iron-Signal-Systems/engineering-standards"

	ValidatorName  = "isras-validator-linux-amd64"
	FrameworkName  = "isras-project-framework.tar.gz"
	ContractsName  = "isras-contracts.tar.gz"
	ProvenanceName = "provenance.json"
	SHA256Name     = "SHA256SUMS"
	SHA512Name     = "SHA512SUMS"

	FrameworkListPath = "release/framework-files.txt"
	ContractsListPath = "release/contract-files.txt"
)

type Options struct {
	Root               string
	OutputDirectory    string
	ExpectedVersion    string
	PublishedAt        string
	ValidationCampaign string
	ReleaseAuthority   string
}

type Result struct {
	SchemaVersion    int              `json:"schema_version"`
	GeneratedAt      time.Time        `json:"generated_at"`
	Profile          string           `json:"profile"`
	Version          string           `json:"version"`
	ReleaseTag       string           `json:"release_tag"`
	SourceRepository string           `json:"source_repository"`
	SourceCommit     string           `json:"source_commit"`
	GoVersion        string           `json:"go_version"`
	OutputDirectory  string           `json:"output_directory"`
	Artifacts        []ArtifactRecord `json:"artifacts"`
	EvidenceJSON     string           `json:"evidence_json"`
	EvidenceText     string           `json:"evidence_text"`
}

type ArtifactRecord struct {
	Kind   string `json:"kind"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
}

type provenance struct {
	SchemaVersion    int                  `json:"schema_version"`
	Profile          string               `json:"profile"`
	Version          string               `json:"version"`
	ReleaseTag       string               `json:"release_tag"`
	SourceRepository string               `json:"source_repository"`
	SourceCommit     string               `json:"source_commit"`
	Build            provenanceBuild      `json:"build"`
	Validation       provenanceValidation `json:"validation"`
	PublishedAt      string               `json:"published_at"`
	ReleaseAuthority string               `json:"release_authority"`
	Limitations      []string             `json:"limitations"`
	Artifacts        []provenanceArtifact `json:"artifacts"`
}

type provenanceBuild struct {
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

type provenanceValidation struct {
	Campaign string `json:"campaign"`
	Commit   string `json:"commit"`
	Status   string `json:"status"`
}

type provenanceArtifact struct {
	Kind   string `json:"kind"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
}
