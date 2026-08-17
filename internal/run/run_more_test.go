package run

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mehulgupta1/fetch/internal/source"
)

func depsWith(f fexec, resolve func(string) (string, bool)) (Deps, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return Deps{Stdout: &out, Stderr: &errb, Exec: f, HTTP: emptyHTTP{}, Resolve: resolve}, &out, &errb
}

func optHeadless() source.Options { return source.Options{Headless: true} }

// ---- C orchestration ----
func TestO3_DOnly(t *testing.T) {
	d, _, errb := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	s := errb.String()
	if strings.Contains(s, "grep started") {
		t.Fatal("O3: grep must NOT run with -d only")
	}
	for _, n := range []string{"subjs", "getJS", "katana", "hakrawler", "urlscan"} {
		if !strings.Contains(s, n+" started") {
			t.Fatalf("O3: %s should run", n)
		}
	}
}

func TestO4_All6(t *testing.T) {
	d, _, errb := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/a.js"}, URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	s := errb.String()
	for _, n := range []string{"grep", "subjs", "getJS", "katana", "hakrawler", "urlscan"} {
		if !strings.Contains(s, n+" started") {
			t.Fatalf("O4: all 6 -> %s missing", n)
		}
	}
}

func TestO12_ToolNotInstalled(t *testing.T) {
	res := func(s string) (string, bool) { return s, s != "subjs" } // subjs absent
	d, _, errb := depsWith(fexec{}, res)
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	if !strings.Contains(errb.String(), "subjs skipped") {
		t.Fatalf("O12: subjs should be skipped: %s", errb.String())
	}
}

func TestO13_AllToolsAbsentGrepWorks(t *testing.T) {
	res := func(s string) (string, bool) { return "", false }
	d, _, _ := depsWith(fexec{}, res)
	out := tmp(t)
	cfg := Config{GrepLines: []string{"https://t/a.js"}, OutPath: out, Silent: true, Timeout: time.Second}
	code := Run(context.Background(), cfg, d)
	data, _ := os.ReadFile(out)
	if code != 0 || !strings.Contains(string(data), "a.js") {
		t.Fatalf("O13: grep works with no tools, got code=%d %q", code, data)
	}
}

func TestO27_AllEmpty(t *testing.T) {
	out := tmp(t)
	d, _, errb := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: out}
	cfg.Timeout = time.Second
	Run(context.Background(), cfg, d)
	if !strings.Contains(errb.String(), "found   0 js") {
		t.Fatalf("O27: report should show 0 found")
	}
	data, _ := os.ReadFile(out)
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("O27: js.txt should be empty, got %q", data)
	}
}

func TestO28_O30_OutputPath(t *testing.T) {
	out := tmp(t)
	os.WriteFile(out, []byte("OLD\n"), 0o644) // pre-existing (O30 overwrite)
	d, _, _ := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/a.js"}, OutPath: out, Silent: true, Timeout: time.Second}
	Run(context.Background(), cfg, d)
	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "OLD") {
		t.Fatal("O30: must overwrite, not append")
	}
}

func TestO31_ParallelRaceClean(t *testing.T) { // run with -race catches issues
	f := fexec{urls: map[string][]string{"subjs": {"https://t/a.js"}, "getJS": {"https://t/b.js"}, "katana": {"https://t/c.js"}, "hakrawler": {"https://t/d.js"}}}
	d, _, _ := depsWith(f, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: tmp(t), Silent: true, Timeout: time.Second}
	if Run(context.Background(), cfg, d) != 0 {
		t.Fatal("O31")
	}
}

// ---- SL silent ----
func TestSL2_NoReport(t *testing.T) {
	d, _, errb := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/a.js"}, Silent: true, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	if strings.Contains(errb.String(), "fetch done") {
		t.Fatal("SL2: --silent must hide report")
	}
}

func TestSL5_SilentPlusFile(t *testing.T) {
	out := tmp(t)
	d, outbuf, _ := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/a.js"}, Silent: true, OutPath: out, Timeout: time.Second}
	Run(context.Background(), cfg, d)
	data, _ := os.ReadFile(out)
	if !strings.Contains(outbuf.String(), "a.js") || !strings.Contains(string(data), "a.js") {
		t.Fatal("SL5: both stdout and file written")
	}
}

func TestSL6_SilentEmpty(t *testing.T) {
	d, outbuf, _ := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/x.css"}, Silent: true, OutPath: tmp(t), Timeout: time.Second}
	if Run(context.Background(), cfg, d) != 0 || strings.TrimSpace(outbuf.String()) != "" {
		t.Fatalf("SL6: empty stdout, got %q", outbuf.String())
	}
}

// ---- PR progress ----
func TestPR_Lines(t *testing.T) {
	res := func(s string) (string, bool) { return s, s != "getJS" } // getJS skipped
	f := fexec{fail: map[string]bool{"katana": true}}
	d, _, errb := depsWith(f, res)
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	s := errb.String()
	if !strings.Contains(s, "[>]") { // PR1 started
		t.Fatal("PR1: started line")
	}
	if !strings.Contains(s, "[ok]") { // PR2 done
		t.Fatal("PR2: done line")
	}
	if !strings.Contains(s, "getJS skipped") { // PR3
		t.Fatal("PR3: skipped line")
	}
	if !strings.Contains(s, "katana failed") { // PR4
		t.Fatal("PR4: failed line")
	}
}

func TestPR9_SilentNoProgress(t *testing.T) {
	d, _, errb := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, Silent: true, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	if strings.Contains(errb.String(), "[>]") {
		t.Fatal("PR9: --silent -> no progress")
	}
}

// ---- RP report ----
func TestRP_Content(t *testing.T) {
	f := fexec{urls: map[string][]string{"subjs": {"https://target.com/a.js"}}}
	d, _, errb := depsWith(f, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://target.com"}, HostForms: []string{"target.com"}, InScope: true, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	s := errb.String()
	for _, want := range []string{"fetch done", "sources", "found", "unique", "in-scope drop", "saved"} {
		if !strings.Contains(s, want) {
			t.Fatalf("RP: report missing %q in %s", want, s)
		}
	}
}

func TestRP6_NoInScopeLine(t *testing.T) {
	d, _, errb := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/a.js"}, OutPath: tmp(t), Timeout: time.Second}
	Run(context.Background(), cfg, d)
	if strings.Contains(errb.String(), "in-scope drop") {
		t.Fatal("RP6: no --in-scope -> no drop line")
	}
}

// ---- EX exit codes ----
func TestEX1_OKZero(t *testing.T) {
	d, _, _ := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/a.js"}, Silent: true, OutPath: tmp(t), Timeout: time.Second}
	if Run(context.Background(), cfg, d) != 0 {
		t.Fatal("EX1")
	}
}

func TestEX5_PartialFailStillZero(t *testing.T) {
	f := fexec{urls: map[string][]string{"subjs": {"https://t/a.js"}}, fail: map[string]bool{"katana": true}}
	d, _, _ := depsWith(f, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, Silent: true, OutPath: tmp(t), Timeout: time.Second}
	if Run(context.Background(), cfg, d) != 0 {
		t.Fatal("EX5: some ok -> exit 0")
	}
}

func TestEX6_ZeroJSStillZero(t *testing.T) {
	d, _, _ := depsWith(fexec{}, func(s string) (string, bool) { return s, true })
	cfg := Config{GrepLines: []string{"https://t/x.css"}, Silent: true, OutPath: tmp(t), Timeout: time.Second}
	if Run(context.Background(), cfg, d) != 0 {
		t.Fatal("EX6: 0 js but ran -> exit 0")
	}
}

// ---- ME memory (only .js kept) ----
func TestME_OnlyJSKept(t *testing.T) {
	f := fexec{urls: map[string][]string{"katana": {"https://t/a.js", "https://t/x.png", "not a url", "https://t/b.JS"}}}
	out := tmp(t)
	d, _, _ := depsWith(f, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, OutPath: out, Silent: true, Timeout: time.Second}
	Run(context.Background(), cfg, d)
	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "png") || strings.Contains(string(data), "not a url") {
		t.Fatalf("ME: only js kept, got %q", data)
	}
}

// ---- RB robustness ----
func TestRB1_AtomicWrite(t *testing.T) {
	out := tmp(t)
	if err := atomicWrite(out, "a.js\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("RB1/RB3: temp file must be gone after rename")
	}
	data, _ := os.ReadFile(out)
	if string(data) != "a.js\n" {
		t.Fatalf("RB1: content %q", data)
	}
}

func TestRB_CreatesParentDir(t *testing.T) { // O29
	out := filepath.Join(t.TempDir(), "sub", "deep", "js.txt")
	if err := atomicWrite(out, "x\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal("O29: parent dir should be created")
	}
}

// ---- HL2 headless no browser (graceful) ----
func TestHL2_KatanaFailsGraceful(t *testing.T) {
	f := fexec{urls: map[string][]string{"subjs": {"https://t/a.js"}}, fail: map[string]bool{"katana": true}}
	d, _, errb := depsWith(f, func(s string) (string, bool) { return s, true })
	cfg := Config{URLForms: []string{"https://t"}, HostForms: []string{"t"}, Opts: optHeadless(), OutPath: tmp(t), Timeout: time.Second}
	code := Run(context.Background(), cfg, d)
	if code != 0 || !strings.Contains(errb.String(), "katana failed") {
		t.Fatalf("HL2: katana fails gracefully, run continues; code=%d", code)
	}
}

func tmp(t *testing.T) string { return filepath.Join(t.TempDir(), "js.txt") }
