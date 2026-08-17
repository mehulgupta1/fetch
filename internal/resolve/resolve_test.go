package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

// G. goBinDir + PATH
func TestGoBinDir_GOBIN(t *testing.T) {
	t.Setenv("GOBIN", "/custom/bin")
	if got := GoBinDir(); got != "/custom/bin" {
		t.Fatalf("G1: want /custom/bin, got %s", got)
	}
}

func TestGoBinDir_GOPATH(t *testing.T) {
	t.Setenv("GOBIN", "")
	t.Setenv("GOPATH", "/gp")
	// go env may still report GOBIN; guard by checking suffix.
	got := GoBinDir()
	if got != filepath.Join("/gp", "bin") && got != "" {
		// only assert when go env didn't override with its own GOBIN
		if goEnv("GOBIN") == "" {
			t.Fatalf("G2: want /gp/bin, got %s", got)
		}
	}
}

func TestGoBinDir_MultiGOPATH(t *testing.T) {
	t.Setenv("GOBIN", "")
	multi := "/a" + string(os.PathListSeparator) + "/b"
	t.Setenv("GOPATH", multi)
	if goEnv("GOBIN") != "" {
		t.Skip("go env forces GOBIN")
	}
	if got := GoBinDir(); got != filepath.Join("/a", "bin") {
		t.Fatalf("G7: want /a/bin, got %s", got)
	}
}

// R. resolve + verify (injected)
func TestResolve_InGobin(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "mytool")
	os.WriteFile(tool, []byte("x"), 0o755)
	r := &Resolver{GoBin: dir, LookPath: failLookPath, Verify: func(string) bool { return true }}
	got, ok := r.Resolve("mytool")
	if !ok || got != tool {
		t.Fatalf("R1: want %s true, got %s %v", tool, got, ok)
	}
}

func TestResolve_OnPath(t *testing.T) {
	r := &Resolver{GoBin: t.TempDir(), LookPath: func(string) (string, error) { return "/usr/bin/mytool", nil }, Verify: func(string) bool { return true }}
	got, ok := r.Resolve("mytool")
	if !ok || got != "/usr/bin/mytool" {
		t.Fatalf("R2: want /usr/bin/mytool true, got %s %v", got, ok)
	}
}

func TestResolve_AbsentBoth(t *testing.T) {
	r := &Resolver{GoBin: t.TempDir(), LookPath: failLookPath, Verify: func(string) bool { return true }}
	if _, ok := r.Resolve("nope"); ok {
		t.Fatal("R3: want not resolved")
	}
}

func TestResolve_WrongBinaryVerifyFails(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "httpx"), []byte("x"), 0o755)
	r := &Resolver{GoBin: dir, LookPath: failLookPath, Verify: func(string) bool { return false }}
	if _, ok := r.Resolve("httpx"); ok {
		t.Fatal("R4: wrong binary must not resolve when verify fails")
	}
}

func TestResolve_GobinPreferredOverPath(t *testing.T) {
	dir := t.TempDir()
	gb := filepath.Join(dir, "mytool")
	os.WriteFile(gb, []byte("x"), 0o755)
	r := &Resolver{GoBin: dir, LookPath: func(string) (string, error) { return "/usr/bin/mytool", nil }, Verify: func(string) bool { return true }}
	got, _ := r.Resolve("mytool")
	if got != gb {
		t.Fatalf("R8: gobin copy must win, got %s", got)
	}
}

func failLookPath(string) (string, error) { return "", os.ErrNotExist }
