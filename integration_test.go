//go:build integration

// Real end-to-end smoke test (IT group). Gated: only runs with
//   go test -tags integration ./...
// Requires the real tools installed + network. NOT run by `go test ./...`.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mehulgupta1/fetch/internal/cli"
)

func realDeps(o, e *bytes.Buffer) cli.Deps {
	return cli.Deps{Stdout: o, Stderr: e, Stdin: strings.NewReader(""), StdinPiped: false}
}

// IT1/IT3/IT5: -l grep over a real list creates js.txt with the .js line.
func TestIT_GrepList(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "js.txt")
	lf := filepath.Join(dir, "l.txt")
	os.WriteFile(lf, []byte("https://example.com/x.js\nhttps://example.com/y.css\n"), 0o644)
	var o, e bytes.Buffer
	code := cli.Run([]string{"-l", lf, "--silent", "-o", out}, realDeps(&o, &e))
	if code != 0 {
		t.Fatalf("IT5: exit %d", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal("IT1: js.txt not created")
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "x.js") || strings.Contains(string(data), "y.css") {
		t.Fatalf("IT3: grep should keep only .js, got %q", data)
	}
}

// IT2/IT4/IT5: a real crawl of example.com runs the actual tools, completes,
// exits 0, and every output line is a real .js url.
func TestIT_RealCrawl(t *testing.T) {
	out := filepath.Join(t.TempDir(), "js.txt")
	var o, e bytes.Buffer
	code := cli.Run([]string{"-d", "example.com", "-timeout", "30s", "--silent", "-o", out}, realDeps(&o, &e))
	if code != 0 {
		t.Fatalf("IT5: exit %d stderr=%s", code, e.String())
	}
	data, _ := os.ReadFile(out)
	for _, ln := range strings.Fields(string(data)) {
		base := strings.SplitN(strings.SplitN(ln, "?", 2)[0], "#", 2)[0]
		if !strings.HasSuffix(strings.ToLower(base), ".js") {
			t.Fatalf("IT2: non-js line in output: %q", ln)
		}
	}
}
