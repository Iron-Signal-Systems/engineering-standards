package projectcommand

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGovulncheckApproval(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "tool-versions.json")
	writeGovulncheckToolConfig(
		t,
		path,
		"golang.org/x/vuln/cmd/govulncheck",
		"v1.6.0",
	)

	approval, err := loadGovulncheckApproval(path)
	if err != nil {
		t.Fatal(err)
	}
	if approval.CommandPackage != govulncheckCommandPackage {
		t.Fatalf(
			"command package = %q",
			approval.CommandPackage,
		)
	}
	if approval.Module != govulncheckModuleRoot {
		t.Fatalf("module = %q", approval.Module)
	}
	if approval.Version != "v1.6.0" {
		t.Fatalf("version = %q", approval.Version)
	}
}

func TestLoadGovulncheckApprovalRejectsHostileConfiguration(
	t *testing.T,
) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "unknown top-level field",
			content: `{
			  "version": 1,
			  "tools": {
			    "govulncheck": {
			      "module": "golang.org/x/vuln/cmd/govulncheck",
			      "version": "v1.6.0"
			    }
			  },
			  "unexpected": true
			}`,
			want: "unknown field",
		},
		{
			name: "missing declaration",
			content: `{
			  "version": 1,
			  "tools": {}
			}`,
			want: "declaration is missing",
		},
		{
			name: "wrong command package",
			content: `{
			  "version": 1,
			  "tools": {
			    "govulncheck": {
			      "module": "example.invalid/govulncheck",
			      "version": "v1.6.0"
			    }
			  }
			}`,
			want: "command package must be",
		},
		{
			name: "non-exact version",
			content: `{
			  "version": 1,
			  "tools": {
			    "govulncheck": {
			      "module": "golang.org/x/vuln/cmd/govulncheck",
			      "version": "latest"
			    }
			  }
			}`,
			want: "exact semantic version",
		},
		{
			name: "unknown tool field",
			content: `{
			  "version": 1,
			  "tools": {
			    "govulncheck": {
			      "module": "golang.org/x/vuln/cmd/govulncheck",
			      "version": "v1.6.0",
			      "digest": "unapproved"
			    }
			  }
			}`,
			want: "unknown field",
		},
		{
			name: "multiple JSON values",
			content: `{
			  "version": 1,
			  "tools": {
			    "govulncheck": {
			      "module": "golang.org/x/vuln/cmd/govulncheck",
			      "version": "v1.6.0"
			    }
			  }
			}
			{}`,
			want: "multiple JSON values",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(
				t.TempDir(),
				"tool-versions.json",
			)
			if err := os.WriteFile(
				path,
				[]byte(test.content),
				0o644,
			); err != nil {
				t.Fatal(err)
			}
			_, err := loadGovulncheckApproval(path)
			if err == nil ||
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLoadGovulncheckApprovalRejectsUnsafeFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.json")
	writeGovulncheckToolConfig(
		t,
		target,
		govulncheckCommandPackage,
		"v1.6.0",
	)
	link := filepath.Join(root, "tool-versions.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	_, err := loadGovulncheckApproval(link)
	if err == nil ||
		!strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyGovulncheckTool(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "tool-versions.json")
	writeGovulncheckToolConfig(
		t,
		config,
		govulncheckCommandPackage,
		"v1.6.0",
	)

	tool := filepath.Join(root, "govulncheck")
	toolBytes := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(tool, toolBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	selectedGo := writeFakeGoIdentity(
		t,
		root,
		tool,
		govulncheckCommandPackage,
		govulncheckModuleRoot,
		"v1.6.0",
	)

	identity, err := verifyGovulncheckTool(
		context.Background(),
		selectedGo,
		tool,
		config,
	)
	if err != nil {
		t.Fatal(err)
	}

	digest := sha256.Sum256(toolBytes)
	if identity.Executable != tool {
		t.Fatalf("executable = %q", identity.Executable)
	}
	if identity.Directory != root {
		t.Fatalf("directory = %q", identity.Directory)
	}
	if identity.CommandPackage != govulncheckCommandPackage {
		t.Fatalf(
			"command package = %q",
			identity.CommandPackage,
		)
	}
	if identity.Module != govulncheckModuleRoot {
		t.Fatalf("module = %q", identity.Module)
	}
	if identity.Version != "v1.6.0" {
		t.Fatalf("version = %q", identity.Version)
	}
	if identity.BuildGoVersion != "go1.26.5-X:nodwarf5" {
		t.Fatalf(
			"build Go version = %q",
			identity.BuildGoVersion,
		)
	}
	if identity.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("SHA256 = %q", identity.SHA256)
	}
}

func TestVerifyGovulncheckToolRejectsIdentityMismatch(t *testing.T) {
	tests := []struct {
		name           string
		commandPackage string
		module         string
		version        string
		want           string
	}{
		{
			name:           "command package",
			commandPackage: "example.invalid/govulncheck",
			module:         govulncheckModuleRoot,
			version:        "v1.6.0",
			want:           "command package mismatch",
		},
		{
			name:           "module",
			commandPackage: govulncheckCommandPackage,
			module:         "example.invalid/module",
			version:        "v1.6.0",
			want:           "module mismatch",
		},
		{
			name:           "version",
			commandPackage: govulncheckCommandPackage,
			module:         govulncheckModuleRoot,
			version:        "v1.5.0",
			want:           "version mismatch",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			config := filepath.Join(
				root,
				"tool-versions.json",
			)
			writeGovulncheckToolConfig(
				t,
				config,
				govulncheckCommandPackage,
				"v1.6.0",
			)
			tool := filepath.Join(root, "govulncheck")
			if err := os.WriteFile(
				tool,
				[]byte("#!/bin/sh\nexit 0\n"),
				0o755,
			); err != nil {
				t.Fatal(err)
			}
			selectedGo := writeFakeGoIdentity(
				t,
				root,
				tool,
				test.commandPackage,
				test.module,
				test.version,
			)

			_, err := verifyGovulncheckTool(
				context.Background(),
				selectedGo,
				tool,
				config,
			)
			if err == nil ||
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestVerifyGovulncheckToolRejectsUnsafeExecutable(
	t *testing.T,
) {
	root := t.TempDir()
	config := filepath.Join(root, "tool-versions.json")
	writeGovulncheckToolConfig(
		t,
		config,
		govulncheckCommandPackage,
		"v1.6.0",
	)
	target := filepath.Join(root, "target")
	if err := os.WriteFile(
		target,
		[]byte("#!/bin/sh\nexit 0\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "govulncheck")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	selectedGo := writeFakeGoIdentity(
		t,
		root,
		target,
		govulncheckCommandPackage,
		govulncheckModuleRoot,
		"v1.6.0",
	)

	_, err := verifyGovulncheckTool(
		context.Background(),
		selectedGo,
		link,
		config,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyGovulncheckToolNeverInstallsMissingTool(
	t *testing.T,
) {
	root := t.TempDir()
	config := filepath.Join(root, "tool-versions.json")
	writeGovulncheckToolConfig(
		t,
		config,
		govulncheckCommandPackage,
		"v1.6.0",
	)
	marker := filepath.Join(root, "selected-go-ran")
	selectedGo := filepath.Join(root, "go")
	script := "#!/bin/sh\n: >" + shellQuote(marker) + "\nexit 1\n"
	if err := os.WriteFile(
		selectedGo,
		[]byte(script),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	_, err := verifyGovulncheckTool(
		context.Background(),
		selectedGo,
		filepath.Join(root, "missing-govulncheck"),
		config,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "inspect govulncheck executable") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("selected Go executed for missing tool: %v", statErr)
	}
}

func writeGovulncheckToolConfig(
	t *testing.T,
	path string,
	module string,
	version string,
) {
	t.Helper()
	content := `{
  "version": 1,
  "tools": {
    "govulncheck": {
      "module": "` + module + `",
      "version": "` + version + `"
    }
  }
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFakeGoIdentity(
	t *testing.T,
	root string,
	tool string,
	commandPackage string,
	module string,
	version string,
) string {
	t.Helper()
	path := filepath.Join(root, "go")
	output := tool + ": go1.26.5-X:nodwarf5\n" +
		"\tpath\t" + commandPackage + "\n" +
		"\tmod\t" + module + "\t" + version + "\th1:test\n"
	script := `#!/bin/sh
if [ "$1" != "version" ] || [ "$2" != "-m" ]; then
	exit 91
fi
cat <<'EOF'
` + output + `EOF
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
