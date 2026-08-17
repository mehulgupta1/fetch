package resolve

import (
	"os"
	"testing"
)

// R5/R6/R7 via realVerify against real binaries.
func TestRealVerify(t *testing.T) {
	// /bin/echo exists and exits 0 with -h -> valid (R5)
	if !realVerify("/bin/echo") {
		t.Error("R5: /bin/echo should verify")
	}
	// /bin/false exits non-zero but RAN -> still valid (R6)
	if !realVerify("/bin/false") {
		t.Error("R6: a binary that runs (non-zero exit) is still valid")
	}
	// missing binary -> invalid (R7)
	if realVerify("/nonexistent/xyztool") {
		t.Error("R7: missing binary must be invalid")
	}
}

// R9: present but not verifiable -> not resolved (injected Verify=false)
func TestResolve_VerifyGate(t *testing.T) {
	dir := t.TempDir()
	writeExe(t, dir, "tool")
	r := &Resolver{GoBin: dir, LookPath: failLookPath, Verify: func(string) bool { return false }}
	if _, ok := r.Resolve("tool"); ok {
		t.Fatal("R9: unverifiable binary must not resolve")
	}
}

func TestVersion_NotFound(t *testing.T) { // CL4 support
	r := &Resolver{GoBin: t.TempDir(), LookPath: failLookPath, Verify: func(string) bool { return false }}
	if r.Version("ghost") != "not found" {
		t.Fatal("missing tool version = not found")
	}
}

func writeExe(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(dir+"/"+name, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
}
