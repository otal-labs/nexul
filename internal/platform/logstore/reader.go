package logstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const (
	searchTimeout = 30 * time.Second
	// base is where the install serves OpenObserve (ZO_BASE_URI).
	base = "/openobserve/api/default"
	// maxResponse bounds what a search answer may hold: 500 lines of up to 2000 characters a column.
	maxResponse = 32 << 20
)

// Config is where the log store is and the login the server reads it with.
type Config struct {
	URL      *url.URL
	Email    string
	Password string
	// HTTP is the client searches go through; nil uses one with the search timeout.
	HTTP *http.Client
}

// Reader reads the nexul stream.
type Reader struct {
	cfg Config
}

// New returns a Reader over cfg.
func New(cfg Config) *Reader {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: searchTimeout + 5*time.Second}
	}
	return &Reader{cfg: cfg}
}

// Page is one newest-first stretch of lines. Until is the time the read is pinned to: pass it back with NextOffset so
// lines logged meanwhile do not shift the pages.
type Page struct {
	Items      []Line    `json:"items"`
	Total      int       `json:"total"`
	HasMore    bool      `json:"has_more"`
	NextOffset int       `json:"next_offset,omitzero"`
	Until      time.Time `json:"until"`
}

// Read answers q: the stream's columns first, so a filter or a column the stream has no line for yet is an empty page
// and not an error, then the count and the page.
func (r *Reader) Read(ctx context.Context, q Query) (Page, error) {
	page := Page{Items: []Line{}, Until: q.Until}
	schema, err := r.schema(ctx)
	if err != nil || len(schema) == 0 {
		return page, err
	}
	clause, none := where(q, func(c string) bool { return slices.Contains(schema, c) })
	if none {
		return page, nil
	}
	rows, err := r.search(ctx, q, countSQL(clause), 0, 1)
	if err != nil || len(rows) == 0 {
		return page, err
	}
	count, ok := rows[0]["c"].(json.Number)
	total, err := count.Int64()
	if !ok || err != nil {
		return page, r.unavailable(ctx, "decode count", fmt.Errorf("count is %v", rows[0]["c"]))
	}
	page.Total = int(total)
	if page.Total <= q.Window.Offset {
		return page, nil
	}
	rows, err = r.search(ctx, q, listSQL(selectable(schema), clause), q.Window.Offset, q.Window.Limit)
	if err != nil {
		return page, err
	}
	for _, row := range rows {
		page.Items = append(page.Items, shape(row))
	}
	end := q.Window.Offset + len(page.Items)
	if page.HasMore = end < page.Total; page.HasMore {
		page.NextOffset = end
	}
	return page, nil
}

// schema lists the stream's columns; none while nothing has been logged yet.
func (r *Reader) schema(ctx context.Context) ([]string, error) {
	body, status, err := r.do(ctx, http.MethodGet, base+"/streams/"+stream+"/schema?type=logs", nil)
	if status == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out struct {
		Schema []struct {
			Name string `json:"name"`
		} `json:"schema"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, r.unavailable(ctx, "decode schema", err)
	}
	cols := make([]string, len(out.Schema))
	for i, f := range out.Schema {
		cols[i] = f.Name
	}
	return cols, nil
}

type searchRequest struct {
	Query struct {
		SQL       string `json:"sql"`
		StartTime int64  `json:"start_time"`
		EndTime   int64  `json:"end_time"`
		From      int    `json:"from"`
		Size      int    `json:"size"`
	} `json:"query"`
	UseCache bool `json:"use_cache"`
	Timeout  int  `json:"timeout"`
}

func (r *Reader) search(ctx context.Context, q Query, sql string, from, size int) ([]map[string]any, error) {
	req := searchRequest{UseCache: false, Timeout: int(searchTimeout / time.Second)}
	req.Query.SQL, req.Query.From, req.Query.Size = sql, from, size
	req.Query.StartTime, req.Query.EndTime = q.Since.UnixMicro(), q.Until.UnixMicro()
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode search: %w", err)
	}
	body, status, err := r.do(ctx, http.MethodPost, base+"/_search", payload)
	if err != nil || status == http.StatusNotFound {
		return nil, err
	}
	var out struct {
		Hits []map[string]any `json:"hits"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, r.unavailable(ctx, "decode search", err)
	}
	return out.Hits, nil
}

// do sends one request with the server's login. A 404 comes back with its status and no error, for the schema of a
// stream not created yet; any other failure is the log store being unavailable.
func (r *Reader) do(ctx context.Context, method, path string, payload []byte) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, searchTimeout+5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, r.cfg.URL.String()+path, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.SetBasicAuth(r.cfg.Email, r.cfg.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.cfg.HTTP.Do(req)
	if err != nil {
		return nil, 0, r.unavailable(ctx, "reach", err)
	}
	defer func() { _ = resp.Body.Close() }() // read-only response body
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return nil, 0, r.unavailable(ctx, "read answer", err)
	}
	if resp.StatusCode == http.StatusNotFound || streamMissing(body) {
		return nil, http.StatusNotFound, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, r.unavailable(ctx, "answered "+strconv.Itoa(resp.StatusCode), errors.New(string(bytes.TrimSpace(body))))
	}
	return body, resp.StatusCode, nil
}

// streamMissing is OpenObserve's search answer before anything has been logged.
func streamMissing(body []byte) bool {
	var e struct {
		Code int `json:"code"`
	}
	return json.Unmarshal(body, &e) == nil && e.Code == 20002
}

// unavailable hides what the store said, which can quote the SQL, and tells the caller what to do.
func (r *Reader) unavailable(ctx context.Context, what string, err error) error {
	slog.WarnContext(ctx, "log store", "step", what, "error", err)
	return fmt.Errorf("%w: the log store (OpenObserve) did not answer; retry in a moment", apperrs.ErrConflict)
}
