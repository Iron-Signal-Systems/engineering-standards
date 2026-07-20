package projectcommand

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGoVersionAtLeastAcceptsLaterCustomToolchain(t *testing.T) {
	if !goVersionAtLeast("go1.26.5-X:nodwarf5", "go1.25.12") {
		t.Fatal("later custom-suffix Go toolchain must satisfy the minimum")
	}
	if !goVersionAtLeast("go1.25.12", "go1.25.12") {
		t.Fatal("exact minimum Go toolchain must be accepted")
	}
	if goVersionAtLeast("go1.24.13", "go1.25.12") {
		t.Fatal("older Go toolchain must be rejected")
	}
}

func TestSelectGoToolchainUsesActivePathAndMinimumSemantics(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell script")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/toolchain\n\ngo 1.25.12\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "selected-go", "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeGo(t, filepath.Join(bin, "go"), "go1.26.5-X:nodwarf5")
	t.Setenv("PATH", bin)

	selection, err := selectGoToolchain(root)
	if err != nil {
		t.Fatalf("select later custom toolchain: %v", err)
	}
	if selection.Actual != "go1.26.5-X:nodwarf5" || selection.Minimum != "go1.25.12" {
		t.Fatalf("unexpected selection: %+v", selection)
	}
	if selection.Directory != bin {
		t.Fatalf("selected directory = %q, want %q", selection.Directory, bin)
	}

	pathValue := sanitizedCommandPath(filepath.Join(root, ".isras", "wrapper"), selection.Directory)
	parts := filepath.SplitList(pathValue)
	if len(parts) == 0 || parts[0] != bin {
		t.Fatalf("selected Go directory must be first in bounded PATH: %q", pathValue)
	}
}

func TestSelectGoToolchainRejectsVersionBelowMinimum(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell script")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/toolchain\n\ngo 1.25.12\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "old-go", "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeGo(t, filepath.Join(bin, "go"), "go1.24.13")
	t.Setenv("PATH", bin)

	_, err := selectGoToolchain(root)
	if err == nil || !strings.Contains(err.Error(), "below project minimum") {
		t.Fatalf("expected minimum-version rejection, got %v", err)
	}
}

func writeFakeGo(t *testing.T, path, version string) {
	t.Helper()
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = env ] && [ \"$2\" = GOVERSION ]; then\n" +
		"  printf '%s\\n' '" + version + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 91\n"
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
}
