package releaseartifact

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
)

const (
	maxArtifactSize      = int64(512 * 1024 * 1024)
	maxTotalArtifactSize = int64(2 * 1024 * 1024 * 1024)
	maxManifestSize      = int64(2 * 1024 * 1024)
	maxProvenanceSize    = int64(2 * 1024 * 1024)
)

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

func VerifyDirectory(pin projectpin.Pin, directory string) (Report, error) {
	now := time.Now().UTC()
	report := newReport("local-directory", filepath.Clean(directory), pin.Standard.ReleaseTag, pin.Standard.SourceCommit, now)
	report.AssetAcquisition = StatusNotPerformed

	if strings.TrimSpace(directory) == "" {
		return finishFailure(report, "source directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return finishFailure(report, "resolve source directory")
	}
	report.SourceLocation = absolute

	results, err := verifyInventoryAndDigests(pin, absolute)
	report.Artifacts = results
	if err != nil {
		if anyDigestFailure(results) {
			report.AssetInventory = StatusPass
			report.PinDigests = StatusFail
		} else {
			report.AssetInventory = StatusFail
		}
		return finishFailure(report, err.Error())
	}
	report.AssetInventory = StatusPass
	report.PinDigests = StatusPass

	if err := verifyManifest(pin, absolute, "sha256-manifest", 64, true, report.Artifacts); err != nil {
		report.SHA256Manifest = StatusFail
		return finishFailure(report, err.Error())
	}
	report.SHA256Manifest = StatusPass
	markManifestStatus(report.Artifacts, true)

	if err := verifyManifest(pin, absolute, "sha512-manifest", 128, false, report.Artifacts); err != nil {
		report.SHA512Manifest = StatusFail
		return finishFailure(report, err.Error())
	}
	report.SHA512Manifest = StatusPass
	markManifestStatus(report.Artifacts, false)

	if err := verifyProvenance(pin, absolute, report.Artifacts); err != nil {
		report.Provenance = StatusFail
		return finishFailure(report, err.Error())
	}
	report.Provenance = StatusPass
	markProvenanceStatus(report.Artifacts)

	report.FinishedAt = time.Now().UTC()
	return report, nil
}

func verifyInventoryAndDigests(pin projectpin.Pin, directory string) ([]ArtifactResult, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, errors.New("read artifact source directory")
	}

	declared := make(map[string]projectpin.Artifact, len(pin.Artifacts))
	for _, artifact := range pin.Artifacts {
		declared[artifact.Name] = artifact
	}
	if len(entries) != len(declared) {
		return nil, errors.New("artifact source directory does not contain the exact declared file set")
	}
	for _, entry := range entries {
		if _, ok := declared[entry.Name()]; !ok {
			return nil, errors.New("artifact source directory contains an undeclared entry")
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			return nil, errors.New("artifact source contains a symlink or directory")
		}
	}

	artifacts := append([]projectpin.Artifact(nil), pin.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	results := make([]ArtifactResult, 0, len(artifacts))
	var total int64
	for _, artifact := range artifacts {
		result, err := hashArtifact(directory, artifact)
		results = append(results, result)
		if err != nil {
			return results, err
		}
		total += result.Size
		if total > maxTotalArtifactSize {
			return results, errors.New("declared artifact set exceeds the total size limit")
		}
	}
	return results, nil
}

func hashArtifact(directory string, artifact projectpin.Artifact) (ArtifactResult, error) {
	result := ArtifactResult{
		Kind: artifact.Kind, Name: artifact.Name, OS: artifact.OS, Arch: artifact.Arch,
		ExpectedSHA256: artifact.SHA256, ExpectedSHA512: artifact.SHA512,
		SHA256Status: StatusNotPerformed, SHA512Status: StatusNotPerformed,
		SHA256Manifest: StatusNotPerformed, SHA512Manifest: StatusNotPerformed,
		ProvenanceBinding: StatusNotPerformed,
	}

	filePath := filepath.Join(directory, artifact.Name)
	before, err := os.Lstat(filePath)
	if err != nil {
		return result, errors.New("declared artifact is missing")
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return result, errors.New("declared artifact is not a regular file")
	}
	if before.Size() <= 0 || before.Size() > maxArtifactSize {
		return result, errors.New("declared artifact violates the size boundary")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return result, errors.New("open declared artifact")
	}
	defer file.Close()

	afterOpen, err := file.Stat()
	if err != nil || !os.SameFile(before, afterOpen) || !afterOpen.Mode().IsRegular() {
		return result, errors.New("declared artifact changed during open")
	}

	h256 := sha256.New()
	h512 := sha512.New()
	written, err := io.Copy(io.MultiWriter(h256, h512), io.LimitReader(file, maxArtifactSize+1))
	if err != nil {
		return result, errors.New("hash declared artifact")
	}
	if written != before.Size() || written > maxArtifactSize {
		return result, errors.New("declared artifact changed during hashing")
	}

	afterHash, err := os.Lstat(filePath)
	if err != nil || !os.SameFile(before, afterHash) || before.Size() != afterHash.Size() {
		return result, errors.New("declared artifact changed during hashing")
	}

	result.Size = written
	result.ObservedSHA256 = hex.EncodeToString(h256.Sum(nil))
	result.ObservedSHA512 = hex.EncodeToString(h512.Sum(nil))
	if secureEqual(result.ExpectedSHA256, result.ObservedSHA256) {
		result.SHA256Status = StatusPass
	} else {
		result.SHA256Status = StatusFail
	}
	if secureEqual(result.ExpectedSHA512, result.ObservedSHA512) {
		result.SHA512Status = StatusPass
	} else {
		result.SHA512Status = StatusFail
	}
	if result.SHA256Status != StatusPass || result.SHA512Status != StatusPass {
		return result, fmt.Errorf("artifact %s digest mismatch", artifact.Name)
	}
	return result, nil
}

func verifyManifest(pin projectpin.Pin, directory, kind string, digestLength int, sha256Manifest bool, results []ArtifactResult) error {
	manifestArtifact, ok := artifactByKind(pin.Artifacts, kind)
	if !ok {
		return errors.New("required checksum manifest is not declared")
	}
	data, err := readBoundedRegularFile(filepath.Join(directory, manifestArtifact.Name), maxManifestSize)
	if err != nil {
		return errors.New("read checksum manifest")
	}
	entries, err := parseManifest(data, digestLength)
	if err != nil {
		return err
	}

	expected := make(map[string]projectpin.Artifact)
	for _, artifact := range pin.Artifacts {
		if artifact.Kind == "sha256-manifest" || artifact.Kind == "sha512-manifest" {
			continue
		}
		expected[artifact.Name] = artifact
	}
	if len(entries) != len(expected) {
		return errors.New("checksum manifest does not contain the exact required artifact set")
	}
	for name, artifact := range expected {
		digest, ok := entries[name]
		if !ok {
			return errors.New("checksum manifest is missing a declared artifact")
		}
		wanted := artifact.SHA512
		observed := observedDigest(results, name, false)
		if sha256Manifest {
			wanted = artifact.SHA256
			observed = observedDigest(results, name, true)
		}
		if !secureEqual(digest, wanted) || !secureEqual(digest, observed) {
			return errors.New("checksum manifest digest does not match the project pin and local bytes")
		}
	}
	return nil
}

func parseManifest(data []byte, digestLength int) (map[string]string, error) {
	if bytes.Contains(data, []byte{'\r'}) {
		return nil, errors.New("checksum manifest contains carriage returns")
	}
	text := string(data)
	if !strings.HasSuffix(text, "\n") {
		return nil, errors.New("checksum manifest must end with a newline")
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	entries := make(map[string]string, len(lines))
	previous := ""
	for _, line := range lines {
		if line == "" {
			return nil, errors.New("checksum manifest contains a blank line")
		}
		parts := strings.Split(line, "  ")
		if len(parts) != 2 || len(parts[0]) != digestLength || parts[1] == "" {
			return nil, errors.New("checksum manifest contains a malformed entry")
		}
		if _, err := hex.DecodeString(parts[0]); err != nil || strings.ToLower(parts[0]) != parts[0] {
			return nil, errors.New("checksum manifest contains an invalid digest")
		}
		if filepath.Base(parts[1]) != parts[1] || strings.Contains(parts[1], "\\") {
			return nil, errors.New("checksum manifest contains an unsafe artifact name")
		}
		if previous != "" && parts[1] <= previous {
			return nil, errors.New("checksum manifest entries must be unique and sorted")
		}
		previous = parts[1]
		entries[parts[1]] = parts[0]
	}
	return entries, nil
}

func verifyProvenance(pin projectpin.Pin, directory string, results []ArtifactResult) error {
	artifact, ok := artifactByKind(pin.Artifacts, "provenance")
	if !ok {
		return errors.New("provenance artifact is not declared")
	}
	data, err := readBoundedRegularFile(filepath.Join(directory, artifact.Name), maxProvenanceSize)
	if err != nil {
		return errors.New("read release provenance")
	}
	if err := rejectDuplicateJSONFields(data); err != nil {
		return errors.New("release provenance contains duplicate or malformed JSON fields")
	}
	var value provenance
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return errors.New("parse release provenance")
	}
	if err := requireJSONEOF(decoder); err != nil {
		return errors.New("release provenance contains trailing data")
	}
	if value.SchemaVersion != 1 || value.Profile != pin.Standard.Profile || value.Version != pin.Standard.Version || value.ReleaseTag != pin.Standard.ReleaseTag || value.SourceRepository != pin.Standard.SourceRepository || value.SourceCommit != pin.Standard.SourceCommit {
		return errors.New("release provenance identity does not match the project pin")
	}
	if value.Build.GoVersion == "" || value.Build.GOOS == "" || value.Build.GOARCH == "" || value.Validation.Campaign == "" || value.Validation.Commit != pin.Standard.SourceCommit || value.Validation.Status != "PASS" || value.PublishedAt == "" || value.ReleaseAuthority == "" || len(value.Limitations) == 0 {
		return errors.New("release provenance is missing required build or validation evidence")
	}
	if _, err := time.Parse(time.RFC3339, value.PublishedAt); err != nil {
		return errors.New("release provenance contains an invalid publication time")
	}
	if !boundedProvenanceText(value.Build.GoVersion, 64) || !boundedProvenanceText(value.Build.GOOS, 32) || !boundedProvenanceText(value.Build.GOARCH, 32) || !boundedProvenanceText(value.Validation.Campaign, 128) || !boundedProvenanceText(value.ReleaseAuthority, 256) {
		return errors.New("release provenance contains invalid or oversized identity text")
	}
	seenLimitations := make(map[string]bool)
	if len(value.Limitations) > 32 {
		return errors.New("release provenance contains too many limitations")
	}
	for _, limitation := range value.Limitations {
		if !boundedProvenanceText(limitation, 1024) || seenLimitations[limitation] {
			return errors.New("release provenance contains an invalid or duplicate limitation")
		}
		seenLimitations[limitation] = true
	}

	expected := make([]projectpin.Artifact, 0)
	for _, item := range pin.Artifacts {
		if item.Kind == "provenance" || item.Kind == "sha256-manifest" || item.Kind == "sha512-manifest" {
			continue
		}
		expected = append(expected, item)
	}
	sort.Slice(expected, func(i, j int) bool { return expected[i].Name < expected[j].Name })
	if len(value.Artifacts) != len(expected) {
		return errors.New("release provenance does not contain the exact core artifact set")
	}
	previous := ""
	for index, declared := range value.Artifacts {
		if declared.Name <= previous {
			return errors.New("release provenance artifacts must be unique and sorted")
		}
		previous = declared.Name
		wanted := expected[index]
		if declared.Kind != wanted.Kind || declared.OS != wanted.OS || declared.Arch != wanted.Arch || declared.Name != wanted.Name || !secureEqual(declared.SHA256, wanted.SHA256) || !secureEqual(declared.SHA512, wanted.SHA512) || !secureEqual(declared.SHA256, observedDigest(results, wanted.Name, true)) || !secureEqual(declared.SHA512, observedDigest(results, wanted.Name, false)) {
			return errors.New("release provenance artifact binding does not match the project pin and local bytes")
		}
	}
	return nil
}

func boundedProvenanceText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && !strings.ContainsAny(value, "\x00\r\n")
}

func readBoundedRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, errors.New("file violates the regular-file or size boundary")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > limit {
		return nil, errors.New("file changed during read or exceeded its size boundary")
	}
	return data, nil
}

func artifactByKind(artifacts []projectpin.Artifact, kind string) (projectpin.Artifact, bool) {
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			return artifact, true
		}
	}
	return projectpin.Artifact{}, false
}

func observedDigest(results []ArtifactResult, name string, sha256Digest bool) string {
	for _, result := range results {
		if result.Name == name {
			if sha256Digest {
				return result.ObservedSHA256
			}
			return result.ObservedSHA512
		}
	}
	return ""
}

func markManifestStatus(results []ArtifactResult, sha256Manifest bool) {
	for index := range results {
		if results[index].Kind == "sha256-manifest" || results[index].Kind == "sha512-manifest" {
			continue
		}
		if sha256Manifest {
			results[index].SHA256Manifest = StatusPass
		} else {
			results[index].SHA512Manifest = StatusPass
		}
	}
}

func markProvenanceStatus(results []ArtifactResult) {
	for index := range results {
		if results[index].Kind == "provenance" || results[index].Kind == "sha256-manifest" || results[index].Kind == "sha512-manifest" {
			continue
		}
		results[index].ProvenanceBinding = StatusPass
	}
}

func anyDigestFailure(results []ArtifactResult) bool {
	for _, result := range results {
		if result.SHA256Status == StatusFail || result.SHA512Status == StatusFail {
			return true
		}
	}
	return false
}

func secureEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func finishFailure(report Report, message string) (Report, error) {
	report.Failure = message
	report.ExecutionAuthorization = AuthorizationDenied
	report.FinishedAt = time.Now().UTC()
	return report, errors.New(message)
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("multiple JSON values")
}

func rejectDuplicateJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("trailing JSON value")
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok || seen[key] {
				return errors.New("duplicate or non-string object key")
			}
			seen[key] = true
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return errors.New("unexpected JSON delimiter")
	}
}
