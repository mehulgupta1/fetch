package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestC7_AtomicNoTemp(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	if err := writeKeyAt(p, "abc"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("C7: temp file must be renamed away")
	}
}

func TestC14_MalformedPreserved(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	os.WriteFile(p, []byte("garbage line\n=oops\nURLSCAN_API_KEY=old\n"), 0o600)
	writeKeyAt(p, "new")
	data, _ := os.ReadFile(p)
	s := string(data)
	if !contains(s, "garbage line") || fileKeyAt(p) != "new" {
		t.Fatalf("C14: malformed lines preserved + key set: %q", s)
	}
}

func TestC21_NoKeyLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	os.WriteFile(p, []byte("OTHER=1\n"), 0o600)
	if fileKeyAt(p) != "" {
		t.Fatal("C21: no key line -> keyless")
	}
}
