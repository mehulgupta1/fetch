package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cliRunP(stdin string, piped bool, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, Deps{Stdout: &out, Stderr: &errb, Stdin: strings.NewReader(stdin), StdinPiped: piped})
	return code, out.String(), errb.String()
}

// ---- V validation (all return exit 1 before any network) ----
func TestV_Validation(t *testing.T) {
	cases := []struct {
		id   string
		args []string
	}{
		{"V1", []string{"-d", "example.com", "--depth", "-1"}},
		{"V2", []string{"-d", "example.com", "-c", "0"}},
		{"V3", []string{"-d", "example.com", "--rate", "-1"}},
		{"V4", []string{"-d", "example.com", "--proxy", "notaurl"}},
		{"V5", []string{"-d", "example.com", "-H", "nocolon"}},
		{"V8", []string{"-d", "example.com", "-timeout", "abc"}},
		{"V10", []string{"-d", "example.com", "--urlscan-limit", "-5"}},
	}
	for _, c := range cases {
		code, _, _ := cliRunP("", false, c.args...)
		if code != 1 {
			t.Errorf("%s: want exit 1, got %d", c.id, code)
		}
	}
}

func TestV6_InScopeNeedsD(t *testing.T) {
	code, _, errs := cliRunP("https://t/a.js\n", true, "--in-scope", "-o", tmpf(t))
	if code != 1 || !strings.Contains(errs, "--in-scope requires -d") {
		t.Fatalf("V6: code=%d err=%q", code, errs)
	}
}

// ---- CL dispatch ----
func TestCL5_UnknownFlag(t *testing.T) {
	code, _, _ := cliRunP("", false, "--nope")
	if code != 1 {
		t.Fatalf("CL5: unknown flag -> exit 1, got %d", code)
	}
}

func TestCL8_CommandBeatsRunFlags(t *testing.T) {
	// --version should win even with an otherwise-invalid run flag
	code, out, _ := cliRunP("", false, "--version", "--depth", "-1")
	if code != 0 || !strings.Contains(out, "fetch "+Version) {
		t.Fatalf("CL8: version must win, got code=%d", code)
	}
}

func TestCL10_DuplicateLastWins(t *testing.T) {
	// last --depth (-1) wins -> validation fails -> exit 1 (if first won, it would run)
	code, _, _ := cliRunP("", false, "-d", "example.com", "--depth", "2", "--depth", "-1")
	if code != 1 {
		t.Fatalf("CL10: last-wins should make depth -1 -> exit 1, got %d", code)
	}
}

// ---- DN normalization (remaining) ----
func TestDN_More(t *testing.T) {
	cases := []struct{ id, in, url, host string }{
		{"DN4", "HTTP://example.com", "http://example.com", "example.com"},
		{"DN5", "example.com:8080", "https://example.com:8080", "example.com"},
		{"DN6", "sub.example.com/path", "https://sub.example.com/path", "sub.example.com"},
		{"DN11", "ftp://example.com", "ftp://example.com", "example.com"},
	}
	for _, c := range cases {
		u, h, ok := normalizeDomain(c.in)
		if !ok || u != c.url || h != c.host {
			t.Errorf("%s normalizeDomain(%q)=(%q,%q,%v)", c.id, c.in, u, h, ok)
		}
	}
}

func TestDN10_DedupHosts(t *testing.T) {
	// example.com AND https://example.com collapse to one url + one host
	dir := t.TempDir()
	f := filepath.Join(dir, "d.txt")
	os.WriteFile(f, []byte("example.com\nhttps://example.com\nExample.com\n"), 0o644)
	urls, hosts, err := resolveD(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 || len(hosts) != 1 {
		t.Fatalf("DN10: want 1 url 1 host, got %v %v", urls, hosts)
	}
}

// ---- ST stdin ----
func TestST1_PipedStdin(t *testing.T) {
	out := tmpf(t)
	code, _, _ := cliRunP("https://t/a.js\nhttps://t/x.css\n", true, "--silent", "-o", out)
	if code != 0 {
		t.Fatalf("ST1: code %d", code)
	}
	data, _ := os.ReadFile(out)
	if strings.TrimSpace(string(data)) != "https://t/a.js" {
		t.Fatalf("ST1: got %q", data)
	}
}

func TestST2_FileWinsOverStdin(t *testing.T) {
	dir := t.TempDir()
	lf := filepath.Join(dir, "l.txt")
	os.WriteFile(lf, []byte("https://f/a.js\n"), 0o644)
	out := filepath.Join(dir, "o.txt")
	cliRunP("https://s/b.js\n", true, "--silent", "-l", lf, "-o", out)
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "f/a.js") || strings.Contains(string(data), "s/b.js") {
		t.Fatalf("ST2: -l file must win over stdin, got %q", data)
	}
}

func TestST3_DashReadsStdin(t *testing.T) {
	out := tmpf(t)
	cliRunP("https://t/a.js\n", false, "--silent", "-l", "-", "-o", out)
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "a.js") {
		t.Fatalf("ST3: -l - reads stdin, got %q", data)
	}
}

func TestST6_EmptyStdinNoInput(t *testing.T) {
	// piped but empty, and no -d -> no input -> usage error
	code, _, errs := cliRunP("", true, "-o", tmpf(t))
	if code != 1 || !strings.Contains(errs, "no input") {
		t.Fatalf("ST6: empty stdin no -d -> exit 1, got %d %q", code, errs)
	}
}

func TestST7_BlankLinesSkipped(t *testing.T) {
	out := tmpf(t)
	cliRunP("\n  \nhttps://t/a.js\n\n", true, "--silent", "-o", out)
	data, _ := os.ReadFile(out)
	if strings.TrimSpace(string(data)) != "https://t/a.js" {
		t.Fatalf("ST7: blanks skipped, got %q", data)
	}
}

// ---- LR large-line / encoding ----
func TestLR1_LargeLine(t *testing.T) {
	// a url line > 64KB must NOT be silently dropped (big-buffer reader)
	big := "https://t.com/" + strings.Repeat("a", 100*1024) + ".js"
	out := tmpf(t)
	cliRunP(big+"\n", true, "--silent", "-o", out)
	data, _ := os.ReadFile(out)
	if len(strings.TrimSpace(string(data))) < 64*1024 {
		t.Fatalf("LR1: >64KB url line dropped (got %d bytes)", len(data))
	}
}

func TestLR3_CRLF(t *testing.T) {
	out := tmpf(t)
	cliRunP("https://t.com/a.js\r\nhttps://t.com/b.js\r\n", true, "--silent", "-o", out)
	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "\r") {
		t.Fatalf("LR3: CRLF \\r must be trimmed, got %q", data)
	}
	if !strings.Contains(string(data), "a.js") {
		t.Fatalf("LR3: content lost, got %q", data)
	}
}

func TestLR4_NoTrailingNewline(t *testing.T) {
	out := tmpf(t)
	cliRunP("https://t.com/a.js", true, "--silent", "-o", out) // no trailing \n
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "a.js") {
		t.Fatalf("LR4: last line without newline lost")
	}
}

// ---- redirect following ----
func TestExpandRedirects_AddsLandingHost(t *testing.T) {
	orig := finalURL
	defer func() { finalURL = orig }()
	finalURL = func(u string) string {
		if strings.Contains(u, "bmw.com") {
			return "https://www.bmw.in/en/index.html"
		}
		return u
	}
	urls, hosts := expandRedirects([]string{"https://bmw.com"})
	if !containsS(urls, "https://bmw.com") {
		t.Fatal("must keep the original target")
	}
	if !containsS(hosts, "bmw.com") || !containsS(hosts, "www.bmw.in") {
		t.Fatalf("must add landing host www.bmw.in, got hosts=%v", hosts)
	}
}

func TestExpandRedirects_NoRedirect(t *testing.T) {
	orig := finalURL
	defer func() { finalURL = orig }()
	finalURL = func(u string) string { return u } // no redirect
	urls, hosts := expandRedirects([]string{"https://x.com"})
	if len(urls) != 1 || len(hosts) != 1 {
		t.Fatalf("same-host landing adds nothing, got %v %v", urls, hosts)
	}
}

func containsS(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func tmpf(t *testing.T) string { return filepath.Join(t.TempDir(), "js.txt") }
