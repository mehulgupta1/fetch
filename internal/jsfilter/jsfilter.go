// Package jsfilter decides if a url is a .js file, dedups, sorts, and
// does --in-scope host matching. This is the tool's core logic.
package jsfilter

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// jsExts are the path suffixes we treat as JavaScript (or its sourcemaps).
// .js/.mjs/.cjs are all JavaScript; the .map variants hold the original
// un-minified source - gold for bug hunting, so we keep them too.
var jsExts = []string{".js", ".mjs", ".cjs", ".js.map", ".mjs.map", ".cjs.map"}

// IsJS reports whether one line is a JS-file url: it must parse as a url AND
// its PATH (query/fragment ignored) end in a JS extension (case-insensitive,
// after percent-decoding). Returns the trimmed line.
func IsJS(line string) (string, bool) {
	s := strings.TrimSpace(line)
	if s == "" {
		return "", false
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", false
	}
	p := strings.ToLower(u.Path)
	for _, ext := range jsExts {
		if strings.HasSuffix(p, ext) {
			return s, true
		}
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

// Content-hash patterns in JS filenames (webpack/vite/rails fingerprints):
//   application-<64hex>.js  main.17afcbe5.js  19.b9434aaa.chunk.js
var reHexHash = regexp.MustCompile(`[-.][0-9a-fA-F]{8,}(\.chunk)?\.js$`)

//   mutations-CZp5U1El.js  index-D25SdecD.js  (vite: dash + 8 base64url chars)
var reB64Hash = regexp.MustCompile(`-([A-Za-z0-9_-]{8})(\.chunk)?\.js$`)

// reWord matches a real filename word: all lowercase, optionally with ONE
// leading capital (settings, Settings, Controls). Content hashes don't fit
// this - they have digits or scattered internal capitals (CZp5U1El, CCSXfehS).
var reWord = regexp.MustCompile(`^[A-Z]?[a-z]+$`)

// looksHashy: an 8-char token is a content hash unless it looks like a real
// word. Real words (Settings, Controls, dropdown) are NEVER stripped, so two
// different-named files can never collapse into one.
func looksHashy(s string) bool {
	return !reWord.MatchString(s)
}

// stripHash removes a content-hash token from a filename path, so all deploy
// versions of one logical file map to the same key.
func stripHash(path string) string {
	if reHexHash.MatchString(path) {
		return reHexHash.ReplaceAllString(path, "$1.js")
	}
	// only treat an 8-char token as a hash if it has an uppercase letter
	// (real filenames are kebab/snake lowercase, so this spares them).
	if m := reB64Hash.FindStringSubmatch(path); m != nil && looksHashy(m[1]) {
		return reB64Hash.ReplaceAllString(path, "$2.js")
	}
	return path
}

// LogicalKey is the dedup key that collapses hash-versions: lowercase host +
// hash-stripped path, ignoring scheme, port, query and fragment.
func LogicalKey(rawurl string) string {
	u, err := url.Parse(strings.TrimSpace(rawurl))
	if err != nil {
		return strings.TrimSpace(rawurl)
	}
	return strings.ToLower(u.Hostname()) + stripHash(u.Path)
}

// DedupLogical keeps ONE real url per logical file (the lexicographically
// smallest, for determinism), collapsing content-hash versions. Sorted.
func DedupLogical(in []string) []string {
	best := make(map[string]string, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		k := LogicalKey(s)
		if cur, ok := best[k]; !ok || s < cur {
			best[k] = s
		}
	}
	out := make([]string, 0, len(best))
	for _, v := range best {
		out = append(out, v)
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
