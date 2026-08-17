package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func has(args []string, want ...string) bool {
	s := strings.Join(args, " ")
	for _, w := range want {
		if !strings.Contains(s, w) {
			return false
		}
	}
	return true
}

// T. FLAG -> ARG TRANSLATION
func TestKatanaArgs(t *testing.T) {
	o := Options{Depth: 3, Concurrency: 20, Rate: 5, Headless: true, Proxy: "http://p:8080", Headers: []string{"A: 1", "B: 2"}}
	a := KatanaArgs(o, "/tmp/l")
	if !has(a, "-d 3", "-c 20", "-rl 5", "-hl", "-proxy http://p:8080", "-list /tmp/l", "-silent") {
		t.Fatalf("T1/6/8/10/12: %v", a)
	}
	if strings.Count(strings.Join(a, " "), "-H ") != 2 {
		t.Fatalf("T15: two headers expected: %v", a)
	}
}

func TestKatanaArgs_SubsScope(t *testing.T) { // T4/T5
	if !has(KatanaArgs(Options{Subs: false}, "l"), "-fs fqdn") {
		t.Fatal("T5: no --subs -> fqdn")
	}
	if has(KatanaArgs(Options{Subs: true}, "l"), "-fs fqdn") {
		t.Fatal("T4: --subs -> rdn (no fqdn)")
	}
}

func TestKatanaArgs_Defaults(t *testing.T) { // T2/T9
	a := KatanaArgs(Options{}, "l")
	if !has(a, "-d 2", "-c 10") {
		t.Fatalf("defaults: %v", a)
	}
}

func TestHakrawlerArgs(t *testing.T) {
	o := Options{Depth: 3, Concurrency: 7, Subs: true, Insecure: true, Proxy: "http://p", Headers: []string{"A: 1", "B: 2"}}
	a := HakrawlerArgs(o)
	if !has(a, "-d 3", "-t 7", "-subs", "-insecure", "-proxy http://p") {
		t.Fatalf("hakrawler: %v", a)
	}
	if !has(a, "-h A: 1;;B: 2") { // T14/T15 joined with ;;
		t.Fatalf("hakrawler headers join: %v", a)
	}
}

func TestSubjsArgs(t *testing.T) { // subjs: no headers/proxy/insecure
	a := SubjsArgs(Options{Concurrency: 9, Headers: []string{"X: 1"}, Insecure: true}, "/tmp/l")
	if !has(a, "-i /tmp/l", "-c 9") {
		t.Fatalf("subjs: %v", a)
	}
	if has(a, "-insecure") || has(a, "-h ") || has(a, "-H ") {
		t.Fatalf("subjs must not get headers/insecure: %v", a)
	}
}

func TestGetjsArgs(t *testing.T) {
	a := GetjsArgs(Options{Concurrency: 4, Headers: []string{"A: 1"}}, "/tmp/l")
	if !has(a, "-input /tmp/l", "-complete", "-threads 4", "-header A: 1") {
		t.Fatalf("getjs: %v", a)
	}
}

func TestProxyEnv(t *testing.T) { // T12
	if ProxyEnv(Options{}) != nil {
		t.Fatal("T13: no proxy -> no env")
	}
	e := ProxyEnv(Options{Proxy: "http://p"})
	if len(e) != 2 || !strings.Contains(e[0], "HTTP_PROXY=http://p") {
		t.Fatalf("proxy env: %v", e)
	}
}

// Grep (S1)
func TestGrep(t *testing.T) {
	r := Grep([]string{"https://t/app.js", "https://t/x.css", "https://t/b.js"})
	if len(r.URLs) != 2 || r.Status != "ok" {
		t.Fatalf("grep: %+v", r)
	}
}

// fakeExec returns canned stdout lines / error.
type fakeExec struct {
	lines  []string
	stderr string
	err    error
}

func (f fakeExec) Run(_ context.Context, _ []string, _ string, _ []string, onLine func(string)) (string, error) {
	for _, l := range f.lines {
		onLine(l)
	}
	return f.stderr, f.err
}

func TestRunTool_OK(t *testing.T) {
	r := RunTool(context.Background(), fakeExec{lines: []string{"https://t/a.js", "https://t/x.png"}}, "katana", "katana", nil, "", nil)
	if len(r.URLs) != 1 || r.Status != "ok" {
		t.Fatalf("O26: only js kept: %+v", r)
	}
}

func TestRunTool_Failed(t *testing.T) { // O15 / DB1
	r := RunTool(context.Background(), fakeExec{stderr: "boom: bad flag", err: fmt.Errorf("exit 1")}, "katana", "katana", nil, "", nil)
	if r.Status != "failed" || r.Reason != "boom: bad flag" {
		t.Fatalf("DB1: want failed reason, got %+v", r)
	}
}

func TestRunTool_PartialKept(t *testing.T) { // best-effort: partial urls kept despite error
	r := RunTool(context.Background(), fakeExec{lines: []string{"https://t/a.js"}, err: fmt.Errorf("killed")}, "katana", "katana", nil, "", nil)
	if len(r.URLs) != 1 || r.Status != "ok" {
		t.Fatalf("partial should be kept ok, got %+v", r)
	}
}

// fakeHTTP for urlscan
type fakeHTTP struct {
	responses map[string]fakeResp
	calls     int
}
type fakeResp struct {
	body       []byte
	status     int
	retryAfter int
}

func (f *fakeHTTP) Get(_ context.Context, url, _ string) ([]byte, int, int, error) {
	f.calls++
	if r, ok := f.responses[url]; ok {
		return r.body, r.status, r.retryAfter, nil
	}
	// default: any result url returns one js
	if strings.Contains(url, "/result/") {
		rr := resultResp{}
		rr.Data.Requests = []struct {
			Request struct {
				Request struct {
					URL string `json:"url"`
				} `json:"request"`
			} `json:"request"`
		}{}
		return []byte(`{"data":{"requests":[{"request":{"request":{"url":"https://t.com/app.js"}}}]}}`), 200, 0, nil
	}
	return nil, 404, 0, nil
}

func searchJSON(ids ...string) []byte {
	type res struct {
		ID     string `json:"_id"`
		Result string `json:"result"`
	}
	var s struct {
		Results []res `json:"results"`
	}
	for _, id := range ids {
		s.Results = append(s.Results, res{ID: id, Result: "https://urlscan.io/api/v1/result/" + id + "/"})
	}
	b, _ := json.Marshal(s)
	return b
}

func TestUrlscan_Parse(t *testing.T) { // UL7 / O23
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=100"
	h := &fakeHTTP{responses: map[string]fakeResp{
		sURL: {body: searchJSON("id1", "id2"), status: 200},
	}}
	r := Urlscan(context.Background(), h, "t.com", "", 0)
	if len(r.URLs) != 2 || r.Status != "ok" { // one js per recording, 2 recordings
		t.Fatalf("UL7: got %+v", r)
	}
}

func TestUrlscan_Cap(t *testing.T) { // UL1 cap honored
	ids := []string{}
	for i := 0; i < 50; i++ {
		ids = append(ids, fmt.Sprintf("id%d", i))
	}
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=5"
	h := &fakeHTTP{responses: map[string]fakeResp{sURL: {body: searchJSON(ids...), status: 200}}}
	r := Urlscan(context.Background(), h, "t.com", "", 5)
	if len(r.URLs) != 5 {
		t.Fatalf("UL1: cap 5 -> 5 recordings opened, got %d", len(r.URLs))
	}
}

func TestUrlscan_429Retry(t *testing.T) { // UR2: 429 then 429 -> failed
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=100"
	h := &fakeHTTP{responses: map[string]fakeResp{sURL: {body: nil, status: 429}}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	r := Urlscan(ctx, h, "t.com", "", 0)
	if r.Status != "failed" {
		t.Fatalf("UR2: want failed on repeated 429, got %+v", r)
	}
}

func TestUrlscan_Empty(t *testing.T) { // UL9 / O24
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=100"
	h := &fakeHTTP{responses: map[string]fakeResp{sURL: {body: []byte(`{"results":[]}`), status: 200}}}
	r := Urlscan(context.Background(), h, "t.com", "", 0)
	if len(r.URLs) != 0 || r.Status != "ok" {
		t.Fatalf("UL9: want empty ok, got %+v", r)
	}
}
