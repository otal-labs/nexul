package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/platform/redact"
)

const (
	// QueueLimit is how many records a sender keeps while the server is unreachable.
	QueueLimit = 1000
	// MaxBatch caps the records one frame or request carries.
	MaxBatch = 50
	// MaxBatchBytes keeps a batch's records well under the server's 64 KiB frame limit.
	MaxBatchBytes = 48 << 10

	maxRecordBytes = 16 << 10
	maxChars       = 2000
	maxKey         = 128
	maxAttrs       = 64
)

// Record is one log line a runner or the automations host sends the server to relay.
type Record struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// Stamp says where a relayed record came from. The server builds it from the authenticated connection, never from the payload.
type Stamp struct {
	Source, Name, ID, Machine, Version string
}

// stamped are the attribute names a relayed record's own attributes may not use.
var stamped = []string{"source", "name", "source_id", "machine", "version"}

// Relay re-emits records a sender shipped to the log store alone; the sender's own journal already holds them.
type Relay struct{ h slog.Handler }

// NewRelay writes relayed records to h.
func NewRelay(h slog.Handler) *Relay { return &Relay{h: h} }

// Emit writes recs stamped with s. A nil Relay, an instance with no log store, drops them. It never fails the
// sender: a record the store cannot take is lost like any other line the exporter drops.
func (r *Relay) Emit(ctx context.Context, s Stamp, recs []Record) {
	if r == nil {
		return
	}
	now := time.Now()
	for _, rec := range recs {
		at := rec.Time
		if at.IsZero() || at.After(now.Add(time.Minute)) {
			at = now
		}
		out := slog.NewRecord(at, relayLevel(rec.Level), redact.Tokens(cut(rec.Message)), 0)
		out.AddAttrs(slog.String("source", s.Source), slog.String("name", s.Name), slog.String("source_id", s.ID),
			slog.String("machine", s.Machine), slog.String("version", s.Version))
		out.AddAttrs(payloadAttrs(rec.Attrs)...)
		_ = r.h.Handle(ctx, out) // best-effort, as the exporter's own drops are
	}
}

func relayLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// payloadAttrs flattens a record's attributes into dotted names, redacted and cut, without the names the server stamps.
func payloadAttrs(attrs map[string]any) []slog.Attr {
	var out []slog.Attr
	flatten("", attrs, &out)
	return out
}

func flatten(prefix string, m map[string]any, out *[]slog.Attr) {
	for _, k := range slices.Sorted(maps.Keys(m)) {
		if len(*out) >= maxAttrs {
			return
		}
		if prefix == "" && slices.Contains(stamped, k) {
			continue
		}
		key := prefix + strings.ToValidUTF8(k[:min(len(k), maxKey)], "")
		switch v := m[k].(type) {
		case nil:
		case map[string]any:
			flatten(key+".", v, out)
		case string:
			*out = append(*out, slog.String(key, redact.Tokens(cut(v))))
		case float64:
			*out = append(*out, slog.Float64(key, v))
		case bool:
			*out = append(*out, slog.Bool(key, v))
		default:
			b, _ := json.Marshal(v) // a value that cannot be encoded is written empty
			*out = append(*out, slog.String(key, redact.Tokens(cut(string(b)))))
		}
	}
}

func cut(s string) string {
	if r := []rune(s); len(r) > maxChars {
		return string(r[:maxChars]) + "…"
	}
	return s
}

// Queue holds the newest QueueLimit records a sender has not delivered. It is read with Peek and emptied with Ack,
// so a batch that failed to send is still there for the next try.
type Queue struct {
	mu      sync.Mutex
	items   []queued
	base    uint64
	dropped int
}

type queued struct {
	rec  Record
	size int
}

// NewQueue is an empty queue.
func NewQueue() *Queue { return &Queue{} }

// Handler is a slog handler that queues every record at level or above, redacted.
func (q *Queue) Handler(level string) slog.Handler {
	return slog.NewJSONHandler(queueWriter{q}, &slog.HandlerOptions{Level: parseLevel(level)})
}

// queueWriter takes the JSON handler's line for a record, so groups and attribute kinds follow slog's own rules.
type queueWriter struct{ q *Queue }

func (w queueWriter) Write(p []byte) (int, error) {
	dec := json.NewDecoder(bytes.NewReader(p))
	var line map[string]any
	if err := dec.Decode(&line); err != nil {
		return 0, fmt.Errorf("decode log line: %w", err)
	}
	rec := Record{Level: strings.ToLower(asString(line[slog.LevelKey])), Message: asString(line[slog.MessageKey])}
	rec.Time, _ = time.Parse(time.RFC3339Nano, asString(line[slog.TimeKey]))
	for _, k := range []string{slog.TimeKey, slog.LevelKey, slog.MessageKey} {
		delete(line, k)
	}
	if len(line) > 0 {
		rec.Attrs = line
	}
	w.q.add(rec)
	return len(p), nil
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

// add queues rec, dropping the oldest when the queue is full and counting it.
func (q *Queue) add(rec Record) {
	rec.Message = redact.Tokens(cut(rec.Message))
	scrub(rec.Attrs)
	size := recordSize(rec)
	if size > maxRecordBytes {
		rec.Attrs, size = nil, recordSize(Record{Time: rec.Time, Level: rec.Level, Message: rec.Message})
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) >= QueueLimit {
		q.items = q.items[1:]
		q.base++
		q.dropped++
	}
	q.items = append(q.items, queued{rec: rec, size: size})
}

func recordSize(rec Record) int {
	b, _ := json.Marshal(rec) // the record held only decoded JSON values, so it encodes
	return len(b)
}

// scrub redacts and cuts every string in attrs, nested values included.
func scrub(attrs map[string]any) {
	for k, v := range attrs {
		switch x := v.(type) {
		case string:
			attrs[k] = redact.Tokens(cut(x))
		case map[string]any:
			scrub(x)
		}
	}
}

// Peek returns the oldest records that fit one batch, and the position to Ack once they are delivered. When lines were
// dropped since the last delivered batch it leads with a warning that says how many, and reports that count for Ack.
func (q *Queue) Peek(maxRecords, maxBytes int) (recs []Record, next uint64, dropped int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	size := 0
	if q.dropped > 0 {
		dropped = q.dropped
		recs = append(recs, Record{Time: time.Now().UTC(), Level: "warn", Message: fmt.Sprintf("dropped %d log lines", dropped)})
		size = 200
	}
	taken := 0
	for _, it := range q.items {
		if len(recs) >= maxRecords || (len(recs) > 0 && size+it.size > maxBytes) {
			break
		}
		recs = append(recs, it.rec)
		size += it.size
		taken++
	}
	return recs, q.base + uint64(taken), dropped
}

// Ack removes what Peek returned once the server has it: the records up to next, and the dropped count that came with them.
func (q *Queue) Ack(next uint64, dropped int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if next > q.base {
		n := min(int(next-q.base), len(q.items))
		q.items = q.items[n:]
		q.base += uint64(n)
	}
	q.dropped = max(q.dropped-dropped, 0)
}

// Len is how many records wait.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}
