// Package run orchestrates the sources: parallel fan-out, merge, filter,
// dedup, write, report.
package run

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mehulgupta1/fetch/internal/jsfilter"
	"github.com/mehulgupta1/fetch/internal/source"
)

// Deps are injectable boundaries.
type Deps struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Exec    source.Exec
	HTTP    source.HTTP
	Resolve func(tool string) (path string, ok bool)
}

// Config is one run's inputs.
type Config struct {
	GrepLines []string // -l lines (for grep S1)
	URLForms  []string // -d normalized urls (for tools)
	HostForms []string // -d hosts (urlscan + in-scope)
	Opts      source.Options
	InScope   bool
	Timeout   time.Duration
	Silent    bool
	Debug     bool
	OutPath   string
}

// Run executes the collection and returns a process exit code.
func Run(ctx context.Context, cfg Config, d Deps) int {
	showProg := !cfg.Silent || cfg.Debug
	pw := d.Stderr
	if !showProg {
		pw = io.Discard
	}
	start := time.Now()

	// temp file of URL forms for tools that read a file (katana/subjs/getJS).
	var listFile string
	if len(cfg.URLForms) > 0 {
		if f, err := os.CreateTemp("", "fetch-urls-*.txt"); err == nil {
			f.WriteString(strings.Join(cfg.URLForms, "\n") + "\n")
			f.Close()
			listFile = f.Name()
			defer os.Remove(listFile)
		}
	}
	stdinURLs := strings.Join(cfg.URLForms, "\n") + "\n"

	type job struct {
		name string
		run  func() source.Result
	}
	var jobs []job

	if len(cfg.GrepLines) > 0 {
		jobs = append(jobs, job{"grep", func() source.Result { return source.Grep(cfg.GrepLines) }})
	}
	if len(cfg.URLForms) > 0 {
		add := func(name string, argv []string, stdin string, env []string) {
			path, ok := d.Resolve(name)
			jobs = append(jobs, job{name, func() source.Result {
				if !ok {
					return source.Skipped(name)
				}
				tctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
				defer cancel()
				return source.RunTool(tctx, d.Exec, name, path, argv, stdin, env)
			}})
		}
		add("subjs", source.SubjsArgs(cfg.Opts, listFile), "", source.ProxyEnv(cfg.Opts))
		add("getJS", source.GetjsArgs(cfg.Opts, listFile), "", source.ProxyEnv(cfg.Opts))
		add("katana", source.KatanaArgs(cfg.Opts, listFile), "", nil)
		add("hakrawler", source.HakrawlerArgs(cfg.Opts), stdinURLs, nil)

		// urlscan (S6): loop hosts
		jobs = append(jobs, job{"urlscan", func() source.Result {
			tctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
			defer cancel()
			agg := source.Result{Name: "urlscan", Status: "ok"}
			failed := 0
			for _, host := range cfg.HostForms {
				r := source.Urlscan(tctx, d.HTTP, host, cfg.Opts.UrlscanKey, cfg.Opts.UrlscanLimit)
				agg.URLs = append(agg.URLs, r.URLs...)
				if r.Status == "failed" {
					failed++
					agg.Reason = r.Reason
				}
			}
			if failed == len(cfg.HostForms) && len(agg.URLs) == 0 && failed > 0 {
				agg.Status = "failed"
			}
			return agg
		}})
	}

	// run all jobs in parallel with live progress.
	// mu guards BOTH the running map AND all writes to pw (parallel goroutines
	// share one writer, so every progress line must be serialized).
	results := make([]source.Result, len(jobs))
	var wg sync.WaitGroup
	var mu sync.Mutex
	running := map[string]bool{}
	logf := func(format string, a ...any) {
		mu.Lock()
		fmt.Fprintf(pw, format, a...)
		mu.Unlock()
	}
	setRunning := func(name string, on bool) {
		mu.Lock()
		if on {
			running[name] = true
		} else {
			delete(running, name)
		}
		mu.Unlock()
	}
	for i, j := range jobs {
		logf("[>]  %s started\n", j.name)
		setRunning(j.name, true)
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			res := j.run()
			setRunning(j.name, false)
			results[i] = res
			switch res.Status {
			case "ok":
				logf("[ok] %s done: %d js\n", res.Name, len(res.URLs))
			case "skipped":
				logf("[-]  %s skipped (%s)\n", res.Name, res.Reason)
			case "failed":
				logf("[x]  %s failed: %s\n", res.Name, res.Reason)
			}
		}(i, j)
	}

	// heartbeat every 15s until done.
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				mu.Lock()
				var names []string
				for n := range running {
					names = append(names, n)
				}
				mu.Unlock()
				if len(names) > 0 {
					logf("[..] %s still running: %s\n",
						time.Since(start).Round(time.Second), strings.Join(names, ", "))
				}
			}
		}
	}()

	wg.Wait()
	close(done)

	// merge
	var all []string
	rawTotal := 0
	for _, r := range results {
		all = append(all, r.URLs...)
		rawTotal += len(r.URLs)
	}

	// in-scope
	dropped := 0
	if cfg.InScope && len(cfg.HostForms) > 0 {
		all, dropped = jsfilter.FilterInScope(all, cfg.HostForms)
	}

	// dedup + sort
	final := jsfilter.Dedup(all)

	// write
	if err := writeOutput(d.Stdout, cfg, final); err != nil {
		fmt.Fprintf(d.Stderr, "error writing output: %v\n", err)
		return 1
	}

	// report
	if !cfg.Silent {
		report(d.Stderr, results, rawTotal, len(final), dropped, cfg, time.Since(start))
	}

	// exit code
	anyOK, anyFailed := false, false
	for _, r := range results {
		if r.Status == "ok" {
			anyOK = true
		}
		if r.Status == "failed" {
			anyFailed = true
		}
	}
	if !anyOK && anyFailed {
		return 1
	}
	return 0
}
