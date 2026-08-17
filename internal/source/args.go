package source

import (
	"strconv"
	"strings"
)

// Options are the run knobs (filled with defaults by the cli layer).
type Options struct {
	Depth        int
	Subs         bool
	Rate         int
	Concurrency  int
	Headless     bool
	Proxy        string
	Headers      []string
	Insecure     bool
	UrlscanKey   string
	UrlscanLimit int
	Debug        bool
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
	return 10
}

// KatanaArgs builds katana's args (real flags, confirmed in VM).
// input via -list <file>. default scope rdn = includes subdomains; without
// --subs we restrict to fqdn (exact host).
func KatanaArgs(o Options, listFile string) []string {
	a := []string{"-silent", "-list", listFile, "-d", itoa(o.depth()), "-c", itoa(o.conc())}
	if !o.Subs {
		a = append(a, "-fs", "fqdn")
	}
	if o.Rate > 0 {
		a = append(a, "-rl", itoa(o.Rate))
	}
	if o.Headless {
		a = append(a, "-hl")
	}
	if o.Proxy != "" {
		a = append(a, "-proxy", o.Proxy)
	}
	for _, h := range o.Headers {
		a = append(a, "-H", h)
	}
	return a
}

// HakrawlerArgs builds hakrawler's args (input via stdin). headers are a
// single ";;"-joined string (hakrawler's format). -proxy is a real flag.
func HakrawlerArgs(o Options) []string {
	a := []string{"-d", itoa(o.depth()), "-t", itoa(o.conc()), "-u"}
	if o.Subs {
		a = append(a, "-subs")
	}
	if o.Insecure {
		a = append(a, "-insecure")
	}
	if o.Proxy != "" {
		a = append(a, "-proxy", o.Proxy)
	}
	if len(o.Headers) > 0 {
		a = append(a, "-h", strings.Join(o.Headers, ";;"))
	}
	return a
}

// SubjsArgs builds subjs's args (input via -i file). subjs has no
// header/proxy/insecure support.
func SubjsArgs(o Options, listFile string) []string {
	return []string{"-i", listFile, "-c", itoa(o.conc())}
}

// GetjsArgs builds getJS's args. -complete makes it emit absolute urls.
// no insecure flag; proxy goes via env.
func GetjsArgs(o Options, listFile string) []string {
	a := []string{"-input", listFile, "-complete", "-threads", itoa(o.conc())}
	for _, h := range o.Headers {
		a = append(a, "-header", h)
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
