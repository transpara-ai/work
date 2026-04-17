package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- Test helpers ---

// newTestServer builds an httptest.Server wired to a fresh eventFanout and
// an authSSE-guarded SSE handler chain. The DB pool is nil so the poller
// is a no-op; tests drive events by calling fanout.Publish directly.
func newTestServer(t *testing.T) (*httptest.Server, *server) {
	t.Helper()
	sv := &server{
		apiKey: "test-key",
		fanout: newEventFanout(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /telemetry/sse", sv.authSSE(sv.telemetrySSE))
	mux.HandleFunc("GET /events/subscribe", sv.authSSE(sv.eventsSubscribe))
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, sv
}

// sseClient opens an SSE connection and returns a channel of `data:` payloads
// and a cancel function. Lines beginning with `:` (comments / keepalive) are
// skipped. The reader goroutine exits when the connection closes.
func sseClient(t *testing.T, ts *httptest.Server, path string) (<-chan string, context.CancelFunc, *http.Response) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		cancel()
		t.Fatalf("sse connect: %v", err)
	}
	frames := make(chan string, 64)
	go func() {
		defer close(frames)
		defer resp.Body.Close()
		br := bufio.NewReader(resp.Body)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					// Expected on ctx cancel / server close.
				}
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "data: ") {
				select {
				case frames <- strings.TrimPrefix(line, "data: "):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return frames, cancel, resp
}

// expectFrame reads the next data frame with a timeout.
func expectFrame(t *testing.T, frames <-chan string, timeout time.Duration) string {
	t.Helper()
	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("sse stream closed before frame arrived")
		}
		return f
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for sse frame")
		return ""
	}
}

// expectNoFrame asserts no frame arrives within dur.
func expectNoFrame(t *testing.T, frames <-chan string, dur time.Duration) {
	t.Helper()
	select {
	case f, ok := <-frames:
		if ok {
			t.Fatalf("unexpected sse frame: %s", f)
		}
	case <-time.After(dur):
	}
}

// drainFrames collects every frame that arrives within dur.
func drainFrames(frames <-chan string, dur time.Duration) []string {
	var out []string
	deadline := time.After(dur)
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				return out
			}
			out = append(out, f)
		case <-deadline:
			return out
		}
	}
}

// withSubscribers spins until either the fanout has n subscribers or the
// timeout fires. Tests use this to avoid publishing before the handler has
// registered its subscription.
func withSubscribers(t *testing.T, sv *server, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sv.fanout.NumSubscribers() >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d subscribers (got %d)", n, sv.fanout.NumSubscribers())
}

func mkEvent(evType, source, summary string) EventFrame {
	return EventFrame{
		Type:       evType,
		Source:     source,
		Summary:    summary,
		RecordedAt: time.Now().UTC(),
	}
}

// --- /telemetry/sse tests ---

func TestSSEKeepaliveComment(t *testing.T) {
	// A plain connect without events should still hold the connection open —
	// the handler writes response headers and calls Flush() before waiting.
	// Verifying we get a 200 + correct Content-Type is enough: the keepalive
	// ticker fires at 30s, longer than any reasonable unit-test budget.
	ts, sv := newTestServer(t)
	frames, cancel, resp := sseClient(t, ts, "/telemetry/sse?key=test-key")
	defer cancel()
	if resp.StatusCode != 200 {
		t.Fatalf("status: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type: want text/event-stream, got %q", ct)
	}
	withSubscribers(t, sv, 1)
	_ = frames
}

func TestSSEEventDelivery(t *testing.T) {
	ts, sv := newTestServer(t)
	frames, cancel, _ := sseClient(t, ts, "/telemetry/sse?key=test-key")
	defer cancel()
	withSubscribers(t, sv, 1)

	// Three distinct (source|prefix) keys so the debouncer does not collapse them.
	sv.fanout.Publish(mkEvent("hive.gap.detected", "cto", "alpha"))
	sv.fanout.Publish(mkEvent("agent.state.changed", "builder-a", "beta"))
	sv.fanout.Publish(mkEvent("site.op.received", "site-gateway", "gamma"))

	for i, want := range []string{"alpha", "beta", "gamma"} {
		f := expectFrame(t, frames, 2*time.Second)
		if !strings.Contains(f, want) {
			t.Fatalf("frame %d: expected to contain %q, got %s", i, want, f)
		}
	}
}

func TestSSEDebounce(t *testing.T) {
	ts, sv := newTestServer(t)
	frames, cancel, _ := sseClient(t, ts, "/telemetry/sse?key=test-key")
	defer cancel()
	withSubscribers(t, sv, 1)

	// Ten state.changed events for the same agent in ~100ms. With a 500ms
	// leading-edge throttle the client should see exactly 1 frame (well
	// within the spec's ≤3 bound).
	for i := 0; i < 10; i++ {
		sv.fanout.Publish(mkEvent("agent.state.changed", "builder-a", fmt.Sprintf("tick-%d", i)))
		time.Sleep(10 * time.Millisecond)
	}
	// Wait past the debounce window to be sure nothing else is in flight.
	time.Sleep(150 * time.Millisecond)
	got := drainFrames(frames, 50*time.Millisecond)
	if len(got) < 1 || len(got) > 3 {
		t.Fatalf("debounce: expected 1–3 frames, got %d (frames: %v)", len(got), got)
	}
}

func TestSSEDisconnectCleanup(t *testing.T) {
	ts, sv := newTestServer(t)
	frames, cancel, _ := sseClient(t, ts, "/telemetry/sse?key=test-key")
	withSubscribers(t, sv, 1)

	sv.fanout.Publish(mkEvent("hive.gap.detected", "cto", "hello"))
	_ = expectFrame(t, frames, 2*time.Second)

	// Cancelling the client context should cause r.Context().Done() to fire
	// in the handler, which returns and triggers defer unsubscribe().
	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sv.fanout.NumSubscribers() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("fanout still has %d subscribers after disconnect", sv.fanout.NumSubscribers())
}

func TestSSEAuthQueryString(t *testing.T) {
	ts, _ := newTestServer(t)

	// Valid key via ?key= — expect 200.
	ok, err := ts.Client().Get(ts.URL + "/telemetry/sse?key=test-key")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if ok.StatusCode != 200 {
		t.Fatalf("valid key: want 200, got %d", ok.StatusCode)
	}
	ok.Body.Close()

	// Invalid key — expect 401.
	bad, err := ts.Client().Get(ts.URL + "/telemetry/sse?key=wrong")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if bad.StatusCode != 401 {
		t.Fatalf("invalid key: want 401, got %d", bad.StatusCode)
	}
	bad.Body.Close()

	// No auth at all — expect 401.
	none, err := ts.Client().Get(ts.URL + "/telemetry/sse")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if none.StatusCode != 401 {
		t.Fatalf("no auth: want 401, got %d", none.StatusCode)
	}
	none.Body.Close()
}

func TestSSEAuthBearer(t *testing.T) {
	ts, _ := newTestServer(t)
	req, _ := http.NewRequest("GET", ts.URL+"/telemetry/sse", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("bearer auth: want 200, got %d", resp.StatusCode)
	}
}

// --- /events/subscribe tests ---

func TestEventsSubscribeNoFilter(t *testing.T) {
	ts, sv := newTestServer(t)
	frames, cancel, _ := sseClient(t, ts, "/events/subscribe?key=test-key")
	defer cancel()
	withSubscribers(t, sv, 1)

	// Five mixed events, no filter ⇒ all five arrive.
	wants := []string{"hive.gap.detected", "agent.state.changed", "site.op.received", "work.task.created", "system.bootstrapped"}
	for i, typ := range wants {
		sv.fanout.Publish(mkEvent(typ, fmt.Sprintf("src-%d", i), typ))
	}
	got := drainFrames(frames, 500*time.Millisecond)
	if len(got) != len(wants) {
		t.Fatalf("want %d frames, got %d (frames: %v)", len(wants), len(got), got)
	}
	for i, f := range got {
		if !strings.Contains(f, wants[i]) {
			t.Errorf("frame %d: want to contain %q, got %s", i, wants[i], f)
		}
	}
}

func TestEventsSubscribeMultiPrefix(t *testing.T) {
	ts, sv := newTestServer(t)
	frames, cancel, _ := sseClient(t, ts, "/events/subscribe?key=test-key&types=hive.*,site.op.*")
	defer cancel()
	withSubscribers(t, sv, 1)

	sv.fanout.Publish(mkEvent("hive.gap.detected", "cto", "h1"))
	sv.fanout.Publish(mkEvent("agent.state.changed", "builder-a", "drop-me"))
	sv.fanout.Publish(mkEvent("site.op.received", "gateway", "s1"))
	sv.fanout.Publish(mkEvent("work.task.created", "worker", "drop-me"))

	got := drainFrames(frames, 500*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("want 2 frames, got %d (frames: %v)", len(got), got)
	}
	if !strings.Contains(got[0], "h1") {
		t.Errorf("frame 0: want hive event, got %s", got[0])
	}
	if !strings.Contains(got[1], "s1") {
		t.Errorf("frame 1: want site event, got %s", got[1])
	}
}

func TestEventsSubscribeEmptyTypes(t *testing.T) {
	ts, sv := newTestServer(t)
	// Empty ?types= value should behave exactly like no filter.
	frames, cancel, _ := sseClient(t, ts, "/events/subscribe?key=test-key&types=")
	defer cancel()
	withSubscribers(t, sv, 1)

	sv.fanout.Publish(mkEvent("hive.gap.detected", "cto", "a"))
	sv.fanout.Publish(mkEvent("agent.state.changed", "builder-a", "b"))
	sv.fanout.Publish(mkEvent("work.task.created", "worker", "c"))

	got := drainFrames(frames, 500*time.Millisecond)
	if len(got) != 3 {
		t.Fatalf("want 3 frames, got %d", len(got))
	}
}

// --- Pure unit tests for filter parsing ---

func TestParseTypeFilters(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"hive.*", []string{"hive."}},
		{"hive.*,site.op.*", []string{"hive.", "site.op."}},
		{" hive.* , site.op.* ", []string{"hive.", "site.op."}},
		{"hive.gap.detected", []string{"hive.gap.detected"}},
		{",,", nil},
	}
	for _, c := range cases {
		got := parseTypeFilters(c.in)
		if len(got) != len(c.want) {
			t.Errorf("parseTypeFilters(%q) len: want %d got %d (%v)", c.in, len(c.want), len(got), got)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseTypeFilters(%q)[%d]: want %q got %q", c.in, i, c.want[i], got[i])
			}
		}
	}
}

func TestEventMatchesFilters(t *testing.T) {
	cases := []struct {
		evType  string
		filters []string
		want    bool
	}{
		{"hive.gap.detected", nil, true},                             // no filter = pass
		{"hive.gap.detected", []string{"hive."}, true},               // prefix match
		{"agent.state.changed", []string{"hive."}, false},            // prefix miss
		{"site.op.received", []string{"hive.", "site.op."}, true},    // multi-prefix match
		{"work.task.created", []string{"hive.", "site.op."}, false},  // multi-prefix miss
		{"hive.gap", []string{"hive.gap.detected"}, false},           // prefix is longer than event type
	}
	for _, c := range cases {
		if got := eventMatchesFilters(c.evType, c.filters); got != c.want {
			t.Errorf("eventMatchesFilters(%q, %v) = %v; want %v", c.evType, c.filters, got, c.want)
		}
	}
}

func TestDebounceKey(t *testing.T) {
	// agent.state.changed and agent.state.transition should collapse.
	a := debounceKey(EventFrame{Type: "agent.state.changed", Source: "builder-a"})
	b := debounceKey(EventFrame{Type: "agent.state.transition", Source: "builder-a"})
	if a != b {
		t.Errorf("want collapse under same family, got %q vs %q", a, b)
	}
	// agent.state.* vs agent.budget.* must stay distinct.
	c := debounceKey(EventFrame{Type: "agent.budget.exhausted", Source: "builder-a"})
	if a == c {
		t.Errorf("want distinct families, got %q == %q", a, c)
	}
	// Different source = different key even on same family.
	d := debounceKey(EventFrame{Type: "agent.state.changed", Source: "builder-b"})
	if a == d {
		t.Errorf("want different source = different key, got %q == %q", a, d)
	}
}
