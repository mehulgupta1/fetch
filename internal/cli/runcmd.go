package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mehulgupta1/fetch/internal/config"
	"github.com/mehulgupta1/fetch/internal/resolve"
	"github.com/mehulgupta1/fetch/internal/run"
	"github.com/mehulgupta1/fetch/internal/source"
)

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(s string) error { *m = append(*m, s); return nil }

var schemeRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*://`)

// dispatchRun parses flags, resolves inputs, validates, and runs.
func dispatchRun(args []string, d Deps) int {
	fs := newFlagSet(d.Stderr)
	f := bindFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 1 // unknown flag / parse error (CL5)
	}

	// inputs
	grepLines, err := readList(*f.lPath, d.Stdin, d.StdinPiped)
	if err != nil {
		fmt.Fprintf(d.Stderr, "error: -l: %v\n", err)
		return 1
	}
	urlForms, hostForms, err := resolveD(*f.dArg)
	if err != nil {
		fmt.Fprintf(d.Stderr, "error: %v\n", err)
		return 1
	}

	// validation
	if len(grepLines) == 0 && len(urlForms) == 0 {
		fmt.Fprintln(d.Stderr, "error: no input - give -l <file>, -d <domain/file>, or pipe urls on stdin")
		return 1
	}
	if *f.inScope && len(hostForms) == 0 {
		fmt.Fprintln(d.Stderr, "error: --in-scope requires -d (it defines the scope)")
		return 1
	}
	if *f.depth < 0 || *f.conc <= 0 || *f.rate < 0 || *f.uLimit < 0 {
		fmt.Fprintln(d.Stderr, "error: --depth/--rate/--urlscan-limit must be >=0 and -c >0")
		return 1
	}
	for _, h := range f.headers {
		if !strings.Contains(h, ":") {
			fmt.Fprintf(d.Stderr, "error: -H %q must be \"Key: Value\"\n", h)
			return 1
		}
	}
	if *f.proxy != "" {
		if u, err := url.Parse(*f.proxy); err != nil || u.Host == "" {
			fmt.Fprintf(d.Stderr, "error: --proxy %q is not a valid url\n", *f.proxy)
			return 1
		}
	}

	uLimit := *f.uLimit
	if uLimit == 0 { // default depends on key presence
		if config.ResolveKey(*f.uKey) != "" {
			uLimit = 100
		} else {
			uLimit = 20
		}
	}

	// follow -d redirects and add the landing host(s) to the target set
	// (so `-d bmw.com`, which bounces to www.bmw.in, crawls bmw.in too).
	if len(urlForms) > 0 && !*f.noRedirect {
		urlForms, hostForms = expandRedirects(urlForms)
	}

	opts := source.Options{
		Depth: *f.depth, Exact: *f.exact, Rate: *f.rate, Concurrency: *f.conc,
		Headless: *f.headless, Proxy: *f.proxy, Headers: []string(f.headers),
		UserAgent: BrowserUA,
		Insecure:  *f.insecure || *f.kAlias, UrlscanKey: config.ResolveKey(*f.uKey),
		UrlscanLimit: uLimit, Debug: *f.debug,
	}

	r := resolve.New()
	cfg := run.Config{
		GrepLines: grepLines, URLForms: urlForms, HostForms: hostForms,
		Opts: opts, InScope: *f.inScope, Timeout: *f.timeout,
		Silent: *f.silent, Debug: *f.debug, OutPath: *f.oPath,
	}
	deps := run.Deps{
		Stdout: d.Stdout, Stderr: d.Stderr,
		Exec: source.RealExec{}, HTTP: source.RealHTTP{},
		Resolve: func(tool string) (string, bool) { return r.Resolve(tool) },
	}
	if !*f.silent {
		fmt.Fprintln(d.Stderr, Banner)
	}
	return run.Run(context.Background(), cfg, deps)
}

// resolveD turns -d into normalized URL forms + host forms.
func resolveD(arg string) (urls, hosts []string, err error) {
	if arg == "" {
		return nil, nil, nil
	}
	var lines []string
	if fi, e := os.Stat(arg); e == nil && !fi.IsDir() {
		fh, e := os.Open(arg)
		if e != nil {
			return nil, nil, e
		}
		defer fh.Close()
		lines = scanLines(fh)
	} else if looksLikeFile(arg) {
		return nil, nil, fmt.Errorf("-d: file not found: %s", arg)
	} else {
		lines = []string{arg}
	}
	seen := map[string]bool{}
	seenHost := map[string]bool{}
	for _, ln := range lines {
		u, h, ok := normalizeDomain(ln)
		if !ok {
			continue
		}
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
		if !seenHost[h] {
			seenHost[h] = true
			hosts = append(hosts, h)
		}
	}
	return urls, hosts, nil
}

func looksLikeFile(s string) bool {
	if strings.HasSuffix(s, ".txt") || strings.HasSuffix(s, ".list") {
		return true
	}
	return strings.Contains(s, "/") && !strings.Contains(s, "://")
}

// normalizeDomain applies the 2c scheme rule; returns (urlForm, host).
func normalizeDomain(entry string) (urlForm, host string, ok bool) {
	e := strings.TrimSpace(entry)
	if e == "" || strings.HasPrefix(e, "#") {
		return "", "", false
	}
	switch {
	case strings.HasPrefix(e, "//"):
		e = "https:" + e
	case !schemeRE.MatchString(e):
		e = "https://" + e
	}
	u, err := url.Parse(e)
	if err != nil || u.Host == "" {
		return "", "", false
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	return u.String(), strings.ToLower(u.Hostname()), true
}

func readList(lPath string, stdin io.Reader, piped bool) ([]string, error) {
	if lPath == "-" {
		return scanLines(stdin), nil
	}
	if lPath != "" {
		fh, err := os.Open(lPath)
		if err != nil {
			return nil, err
		}
		defer fh.Close()
		return scanLines(fh), nil
	}
	if piped {
		return scanLines(stdin), nil
	}
	return nil, nil
}

// scanLines reads non-blank lines with a large buffer (>64KB safe).
func scanLines(r io.Reader) []string {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	var out []string
	for sc.Scan() {
		if s := strings.TrimSpace(sc.Text()); s != "" && !strings.HasPrefix(s, "#") {
			out = append(out, s)
		}
	}
	return out
}

// runFlags holds parsed flag pointers.
type runFlags struct {
	lPath, dArg, oPath, proxy, uKey  *string
	silent, debug, subs, headless    *bool
	exact, insecure, kAlias, inScope *bool
	noRedirect                       *bool
	depth, conc, rate, uLimit        *int
	timeout                          *time.Duration
	headers                          multiFlag
}

func bindFlags(fs *flagSet) *runFlags {
	f := &runFlags{}
	f.lPath = fs.String("l", "")
	f.dArg = fs.String("d", "")
	f.oPath = fs.String("o", "")
	f.proxy = fs.String("proxy", "")
	f.uKey = fs.String("urlscan-key", "")
	f.silent = fs.Bool("silent")
	f.debug = fs.Bool("debug")
	f.subs = fs.Bool("subs")   // accepted for compatibility; subdomains are on by default
	f.exact = fs.Bool("exact") // restrict to the exact host
	f.headless = fs.Bool("headless")
	f.insecure = fs.Bool("insecure")
	f.kAlias = fs.Bool("k")
	f.inScope = fs.Bool("in-scope")
	f.noRedirect = fs.Bool("no-redirect")
	f.depth = fs.Int("depth", 2)
	f.conc = fs.Int("c", 5)
	f.rate = fs.Int("rate", 100)
	f.uLimit = fs.Int("urlscan-limit", 0)
	f.timeout = fs.Duration("timeout", 10*time.Minute)
	fs.set.Var(&f.headers, "H", "")
	return f
}
