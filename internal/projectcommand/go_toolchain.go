package projectcommand

import (
	"bufio"
	"errors"
	"fmt"
	goversion "go/version"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type goToolchainSelection struct {
	Executable string
	Directory  string
	Minimum    string
	Actual     string
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
	minimum, err := readMinimumGoVersion(root)
	if err != nil {
		return goToolchainSelection{}, err
	}

	executable, err := exec.LookPath("go")
	if err != nil {
		return goToolchainSelection{}, errors.New("Go-profile project requires an available Go toolchain")
	}
	if !filepath.IsAbs(executable) {
		executable, err = filepath.Abs(executable)
		if err != nil {
			return goToolchainSelection{}, errors.New("resolve selected Go executable")
		}
	}
	executable = filepath.Clean(executable)
	info, err := os.Stat(executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return goToolchainSelection{}, errors.New("selected Go executable is not a regular executable file")
	}

	command := exec.Command(executable, "env", "GOVERSION")
	command.Dir = root
	command.Env = goVersionEnvironment()
	output, err := command.Output()
	if err != nil {
		return goToolchainSelection{}, errors.New("query selected Go toolchain version")
	}
	actual := strings.TrimSpace(string(output))
	if !goversion.IsValid(actual) {
		return goToolchainSelection{}, fmt.Errorf("selected Go toolchain reported an invalid version: %q", actual)
	}
	if !goVersionAtLeast(actual, minimum) {
		return goToolchainSelection{}, fmt.Errorf("selected Go toolchain %s is below project minimum %s", actual, minimum)
	}

	return goToolchainSelection{
		Executable: executable,
		Directory:  filepath.Dir(executable),
		Minimum:    minimum,
		Actual:     actual,
	}, nil
}

func readMinimumGoVersion(root string) (string, error) {
	file, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", errors.New("read Go-profile project go.mod")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || fields[0] != "go" {
			continue
		}
		minimum := "go" + fields[1]
		if !goversion.IsValid(minimum) {
			return "", fmt.Errorf("project go.mod contains an invalid go directive: %q", fields[1])
		}
		return minimum, nil
	}
	if err := scanner.Err(); err != nil {
		return "", errors.New("scan Go-profile project go.mod")
	}
	return "", errors.New("Go-profile project go.mod does not declare a go directive")
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
