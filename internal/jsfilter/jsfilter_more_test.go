package jsfilter

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestD6_SameUrlManySources(t *testing.T) {
	got := Dedup([]string{"https://t/a.js", "https://t/a.js", "https://t/a.js"})
	if len(got) != 1 {
		t.Fatalf("D6: want 1, got %v", got)
	}
}

func TestD8_LargeN(t *testing.T) {
	var in []string
	for i := 0; i < 5000; i++ {
		in = append(in, fmt.Sprintf("https://t/%d.js", i%1000)) // 1000 unique, 5x dupes
	}
	got := Dedup(in)
	if len(got) != 1000 {
		t.Fatalf("D8: want 1000 unique, got %d", len(got))
	}
	if !sort.StringsAreSorted(got) {
		t.Fatal("D8: output must be sorted")
	}
}

func TestP7_MultiTargetFilter(t *testing.T) {
	urls := []string{"https://a.com/x.js", "https://b.com/y.js", "https://c.com/z.js"}
	kept, dropped := FilterInScope(urls, []string{"a.com", "b.com"})
	if len(kept) != 2 || dropped != 1 {
		t.Fatalf("P7: want 2 kept 1 dropped, got %d %d", len(kept), dropped)
	}
}

func TestLogicalKey_Collapses(t *testing.T) {
	pairs := [][2]string{
		{ // rails 64-hex fingerprint
			"https://h.com/assets/application-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.js",
			"https://h.com/assets/application-f2599f37d46071b2056db749cbb69d488060df26503180c6502dc7320c0d7b6a.js",
		},
		{"https://h.com/a/main.17afcbe5.js", "https://h.com/a/main.abcd1234.js"},        // webpack .hex.
		{"https://h.com/a/mutations-CZp5U1El.js", "https://h.com/a/mutations-D25SdecD.js"}, // vite base64
		{"https://h.com/js/19.b9434aaa.chunk.js", "https://h.com/js/19.abcdef01.chunk.js"}, // chunk
	}
	for _, p := range pairs {
		if LogicalKey(p[0]) != LogicalKey(p[1]) {
			t.Errorf("should collapse:\n  %s -> %s\n  %s -> %s", p[0], LogicalKey(p[0]), p[1], LogicalKey(p[1]))
		}
	}
}

func TestLogicalKey_NeverMergesDifferentFiles(t *testing.T) {
	// two DIFFERENT 8-char CamelCase names (no digit) must stay distinct
	a := LogicalKey("https://h.com/mod-Settings.js")
	b := LogicalKey("https://h.com/mod-Controls.js")
	if a == b {
		t.Fatalf("SAFETY BREACH: different files merged (%s == %s)", a, b)
	}
	if a != "h.com/mod-Settings.js" {
		t.Fatalf("real name wrongly stripped: %s", a)
	}
}

func TestLogicalKey_KeepsDistinct(t *testing.T) {
	if LogicalKey("https://h.com/app.js") == LogicalKey("https://h.com/vendor.js") {
		t.Fatal("distinct files must not collapse")
	}
	// plain lowercase names must NOT be treated as hashes
	for _, u := range []string{"https://h.com/settings.js", "https://h.com/some-config.js"} {
		if got := LogicalKey(u); got != "h.com"+u[len("https://h.com"):] {
			t.Errorf("plain name altered: %s -> %s", u, got)
		}
	}
}

func TestDedupLogical(t *testing.T) {
	in := []string{
		"https://h.com/app-aaaa1111.js",
		"https://h.com/app-bbbb2222.js",
		"https://h.com/app-cccc3333.js",
		"https://h.com/vendor.js",
	}
	out := DedupLogical(in)
	if len(out) != 2 {
		t.Fatalf("want 2 logical files, got %d: %v", len(out), out)
	}
	// representative is a real (working) url, not the hash-stripped key
	if !strings.HasSuffix(out[0], ".js") || !strings.Contains(out[0], "app-") {
		t.Fatalf("representative should be a real hashed url: %v", out)
	}
}

func TestP10_ExactSubdomainScope(t *testing.T) {
	// scope is api.target.com only -> target.com root is OUT
	if InScope("https://target.com/a.js", []string{"api.target.com"}) {
		t.Fatal("P10: root must be out of scope when only a subdomain is listed")
	}
	if !InScope("https://x.api.target.com/a.js", []string{"api.target.com"}) {
		t.Fatal("P10: deeper subdomain in scope")
	}
}
