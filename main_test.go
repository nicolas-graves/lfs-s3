package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles the package into a temp binary and returns its path.
// Version is injected via ldflags, mirroring the Makefile's build command.
func buildBinary(t *testing.T, version string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "lfs-s3-test")
	cmd := exec.Command("go", "build",
		"-ldflags", "-X main.Version="+version,
		"-o", bin, ".")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// TestVersionFlag is the acceptance test: `lfs-s3 --version` must print the
// injected version and exit 0 without starting the transfer agent loop.
func TestVersionFlag(t *testing.T) {
	const want = "9.9.9-test"
	bin := buildBinary(t, want)

	cmd := exec.Command(bin, "--version")
	// No stdin provided: if --version did not short-circuit, Serve would
	// block/fail reading the LFS protocol from stdin.
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 for --version, got error: %v\noutput: %s", err, out)
	}
	got := strings.TrimSpace(string(out))
	if !strings.Contains(got, want) {
		t.Fatalf("expected --version output to contain %q, got %q", want, got)
	}
}
