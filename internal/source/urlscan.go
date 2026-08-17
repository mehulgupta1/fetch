package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mehulgupta1/fetch/internal/jsfilter"
)

// HTTP is urlscan's http getter (injectable for tests).
type HTTP interface {
	// Get fetches url with an optional urlscan API-Key header.
	// retryAfter is seconds parsed from a 429 Retry-After header (0 if none).
	Get(ctx context.Context, url, key string) (body []byte, status, retryAfter int, err error)
}

// RealHTTP calls the real urlscan.io API (trusted https endpoint).
type RealHTTP struct{ Client *http.Client }

func (h RealHTTP) Get(ctx context.Context, url, key string) ([]byte, int, int, error) {
	c := h.Client
	if c == nil {
		c = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	req.Header.Set("User-Agent", "fetch/0.1 (+js-collector)")
	if key != "" {
		req.Header.Set("API-Key", key)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ra := 0
	if v := resp.Header.Get("Retry-After"); v != "" {
		ra, _ = strconv.Atoi(v)
	}
	return body, resp.StatusCode, ra, nil
}

const urlscanBase = "https://urlscan.io/api/v1"

type searchResp struct {
	Results []struct {
		ID     string `json:"_id"`
		Result string `json:"result"`
	} `json:"results"`
}

type resultResp struct {
	Data struct {
		Requests []struct {
			Request struct {
				Request struct {
					URL string `json:"url"`
				} `json:"request"`
			} `json:"request"`
		} `json:"requests"`
	} `json:"data"`
}

// Urlscan is source S6: search a host, open up to `limit` newest recordings,
// collect their .js resource urls. Retries a 429 once.
func Urlscan(ctx context.Context, h HTTP, host, key string, limit int) Result {
	r := Result{Name: "urlscan", Status: "ok"}
	if limit <= 0 || limit > 100 {
		limit = 100 // one page; pagination deferred
	}
	searchURL := fmt.Sprintf("%s/search/?q=domain:%s&size=%d", urlscanBase, host, limit)
	body, status, err := getWithRetry(ctx, h, searchURL, key)
	if err != nil {
		r.Status, r.Reason = "failed", err.Error()
		return r
	}
	if status == 429 {
		r.Status, r.Reason = "failed", "429 rate-limited"
		return r
	}
	if status != 200 {
		r.Status, r.Reason = "failed", fmt.Sprintf("status %d", status)
		return r
	}
	var sr searchResp
	if err := json.Unmarshal(body, &sr); err != nil {
		r.Status, r.Reason = "failed", "bad search json"
		return r
	}
	n := len(sr.Results)
	if n > limit {
		n = limit
	}
	for i := 0; i < n; i++ {
		resURL := sr.Results[i].Result
		if resURL == "" && sr.Results[i].ID != "" {
			resURL = fmt.Sprintf("%s/result/%s/", urlscanBase, sr.Results[i].ID)
		}
		if resURL == "" {
			continue
		}
		db, st, derr := getWithRetry(ctx, h, resURL, key)
		if derr != nil || st != 200 {
			continue // skip this recording, keep going
		}
		var rr resultResp
		if json.Unmarshal(db, &rr) != nil {
			continue
		}
		for _, req := range rr.Data.Requests {
			if u, ok := jsfilter.IsJS(req.Request.Request.URL); ok {
				r.URLs = append(r.URLs, u)
			}
		}
	}
	return r
}

// getWithRetry does one 429 retry (honor Retry-After, else short backoff).
func getWithRetry(ctx context.Context, h HTTP, url, key string) ([]byte, int, error) {
	body, status, ra, err := h.Get(ctx, url, key)
	if err != nil {
		return nil, 0, err
	}
	if status == 429 {
		wait := time.Duration(ra) * time.Second
		if wait <= 0 {
			wait = 5 * time.Second
		}
		select {
		case <-ctx.Done():
			return nil, status, ctx.Err()
		case <-time.After(wait):
		}
		body, status, _, err = h.Get(ctx, url, key)
		if err != nil {
			return nil, 0, err
		}
	}
	return body, status, nil
}
