package hostedtrust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type signerRecord struct {
	Principal   string `json:"principal"`
	Fingerprint string `json:"fingerprint"`
}

type trustManifest struct {
	SchemaVersion int            `json:"schema_version"`
	Authority     string         `json:"authority"`
	File          string         `json:"file"`
	SHA256        string         `json:"sha256"`
	Signers       []signerRecord `json:"signers"`
}

func TestHostedSSHSignerTrustBootstrap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SSH signing regression suite requires a Unix-like runner")
	}
	for _, command := range []string{"bash", "git", "python3", "sha256sum", "ssh-keygen"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("required command unavailable: %s", command)
		}
	}

	correct := generateKey(t, "correct")
	wrong := generateKey(t, "wrong")
	principal := "isras-test@example.invalid"
	target := createSignedTarget(t, correct.private, principal)

	t.Run("correct key and principal", func(t *testing.T) {
		standard := createStandard(t, principal, correct.public, false)
		result := runBootstrap(t, standard, target)
		if result.err != nil {
			t.Fatalf("correct trust was rejected: %v\n%s", result.err, result.output)
		}
		if !strings.Contains(result.output, "HOSTED SSH SIGNER TRUST: PASS") {
			t.Fatalf("success output is incomplete:\n%s", result.output)
		}
	})

	t.Run("missing trust source", func(t *testing.T) {
		standard := createStandard(t, principal, correct.public, false)
		if err := os.Remove(filepath.Join(standard, "trust", "ssh", "iron-signal-systems.allowed-signers")); err != nil {
			t.Fatal(err)
		}
		result := runBootstrap(t, standard, target)
		if result.err == nil {
			t.Fatal("missing trust source was accepted")
		}
	})

	t.Run("altered trust source", func(t *testing.T) {
		standard := createStandard(t, principal, correct.public, false)
		path := filepath.Join(standard, "trust", "ssh", "iron-signal-systems.allowed-signers")
		if err := os.WriteFile(path, []byte(principal+" "+wrong.public+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		result := runBootstrap(t, standard, target)
		if result.err == nil {
			t.Fatal("altered trust source was accepted")
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		standard := createStandard(t, principal, wrong.public, false)
		result := runBootstrap(t, standard, target)
		if result.err == nil {
			t.Fatal("wrong signing key was accepted")
		}
	})

	t.Run("wrong principal", func(t *testing.T) {
		standard := createStandard(t, "wrong-principal@example.invalid", correct.public, false)
		result := runBootstrap(t, standard, target)
		if result.err == nil {
			t.Fatal("wrong signer principal was accepted")
		}
		if !strings.Contains(result.output, "does not match the exact commit committer email") {
			t.Fatalf("wrong-principal failure was not explicit:\n%s", result.output)
		}
	})
}

type keyPair struct {
	private string
	public  string
}

func generateKey(t *testing.T, name string) keyPair {
	t.Helper()
	directory := t.TempDir()
	private := filepath.Join(directory, name)
	run(t, directory, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", private)
	publicBytes, err := os.ReadFile(private + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(publicBytes))
	if len(fields) < 2 {
		t.Fatal("generated public key is invalid")
	}
	return keyPair{private: private, public: fields[0] + " " + fields[1]}
}

func createSignedTarget(t *testing.T, private, principal string) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "git", "init", "-b", "dev")
	run(t, root, "git", "config", "user.name", "ISRAS Test")
	run(t, root, "git", "config", "user.email", principal)
	run(t, root, "git", "config", "gpg.format", "ssh")
	run(t, root, "git", "config", "user.signingkey", private)
	run(t, root, "git", "config", "commit.gpgsign", "true")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "git", "add", "README.md")
	run(t, root, "git", "commit", "-S", "-m", "signed fixture")
	return root
}

func createStandard(t *testing.T, principal, public string, alterAfterCommit bool) string {
	t.Helper()
	root := t.TempDir()
	tools := filepath.Join(root, "tools")
	trust := filepath.Join(root, "trust", "ssh")
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(trust, 0o755); err != nil {
		t.Fatal(err)
	}

	sourceScript := repositoryFile(t, "tools", "configure-hosted-ssh-signing-trust.sh")
	scriptBytes, err := os.ReadFile(sourceScript)
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(tools, "configure-hosted-ssh-signing-trust.sh")
	if err := os.WriteFile(script, scriptBytes, 0o755); err != nil {
		t.Fatal(err)
	}

	allowedName := "iron-signal-systems.allowed-signers"
	allowed := []byte(principal + " " + public + "\n")
	allowedPath := filepath.Join(trust, allowedName)
	if err := os.WriteFile(allowedPath, allowed, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(allowed)
	digestText := hex.EncodeToString(digest[:])
	if err := os.WriteFile(
		filepath.Join(trust, allowedName+".sha256"),
		[]byte(digestText+"  "+allowedName+"\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	fingerprint := fingerprint(t, public)
	manifest := trustManifest{
		SchemaVersion: 1,
		Authority:     "Iron Signal Systems Engineering Standards",
		File:          allowedName,
		SHA256:        digestText,
		Signers:       []signerRecord{{Principal: principal, Fingerprint: fingerprint}},
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := os.WriteFile(filepath.Join(trust, "manifest.json"), manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	run(t, root, "git", "init", "-b", "dev")
	run(t, root, "git", "config", "user.name", "ISRAS Test")
	run(t, root, "git", "config", "user.email", "isras-test@example.invalid")
	run(t, root, "git", "add", ".")
	run(t, root, "git", "-c", "commit.gpgsign=false", "commit", "-m", "trust fixture")
	if alterAfterCommit {
		if err := os.WriteFile(allowedPath, append(allowed, '#'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func fingerprint(t *testing.T, public string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.pub")
	if err := os.WriteFile(path, []byte(public+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := run(t, filepath.Dir(path), "ssh-keygen", "-lf", path, "-E", "sha256")
	fields := strings.Fields(output)
	if len(fields) < 2 {
		t.Fatalf("fingerprint output is invalid: %s", output)
	}
	return fields[1]
}

type commandResult struct {
	output string
	err    error
}

func runBootstrap(t *testing.T, standard, target string) commandResult {
	t.Helper()
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	script := filepath.Join(standard, "tools", "configure-hosted-ssh-signing-trust.sh")
	command := exec.Command("bash", script, "--target", target, "--runtime-root", runtimeRoot)
	output, err := command.CombinedOutput()
	return commandResult{output: string(output), err: err}
}

func repositoryFile(t *testing.T, parts ...string) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	return filepath.Join(append([]string{root}, parts...)...)
}

func run(t *testing.T, directory, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return string(output)
}
