package source

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ---- T translation (remaining) ----
func TestT3_DepthZero(t *testing.T) {
	if !has(KatanaArgs(Options{Depth: 0}, "l"), "-d 2") { // 0 -> default 2
		t.Fatal("depth 0 uses default")
	}
	// explicit want: passing Depth stays; depth() maps <=0 to 2 by design
}

func TestT7_NoRate(t *testing.T) {
	if has(KatanaArgs(Options{}, "l"), "-rl") {
		t.Fatal("T7: no --rate -> no -rl")
	}
}

func TestT11_NoHeadless(t *testing.T) {
	if has(KatanaArgs(Options{}, "l"), "-hl") {
		t.Fatal("T11: no --headless -> no -hl")
	}
}

func TestT13_NoProxy(t *testing.T) {
	if ProxyEnv(Options{}) != nil || has(KatanaArgs(Options{}, "l"), "-proxy") || has(HakrawlerArgs(Options{}), "-proxy") {
		t.Fatal("T13: no proxy anywhere")
	}
}

func TestT17_Insecure(t *testing.T) {
	if !has(HakrawlerArgs(Options{Insecure: true}), "-insecure") {
		t.Fatal("T17: hakrawler -insecure")
	}
	// subjs/getjs have no insecure flag
	if has(SubjsArgs(Options{Insecure: true}, "l"), "-insecure") || has(GetjsArgs(Options{Insecure: true}, "l"), "-insecure") {
		t.Fatal("T17: subjs/getjs must not get -insecure")
	}
}

func TestT18_NoInsecure(t *testing.T) {
	if has(HakrawlerArgs(Options{}), "-insecure") {
		t.Fatal("T18: default verifies TLS")
	}
}

func TestT20_UnsupportedSkip(t *testing.T) {
	// --rate only reaches katana; subjs/getjs/hakrawler never get -rl
	if has(HakrawlerArgs(Options{Rate: 5}), "-rl") || has(SubjsArgs(Options{Rate: 5}, "l"), "-rl") {
		t.Fatal("T20: rate must not leak to non-katana tools")
	}
}

func TestHeadless_HL1(t *testing.T) { // HL1: --headless -> katana -hl + -nos (root)
	a := KatanaArgs(Options{Headless: true}, "l")
	if !has(a, "-hl") || !has(a, "-nos") {
		t.Fatalf("HL1: headless needs -hl and -nos, got %v", a)
	}
}

// ---- IN input feeding ----
func TestIN1_KatanaList(t *testing.T) {
	if !has(KatanaArgs(Options{}, "/tmp/x"), "-list /tmp/x") {
		t.Fatal("IN1: katana -list file")
	}
}

func TestIN2_HakrawlerStdin(t *testing.T) {
	// hakrawler args reference NO file (input via stdin)
	if strings.Contains(strings.Join(HakrawlerArgs(Options{}), " "), "/tmp") {
		t.Fatal("IN2: hakrawler takes stdin, no file arg")
	}
}

func TestIN3_SubjsFile(t *testing.T) {
	if !has(SubjsArgs(Options{}, "/tmp/x"), "-i /tmp/x") {
		t.Fatal("IN3: subjs -i file")
	}
}

func TestIN4_GetjsFileComplete(t *testing.T) {
	a := GetjsArgs(Options{}, "/tmp/x")
	if !has(a, "-input /tmp/x", "-complete") {
		t.Fatal("IN4: getjs -input + -complete")
	}
}

// ---- UR retry / UL depth ----

// seqHTTP returns queued statuses in order per any url.
type seqHTTP struct {
	statuses []int
	bodies   [][]byte
	i        int
	calls    int
}

func (s *seqHTTP) Get(_ context.Context, url, _ string) ([]byte, int, int, error) {
	s.calls++
	st := 200
	var body []byte
	if s.i < len(s.statuses) {
		st = s.statuses[s.i]
	}
	if s.i < len(s.bodies) {
		body = s.bodies[s.i]
	}
	s.i++
	if body == nil && st == 200 {
		if strings.Contains(url, "/result/") {
			body = []byte(`{"data":{"requests":[{"request":{"request":{"url":"https://t/a.js"}}}]}}`)
		} else {
			body = searchJSON("id1")
		}
	}
	return body, st, 0, nil
}

func TestUR1_429ThenOK(t *testing.T) {
	// search: 429 then 200(search), then result 200
	h := &seqHTTP{
		statuses: []int{429, 200, 200},
		bodies:   [][]byte{nil, searchJSON("id1"), []byte(`{"data":{"requests":[{"request":{"request":{"url":"https://t/a.js"}}}]}}`)},
	}
	r := Urlscan(context.Background(), h, "t.com", "", 0)
	if r.Status != "ok" || len(r.URLs) != 1 {
		t.Fatalf("UR1: want ok 1 js after retry, got %+v", r)
	}
}

func TestUR5_NoRetryOn200(t *testing.T) {
	h := &seqHTTP{statuses: []int{200, 200}}
	Urlscan(context.Background(), h, "t.com", "", 0)
	if h.calls != 2 { // 1 search + 1 detail, no extra retry
		t.Fatalf("UR5: want 2 calls, got %d", h.calls)
	}
}

func TestUR6_Non429NotRetried(t *testing.T) {
	h := &seqHTTP{statuses: []int{500}}
	r := Urlscan(context.Background(), h, "t.com", "", 0)
	if r.Status != "failed" || h.calls != 1 {
		t.Fatalf("UR6: 500 not retried -> failed, calls=%d %+v", h.calls, r)
	}
}

func TestUL2_FewerThanN(t *testing.T) {
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=100"
	h := &fakeHTTP{responses: map[string]fakeResp{sURL: {body: searchJSON("id1", "id2"), status: 200}}}
	r := Urlscan(context.Background(), h, "t.com", "", 0)
	if len(r.URLs) != 2 {
		t.Fatalf("UL2: 2 results -> 2 opened, got %d", len(r.URLs))
	}
}

func TestUL3_LimitOverride(t *testing.T) {
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=3"
	h := &fakeHTTP{responses: map[string]fakeResp{sURL: {body: searchJSON("a", "b", "c", "d"), status: 200}}}
	r := Urlscan(context.Background(), h, "t.com", "", 3)
	if len(r.URLs) != 3 {
		t.Fatalf("UL3: limit 3 honored, got %d", len(r.URLs))
	}
}

func TestUL6_ZeroNoCap(t *testing.T) {
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=100"
	h := &fakeHTTP{responses: map[string]fakeResp{sURL: {body: searchJSON("a"), status: 200}}}
	r := Urlscan(context.Background(), h, "t.com", "", 0) // 0 -> size 100
	if r.Status != "ok" {
		t.Fatalf("UL6: %+v", r)
	}
}

func TestUL8_RecordingNoJS(t *testing.T) {
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=100"
	rURL := "https://urlscan.io/api/v1/result/id1/"
	h := &fakeHTTP{responses: map[string]fakeResp{
		sURL: {body: searchJSON("id1"), status: 200},
		rURL: {body: []byte(`{"data":{"requests":[{"request":{"request":{"url":"https://t/x.png"}}}]}}`), status: 200},
	}}
	r := Urlscan(context.Background(), h, "t.com", "", 0)
	if len(r.URLs) != 0 || r.Status != "ok" {
		t.Fatalf("UL8: no js -> empty ok, got %+v", r)
	}
}

func TestUL10_DetailFailSkip(t *testing.T) {
	sURL := "https://urlscan.io/api/v1/search/?q=domain:t.com&size=100"
	r1 := "https://urlscan.io/api/v1/result/id1/"
	// id2 has no explicit response -> default handler returns one js
	h := &fakeHTTP{responses: map[string]fakeResp{
		sURL: {body: searchJSON("id1", "id2"), status: 200},
		r1:   {status: 500}, // id1 detail fails
	}}
	r := Urlscan(context.Background(), h, "t.com", "", 0)
	if len(r.URLs) != 1 { // id1 skipped, id2 default gives 1
		t.Fatalf("UL10: want 1 (id1 skipped), got %d", len(r.URLs))
	}
}

// ---- DB diagnostics (Reason) ----
func TestDB2_ReasonTimeout(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if Reason(ctx, "", context.DeadlineExceeded) != "timeout" {
		t.Fatal("DB2")
	}
}

func TestDB3_ReasonStderr(t *testing.T) { // last non-empty stderr line
	if Reason(context.Background(), "warn\nfatal: boom\n", errBoom{}) != "fatal: boom" {
		t.Fatal("DB3 stderr line")
	}
}

func TestDB13_SuccessNoReason(t *testing.T) {
	r := RunTool(context.Background(), fakeExec{lines: []string{"https://t/a.js"}}, "x", "x", nil, "", nil)
	if r.Reason != "" || r.Status != "ok" {
		t.Fatalf("DB13: %+v", r)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

// ---- default User-Agent injection ----
func TestUA_Injected(t *testing.T) {
	o := Options{UserAgent: "TestUA"}
	if !has(KatanaArgs(o, "l"), "User-Agent: TestUA") {
		t.Fatal("katana should carry default UA header")
	}
	if !has(GetjsArgs(o, "l"), "User-Agent: TestUA") {
		t.Fatal("getjs should carry default UA header")
	}
	if !has(HakrawlerArgs(o), "User-Agent: TestUA") {
		t.Fatal("hakrawler should carry default UA")
	}
	if !has(SubjsArgs(o, "l"), "-ua TestUA") {
		t.Fatal("subjs should carry -ua")
	}
}

func TestUA_UserHeaderWins(t *testing.T) {
	o := Options{UserAgent: "Default", Headers: []string{"User-Agent: Custom"}}
	if has(KatanaArgs(o, "l"), "User-Agent: Default") {
		t.Fatal("must not inject default UA when user set one")
	}
	if !has(SubjsArgs(o, "l"), "-ua Custom") {
		t.Fatal("subjs should use the user's UA value")
	}
}
