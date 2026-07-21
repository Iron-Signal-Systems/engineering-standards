package projectcommand

import (
	"errors"
	"fmt"
	goversion "go/version"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const maxGoModBytes = 1024 * 1024

type goToolchainSelection struct {
	Executable       string
	Directory        string
	Minimum          string
	Toolchain        string
	Actual           string
	MinimumSatisfied bool
	Modules          []goModuleSelection
}

type goModuleSelection struct {
	GoModPath        string
	Directory        string
	ModulePath       string
	Minimum          string
	Toolchain        string
	MinimumSatisfied bool
}

type goModuleDeclaration struct {
	Module    string
	Minimum   string
	Toolchain string
}

func projectUsesGoProfile(request Request) bool {
	for _, profile := range request.Pin.Profiles {
		if profile == "go" {
			return true
		}
	}
	return false
}

func selectGoToolchain(root string) (goToolchainSelection, error) {
	selection := goToolchainSelection{}

	modules, err := discoverGoModules(root)
	if err != nil {
		return selection, err
	}
	selection.Modules = modules

	for _, module := range modules {
		if selection.Minimum == "" ||
			goversion.Compare(module.Minimum, selection.Minimum) > 0 {
			selection.Minimum = module.Minimum
		}
		if module.GoModPath == "go.mod" {
			selection.Toolchain = module.Toolchain
		}
	}

	executable, err := exec.LookPath("go")
	if err != nil {
		return selection, errors.New(
			"Go-profile project requires an available Go toolchain",
		)
	}
	if !filepath.IsAbs(executable) {
		executable, err = filepath.Abs(executable)
		if err != nil {
			return selection, errors.New("resolve selected Go executable")
		}
	}
	executable = filepath.Clean(executable)
	selection.Executable = executable
	selection.Directory = filepath.Dir(executable)

	info, err := os.Stat(executable)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm()&0o111 == 0 {
		return selection, errors.New(
			"selected Go executable is not a regular executable file",
		)
	}

	command := exec.Command(executable, "env", "GOVERSION")
	command.Dir = root
	command.Env = goVersionEnvironment()
	output, err := command.Output()
	if err != nil {
		return selection, errors.New(
			"query selected Go toolchain version",
		)
	}

	selection.Actual = strings.TrimSpace(string(output))
	if !goversion.IsValid(selection.Actual) {
		return selection, fmt.Errorf(
			"selected Go toolchain reported an invalid version: %q",
			selection.Actual,
		)
	}

	selection.MinimumSatisfied = true
	for index := range selection.Modules {
		satisfied := goVersionAtLeast(
			selection.Actual,
			selection.Modules[index].Minimum,
		)
		selection.Modules[index].MinimumSatisfied = satisfied
		if !satisfied {
			selection.MinimumSatisfied = false
		}
	}

	if !selection.MinimumSatisfied {
		for _, module := range selection.Modules {
			if !module.MinimumSatisfied {
				return selection, fmt.Errorf(
					"selected Go toolchain %s is below project minimum %s for module %s",
					selection.Actual,
					module.Minimum,
					module.GoModPath,
				)
			}
		}
		return selection, fmt.Errorf(
			"selected Go toolchain %s is below project minimum %s",
			selection.Actual,
			selection.Minimum,
		)
	}

	return selection, nil
}

func discoverGoModules(root string) ([]goModuleSelection, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.New("resolve Go module repository root")
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	rootGoMod := filepath.Join(absoluteRoot, "go.mod")
	rootInfo, err := os.Lstat(rootGoMod)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New(
			"Go-profile project root does not contain a governed go.mod",
		)
	}
	if err != nil {
		return nil, errors.New("inspect Go-profile project root go.mod")
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 ||
		!rootInfo.Mode().IsRegular() {
		_, parseErr := readGoModuleDeclaration(
			absoluteRoot,
			"go.mod",
		)
		if parseErr != nil {
			return nil, parseErr
		}
		return nil, errors.New(
			"Go module inventory contains a non-regular root go.mod path",
		)
	}

	relativePaths, err := repositoryGoModulePaths(absoluteRoot)
	if err != nil {
		return nil, err
	}

	var modules []goModuleSelection
	modulePaths := make(map[string]string)
	rootFound := false

	for _, relative := range relativePaths {
		declaration, err := readGoModuleDeclaration(
			absoluteRoot,
			relative,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"validate Go module %s: %w",
				relative,
				err,
			)
		}

		if prior, exists := modulePaths[declaration.Module]; exists {
			return nil, fmt.Errorf(
				"Go module path %q is declared by both %s and %s",
				declaration.Module,
				prior,
				relative,
			)
		}
		modulePaths[declaration.Module] = relative

		directory := filepath.ToSlash(filepath.Dir(relative))
		if directory == "" || directory == "." {
			directory = "."
		}
		if relative == "go.mod" {
			rootFound = true
		}

		modules = append(modules, goModuleSelection{
			GoModPath:  relative,
			Directory:  directory,
			ModulePath: declaration.Module,
			Minimum:    declaration.Minimum,
			Toolchain:  declaration.Toolchain,
		})
	}

	if !rootFound {
		return nil, errors.New(
			"Go-profile project root go.mod is excluded from the repository-owned source inventory",
		)
	}
	if len(modules) == 0 {
		return nil, errors.New(
			"Go-profile project does not declare any repository-owned Go modules",
		)
	}

	return modules, nil
}

func repositoryGoModulePaths(root string) ([]string, error) {
	gitExecutable, err := boundedSystemExecutable("git")
	if err != nil {
		return nil, err
	}

	command := exec.Command(
		gitExecutable,
		"ls-files",
		"-z",
		"--cached",
		"--others",
		"--exclude-standard",
		"--",
		"go.mod",
		":(glob)**/go.mod",
	)
	command.Dir = root
	command.Env = []string{
		"HOME=" + root,
		"LANG=C",
		"LC_ALL=C",
		"PATH=" + sanitizedCommandPath(gitExecutable),
	}

	output, err := command.Output()
	if err != nil {
		return nil, errors.New(
			"enumerate repository-owned Go module files",
		)
	}

	seen := make(map[string]bool)
	var paths []string

	for _, raw := range strings.Split(string(output), "\x00") {
		if raw == "" {
			continue
		}
		if filepath.IsAbs(raw) ||
			strings.Contains(raw, "\\") {
			return nil, errors.New(
				"repository Go module inventory contains an unsafe path",
			)
		}

		normalized := filepath.ToSlash(
			filepath.Clean(filepath.FromSlash(raw)),
		)
		if normalized != raw ||
			normalized == "." ||
			normalized == ".." ||
			strings.HasPrefix(normalized, "../") {
			return nil, errors.New(
				"repository Go module inventory contains an unsafe path",
			)
		}
		if normalized != "go.mod" &&
			!strings.HasSuffix(normalized, "/go.mod") {
			return nil, errors.New(
				"repository Go module inventory contains a non-go.mod path",
			)
		}
		if localRuntimePath(normalized) {
			continue
		}
		if seen[normalized] {
			continue
		}

		seen[normalized] = true
		paths = append(paths, normalized)
	}

	sort.Strings(paths)
	return paths, nil
}

func localRuntimePath(relative string) bool {
	return relative == ".local/go.mod" ||
		strings.HasPrefix(relative, ".local/")
}

func boundedSystemExecutable(name string) (string, error) {
	if name == "" || filepath.Base(name) != name {
		return "", errors.New("bounded system executable name is invalid")
	}

	for _, directory := range filepath.SplitList(
		sanitizedCommandPath(""),
	) {
		candidate := filepath.Join(directory, name)
		info, err := os.Stat(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf(
				"inspect bounded system executable %s: %w",
				name,
				err,
			)
		}
		if !info.Mode().IsRegular() ||
			info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}

	return "", fmt.Errorf(
		"required bounded system executable %q is unavailable",
		name,
	)
}

func readGoModuleDeclaration(root, relativePath string) (goModuleDeclaration, error) {
	if relativePath == "" || filepath.IsAbs(relativePath) || strings.Contains(relativePath, "\\") {
		return goModuleDeclaration{}, errors.New("Go module file path must be a relative slash-separated path")
	}
	normalized := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath)))
	if normalized != relativePath || normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return goModuleDeclaration{}, errors.New("Go module file path is unsafe")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return goModuleDeclaration{}, errors.New("resolve Go module repository root")
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	candidate := filepath.Join(absoluteRoot, filepath.FromSlash(normalized))
	if !pathWithin(absoluteRoot, candidate) {
		return goModuleDeclaration{}, errors.New("Go module file escapes the target repository")
	}
	if err := rejectGoModuleSymlinkPath(absoluteRoot, candidate); err != nil {
		return goModuleDeclaration{}, err
	}
	info, err := os.Lstat(candidate)
	if err != nil {
		return goModuleDeclaration{}, errors.New("read Go-profile project go.mod")
	}
	if !info.Mode().IsRegular() {
		return goModuleDeclaration{}, errors.New("Go module file is non-regular; expected a regular file")
	}
	if info.Size() > maxGoModBytes {
		return goModuleDeclaration{}, fmt.Errorf("Go module file exceeds %d-byte limit", maxGoModBytes)
	}
	file, err := os.Open(candidate)
	if err != nil {
		return goModuleDeclaration{}, errors.New("read Go-profile project go.mod")
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxGoModBytes+1))
	if err != nil {
		return goModuleDeclaration{}, errors.New("read Go-profile project go.mod")
	}
	if len(content) > maxGoModBytes {
		return goModuleDeclaration{}, fmt.Errorf("Go module file exceeds %d-byte limit", maxGoModBytes)
	}
	return parseGoModuleDeclaration(content)
}

func parseGoModuleDeclaration(content []byte) (goModuleDeclaration, error) {
	stripped, err := stripGoModComments(content)
	if err != nil {
		return goModuleDeclaration{}, err
	}

	var declaration goModuleDeclaration
	moduleCount := 0
	goCount := 0
	toolchainCount := 0

	for lineNumber, line := range strings.Split(string(stripped), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "module":
			moduleCount++
			if moduleCount > 1 {
				return goModuleDeclaration{}, errors.New(
					"project go.mod contains duplicate module directives",
				)
			}
			if len(fields) != 2 {
				return goModuleDeclaration{}, fmt.Errorf(
					"project go.mod contains a malformed module directive on line %d",
					lineNumber+1,
				)
			}
			value, err := parseGoModDirectiveValue(fields[1])
			if err != nil || !validGoModulePath(value) {
				return goModuleDeclaration{}, fmt.Errorf(
					"project go.mod contains an invalid module directive on line %d",
					lineNumber+1,
				)
			}
			declaration.Module = value

		case "go":
			goCount++
			if goCount > 1 {
				return goModuleDeclaration{}, errors.New(
					"project go.mod contains duplicate go directives",
				)
			}
			if len(fields) != 2 {
				return goModuleDeclaration{}, fmt.Errorf(
					"project go.mod contains a malformed go directive on line %d",
					lineNumber+1,
				)
			}
			value, err := parseGoModDirectiveValue(fields[1])
			if err != nil || !goversion.IsValid("go"+value) {
				return goModuleDeclaration{}, fmt.Errorf(
					"project go.mod contains an invalid go directive on line %d",
					lineNumber+1,
				)
			}
			declaration.Minimum = "go" + value

		case "toolchain":
			toolchainCount++
			if toolchainCount > 1 {
				return goModuleDeclaration{}, errors.New(
					"project go.mod contains duplicate toolchain directives",
				)
			}
			if len(fields) != 2 {
				return goModuleDeclaration{}, fmt.Errorf(
					"project go.mod contains a malformed toolchain directive on line %d",
					lineNumber+1,
				)
			}
			value, err := parseGoModDirectiveValue(fields[1])
			if err != nil ||
				value != "default" &&
					!goversion.IsValid(value) {
				return goModuleDeclaration{}, fmt.Errorf(
					"project go.mod contains an invalid toolchain directive on line %d",
					lineNumber+1,
				)
			}
			declaration.Toolchain = value
		}
	}

	if declaration.Module == "" {
		return goModuleDeclaration{}, errors.New(
			"Go-profile project go.mod does not declare a module directive",
		)
	}
	if declaration.Minimum == "" {
		return goModuleDeclaration{}, errors.New(
			"Go-profile project go.mod does not declare a go directive",
		)
	}
	if declaration.Toolchain != "" &&
		declaration.Toolchain != "default" &&
		goversion.Compare(
			declaration.Toolchain,
			declaration.Minimum,
		) < 0 {
		return goModuleDeclaration{}, errors.New(
			"project go.mod toolchain directive is below the go directive minimum",
		)
	}
	return declaration, nil
}

func validGoModulePath(value string) bool {
	if value == "" ||
		strings.ContainsAny(value, "\x00\r\n\t \\") ||
		strings.HasPrefix(value, "/") ||
		strings.HasSuffix(value, "/") ||
		strings.Contains(value, "//") {
		return false
	}

	for _, component := range strings.Split(value, "/") {
		if component == "" ||
			component == "." ||
			component == ".." {
			return false
		}
	}
	return true
}

func parseGoModDirectiveValue(value string) (string, error) {
	if value == "" {
		return "", errors.New("empty directive value")
	}
	if value[0] != '"' && value[0] != '`' {
		return value, nil
	}
	decoded, err := strconv.Unquote(value)
	if err != nil || decoded == "" {
		return "", errors.New("invalid quoted directive value")
	}
	return decoded, nil
}

func stripGoModComments(content []byte) ([]byte, error) {
	if strings.IndexByte(string(content), 0) >= 0 {
		return nil, errors.New("project go.mod contains a prohibited NUL byte")
	}
	out := make([]byte, 0, len(content))
	state := 0
	escaped := false
	for i := 0; i < len(content); i++ {
		c := content[i]
		var n byte
		if i+1 < len(content) {
			n = content[i+1]
		}
		switch state {
		case 0:
			switch {
			case c == '/' && n == '/':
				state = 1
				out = append(out, ' ', ' ')
				i++
			case c == '/' && n == '*':
				state = 2
				out = append(out, ' ', ' ')
				i++
			case c == '"':
				state = 3
				out = append(out, c)
			case c == '`':
				state = 4
				out = append(out, c)
			default:
				out = append(out, c)
			}
		case 1:
			if c == '\n' {
				state = 0
				out = append(out, c)
			} else {
				out = append(out, ' ')
			}
		case 2:
			if c == '*' && n == '/' {
				out = append(out, ' ', ' ')
				i++
				state = 0
			} else if c == '\n' {
				out = append(out, '\n')
			} else {
				out = append(out, ' ')
			}
		case 3:
			out = append(out, c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				state = 0
			}
		case 4:
			out = append(out, c)
			if c == '`' {
				state = 0
			}
		}
	}
	if state == 2 {
		return nil, errors.New("project go.mod contains an unterminated block comment")
	}
	if state == 3 || state == 4 {
		return nil, errors.New("project go.mod contains an unterminated quoted value")
	}
	return out, nil
}

func rejectGoModuleSymlinkPath(root, candidate string) error {
	relative, err := filepath.Rel(root, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("Go module file escapes the target repository")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New("inspect Go module file path")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("Go module file path contains a symbolic link")
		}
	}
	return nil
}

func goVersionAtLeast(actual, minimum string) bool {
	return goversion.IsValid(actual) && goversion.IsValid(minimum) && goversion.Compare(actual, minimum) >= 0
}

func goVersionEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		name, _, found := strings.Cut(entry, "=")
		if found && (name == "GOTOOLCHAIN" || name == "GOENV") {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, "GOTOOLCHAIN=local", "GOENV=off")
}
