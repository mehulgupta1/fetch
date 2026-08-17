package run

import (
	"fmt"
	"io"
	"time"

	"github.com/mehulgupta1/fetch/internal/source"
)

// report prints the final summary to stderr (see JS_PLAN 3c).
func report(w io.Writer, results []source.Result, rawTotal, unique, dropped int, cfg Config, elapsed time.Duration) {
	fmt.Fprintf(w, "-------- fetch done (%s) --------\n", elapsed.Round(time.Second))
	fmt.Fprintln(w, " sources")
	for _, r := range results {
		switch r.Status {
		case "ok":
			fmt.Fprintf(w, "   %-11s %d\n", r.Name, len(r.URLs))
		case "skipped":
			fmt.Fprintf(w, "   %-11s skipped (%s)\n", r.Name, r.Reason)
		case "failed":
			fmt.Fprintf(w, "   %-11s failed (%s)\n", r.Name, r.Reason)
		}
	}
	fmt.Fprintf(w, " found   %d js  ->  %d unique\n", rawTotal, unique)
	if cfg.InScope {
		fmt.Fprintf(w, " in-scope drop   %d\n", dropped)
	}
	out := cfg.OutPath
	if out == "" {
		out = "js.txt"
	}
	if !cfg.Silent {
		fmt.Fprintf(w, " saved   %s\n", out)
	}
	fmt.Fprintln(w, "----------------------------------")
}
