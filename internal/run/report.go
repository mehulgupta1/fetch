package run

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mehulgupta1/fetch/internal/source"
)

// report prints the compact final summary to stderr.
func report(w io.Writer, results []source.Result, rawTotal, unique, dropped int, cfg Config, elapsed time.Duration) {
	fmt.Fprintf(w, "\n  done in %s\n", elapsed.Round(time.Second))

	var parts []string
	for _, r := range results {
		switch r.Status {
		case "ok":
			parts = append(parts, fmt.Sprintf("%s %d", r.Name, len(r.URLs)))
		case "skipped":
			parts = append(parts, r.Name+" skip")
		case "failed":
			parts = append(parts, r.Name+" fail")
		}
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(parts, " | "))

	scope := ""
	if cfg.InScope {
		scope = fmt.Sprintf("  (%d out-of-scope dropped)", dropped)
	}
	out := cfg.OutPath
	if out == "" {
		out = "js.txt"
	}
	fmt.Fprintf(w, "  %d found -> %d unique%s -> %s\n", rawTotal, unique, scope, out)
}
