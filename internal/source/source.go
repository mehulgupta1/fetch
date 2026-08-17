// Package source runs the 6 JS-finding sources and filters output to .js.
package source

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mehulgupta1/fetch/internal/jsfilter"
)

// Result is one source's contribution.
type Result struct {
	Name   string
	URLs   []string
	Status string // "ok" | "skipped" | "failed"
	Reason string
}

// Exec runs an external tool, streaming each stdout line to onLine so we
// filter to .js while reading (bounds memory). Injectable for tests.
type Exec interface {
	Run(ctx context.Context, argv []string, stdin string, extraEnv []string, onLine func(string)) (stderr string, err error)
}

// RealExec is the production Exec (large-line safe reader, stderr captured).
type RealExec struct{}

func (RealExec) Run(ctx context.Context, argv []string, stdin string, extraEnv []string, onLine func(string)) (string, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	var errbuf bytes.Buffer
	cmd.Stderr = &errbuf
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return errbuf.String(), err
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 1024*1024), 64*1024*1024) // >64KB lines safe
	for sc.Scan() {
		onLine(sc.Text())
	}
	return errbuf.String(), cmd.Wait()
}

// Grep is source S1: keep .js lines from a url list (no external tool).
func Grep(lines []string) Result {
	r := Result{Name: "grep", Status: "ok"}
	for _, ln := range lines {
		if u, ok := jsfilter.IsJS(ln); ok {
			r.URLs = append(r.URLs, u)
		}
	}
	return r
}

// RunTool runs one external tool and keeps only .js from its stdout.
// Partial output is kept (best-effort); "failed" only when nothing came
// back AND the tool errored.
func RunTool(ctx context.Context, ex Exec, name, path string, args []string, stdin string, env []string) Result {
	r := Result{Name: name, Status: "ok"}
	argv := append([]string{path}, args...)
	stderr, err := ex.Run(ctx, argv, stdin, env, func(ln string) {
		if u, ok := jsfilter.IsJS(ln); ok {
			r.URLs = append(r.URLs, u)
		}
	})
	if err != nil && len(r.URLs) == 0 {
		r.Status = "failed"
		r.Reason = Reason(ctx, stderr, err)
	}
	return r
}

// Reason derives a short failure reason.
func Reason(ctx context.Context, stderr string, err error) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "timeout"
	}
	if l := lastLine(stderr); l != "" {
		return l
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return fmt.Sprintf("exit code %d", ee.ExitCode())
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" {
			return t
		}
	}
	return ""
}

// Skipped builds a skipped result (tool not installed).
func Skipped(name string) Result {
	return Result{Name: name, Status: "skipped", Reason: "not installed"}
}
