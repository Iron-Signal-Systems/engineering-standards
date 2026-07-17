package projectpin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	MetadataPath           = ".isras/project.json"
	SchemaVersion          = 1
	Profile                = "ISRAS-SD"
	SourceRepository       = "github.com/Iron-Signal-Systems/engineering-standards"
	ReusableWorkflowPath   = ".github/workflows/validate-project.yml"
	MaxFileSize            = 256 * 1024
	maxArtifacts           = 32
	maxProfiles            = 8
	maxCommands            = 64
	maxArgumentsPerCommand = 64
	maxArgumentBytes       = 4096
	maxTotalCommandBytes   = 64 * 1024
	maxRelativePathBytes   = 255
)

var (
	stableVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	commitPattern        = regexp.MustCompile(`^[0-9a-f]{40}$`)
	digest256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	digest512Pattern     = regexp.MustCompile(`^[0-9a-f]{128}$`)
	projectPattern       = regexp.MustCompile(`^github\.com/Iron-Signal-Systems/[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	identifierPattern    = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	commandNamePattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	artifactNamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

var supportedArtifactKinds = map[string]bool{
	"validator":       true,
	"framework":       true,
	"contracts":       true,
	"provenance":      true,
	"sha256-manifest": true,
	"sha512-manifest": true,
	"migration":       true,
}

var supportedProfiles = map[string]bool{"go": true}

var requiredGoCommands = []string{
	"format_check",
	"static_analysis",
	"test",
	"build",
	"module_consistency",
	"module_integrity",
	"known_vulnerabilities",
}

var requiredArtifactKinds = []string{
	"validator",
	"framework",
	"contracts",
	"provenance",
	"sha256-manifest",
	"sha512-manifest",
}

type Pin struct {
	SchemaVersion int                 `json:"schema_version"`
	Project       Project             `json:"project"`
	Standard      Standard            `json:"standard"`
	Artifacts     []Artifact          `json:"artifacts"`
	Workflow      Workflow            `json:"workflow"`
	Profiles      []string            `json:"profiles"`
	Commands      map[string][]string `json:"commands"`
	Evidence      Evidence            `json:"evidence"`
}

type Project struct {
	Repository string `json:"repository"`
}

type Standard struct {
	Profile          string `json:"profile"`
	Version          string `json:"version"`
	ReleaseTag       string `json:"release_tag"`
	SourceRepository string `json:"source_repository"`
	SourceCommit     string `json:"source_commit"`
}

type Artifact struct {
	Kind   string `json:"kind"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
}

type Workflow struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
	Commit     string `json:"commit"`
}

type Evidence struct {
	Directory string `json:"directory"`
}

func Load(root string) (Pin, error) {
	filePath := filepath.Join(root, filepath.FromSlash(MetadataPath))
	file, err := os.Open(filePath)
	if err != nil {
		return Pin{}, fmt.Errorf("read project pin %s: %w", MetadataPath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, MaxFileSize+1))
	if err != nil {
		return Pin{}, fmt.Errorf("read project pin %s: %w", MetadataPath, err)
	}
	if len(data) > MaxFileSize {
		return Pin{}, fmt.Errorf("project pin exceeds %d-byte limit", MaxFileSize)
	}
	return Parse(data)
}

func Parse(data []byte) (Pin, error) {
	if len(data) == 0 {
		return Pin{}, errors.New("project pin is empty")
	}
	if len(data) > MaxFileSize {
		return Pin{}, fmt.Errorf("project pin exceeds %d-byte limit", MaxFileSize)
	}
	if err := rejectDuplicateFields(data); err != nil {
		return Pin{}, err
	}

	var pin Pin
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pin); err != nil {
		if strings.Contains(err.Error(), "unknown field") {
			return Pin{}, errors.New("project pin contains an unknown field")
		}
		return Pin{}, fmt.Errorf("parse project pin: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Pin{}, err
	}
	if err := validate(pin); err != nil {
		return Pin{}, err
	}
	return pin, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("parse trailing project pin data: %w", err)
	}
	return errors.New("project pin contains multiple JSON values")
}

func rejectDuplicateFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder, "$"); err != nil {
		return fmt.Errorf("parse project pin structure: %w", err)
	}
	if _, err := decoder.Token(); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("parse trailing project pin structure: %w", err)
	}
	return errors.New("project pin contains a trailing JSON value")
}

func scanJSONValue(decoder *json.Decoder, location string) error {
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
			if !ok {
				return errors.New("project pin contains a non-string object key")
			}
			if seen[key] {
				return errors.New("project pin contains a duplicate field")
			}
			seen[key] = true
			if err := scanJSONValue(decoder, location+"."+key); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return errors.New("project pin contains an unexpected object terminator")
		}
	case '[':
		index := 0
		for decoder.More() {
			if err := scanJSONValue(decoder, fmt.Sprintf("%s[%d]", location, index)); err != nil {
				return err
			}
			index++
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return errors.New("project pin contains an unexpected array terminator")
		}
	default:
		return errors.New("project pin contains an unexpected JSON delimiter")
	}
	return nil
}

func validate(pin Pin) error {
	if pin.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported project pin schema version %d", pin.SchemaVersion)
	}
	if !projectPattern.MatchString(pin.Project.Repository) {
		return errors.New("invalid project repository identity")
	}
	if pin.Standard.Profile != Profile {
		return errors.New("unexpected ISRAS profile")
	}
	if !stableVersionPattern.MatchString(pin.Standard.Version) {
		return errors.New("project pin requires a stable MAJOR.MINOR.PATCH version")
	}
	if pin.Standard.ReleaseTag != "isras-v"+pin.Standard.Version {
		return errors.New("release tag does not match the pinned version")
	}
	if pin.Standard.SourceRepository != SourceRepository {
		return errors.New("unexpected ISRAS source repository")
	}
	if err := validateCommit("ISRAS source commit", pin.Standard.SourceCommit); err != nil {
		return err
	}
	if err := validateArtifacts(pin.Artifacts); err != nil {
		return err
	}
	if pin.Workflow.Repository != SourceRepository {
		return errors.New("unexpected reusable workflow repository")
	}
	if pin.Workflow.Path != ReusableWorkflowPath {
		return errors.New("unexpected reusable workflow path")
	}
	if err := validateCommit("reusable workflow commit", pin.Workflow.Commit); err != nil {
		return err
	}
	if pin.Workflow.Commit != pin.Standard.SourceCommit {
		return errors.New("reusable workflow commit does not match the pinned ISRAS source commit")
	}
	if err := validateProfiles(pin.Profiles); err != nil {
		return err
	}
	if err := validateCommands(pin.Commands, pin.Profiles); err != nil {
		return err
	}
	if err := validateRelativePath("evidence directory", pin.Evidence.Directory); err != nil {
		return err
	}
	return nil
}

func validateArtifacts(artifacts []Artifact) error {
	if len(artifacts) == 0 {
		return errors.New("project pin requires release artifacts")
	}
	if len(artifacts) > maxArtifacts {
		return fmt.Errorf("project pin declares %d artifacts; maximum is %d", len(artifacts), maxArtifacts)
	}

	kindCounts := make(map[string]int)
	platforms := make(map[string]bool)
	names := make(map[string]bool)
	for index, artifact := range artifacts {
		if !identifierPattern.MatchString(artifact.Kind) {
			return fmt.Errorf("artifact %d has an invalid kind", index)
		}
		if !supportedArtifactKinds[artifact.Kind] {
			return fmt.Errorf("artifact %d has an unsupported kind", index)
		}
		if !artifactNamePattern.MatchString(artifact.Name) || path.Base(artifact.Name) != artifact.Name || strings.Contains(artifact.Name, "\\") {
			return fmt.Errorf("artifact %d has an unsafe name", index)
		}
		if names[artifact.Name] {
			return errors.New("project pin contains a duplicate artifact name")
		}
		names[artifact.Name] = true
		if err := validateDigest("artifact SHA-256", artifact.SHA256, digest256Pattern); err != nil {
			return fmt.Errorf("artifact %d: %w", index, err)
		}
		if err := validateDigest("artifact SHA-512", artifact.SHA512, digest512Pattern); err != nil {
			return fmt.Errorf("artifact %d: %w", index, err)
		}

		kindCounts[artifact.Kind]++
		if artifact.Kind == "validator" {
			if !identifierPattern.MatchString(artifact.OS) || !identifierPattern.MatchString(artifact.Arch) {
				return fmt.Errorf("validator artifact %d requires valid os and arch identifiers", index)
			}
			platform := artifact.OS + "/" + artifact.Arch
			if platforms[platform] {
				return errors.New("project pin contains a duplicate validator artifact platform")
			}
			platforms[platform] = true
		} else if artifact.OS != "" || artifact.Arch != "" {
			return fmt.Errorf("non-validator artifact %d must not declare os or arch", index)
		}
	}

	if kindCounts["migration"] > 1 {
		return errors.New("project pin may declare at most one migration artifact")
	}

	for _, kind := range requiredArtifactKinds {
		count := kindCounts[kind]
		if kind == "validator" {
			if count == 0 {
				return errors.New("project pin requires at least one validator artifact")
			}
			continue
		}
		if count != 1 {
			return fmt.Errorf("project pin requires exactly one %s artifact, found %d", kind, count)
		}
	}
	return nil
}

func validateProfiles(profiles []string) error {
	if len(profiles) == 0 {
		return errors.New("project pin requires at least one project profile")
	}
	if len(profiles) > maxProfiles {
		return fmt.Errorf("project pin declares %d profiles; maximum is %d", len(profiles), maxProfiles)
	}
	seen := make(map[string]bool)
	for _, profile := range profiles {
		if !identifierPattern.MatchString(profile) {
			return errors.New("project pin contains an invalid project profile")
		}
		if seen[profile] {
			return errors.New("project pin contains a duplicate project profile")
		}
		if !supportedProfiles[profile] {
			return errors.New("project pin contains an unsupported project profile")
		}
		seen[profile] = true
	}
	return nil
}

func validateCommands(commands map[string][]string, profiles []string) error {
	if len(commands) == 0 {
		return errors.New("project pin requires project-owned validation commands")
	}
	if len(commands) > maxCommands {
		return fmt.Errorf("project pin declares %d commands; maximum is %d", len(commands), maxCommands)
	}

	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !commandNamePattern.MatchString(name) {
			return errors.New("project pin contains an invalid command name")
		}
		arguments := commands[name]
		if len(arguments) == 0 {
			return fmt.Errorf("command %q has no executable", name)
		}
		if len(arguments) > maxArgumentsPerCommand {
			return fmt.Errorf("command %q has %d arguments; maximum is %d", name, len(arguments), maxArgumentsPerCommand)
		}
		total := 0
		for index, argument := range arguments {
			if argument == "" {
				return fmt.Errorf("command %q argument %d is empty", name, index)
			}
			if len(argument) > maxArgumentBytes {
				return fmt.Errorf("command %q argument %d exceeds %d bytes", name, index, maxArgumentBytes)
			}
			if strings.ContainsAny(argument, "\x00\r\n") {
				return fmt.Errorf("command %q argument %d contains a prohibited control character", name, index)
			}
			total += len(argument)
		}
		if total > maxTotalCommandBytes {
			return fmt.Errorf("command %q exceeds %d total argument bytes", name, maxTotalCommandBytes)
		}
		if strings.TrimSpace(arguments[0]) != arguments[0] || strings.ContainsAny(arguments[0], " \t") {
			return fmt.Errorf("command %q executable must be one argument without whitespace", name)
		}
	}

	if contains(profiles, "go") {
		for _, required := range requiredGoCommands {
			if _, ok := commands[required]; !ok {
				return fmt.Errorf("Go profile requires command %q", required)
			}
		}
	}
	return nil
}

func validateCommit(label, value string) error {
	if !commitPattern.MatchString(value) || allZero(value) {
		return fmt.Errorf("invalid %s", label)
	}
	return nil
}

func validateDigest(label, value string, pattern *regexp.Regexp) error {
	if !pattern.MatchString(value) || allZero(value) {
		return fmt.Errorf("invalid %s", label)
	}
	return nil
}

func validateRelativePath(label, value string) error {
	if len(value) > maxRelativePathBytes {
		return fmt.Errorf("%s exceeds %d bytes", label, maxRelativePathBytes)
	}
	if value == "" || strings.Contains(value, "\\") || strings.ContainsAny(value, "\x00\r\n\t") {
		return fmt.Errorf("invalid %s", label)
	}
	if strings.HasPrefix(value, "/") || value == "." || value == ".." {
		return fmt.Errorf("invalid %s", label)
	}
	cleaned := path.Clean(value)
	if cleaned != value || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("invalid %s", label)
	}
	if cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
		return fmt.Errorf("%s must not be inside .git", label)
	}
	return nil
}

func allZero(value string) bool {
	return strings.Trim(value, "0") == ""
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
