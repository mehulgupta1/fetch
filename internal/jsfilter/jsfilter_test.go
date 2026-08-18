package jsfilter

import (
	"reflect"
	"testing"
)

// A. FILTER (isJS)
func TestIsJS(t *testing.T) {
	cases := []struct {
		id   string
		in   string
		want bool
	}{
		{"F1", "https://t.com/app.js", true},
		{"F2", "https://t.com/app.js?v=1", true},
		{"F3", "https://t.com/app.js#frag", true},
		{"F4", "https://t.com/a/b/app.min.js", true},
		{"F5", "https://t.com/data.json", false},
		{"F6", "https://t.com/app.js/", false},
		{"F7", "https://t.com/APP.JS", true},
		{"F8", "https://t.com/", false},
		{"F9", "https://t.com/jsstuff", false},
		{"F10", "garbage line with spaces", false},
		{"F11", "//cdn.x.com/app.js", true},
		{"F12", "/app.js", true},
		{"F13", "", false},
		{"F14", "https://t.com/app.js?file=x.css", true},
		{"F15", "https://t.com/app.mjs", true}, // .mjs is JavaScript
		{"F16", "  https://t.com/a.js  ", true},
		{"F17", "https://t.com/app.JS?x=1", true},
		{"F18", "https://t.com/.js", true},
		{"F19", "https://t.com:8443/app.js", true},
		{"F20", "HTTPS://t.com/a.js", true},
		{"F21", "https://t.com/app%2Ejs", true},
		{"F22", "https://t.com/app.js.map", true}, // sourcemap - keep (juicy)
		{"F23", "https://t.com/app.min.js.gz", false},
		{"F24", "https://t.com/page?redirect=/x.js", false},
		{"F25", "https://t.com/page#/x.js", false},
		{"F26", "app.js\t", true},
		{"F27", "app.js", true},
		{"F28", "http://", false},
		{"F29", "https://t.com/app.js%00.png", false},
		{"F30", "ftp://t.com/app.js", true},
		{"F31", "https://t.com/appjs", false},
		{"F32", "https://t.com/a.js?x=/y.css#z.json", true},
		{"F33", "https://t.com/mod.cjs", true},         // CommonJS
		{"F34", "https://t.com/app.mjs?v=1", true},      // ES module + query
		{"F35", "https://t.com/style.css.map", false},   // css sourcemap, not js
		{"F36", "https://t.com/vendor.js.map", true},    // js sourcemap
	}
	for _, c := range cases {
		_, got := IsJS(c.in)
		if got != c.want {
			t.Errorf("%s IsJS(%q)=%v want %v", c.id, c.in, got, c.want)
		}
	}
}

// B. DEDUP + SORT
func TestDedup(t *testing.T) {
	in := []string{"b.js", "a.js", "b.js", " a.js ", "", "c.js"}
	got := Dedup(in)
	want := []string{"a.js", "b.js", "c.js"} // D1,D4,D5,D10
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Dedup got %v want %v", got, want)
	}
}

func TestDedup_LooseKeepsVariants(t *testing.T) { // D2, D3, D7
	in := []string{"x.js?v=2", "x.js?v=1", "HTTP://T/a.js", "http://t/a.js", "y.js#a", "y.js#b"}
	got := Dedup(in)
	if len(got) != 6 {
		t.Fatalf("loose dedup should keep all 6 variants, got %d: %v", len(got), got)
	}
}

func TestDedup_Empty(t *testing.T) { // D9
	if got := Dedup(nil); len(got) != 0 {
		t.Fatalf("D9: want empty, got %v", got)
	}
}

// P. --in-scope
func TestInScope(t *testing.T) {
	tg := []string{"target.com"}
	cases := []struct {
		id   string
		url  string
		want bool
	}{
		{"P1", "https://target.com/app.js", true},
		{"P2", "https://api.target.com/a.js", true},
		{"P3", "https://cdn.jsdelivr.net/x.js", false},
		{"P4", "https://TARGET.com/a.js", true},
		{"P5", "https://eviltarget.com/a.js", false},
		{"P12", "https://target.com:8443/app.js", true},
		{"P13", "/app.js", false},
		{"P14", "//cdn.x.com/app.js", false},
	}
	for _, c := range cases {
		if got := InScope(c.url, tg); got != c.want {
			t.Errorf("%s InScope(%q)=%v want %v", c.id, c.url, got, c.want)
		}
	}
}

func TestInScope_MultiTarget(t *testing.T) { // P6
	tg := []string{"a.com", "b.com"}
	if !InScope("https://x.b.com/f.js", tg) {
		t.Fatal("P6: should match any target")
	}
	if InScope("https://c.com/f.js", tg) {
		t.Fatal("P6: c.com not in scope")
	}
}

func TestFilterInScope(t *testing.T) { // P9 count
	urls := []string{"https://target.com/a.js", "https://cdn.net/b.js"}
	kept, dropped := FilterInScope(urls, []string{"target.com"})
	if len(kept) != 1 || dropped != 1 {
		t.Fatalf("want 1 kept 1 dropped, got %d %d", len(kept), dropped)
	}
}
