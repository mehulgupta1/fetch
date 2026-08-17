package run

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fake exec keyed by tool path (== name in tests)
type fexec struct {
	urls map[string][]string
	fail map[string]bool
}

func (f fexec) Run(_ context.Context, argv []string, _ string, _ []string, onLine func(string)) (string, error) {
	for _, u := range f.urls[argv[0]] {
		onLine(u)
	}
	if f.fail[argv[0]] {
		return "boom", errFail
	}
	return "", nil
}

var errFail = &exitErr{}

type exitErr struct{}

func (*exitErr) Error() string { return "exit 1" }

// urlscan http fake: empty search
type emptyHTTP struct{}

func (emptyHTTP) Get(_ context.Context, url, _ string) ([]byte, int, int, error) {
	return []byte(`{"results":[]}`), 200, 0, nil
}

func baseDeps(f fexec) Deps {
	return Deps{
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Exec:    f,
		HTTP:    emptyHTTP{},
		Resolve: func(t string) (string, bool) { return t, true },
	}
}

func TestRun_LOnly(t *testing.T) { // O2: grep only
	out := filepath.Join(t.TempDir(), "js.txt")
	cfg := Config{GrepLines: []string{"https://t/a.js", "https://t/x.css"}, OutPath: out, Silent: true, Timeout: time.Second}
	d := baseDeps(fexec{})
	d.Stdout = &bytes.Buffer{}
	code := Run(context.Background(), cfg, d)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	data, _ := os.ReadFile(out)
	if strings.TrimSpace(string(data)) != "https://t/a.js" {
		t.Fatalf("O2: want a.js only, got %q", data)
	}
}

func TestRun_MergeDedup(t *testing.T) { // O25 merge dedup
	out := filepath.Join(t.TempDir(), "js.txt")
	f := fexec{urls: map[string][]string{
		"subjs":     {"https://t/a.js"},
		"getJS":     {"https://t/a.js", "https://t/b.js"},
		"katana":    {"https://t/c.js"},
		"hakrawler": {},
	}}
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: out, Silent: true, Timeout: time.Second}
	code := Run(context.Background(), cfg, baseDeps(f))
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	data, _ := os.ReadFile(out)
	got := strings.Fields(string(data))
	if len(got) != 3 { // a,b,c unique
		t.Fatalf("O25: want 3 unique, got %v", got)
	}
}

func TestRun_InScope(t *testing.T) { // in-scope drop
	out := filepath.Join(t.TempDir(), "js.txt")
	cfg := Config{
		GrepLines: []string{"https://target.com/a.js", "https://cdn.net/b.js"},
		HostForms: []string{"target.com"}, InScope: true,
		OutPath: out, Silent: true, Timeout: time.Second,
	}
	Run(context.Background(), cfg, baseDeps(fexec{}))
	data, _ := os.ReadFile(out)
	if strings.TrimSpace(string(data)) != "https://target.com/a.js" {
		t.Fatalf("in-scope: got %q", data)
	}
}

func TestRun_AllToolsFail(t *testing.T) { // EX4 all failed -> exit 1
	f := fexec{fail: map[string]bool{"subjs": true, "getJS": true, "katana": true, "hakrawler": true}}
	// urlscan empty-ok would make anyOK true; use a host that fails via 404 http
	d := baseDeps(f)
	d.HTTP = failHTTP{}
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: filepath.Join(t.TempDir(), "o"), Silent: true, Timeout: time.Second}
	code := Run(context.Background(), cfg, d)
	if code != 1 {
		t.Fatalf("EX4: want exit 1, got %d", code)
	}
}

type failHTTP struct{}

func (failHTTP) Get(_ context.Context, url, _ string) ([]byte, int, int, error) {
	return nil, 500, 0, nil
}

func TestRun_Report(t *testing.T) { // RP: report to stderr
	var errb bytes.Buffer
	d := baseDeps(fexec{})
	d.Stderr = &errb
	cfg := Config{GrepLines: []string{"https://t/a.js"}, OutPath: filepath.Join(t.TempDir(), "o"), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	s := errb.String()
	if !strings.Contains(s, "fetch done") || !strings.Contains(s, "found") {
		t.Fatalf("RP: report missing: %q", s)
	}
}

func TestRun_SilentStdout(t *testing.T) { // SL3
	var out bytes.Buffer
	d := baseDeps(fexec{})
	d.Stdout = &out
	cfg := Config{GrepLines: []string{"https://t/a.js"}, Silent: true, Timeout: time.Second, OutPath: filepath.Join(t.TempDir(), "o")}
	Run(context.Background(), cfg, d)
	if strings.TrimSpace(out.String()) != "https://t/a.js" {
		t.Fatalf("SL3: stdout got %q", out.String())
	}
}
