package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/projectpin"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/releaseartifact"
)

func verifyProjectArtifacts(ctx context.Context, root string, pin projectpin.Pin, args []string) (releaseartifact.Report, string, string, error) {
	sourceDirectory, err := parseProjectArtifactArgs(args)
	if err != nil {
		return releaseartifact.Report{}, "", "", err
	}

	var report releaseartifact.Report
	var verifyErr error
	if sourceDirectory == "" {
		report, verifyErr = releaseartifact.VerifyGitHub(ctx, pin)
	} else {
		report, verifyErr = releaseartifact.VerifyDirectory(pin, sourceDirectory)
	}

	jsonPath, textPath, evidenceErr := releaseartifact.WriteEvidence(root, pin.Evidence.Directory, report)
	if evidenceErr != nil {
		report.ExecutionAuthorization = releaseartifact.AuthorizationDenied
		if report.Failure == "" {
			report.Failure = "artifact verification evidence could not be written"
		}
		if verifyErr != nil {
			return report, "", "", errors.New("artifact verification failed and evidence could not be written")
		}
		return report, "", "", evidenceErr
	}
	return report, jsonPath, textPath, verifyErr
}

func parseProjectArtifactArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	if len(args) != 2 || args[0] != "--source-directory" || strings.TrimSpace(args[1]) == "" {
		return "", errors.New("verify-artifacts accepts either no options or --source-directory PATH")
	}
	return args[1], nil
}

func renderProjectArtifactVerification(writer io.Writer, root string, report releaseartifact.Report, jsonPath, textPath string) {
	fmt.Fprintln(writer, "ISRAS RELEASE ARTIFACT VERIFICATION")
	fmt.Fprintln(writer, "===================================")
	fmt.Fprintf(writer, "Source mode:             %s\n", report.SourceMode)
	fmt.Fprintf(writer, "Release tag:             %s\n", report.ReleaseTag)
	fmt.Fprintf(writer, "Source commit:           %s\n", report.SourceCommit)
	fmt.Fprintf(writer, "Release record:          %s\n", report.ReleaseRecord)
	fmt.Fprintf(writer, "Signed annotated tag:    %s\n", report.SignedTag)
	fmt.Fprintf(writer, "Asset acquisition:       %s\n", report.AssetAcquisition)
	fmt.Fprintf(writer, "Exact asset inventory:   %s\n", report.AssetInventory)
	fmt.Fprintf(writer, "Pin SHA-256/SHA-512:     %s\n", report.PinDigests)
	fmt.Fprintf(writer, "SHA-256 manifest:        %s\n", report.SHA256Manifest)
	fmt.Fprintf(writer, "SHA-512 manifest:        %s\n", report.SHA512Manifest)
	fmt.Fprintf(writer, "Provenance binding:      %s\n", report.Provenance)
	fmt.Fprintf(writer, "Execution authorization: %s\n", report.ExecutionAuthorization)
	if report.Failure != "" {
		fmt.Fprintf(writer, "Failure:                 %s\n", report.Failure)
	}
	fmt.Fprintf(writer, "Artifacts evaluated:     %d\n", len(report.Artifacts))
	for index, artifact := range report.Artifacts {
		label := artifact.Kind
		if artifact.OS != "" || artifact.Arch != "" {
			label += " (" + artifact.OS + "/" + artifact.Arch + ")"
		}
		fmt.Fprintf(writer, "  %d. %s: %s\n", index+1, label, artifact.Name)
		fmt.Fprintf(writer, "     Size: %d bytes\n", artifact.Size)
		fmt.Fprintf(writer, "     Pin SHA-256: %s\n", artifact.SHA256Status)
		fmt.Fprintf(writer, "     Pin SHA-512: %s\n", artifact.SHA512Status)
		if artifact.Kind != "sha256-manifest" && artifact.Kind != "sha512-manifest" {
			fmt.Fprintf(writer, "     SHA-256 manifest: %s\n", artifact.SHA256Manifest)
			fmt.Fprintf(writer, "     SHA-512 manifest: %s\n", artifact.SHA512Manifest)
		}
		if artifact.Kind != "provenance" && artifact.Kind != "sha256-manifest" && artifact.Kind != "sha512-manifest" {
			fmt.Fprintf(writer, "     Provenance: %s\n", artifact.ProvenanceBinding)
		}
	}
	if jsonPath != "" {
		fmt.Fprintf(writer, "Evidence JSON:           %s\n", relative(root, jsonPath))
	}
	if textPath != "" {
		fmt.Fprintf(writer, "Evidence text:           %s\n", relative(root, textPath))
	}
}
