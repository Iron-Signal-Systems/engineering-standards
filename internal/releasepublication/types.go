package releasepublication

import "time"

const (
	ActionCheck   Action = "check"
	ActionPublish Action = "publish"

	StatusPass         = "PASS"
	StatusFail         = "FAIL"
	StatusNotPerformed = "NOT PERFORMED"
)

type Action string

type Options struct {
	Root              string
	Action            Action
	ExpectedVersion   string
	Branch            string
	Remote            string
	GitHubRepository  string
	ArtifactDirectory string
	BuildEvidence     string
	NotesFile         string
	Title             string
	Confirm           bool
}

type Result struct {
	SchemaVersion     int              `json:"schema_version"`
	StartedAt         time.Time        `json:"started_at"`
	FinishedAt        time.Time        `json:"finished_at"`
	Action            string           `json:"action"`
	RepositoryRoot    string           `json:"repository_root"`
	SourceRepository  string           `json:"source_repository"`
	SourceCommit      string           `json:"source_commit"`
	Version           string           `json:"version"`
	ReleaseTag        string           `json:"release_tag"`
	GitHubRepository  string           `json:"github_repository"`
	ArtifactDirectory string           `json:"artifact_directory"`
	BuildEvidence     string           `json:"build_evidence"`
	NotesFile         string           `json:"notes_file"`
	Title             string           `json:"title"`
	LocalVerification string           `json:"local_verification"`
	RemoteTag         string           `json:"remote_tag"`
	ReleaseAbsence    string           `json:"release_absence"`
	DraftCreation     string           `json:"draft_creation"`
	AssetUpload       string           `json:"asset_upload"`
	DraftVerification string           `json:"draft_verification"`
	Publication       string           `json:"publication"`
	FinalVerification string           `json:"final_verification"`
	Cleanup           string           `json:"cleanup"`
	ReleaseID         int64            `json:"release_id,omitempty"`
	ReleaseURL        string           `json:"release_url,omitempty"`
	EvidenceJSON      string           `json:"evidence_json"`
	EvidenceText      string           `json:"evidence_text"`
	Failure           string           `json:"failure,omitempty"`
	Artifacts         []ArtifactResult `json:"artifacts"`
}

type ArtifactResult struct {
	Kind           string `json:"kind"`
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
	SHA512         string `json:"sha512"`
	RemoteAssetID  int64  `json:"remote_asset_id,omitempty"`
	RemoteSize     int64  `json:"remote_size,omitempty"`
	RemoteDigest   string `json:"remote_digest,omitempty"`
	UploadStatus   string `json:"upload_status"`
	DownloadStatus string `json:"download_status"`
}
