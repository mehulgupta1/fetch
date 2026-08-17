// Package jsfilter decides if a url is a .js file, dedups, sorts, and
// does --in-scope host matching. This is the tool's core logic.
package jsfilter

import (
	"net/url"
	"sort"
	"strings"
)

// IsJS reports whether one line is a JS-file url (loose rules):
// it must parse as a url AND its PATH (query/fragment ignored) end in ".js"
// (case-insensitive, after percent-decoding). Returns the trimmed line.
func IsJS(line string) (string, bool) {
	s := strings.TrimSpace(line)
	if s == "" {
		return "", false
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", false
	}
	if strings.HasSuffix(strings.ToLower(u.Path), ".js") {
		return s, true
	}
	return "", false
}

// Dedup trims, drops blanks, removes exact duplicates, and sorts (ASCII).
func Dedup(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// Host returns the lowercased hostname (port stripped) of a url, or "".
func Host(rawurl string) string {
	u, err := url.Parse(strings.TrimSpace(rawurl))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// InScope reports whether a url's host equals, or is a subdomain of, ANY
// target host. A hostless url is never in scope. Targets are lowercased.
func InScope(rawurl string, targets []string) bool {
	h := Host(rawurl)
	if h == "" {
		return false
	}
	for _, t := range targets {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if h == t || strings.HasSuffix(h, "."+t) {
			return true
		}
	}
	return false
}

// FilterInScope keeps only in-scope urls; returns kept + dropped count.
func FilterInScope(urls, targets []string) (kept []string, dropped int) {
	for _, u := range urls {
		if InScope(u, targets) {
			kept = append(kept, u)
		} else {
			dropped++
		}
	}
	return kept, dropped
}
