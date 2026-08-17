package cli

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BrowserUA is the default User-Agent fetch presents to targets, so
// bot-protected sites are less likely to block the crawlers.
const BrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// finalURL follows redirects (browser UA) and returns the landing url, or ""
// if it can't be reached. Overridable in tests.
var finalURL = func(rawurl string) string {
	client := &http.Client{Timeout: 12 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", BrowserUA)
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	return resp.Request.URL.String() // final url after redirect chain
}

func hostOf(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// expandRedirects follows each target's redirect and, when it lands on a
// different host, ADDS that landing url+host to the target set. So
// `-d bmw.com` (which bounces to www.bmw.in) ends up crawling bmw.in too.
func expandRedirects(urls []string) (outURLs, outHosts []string) {
	seenU := map[string]bool{}
	seenH := map[string]bool{}
	addU := func(u string) {
		if u != "" && !seenU[u] {
			seenU[u] = true
			outURLs = append(outURLs, u)
		}
	}
	addH := func(h string) {
		if h != "" && !seenH[h] {
			seenH[h] = true
			outHosts = append(outHosts, h)
		}
	}

	// resolve landings concurrently (bounded).
	landings := make([]string, len(urls))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, u := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, u string) {
			defer wg.Done()
			defer func() { <-sem }()
			landings[i] = finalURL(u)
		}(i, u)
	}
	wg.Wait()

	for i, u := range urls {
		addU(u)
		addH(hostOf(u))
		if l := landings[i]; l != "" && hostOf(l) != "" && hostOf(l) != hostOf(u) {
			addU(l)
			addH(hostOf(l))
		}
	}
	return outURLs, outHosts
}
