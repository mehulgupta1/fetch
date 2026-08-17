package jsfilter

import (
	"fmt"
	"sort"
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

func TestP10_ExactSubdomainScope(t *testing.T) {
	// scope is api.target.com only -> target.com root is OUT
	if InScope("https://target.com/a.js", []string{"api.target.com"}) {
		t.Fatal("P10: root must be out of scope when only a subdomain is listed")
	}
	if !InScope("https://x.api.target.com/a.js", []string{"api.target.com"}) {
		t.Fatal("P10: deeper subdomain in scope")
	}
}
