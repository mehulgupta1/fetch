package source

import (
	"strconv"
	"strings"
)

// Options are the run knobs (filled with defaults by the cli layer).
type Options struct {
	Depth        int
	Exact        bool // restrict to the exact host (default: include subdomains)
	Rate         int
	Concurrency  int
	Headless     bool
	Proxy        string
	Headers      []string
	UserAgent    string
	Insecure     bool
	UrlscanKey   string
	UrlscanLimit int
	Debug        bool
}

// effUA returns the effective User-Agent: a user-supplied "User-Agent:" header
// wins over the default UserAgent.
func (o Options) effUA() string {
	for _, h := range o.Headers {
		if k, v, ok := strings.Cut(h, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "user-agent") {
			return strings.TrimSpace(v)
		}
	}
	return o.UserAgent
}

// addUAHeader reports whether to inject the default UA as a header (only when
// the user did NOT already provide a User-Agent header).
func (o Options) addUAHeader() bool {
	if o.UserAgent == "" {
		return false
	}
	for _, h := range o.Headers {
		if k, _, ok := strings.Cut(h, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "user-agent") {
			return false
		}
	}
	return true
}

func itoa(n int) string { return strconv.Itoa(n) }

func (o Options) depth() int {
	if o.Depth > 0 {
		return o.Depth
	}
	return 2
}

func (o Options) conc() int {
	if o.Concurrency > 0 {
		return o.Concurrency
	}
	return 5 // polite default: 4 tools x 5 = 20 concurrent, not 40
}

// KatanaArgs builds katana's args (real flags, confirmed in VM).
// input via -list <file>. default scope rdn = includes subdomains; without
// --subs we restrict to fqdn (exact host).
func KatanaArgs(o Options, listFile string) []string {
	a := []string{"-silent", "-list", listFile, "-d", itoa(o.depth()), "-c", itoa(o.conc())}
	if o.Exact {
		a = append(a, "-fs", "fqdn") // strict: exact host only
	} // default = katana's rdn scope (includes subdomains)
	if o.Rate > 0 {
		a = append(a, "-rl", itoa(o.Rate))
	}
	if o.Headless {
		// -nos (no-sandbox) is required for headless Chrome to start as root.
		a = append(a, "-hl", "-nos")
	}
	if o.Proxy != "" {
		a = append(a, "-proxy", o.Proxy)
	}
	for _, h := range o.Headers {
		a = append(a, "-H", h)
	}
	if o.addUAHeader() {
		a = append(a, "-H", "User-Agent: "+o.UserAgent)
	}
	return a
}

// HakrawlerArgs builds hakrawler's args (input via stdin). headers are a
// single ";;"-joined string (hakrawler's format). -proxy is a real flag.
func HakrawlerArgs(o Options) []string {
	a := []string{"-d", itoa(o.depth()), "-t", itoa(o.conc()), "-u"}
	if !o.Exact {
		a = append(a, "-subs") // default: include subdomains
	}
	if o.Insecure {
		a = append(a, "-insecure")
	}
	if o.Proxy != "" {
		a = append(a, "-proxy", o.Proxy)
	}
	hdrs := append([]string{}, o.Headers...)
	if o.addUAHeader() {
		hdrs = append(hdrs, "User-Agent: "+o.UserAgent)
	}
	if len(hdrs) > 0 {
		a = append(a, "-h", strings.Join(hdrs, ";;"))
	}
	return a
}

// SubjsArgs builds subjs's args (input via -i file). subjs has no
// header/proxy/insecure support.
func SubjsArgs(o Options, listFile string) []string {
	a := []string{"-i", listFile, "-c", itoa(o.conc())}
	if ua := o.effUA(); ua != "" {
		a = append(a, "-ua", ua)
	}
	return a
}

// GetjsArgs builds getJS's args. -complete makes it emit absolute urls.
// no insecure flag; proxy goes via env.
func GetjsArgs(o Options, listFile string) []string {
	a := []string{"-input", listFile, "-complete", "-threads", itoa(o.conc())}
	for _, h := range o.Headers {
		a = append(a, "-header", h)
	}
	if o.addUAHeader() {
		a = append(a, "-header", "User-Agent: "+o.UserAgent)
	}
	return a
}

// ProxyEnv returns HTTP(S)_PROXY env for tools without a proxy flag.
func ProxyEnv(o Options) []string {
	if o.Proxy == "" {
		return nil
	}
	return []string{"HTTP_PROXY=" + o.Proxy, "HTTPS_PROXY=" + o.Proxy}
}
