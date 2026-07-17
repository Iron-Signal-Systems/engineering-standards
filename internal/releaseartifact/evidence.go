package releaseartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func WriteEvidence(root, relativeDirectory string, report Report) (string, string, error) {
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return "", "", errors.New("resolve repository root for evidence")
	}
	directory := filepath.Join(rootAbsolute, filepath.FromSlash(relativeDirectory))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", "", errors.New("create artifact verification evidence directory")
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", "", errors.New("resolve artifact verification evidence directory")
	}
	relative, err := filepath.Rel(rootAbsolute, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", errors.New("artifact verification evidence directory escapes the repository")
	}

	stamp := report.FinishedAt.UTC().Format("20060102T150405.000000000Z")
	if report.FinishedAt.IsZero() {
		stamp = time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	base := "isras-artifact-verification-" + stamp
	jsonPath := filepath.Join(resolved, base+".json")
	textPath := filepath.Join(resolved, base+".txt")

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", "", errors.New("encode artifact verification JSON evidence")
	}
	jsonData = append(jsonData, '\n')
	if err := writeExclusive(jsonPath, jsonData); err != nil {
		return "", "", errors.New("write artifact verification JSON evidence")
	}

	textData := renderEvidenceText(report)
	if err := writeExclusive(textPath, textData); err != nil {
		_ = os.Remove(jsonPath)
		return "", "", errors.New("write artifact verification text evidence")
	}
	return filepath.ToSlash(jsonPath), filepath.ToSlash(textPath), nil
}

func renderEvidenceText(report Report) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "ISRAS RELEASE ARTIFACT VERIFICATION EVIDENCE")
	fmt.Fprintln(&output, "============================================")
	fmt.Fprintf(&output, "Started:                 %s\n", report.StartedAt.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&output, "Finished:                %s\n", report.FinishedAt.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&output, "Source mode:             %s\n", report.SourceMode)
	fmt.Fprintf(&output, "Source location:         %s\n", report.SourceLocation)
	fmt.Fprintf(&output, "Release tag:             %s\n", report.ReleaseTag)
	fmt.Fprintf(&output, "Source commit:           %s\n", report.SourceCommit)
	fmt.Fprintf(&output, "Release record:          %s\n", report.ReleaseRecord)
	fmt.Fprintf(&output, "Signed tag:              %s\n", report.SignedTag)
	fmt.Fprintf(&output, "Asset acquisition:       %s\n", report.AssetAcquisition)
	fmt.Fprintf(&output, "Asset inventory:         %s\n", report.AssetInventory)
	fmt.Fprintf(&output, "Pin digests:             %s\n", report.PinDigests)
	fmt.Fprintf(&output, "SHA-256 manifest:        %s\n", report.SHA256Manifest)
	fmt.Fprintf(&output, "SHA-512 manifest:        %s\n", report.SHA512Manifest)
	fmt.Fprintf(&output, "Provenance:              %s\n", report.Provenance)
	fmt.Fprintf(&output, "Execution authorization: %s\n", report.ExecutionAuthorization)
	if report.Failure != "" {
		fmt.Fprintf(&output, "Failure:                 %s\n", report.Failure)
	}
	for index, artifact := range report.Artifacts {
		fmt.Fprintf(&output, "\nArtifact %d\n", index+1)
		fmt.Fprintf(&output, "  Kind:               %s\n", artifact.Kind)
		fmt.Fprintf(&output, "  Name:               %s\n", artifact.Name)
		fmt.Fprintf(&output, "  OS/architecture:    %s/%s\n", artifact.OS, artifact.Arch)
		fmt.Fprintf(&output, "  Size:               %d\n", artifact.Size)
		fmt.Fprintf(&output, "  Remote size:        %d\n", artifact.RemoteSize)
		fmt.Fprintf(&output, "  Expected SHA-256:   %s\n", artifact.ExpectedSHA256)
		fmt.Fprintf(&output, "  Observed SHA-256:   %s\n", artifact.ObservedSHA256)
		fmt.Fprintf(&output, "  SHA-256 status:     %s\n", artifact.SHA256Status)
		fmt.Fprintf(&output, "  Expected SHA-512:   %s\n", artifact.ExpectedSHA512)
		fmt.Fprintf(&output, "  Observed SHA-512:   %s\n", artifact.ObservedSHA512)
		fmt.Fprintf(&output, "  SHA-512 status:     %s\n", artifact.SHA512Status)
		fmt.Fprintf(&output, "  SHA-256 manifest:   %s\n", artifact.SHA256Manifest)
		fmt.Fprintf(&output, "  SHA-512 manifest:   %s\n", artifact.SHA512Manifest)
		fmt.Fprintf(&output, "  Provenance binding: %s\n", artifact.ProvenanceBinding)
	}
	return output.Bytes()
}

func writeExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	return file.Close()
}
