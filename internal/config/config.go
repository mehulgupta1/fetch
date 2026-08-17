// Package config stores/reads the urlscan API key in ~/.config/fetch/config.
package config

import (
	"os"
	"path/filepath"
	"strings"
)

const keyName = "URLSCAN_API_KEY"

// Dir returns the config directory (honors XDG_CONFIG_HOME).
func Dir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "fetch")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "fetch")
}

// Path returns the config file path.
func Path() string { return filepath.Join(Dir(), "config") }

// FileKey reads URLSCAN_API_KEY from the config file ("" if absent/none).
func FileKey() string { return fileKeyAt(Path()) }

func fileKeyAt(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "#") {
			continue
		}
		if k, v, ok := strings.Cut(ln, "="); ok && strings.TrimSpace(k) == keyName {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// ResolveKey applies the lookup order: flag > env > config file > "".
func ResolveKey(flag string) string {
	if flag != "" {
		return flag
	}
	if e := os.Getenv(keyName); e != "" {
		return e
	}
	return FileKey()
}

// WriteKey stores the key atomically (file 0600, dir 0700), preserving any
// other lines and uncommenting a commented key line. Empty key = no write.
func WriteKey(key string) error { return writeKeyAt(Path(), key) }

func writeKeyAt(path, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil // skip
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var lines []string
	found := false
	if data, err := os.ReadFile(path); err == nil {
		for _, ln := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ln), "#"))
			if k, _, ok := strings.Cut(trimmed, "="); ok && strings.TrimSpace(k) == keyName {
				lines = append(lines, keyName+"="+key)
				found = true
				continue
			}
			lines = append(lines, ln)
		}
	}
	if !found {
		lines = append(lines, keyName+"="+key)
	}
	out := strings.Join(dropTrailingBlank(lines), "\n") + "\n"

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func dropTrailingBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// Mask hides all but the last 4 chars of a secret for display.
func Mask(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}
