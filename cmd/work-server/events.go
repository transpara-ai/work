package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// EventFrame is the per-event payload pushed to SSE subscribers.
//
// The shape mirrors the subset of eventgraph's event.Event that the hive's
// `Writer.SubscribeToBus` persists to telemetry_event_stream. Keeping the JSON
// shape stable lets dashboard consumers decode frames the same way whether the
// work-server is later rewired to a shared bus or remains on the poll bridge.
type EventFrame struct {
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	Summary    string          `json:"summary,omitempty"`
	Content    json.RawMessage `json:"content,omitempty"`
	RecordedAt time.Time       `json:"recorded_at"`
}

// eventFanout broadcasts EventFrames to all currently-subscribed SSE handlers.
//
// Subscribers are identified by an internal id (so we don't leak channels to
// callers). Each subscriber gets a buffered channel; publishes that would block
// are dropped silently — telemetry is best-effort, matching the pattern at
// `lovyou-ai-hive/pkg/telemetry/writer.go:204-237`.
type eventFanout struct {
	mu     sync.Mutex
	subs   map[int64]chan EventFrame
	nextID int64
}

func newEventFanout() *eventFanout {
	return &eventFanout{subs: make(map[int64]chan EventFrame)}
}

// Subscribe registers a new subscriber with a buffered channel of the given
// size and returns the read-only channel plus an unsubscribe function that
// removes the subscription and closes the channel.
func (f *eventFanout) Subscribe(buf int) (<-chan EventFrame, func()) {
	if buf <= 0 {
		buf = 256
	}
	ch := make(chan EventFrame, buf)
	f.mu.Lock()
	id := f.nextID
	f.nextID++
	f.subs[id] = ch
	f.mu.Unlock()

	unsub := func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		if existing, ok := f.subs[id]; ok {
			close(existing)
			delete(f.subs, id)
		}
	}
	return ch, unsub
}

// Publish delivers ev to every current subscriber. A full subscriber channel
// is silently skipped so one slow consumer cannot stall the others.
func (f *eventFanout) Publish(ev EventFrame) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ch := range f.subs {
		select {
		case ch <- ev:
		default:
			// Subscriber is backed up; drop rather than block.
		}
	}
}

// NumSubscribers reports the current subscriber count. Used in tests to
// assert cleanup when a connection disconnects.
func (f *eventFanout) NumSubscribers() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.subs)
}

// --- Type-prefix filtering (used by /events/subscribe) ---

// parseTypeFilters splits a comma-separated ?types= value into a slice of
// prefixes. A trailing "*" is stripped so callers may write either
// `hive.*` or `hive.` interchangeably.
func parseTypeFilters(q string) []string {
	if q == "" {
		return nil
	}
	parts := strings.Split(q, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimSuffix(p, "*")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// eventMatchesFilters reports whether evType starts with any filter. An empty
// filter slice is pass-through (no filter applied).
func eventMatchesFilters(evType string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if strings.HasPrefix(evType, f) {
			return true
		}
	}
	return false
}

// --- SSE auth with query-string fallback ---

// authSSE is the auth middleware for SSE endpoints. It accepts three sources,
// in order: Authorization: Bearer header, ws_key cookie, and `?key=` query
// parameter. The query-string arm exists because browser `EventSource` cannot
// set custom headers; callers that use it get a warning log (once per
// connection) recommending cookie auth.
func (sv *server) authSSE(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); found && token == sv.apiKey {
			next(w, r)
			return
		}
		if c, err := r.Cookie("ws_key"); err == nil && c.Value == sv.apiKey {
			next(w, r)
			return
		}
		if k := r.URL.Query().Get("key"); k != "" && k == sv.apiKey {
			log.Printf("sse auth via query string from %s — key will appear in access logs, prefer cookie auth", r.RemoteAddr)
			next(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "invalid or missing API key")
	}
}

// --- /events/subscribe handler ---

// eventsSubscribe handles GET /events/subscribe[?types=<prefix1>,<prefix2>] —
// a raw, filtered SSE feed with no debounce. Accepts the same auth sources as
// /telemetry/sse.
func (sv *server) eventsSubscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	filters := parseTypeFilters(r.URL.Query().Get("types"))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	events, unsubscribe := sv.fanout.Subscribe(256)
	defer unsubscribe()

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case ev, ok := <-events:
			if !ok {
				return
			}
			if !eventMatchesFilters(ev.Type, filters) {
				continue
			}
			if !writeEventFrame(w, flusher, ev) {
				return
			}
		}
	}
}

// writeEventFrame marshals ev and writes a single SSE `data:` frame. Returns
// false on write error so the caller can exit the loop.
func writeEventFrame(w http.ResponseWriter, flusher http.Flusher, ev EventFrame) bool {
	b, err := json.Marshal(ev)
	if err != nil {
		return true // skip this frame; keep connection open
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

// --- Poll bridge: hive.telemetry_event_stream → fanout ---
//
// The work-server and hive run as separate binaries per the nucbuntu
// deployment, so they cannot share an in-process bus. Instead, hive's
// Writer.SubscribeToBus persists every bus event to telemetry_event_stream
// with a monotonic BIGSERIAL id; we tail it with `WHERE id > last_seen` at
// 500ms. If a future refactor colocates the two servers, replace this with
// a direct bus.Subscribe() call.
const (
	eventPollInterval = 500 * time.Millisecond
	eventPollBatch    = 200
)

// runEventPoller tails telemetry_event_stream and republishes each new row on
// the fanout. It exits when ctx is done.
func (sv *server) runEventPoller(ctx context.Context) {
	if sv.pool == nil || sv.fanout == nil {
		return
	}

	var lastID int64
	// Seed from the current max id so we only push events recorded AFTER
	// startup. Without this, every restart would replay the ring buffer.
	if err := sv.pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) FROM telemetry_event_stream`).Scan(&lastID); err != nil {
		// Table may not exist yet; start from 0 and let the poller catch up.
		lastID = 0
	}

	ticker := time.NewTicker(eventPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			newLast, err := sv.pollAndPublish(ctx, lastID)
			if err != nil {
				// Log at most once per failure; keep the poller alive —
				// transient DB hiccups must not kill the event stream.
				continue
			}
			if newLast > lastID {
				lastID = newLast
			}
		}
	}
}

// pollAndPublish fetches rows with id > after and publishes each to the
// fanout. Returns the largest id seen.
func (sv *server) pollAndPublish(ctx context.Context, after int64) (int64, error) {
	qctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const q = `SELECT id, event_type, actor_role, summary, raw_content, recorded_at
		FROM telemetry_event_stream
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2`
	rows, err := sv.pool.Query(qctx, q, after, eventPollBatch)
	if err != nil {
		return after, err
	}
	defer rows.Close()

	max := after
	for rows.Next() {
		var (
			id         int64
			evType     string
			role       string
			summary    *string
			rawContent []byte
			recordedAt time.Time
		)
		if err := rows.Scan(&id, &evType, &role, &summary, &rawContent, &recordedAt); err != nil {
			return max, err
		}
		frame := EventFrame{
			Type:       evType,
			Source:     role,
			RecordedAt: recordedAt,
			Content:    rawContent,
		}
		if summary != nil {
			frame.Summary = *summary
		}
		sv.fanout.Publish(frame)
		if id > max {
			max = id
		}
	}
	return max, rows.Err()
}
