package releaseartifact

import "time"

const (
	StatusPass         = "PASS"
	StatusFail         = "FAIL"
	StatusNotPerformed = "NOT PERFORMED"

	AuthorizationGranted = "GRANTED"
	AuthorizationDenied  = "DENIED"
)

type Report struct {
	SchemaVersion          int              `json:"schema_version"`
	StartedAt              time.Time        `json:"started_at"`
	FinishedAt             time.Time        `json:"finished_at"`
	SourceMode             string           `json:"source_mode"`
	SourceLocation         string           `json:"source_location"`
	ReleaseTag             string           `json:"release_tag"`
	SourceCommit           string           `json:"source_commit"`
	ReleaseRecord          string           `json:"release_record"`
	SignedTag              string           `json:"signed_tag"`
	AssetAcquisition       string           `json:"asset_acquisition"`
	AssetInventory         string           `json:"asset_inventory"`
	PinDigests             string           `json:"pin_digests"`
	SHA256Manifest         string           `json:"sha256_manifest"`
	SHA512Manifest         string           `json:"sha512_manifest"`
	Provenance             string           `json:"provenance"`
	ExecutionAuthorization string           `json:"execution_authorization"`
	Failure                string           `json:"failure,omitempty"`
	Artifacts              []ArtifactResult `json:"artifacts"`
}

type ArtifactResult struct {
	Kind              string `json:"kind"`
	Name              string `json:"name"`
	OS                string `json:"os,omitempty"`
	Arch              string `json:"arch,omitempty"`
	Size              int64  `json:"size"`
	RemoteSize        int64  `json:"remote_size,omitempty"`
	ExpectedSHA256    string `json:"expected_sha256"`
	ObservedSHA256    string `json:"observed_sha256,omitempty"`
	ExpectedSHA512    string `json:"expected_sha512"`
	ObservedSHA512    string `json:"observed_sha512,omitempty"`
	SHA256Status      string `json:"sha256_status"`
	SHA512Status      string `json:"sha512_status"`
	SHA256Manifest    string `json:"sha256_manifest"`
	SHA512Manifest    string `json:"sha512_manifest"`
	ProvenanceBinding string `json:"provenance_binding"`
}

func newReport(mode, location, releaseTag, sourceCommit string, now time.Time) Report {
	return Report{
		SchemaVersion:          1,
		StartedAt:              now.UTC(),
		SourceMode:             mode,
		SourceLocation:         location,
		ReleaseTag:             releaseTag,
		SourceCommit:           sourceCommit,
		ReleaseRecord:          StatusNotPerformed,
		SignedTag:              StatusNotPerformed,
		AssetAcquisition:       StatusNotPerformed,
		AssetInventory:         StatusNotPerformed,
		PinDigests:             StatusNotPerformed,
		SHA256Manifest:         StatusNotPerformed,
		SHA512Manifest:         StatusNotPerformed,
		Provenance:             StatusNotPerformed,
		ExecutionAuthorization: AuthorizationDenied,
	}
}
