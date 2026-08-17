// Package cli parses args, dispatches commands, and runs the collection.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mehulgupta1/fetch/internal/config"
	"github.com/mehulgupta1/fetch/internal/resolve"
	"github.com/mehulgupta1/fetch/internal/setup"
)

// Deps are the injectable I/O boundaries.
type Deps struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	StdinPiped bool // true when stdin is a pipe (not a tty)
}

// goPkgs maps each tool to its go-install module path.
// NOTE: paths are best-guess; corrected in the VM at build.
var goPkgs = map[string]string{
	"subjs":     "github.com/lc/subjs@latest",
	"getJS":     "github.com/003random/getJS/v2@latest",
	"katana":    "github.com/projectdiscovery/katana/cmd/katana@latest",
	"hakrawler": "github.com/hakluke/hakrawler@latest",
}

var toolOrder = []string{"subjs", "getJS", "katana", "hakrawler"}

// Run dispatches on args (command precedence, see JS_PLAN 2e).
func Run(args []string, d Deps) int {
	switch {
	case hasAny(args, "-h", "--help"):
		fmt.Fprint(d.Stdout, Help)
		return 0
	case len(args) == 0 && !d.StdinPiped:
		fmt.Fprint(d.Stdout, Help)
		return 0
	case hasAny(args, "--version", "-version"):
		return dispatchVersion(d)
	case hasAny(args, "-setup", "--setup"):
		return dispatchSetup(d)
	case hasAny(args, "-config", "--config"):
		return dispatchConfig(d, args)
	}
	return dispatchRun(args, d)
}

func hasAny(args []string, flags ...string) bool {
	for _, a := range args {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

func dispatchVersion(d Deps) int {
	fmt.Fprintf(d.Stdout, "fetch %s\n", Version)
	r := resolve.New()
	for _, t := range toolOrder {
		fmt.Fprintf(d.Stdout, "  %-10s %s\n", t, r.Version(t))
	}
	return 0
}

func dispatchSetup(d Deps) int {
	if !setup.HasGo(exec.LookPath) {
		fmt.Fprintln(d.Stderr, "error: go not found - install Go first, then re-run `fetch -setup`")
		return 1
	}
	r := resolve.New()
	deps := setup.Deps{
		IsInstalled: func(tool string) bool { _, ok := r.Resolve(tool); return ok },
		Install: func(tool string) error {
			pkg, ok := goPkgs[tool]
			if !ok {
				return fmt.Errorf("no install recipe for %s", tool)
			}
			return runVisible(d, setup.GoInstallCmd(pkg))
		},
	}
	fmt.Fprintln(d.Stdout, "fetch -setup: checking subjs + getJS + katana + hakrawler ...")
	fmt.Fprintln(d.Stdout)
	rep := setup.Run(toolOrder, deps)

	for _, t := range toolOrder {
		switch {
		case containsStr(rep.Installed, t):
			p, _ := r.Resolve(t)
			fmt.Fprintf(d.Stdout, "  [+] %-10s installed        %s\n", t, p)
		case containsStr(rep.Skipped, t):
			p, _ := r.Resolve(t)
			fmt.Fprintf(d.Stdout, "  [=] %-10s already present  %s\n", t, p)
		case containsStr(rep.Failed, t):
			fmt.Fprintf(d.Stdout, "  [x] %-10s FAILED - see errors above\n", t)
		}
	}

	ready := len(rep.Installed) + len(rep.Skipped)
	fmt.Fprintf(d.Stdout, "\n  %d/%d tools ready", ready, len(toolOrder))
	if len(rep.Failed) > 0 {
		fmt.Fprintf(d.Stdout, "  -  %d failed", len(rep.Failed))
	}
	fmt.Fprintln(d.Stdout)

	gobin := resolve.GoBinDir()
	if !strings.Contains(os.Getenv("PATH"), gobin) {
		fmt.Fprintf(d.Stdout, "\nNOTE: add gobin to PATH so the tools are found at run time:\n  export PATH=\"$PATH:%s\"\n", gobin)
	}
	if len(rep.Failed) > 0 {
		return 1
	}
	return 0
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func dispatchConfig(d Deps, args []string) int {
	key := flagValue(args, "--urlscan-key")
	if key == "" {
		fmt.Fprint(d.Stdout, "urlscan api key (Enter to skip): ")
		sc := bufio.NewScanner(d.Stdin)
		if sc.Scan() {
			key = strings.TrimSpace(sc.Text())
		}
	}
	if key == "" {
		fmt.Fprintln(d.Stdout, "no key entered; nothing changed")
		return 0
	}
	if err := config.WriteKey(key); err != nil {
		fmt.Fprintf(d.Stderr, "error writing config: %v\n", err)
		return 1
	}
	fmt.Fprintf(d.Stdout, "saved urlscan key %s to %s\n", config.Mask(key), config.Path())
	return 0
}

// flagValue returns the value after `name` in args ("" if absent).
func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(a, name+"="); ok {
			return v
		}
	}
	return ""
}

// runVisible streams a command's output (and any prompt) to the user.
func runVisible(d Deps, argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty command")
	}
	fmt.Fprintf(d.Stdout, "  $ %s\n", strings.Join(argv, " "))
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = d.Stdout
	cmd.Stderr = d.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

