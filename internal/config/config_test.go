package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteKey_Create(t *testing.T) { // C1/C4/C6
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "config")
	if err := writeKeyAt(p, "abc123"); err != nil {
		t.Fatal(err)
	}
	if fileKeyAt(p) != "abc123" {
		t.Fatalf("C1: key not stored")
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("C6: want 0600, got %o", fi.Mode().Perm())
	}
	di, _ := os.Stat(filepath.Dir(p))
	if di.Mode().Perm() != 0o700 {
		t.Fatalf("C5: dir want 0700, got %o", di.Mode().Perm())
	}
}

func TestWriteKey_EmptySkips(t *testing.T) { // C3/C11
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	if err := writeKeyAt(p, "   "); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) { // C22
		t.Fatal("C22: empty key must not create a file")
	}
}

func TestWriteKey_PreservesOtherLines(t *testing.T) { // C8
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	os.WriteFile(p, []byte("OTHER=keep\nURLSCAN_API_KEY=old\n"), 0o600)
	writeKeyAt(p, "new")
	data, _ := os.ReadFile(p)
	s := string(data)
	if !contains(s, "OTHER=keep") {
		t.Fatalf("C8: other line lost: %q", s)
	}
	if fileKeyAt(p) != "new" { // C9 updated in place
		t.Fatalf("C9: want new, got %q", fileKeyAt(p))
	}
	if count(s, "URLSCAN_API_KEY") != 1 {
		t.Fatalf("C9: key duplicated: %q", s)
	}
}

func TestWriteKey_Uncomments(t *testing.T) { // C10
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	os.WriteFile(p, []byte("#URLSCAN_API_KEY=\n"), 0o600)
	writeKeyAt(p, "xyz")
	if fileKeyAt(p) != "xyz" {
		t.Fatalf("C10: commented key not set")
	}
}

func TestWriteKey_Trims(t *testing.T) { // C12
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	writeKeyAt(p, "  spaced  ")
	if fileKeyAt(p) != "spaced" {
		t.Fatalf("C12: want trimmed, got %q", fileKeyAt(p))
	}
}

func TestResolveKey_Order(t *testing.T) { // C16-C21
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	writeKeyAt(p, "fromfile")
	t.Setenv("XDG_CONFIG_HOME", dir) // makes Path() point at dir/fetch/config
	// move file to where Path() expects
	os.MkdirAll(filepath.Join(dir, "fetch"), 0o700)
	writeKeyAt(filepath.Join(dir, "fetch", "config"), "fromfile")

	t.Setenv("URLSCAN_API_KEY", "")
	if got := ResolveKey("flagkey"); got != "flagkey" { // C16 flag wins
		t.Fatalf("C16: want flagkey, got %s", got)
	}
	t.Setenv("URLSCAN_API_KEY", "envkey")
	if got := ResolveKey(""); got != "envkey" { // C17 env over file
		t.Fatalf("C17: want envkey, got %s", got)
	}
	t.Setenv("URLSCAN_API_KEY", "")
	if got := ResolveKey(""); got != "fromfile" { // C18 file
		t.Fatalf("C18: want fromfile, got %s", got)
	}
}

func TestResolveKey_MissingConfig(t *testing.T) { // C19/C20
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("URLSCAN_API_KEY", "")
	if got := ResolveKey(""); got != "" {
		t.Fatalf("C20: want keyless, got %s", got)
	}
}

func TestMask(t *testing.T) { // C13
	if Mask("abcdef12") != "****ef12" {
		t.Fatalf("C13: got %s", Mask("abcdef12"))
	}
	if Mask("abc") != "***" {
		t.Fatalf("C13: short got %s", Mask("abc"))
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func count(s, sub string) int {
	n, i := 0, 0
	for {
		j := indexOfFrom(s, sub, i)
		if j < 0 {
			return n
		}
		n++
		i = j + len(sub)
	}
}
func indexOf(s, sub string) int { return indexOfFrom(s, sub, 0) }
func indexOfFrom(s, sub string, from int) int {
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
