package run

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// writeOutput writes the url list. Under --silent it prints to stdout;
// otherwise it writes the -o file atomically (temp+rename). If -o is given
// under --silent, the file is ALSO written.
func writeOutput(stdout io.Writer, cfg Config, urls []string) error {
	body := strings.Join(urls, "\n")
	if len(urls) > 0 {
		body += "\n"
	}
	if cfg.Silent {
		fmt.Fprint(stdout, body)
		if cfg.OutPath == "" {
			return nil
		}
	}
	return atomicWrite(cfg.OutPath, body)
}

func atomicWrite(path, body string) error {
	if path == "" {
		path = "js.txt"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
