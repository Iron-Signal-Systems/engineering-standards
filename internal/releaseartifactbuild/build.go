package releaseartifactbuild

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	maxArtifactBytes = int64(512 * 1024 * 1024)
	maxTotalBytes    = int64(2 * 1024 * 1024 * 1024)
)

type Builder struct {
	Runner commandRunner
	Now    func() time.Time
}

func Build(ctx context.Context, options Options) (Result, error) {
	builder := Builder{Runner: osCommandRunner{}, Now: time.Now}
	return builder.Build(ctx, options)
}

func (builder Builder) Build(ctx context.Context, options Options) (result Result, err error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Result{}, errors.New("release artifact production currently requires a linux/amd64 build host")
	}
	if builder.Runner == nil {
		builder.Runner = osCommandRunner{}
	}
	if builder.Now == nil {
		builder.Now = time.Now
	}
	if err := validateOptions(options); err != nil {
		return Result{}, err
	}

	boundary, err := inspectSource(ctx, builder.Runner, options.Root, options.ExpectedVersion)
	if err != nil {
		return Result{}, err
	}
	publishedAt, err := time.Parse(time.RFC3339, options.PublishedAt)
	if err != nil {
		return Result{}, errors.New("published-at must be an RFC3339 timestamp")
	}
	publishedAt = publishedAt.UTC()

	outputDirectory := options.OutputDirectory
	if strings.TrimSpace(outputDirectory) == "" {
		outputDirectory = filepath.Join(boundary.Root, ".local", "releases", boundary.Tag, "assets")
	} else if !filepath.IsAbs(outputDirectory) {
		outputDirectory = filepath.Join(boundary.Root, outputDirectory)
	}
	outputDirectory = filepath.Clean(outputDirectory)
	releaseRoot := filepath.Join(boundary.Root, ".local", "releases")
	relativeOutput, relativeErr := filepath.Rel(releaseRoot, outputDirectory)
	if relativeErr != nil || relativeOutput == "." || relativeOutput == ".." || strings.HasPrefix(relativeOutput, ".."+string(filepath.Separator)) {
		return Result{}, errors.New("release artifact output must be below the repository .local/releases directory")
	}
	if info, statErr := os.Lstat(outputDirectory); statErr == nil {
		if info.IsDir() {
			return Result{}, errors.New("release artifact output directory already exists")
		}
		return Result{}, errors.New("release artifact output path already exists")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return Result{}, errors.New("inspect release artifact output path")
	}

	parent := filepath.Dir(outputDirectory)
	if err := ensurePrivateDirectory(boundary.Root, parent); err != nil {
		return Result{}, errors.New("create secure release artifact output parent")
	}
	temporary, err := os.MkdirTemp(parent, ".isras-artifacts-")
	if err != nil {
		return Result{}, errors.New("create temporary release artifact directory")
	}
	if err := os.Chmod(temporary, 0o700); err != nil {
		_ = os.RemoveAll(temporary)
		return Result{}, errors.New("secure temporary release artifact directory")
	}
	committedOutput := false
	defer func() {
		if !committedOutput {
			_ = os.RemoveAll(temporary)
			_ = os.RemoveAll(outputDirectory)
		}
	}()

	if err := builder.buildValidator(ctx, boundary, filepath.Join(temporary, ValidatorName)); err != nil {
		return Result{}, err
	}
	if err := buildArchive(ctx, builder.Runner, boundary, FrameworkListPath, "isras-project-framework", filepath.Join(temporary, FrameworkName)); err != nil {
		return Result{}, err
	}
	if err := buildArchive(ctx, builder.Runner, boundary, ContractsListPath, "isras-contracts", filepath.Join(temporary, ContractsName)); err != nil {
		return Result{}, err
	}

	core, err := hashNamedArtifacts(temporary, []artifactDescriptor{
		{Kind: "validator", OS: "linux", Arch: "amd64", Name: ValidatorName},
		{Kind: "framework", Name: FrameworkName},
		{Kind: "contracts", Name: ContractsName},
	})
	if err != nil {
		return Result{}, err
	}

	provenanceValue := provenance{
		SchemaVersion:    1,
		Profile:          Profile,
		Version:          boundary.Version,
		ReleaseTag:       boundary.Tag,
		SourceRepository: SourceRepository,
		SourceCommit:     boundary.Commit,
		Build: provenanceBuild{
			GoVersion: boundary.GoVersion,
			GOOS:      "linux",
			GOARCH:    "amd64",
		},
		Validation: provenanceValidation{
			Campaign: options.ValidationCampaign,
			Commit:   boundary.Commit,
			Status:   "PASS",
		},
		PublishedAt:      publishedAt.Format(time.RFC3339),
		ReleaseAuthority: options.ReleaseAuthority,
		Limitations: []string{
			"self-authored and self-validated release evidence",
			"no independent audit, certification, or universal production-fitness claim",
			"validator binary produced only for linux/amd64 in this release artifact set",
		},
		Artifacts: provenanceArtifacts(core),
	}
	if err := writeJSON(filepath.Join(temporary, ProvenanceName), provenanceValue, 0o644); err != nil {
		return Result{}, errors.New("write release provenance")
	}

	nonManifest, err := hashNamedArtifacts(temporary, []artifactDescriptor{
		{Kind: "validator", OS: "linux", Arch: "amd64", Name: ValidatorName},
		{Kind: "framework", Name: FrameworkName},
		{Kind: "contracts", Name: ContractsName},
		{Kind: "provenance", Name: ProvenanceName},
	})
	if err != nil {
		return Result{}, err
	}
	if err := writeManifest(filepath.Join(temporary, SHA256Name), nonManifest, true); err != nil {
		return Result{}, errors.New("write SHA-256 manifest")
	}
	if err := writeManifest(filepath.Join(temporary, SHA512Name), nonManifest, false); err != nil {
		return Result{}, errors.New("write SHA-512 manifest")
	}

	artifacts, err := hashNamedArtifacts(temporary, []artifactDescriptor{
		{Kind: "validator", OS: "linux", Arch: "amd64", Name: ValidatorName},
		{Kind: "framework", Name: FrameworkName},
		{Kind: "contracts", Name: ContractsName},
		{Kind: "provenance", Name: ProvenanceName},
		{Kind: "sha256-manifest", Name: SHA256Name},
		{Kind: "sha512-manifest", Name: SHA512Name},
	})
	if err != nil {
		return Result{}, err
	}
	if err := verifyExactOutputSet(temporary, artifacts); err != nil {
		return Result{}, err
	}

	finalBoundary, err := inspectSource(ctx, builder.Runner, boundary.Root, boundary.Version)
	if err != nil {
		return Result{}, errors.New("revalidate release source boundary after artifact production")
	}
	if finalBoundary.Commit != boundary.Commit || finalBoundary.Tag != boundary.Tag || finalBoundary.GoVersion != boundary.GoVersion {
		return Result{}, errors.New("release source boundary changed during artifact production")
	}

	result = Result{
		SchemaVersion:    1,
		GeneratedAt:      builder.Now().UTC(),
		Profile:          Profile,
		Version:          boundary.Version,
		ReleaseTag:       boundary.Tag,
		SourceRepository: SourceRepository,
		SourceCommit:     boundary.Commit,
		GoVersion:        boundary.GoVersion,
		OutputDirectory:  outputDirectory,
		Artifacts:        artifacts,
	}
	evidenceDirectory := filepath.Join(boundary.Root, ".local", "validation", "releases", boundary.Tag)
	jsonPath := filepath.Join(evidenceDirectory, "artifact-build.json")
	textPath := filepath.Join(evidenceDirectory, "artifact-build.txt")
	result.EvidenceJSON = jsonPath
	result.EvidenceText = textPath

	if err := os.Rename(temporary, outputDirectory); err != nil {
		return Result{}, errors.New("commit release artifact output directory")
	}
	committedOutput = true
	if err := writeEvidence(result, boundary.Root, evidenceDirectory); err != nil {
		committedOutput = false
		return Result{}, err
	}
	return result, nil
}

func validateOptions(options Options) error {
	if !boundedText(options.ValidationCampaign, 128) {
		return errors.New("validation campaign is required and must be a bounded single-line value")
	}
	if !boundedText(options.ReleaseAuthority, 256) {
		return errors.New("release authority is required and must be a bounded single-line value")
	}
	if strings.TrimSpace(options.PublishedAt) == "" {
		return errors.New("published-at is required")
	}
	if options.ExpectedVersion != "" && !stableVersionPattern.MatchString(options.ExpectedVersion) {
		return errors.New("expected version must be MAJOR.MINOR.PATCH")
	}
	return nil
}

func boundedText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func (builder Builder) buildValidator(ctx context.Context, boundary sourceBoundary, outputPath string) error {
	linkerFlags := strings.Join([]string{
		"-s",
		"-w",
		"-buildid=",
		"-X=github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity.releaseVersion=" + boundary.Version,
		"-X=github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity.releaseTag=" + boundary.Tag,
		"-X=github.com/Iron-Signal-Systems/engineering-standards/internal/validatoridentity.releaseSourceCommit=" + boundary.Commit,
	}, " ")
	environment := sanitizedEnvironment(
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)
	_, _, err := builder.Runner.Run(ctx, boundary.Root, environment,
		"go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false",
		"-ldflags", linkerFlags, "-o", outputPath, "./cmd/isras-validate")
	if err != nil {
		return errors.New("build release validator")
	}
	if err := os.Chmod(outputPath, 0o755); err != nil {
		return errors.New("set release validator mode")
	}
	identityOutput, _, err := builder.Runner.Run(ctx, boundary.Root, sanitizedEnvironment(), outputPath, "version")
	if err != nil {
		return errors.New("execute release validator identity check")
	}
	for _, expected := range []string{
		"Standard version:  " + boundary.Version,
		"Ownership:         release-artifact",
		"Release tag:       " + boundary.Tag,
		"Source repository: " + SourceRepository,
		"Source commit:     " + boundary.Commit,
	} {
		if !strings.Contains(identityOutput, expected) {
			return errors.New("release validator embedded identity check failed")
		}
	}
	return nil
}

type artifactDescriptor struct {
	Kind string
	OS   string
	Arch string
	Name string
}

func hashNamedArtifacts(directory string, descriptors []artifactDescriptor) ([]ArtifactRecord, error) {
	records := make([]ArtifactRecord, 0, len(descriptors))
	var total int64
	for _, descriptor := range descriptors {
		record, err := hashFile(filepath.Join(directory, descriptor.Name), descriptor)
		if err != nil {
			return nil, err
		}
		total += record.Size
		if total > maxTotalBytes {
			return nil, errors.New("release artifact set exceeds the total size boundary")
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	return records, nil
}

func hashFile(filePath string, descriptor artifactDescriptor) (ArtifactRecord, error) {
	info, err := os.Lstat(filePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maxArtifactBytes {
		return ArtifactRecord{}, fmt.Errorf("artifact %s violates the regular-file or size boundary", descriptor.Name)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("open artifact %s", descriptor.Name)
	}
	defer file.Close()
	h256 := sha256.New()
	h512 := sha512.New()
	written, err := io.Copy(io.MultiWriter(h256, h512), io.LimitReader(file, maxArtifactBytes+1))
	if err != nil || written != info.Size() || written > maxArtifactBytes {
		return ArtifactRecord{}, fmt.Errorf("hash artifact %s", descriptor.Name)
	}
	after, err := os.Lstat(filePath)
	if err != nil || !os.SameFile(info, after) || after.Size() != info.Size() {
		return ArtifactRecord{}, fmt.Errorf("artifact %s changed during hashing", descriptor.Name)
	}
	return ArtifactRecord{
		Kind: descriptor.Kind, OS: descriptor.OS, Arch: descriptor.Arch, Name: descriptor.Name,
		Size:   written,
		SHA256: hex.EncodeToString(h256.Sum(nil)),
		SHA512: hex.EncodeToString(h512.Sum(nil)),
	}, nil
}

func provenanceArtifacts(records []ArtifactRecord) []provenanceArtifact {
	out := make([]provenanceArtifact, 0, len(records))
	for _, record := range records {
		out = append(out, provenanceArtifact{
			Kind: record.Kind, OS: record.OS, Arch: record.Arch, Name: record.Name,
			SHA256: record.SHA256, SHA512: record.SHA512,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func writeManifest(filePath string, records []ArtifactRecord, useSHA256 bool) error {
	copyRecords := append([]ArtifactRecord(nil), records...)
	sort.Slice(copyRecords, func(i, j int) bool { return copyRecords[i].Name < copyRecords[j].Name })
	var builder strings.Builder
	for _, record := range copyRecords {
		digest := record.SHA512
		if useSHA256 {
			digest = record.SHA256
		}
		fmt.Fprintf(&builder, "%s  %s\n", digest, record.Name)
	}
	return writeFileAtomic(filePath, []byte(builder.String()), 0o644)
}

func writeJSON(filePath string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(filePath, data, mode)
}

func writeFileAtomic(filePath string, data []byte, mode os.FileMode) error {
	temporary := filePath + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	failed := true
	defer func() {
		_ = file.Close()
		if failed {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, filePath); err != nil {
		return err
	}
	failed = false
	return nil
}

func verifyExactOutputSet(directory string, records []ArtifactRecord) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return errors.New("read release artifact output directory")
	}
	if len(entries) != len(records) {
		return errors.New("release artifact output directory does not contain the exact artifact set")
	}
	expected := make(map[string]bool, len(records))
	for _, record := range records {
		expected[record.Name] = true
	}
	for _, entry := range entries {
		if !expected[entry.Name()] || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("release artifact output directory contains an unexpected entry")
		}
	}
	return nil
}

func ensurePrivateDirectory(root, directory string) error {
	root = filepath.Clean(root)
	directory = filepath.Clean(directory)
	relative, err := filepath.Rel(root, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("private directory escapes the repository")
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			return errors.New("private directory contains an unsafe component")
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return err
			}
			continue
		}
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("private directory path contains a symlink or non-directory")
		}
	}
	return nil
}

func writeEvidence(result Result, repositoryRoot, directory string) error {
	if err := ensurePrivateDirectory(repositoryRoot, directory); err != nil {
		return errors.New("create secure release artifact evidence directory")
	}
	if err := writeJSON(result.EvidenceJSON, result, 0o600); err != nil {
		return errors.New("write release artifact JSON evidence")
	}
	var builder strings.Builder
	fmt.Fprintln(&builder, "ISRAS RELEASE ARTIFACT BUILD")
	fmt.Fprintln(&builder, "============================")
	fmt.Fprintf(&builder, "Status:            PASS\n")
	fmt.Fprintf(&builder, "Version:           %s\n", result.Version)
	fmt.Fprintf(&builder, "Release tag:       %s\n", result.ReleaseTag)
	fmt.Fprintf(&builder, "Source commit:     %s\n", result.SourceCommit)
	fmt.Fprintf(&builder, "Go version:        %s\n", result.GoVersion)
	fmt.Fprintf(&builder, "Output directory:  %s\n", result.OutputDirectory)
	fmt.Fprintf(&builder, "Artifacts:         %d\n", len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		fmt.Fprintf(&builder, "\n%s\n", artifact.Name)
		fmt.Fprintf(&builder, "  Kind:     %s\n", artifact.Kind)
		fmt.Fprintf(&builder, "  Size:     %d\n", artifact.Size)
		fmt.Fprintf(&builder, "  SHA-256:  %s\n", artifact.SHA256)
		fmt.Fprintf(&builder, "  SHA-512:  %s\n", artifact.SHA512)
	}
	if err := writeFileAtomic(result.EvidenceText, []byte(builder.String()), 0o600); err != nil {
		_ = os.Remove(result.EvidenceJSON)
		return errors.New("write release artifact text evidence")
	}
	return nil
}
