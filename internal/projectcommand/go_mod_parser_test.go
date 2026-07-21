package projectcommand

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseGoModuleDeclarationAcceptsGovernedSyntax(t *testing.T) {
	tests := []struct{ name, content, minimum, toolchain string }{
		{"ordinary", "module example.test/root\n\ngo 1.25.12\n", "go1.25.12", ""},
		{"line comment", "module example.test/root\n\ngo 1.25.12 // minimum\n", "go1.25.12", ""},
		{"block comment", "module example.test/root\n\ngo 1.25.12 /* minimum */\n", "go1.25.12", ""},
		{"multiline block", "module example.test/root\n\n/* governed\ncomment */\ngo 1.25.12\n", "go1.25.12", ""},
		{"quoted", "module example.test/root\n\ngo \"1.25.12\"\n", "go1.25.12", ""},
		{"toolchain", "module example.test/root\n\ngo 1.25.12\ntoolchain go1.26.5-X:nodwarf5\n", "go1.25.12", "go1.26.5-X:nodwarf5"},
		{"default", "module example.test/root\n\ngo 1.25.12\ntoolchain default\n", "go1.25.12", "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseGoModuleDeclaration([]byte(tt.content))
			if err != nil {
				t.Fatal(err)
			}
			if d.Minimum != tt.minimum || d.Toolchain != tt.toolchain {
				t.Fatalf("%+v", d)
			}
		})
	}
}

func TestParseGoModuleDeclarationRejectsInvalidBoundaries(t *testing.T) {
	tests := []struct{ name, content, want string }{
		{"missing", "module x\n", "does not declare"}, {"malformed", "module x\n\ngo\n", "malformed go"}, {"duplicate", "module x\n\ngo 1.25.12\ngo 1.26.0\n", "duplicate go"}, {"invalid", "module x\n\ngo banana\n", "invalid go"}, {"duplicate toolchain", "module x\n\ngo 1.25.12\ntoolchain default\ntoolchain go1.26.0\n", "duplicate toolchain"}, {"bad toolchain", "module x\n\ngo 1.25.12\ntoolchain banana\n", "invalid toolchain"}, {"old toolchain", "module x\n\ngo 1.26.0\ntoolchain go1.25.12\n", "below the go directive"}, {"comment", "module x\n\ngo 1.25.12 /*", "unterminated block"}, {"quote", "module x\n\ngo \"1.25.12\n", "unterminated quoted"}, {"nul", "module x\n\ngo 1.25.12\x00\n", "NUL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseGoModuleDeclaration([]byte(tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err=%v want=%q", err, tt.want)
			}
		})
	}
}

func TestReadGoModuleDeclarationSupportsNestedModules(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "modules", "worker")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "go.mod"), []byte("module example.test/worker\n\ngo 1.26.0\ntoolchain default\n"), 0600); err != nil {
		t.Fatal(err)
	}
	d, err := readGoModuleDeclaration(root, "modules/worker/go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if d.Minimum != "go1.26.0" || d.Toolchain != "default" {
		t.Fatalf("%+v", d)
	}
}

func TestReadGoModuleDeclarationRejectsUnsafeFiles(t *testing.T) {
	t.Run("escape", func(t *testing.T) {
		_, err := readGoModuleDeclaration(t.TempDir(), "../go.mod")
		if err == nil || !strings.Contains(err.Error(), "unsafe") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		external := filepath.Join(t.TempDir(), "go.mod")
		_ = os.WriteFile(external, []byte("module x\n\ngo 1.25.12\n"), 0600)
		_ = os.Symlink(external, filepath.Join(root, "go.mod"))
		_, err := readGoModuleDeclaration(root, "go.mod")
		if err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("nonregular", func(t *testing.T) {
		root := t.TempDir()
		_ = os.Mkdir(filepath.Join(root, "go.mod"), 0700)
		_, err := readGoModuleDeclaration(root, "go.mod")
		if err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("unreadable", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip()
		}
		root := t.TempDir()
		p := filepath.Join(root, "go.mod")
		_ = os.WriteFile(p, []byte("module x\n\ngo 1.25.12\n"), 0000)
		defer os.Chmod(p, 0600)
		_, err := readGoModuleDeclaration(root, "go.mod")
		if err == nil || !strings.Contains(err.Error(), "read Go-profile") {
			t.Fatalf("%v", err)
		}
	})
}
