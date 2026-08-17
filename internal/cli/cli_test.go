package cli

import (
	"bytes"
	"strings"
	"testing"
)

func cliRun(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, Deps{Stdout: &out, Stderr: &errb, Stdin: strings.NewReader("")})
	return code, out.String(), errb.String()
}

func TestDispatch_BareHelp(t *testing.T) { // CL1
	code, out, _ := cliRun()
	if code != 0 || !strings.Contains(out, "js file collector") {
		t.Fatalf("CL1: code=%d out=%q", code, out)
	}
}

func TestDispatch_HelpFlag(t *testing.T) { // CL2
	for _, f := range []string{"-h", "--help"} {
		code, out, _ := cliRun(f)
		if code != 0 || !strings.Contains(out, "COMMANDS") {
			t.Fatalf("CL2 %s: code=%d", f, code)
		}
	}
}

func TestDispatch_Version(t *testing.T) { // CL3/CL4
	code, out, _ := cliRun("--version")
	if code != 0 || !strings.Contains(out, "fetch "+Version) {
		t.Fatalf("CL3: code=%d out=%q", code, out)
	}
	// tool lines present (value is "not found" in a bare test env = CL4)
	if !strings.Contains(out, "subjs") || !strings.Contains(out, "katana") {
		t.Fatalf("CL4: tool versions missing: %q", out)
	}
}

func TestDispatch_ConfigFlag(t *testing.T) { // CL7 + C1 path
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, out, _ := cliRun("-config", "--urlscan-key", "secretKEY")
	if code != 0 || !strings.Contains(out, "saved urlscan key") {
		t.Fatalf("CL7: code=%d out=%q", code, out)
	}
	if strings.Contains(out, "secretKEY") { // masked
		t.Fatalf("key must be masked in output: %q", out)
	}
}

func TestDispatch_ConfigSkip(t *testing.T) { // C3
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, out, _ := cliRun("-config") // no key, empty stdin
	if code != 0 || !strings.Contains(out, "nothing changed") {
		t.Fatalf("C3: code=%d out=%q", code, out)
	}
}

func TestDispatch_RunNoInput(t *testing.T) { // CL9 / O1
	code, _, errs := cliRun("--depth", "3")
	if code != 1 || !strings.Contains(errs, "no input") {
		t.Fatalf("CL9: code=%d err=%q", code, errs)
	}
}

func TestNormalizeDomain(t *testing.T) { // DN
	cases := []struct{ in, url, host string }{
		{"example.com", "https://example.com", "example.com"},       // DN1
		{"https://example.com", "https://example.com", "example.com"}, // DN2
		{"http://example.com", "http://example.com", "example.com"},   // DN3
		{"Example.com", "https://example.com", "example.com"},         // DN13 lowercase
		{"//example.com", "https://example.com", "example.com"},       // DN7
		{"  example.com  ", "https://example.com", "example.com"},     // DN8
	}
	for _, c := range cases {
		u, h, ok := normalizeDomain(c.in)
		if !ok || u != c.url || h != c.host {
			t.Errorf("normalizeDomain(%q)=(%q,%q,%v) want (%q,%q)", c.in, u, h, ok, c.url, c.host)
		}
	}
	if _, _, ok := normalizeDomain("# comment"); ok {
		t.Error("DN9: comment should be skipped")
	}
}

func TestResolveD_MissingFileGuard(t *testing.T) { // O11b
	if _, _, err := resolveD("subs.txt"); err == nil {
		t.Fatal("O11b: missing .txt file must error")
	}
	if _, _, err := resolveD("example.com"); err != nil {
		t.Fatalf("O11: bare domain must not error: %v", err)
	}
}

func TestFlagValue(t *testing.T) {
	if flagValue([]string{"--urlscan-key", "abc"}, "--urlscan-key") != "abc" {
		t.Fatal("space form")
	}
	if flagValue([]string{"--urlscan-key=xyz"}, "--urlscan-key") != "xyz" {
		t.Fatal("equals form")
	}
	if flagValue([]string{"--other"}, "--urlscan-key") != "" {
		t.Fatal("absent")
	}
}
