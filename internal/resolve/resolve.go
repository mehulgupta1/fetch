// Package resolve locates external tools the way `go install` places them:
// gobin first, then PATH. Verify = "does it actually run".
package resolve

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// goEnv reads `go env <key>` (empty if go missing / key unset).
func goEnv(key string) string {
	out, err := exec.Command("go", "env", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GoBinDir returns where `go install` drops binaries:
// GOBIN, else first GOPATH/bin, else ~/go/bin.
func GoBinDir() string {
	if b := goEnv("GOBIN"); b != "" {
		return b
	}
	gp := goEnv("GOPATH")
	if gp == "" {
		gp = os.Getenv("GOPATH")
	}
	if gp != "" {
		first := strings.Split(gp, string(os.PathListSeparator))[0]
		return filepath.Join(first, "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}

// Resolver finds tools. Fields are injectable for tests.
type Resolver struct {
	GoBin    string
	LookPath func(string) (string, error)
	Verify   func(path string) bool
}

func New() *Resolver {
	return &Resolver{
		GoBin:    GoBinDir(),
		LookPath: exec.LookPath,
		Verify:   realVerify,
	}
}

// Resolve returns the tool's path if it is found AND verified.
// gobin dir is checked first (a fetch-installed copy wins over a
// same-named binary on PATH, e.g. Kali's python httpx problem).
func (r *Resolver) Resolve(tool string) (string, bool) {
	cand := filepath.Join(r.GoBin, tool)
	if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
		if r.Verify(cand) {
			return cand, true
		}
	}
	if p, err := r.LookPath(tool); err == nil {
		if r.Verify(p) {
			return p, true
		}
	}
	return "", false
}

// realVerify runs the binary with -h. "ran at all" (even non-zero exit)
// counts as valid; only a failure to START or a hang counts as invalid.
// stdin is nil so stdin-reading tools (hakrawler) can't block on us.
func realVerify(path string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-h")
	cmd.Stdin = nil
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return false
	}
	var ee *exec.ExitError
	if err == nil || errors.As(err, &ee) {
		return true
	}
	return false // failed to start (not found / not executable)
}

// Version returns a one-line version string for a tool (best-effort),
// or "not found" if it can't be resolved.
func (r *Resolver) Version(tool string) string {
	path, ok := r.Resolve(tool)
	if !ok {
		return "not found"
	}
	for _, flag := range []string{"-version", "--version", "-v"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, path, flag)
		cmd.Stdin = nil
		out, err := cmd.CombinedOutput()
		cancel()
		if line := firstNonEmpty(string(out)); err == nil && line != "" {
			return line
		}
	}
	return "installed"
}

func firstNonEmpty(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}
