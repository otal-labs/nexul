package logstore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeStore is an OpenObserve that records what it was asked and answers from canned replies.
type fakeStore struct {
	mu      sync.Mutex
	schema  func(w http.ResponseWriter)
	search  func(w http.ResponseWriter, sql string)
	auth    string
	reqs    []searchRequest
	schemas int
}

const fullSchema = `{"schema":[{"name":"_timestamp"},{"name":"body"},{"name":"severity"},{"name":"trace_id"},{"name":"source"},{"name":"api_token"},{"name":"span_id"},{"name":"error"}]}`

func (f *fakeStore) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.auth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/openobserve/api/default/streams/nexul/schema":
			assert.Equal(t, "logs", r.URL.Query().Get("type"))
			f.schemas++
			f.schema(w)
		case "/openobserve/api/default/_search":
			var req searchRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			f.reqs = append(f.reqs, req)
			f.search(w, req.Query.SQL)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	})
}

func newStore(t *testing.T, f *fakeStore) *Reader {
	t.Helper()
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return New(Config{URL: u, Email: "nexul@example.com", Password: "pw"})
}

func reply(body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) { _, _ = w.Write([]byte(body)) }
}

func mustQuery(t *testing.T, p Params) Query {
	t.Helper()
	q, err := p.Query(time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	return q
}

func TestReader_Read_AsksForNamedColumnsOnly(t *testing.T) {
	f := &fakeStore{
		schema: reply(fullSchema),
		search: func(w http.ResponseWriter, sql string) {
			if strings.Contains(sql, "count(*)") {
				_, _ = w.Write([]byte(`{"hits":[{"c":3}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"hits":[{"_timestamp":1790000000000000,"severity":"ERROR","body":"boom","error":"exit 1"}]}`))
		},
	}
	page, err := newStore(t, f).Read(t.Context(), mustQuery(t, Params{Level: "warn", Limit: 1, Offset: 1}))
	require.NoError(t, err)

	assert.Equal(t, "Basic bmV4dWxAZXhhbXBsZS5jb206cHc=", f.auth)
	require.Len(t, f.reqs, 2)
	assert.Equal(t, `SELECT count(*) AS c FROM "nexul" WHERE severity IN ('WARN', 'ERROR')`, f.reqs[0].Query.SQL)
	assert.Equal(t, `SELECT "_timestamp", "body", "severity", "trace_id", "source", "error" FROM "nexul" WHERE severity IN ('WARN', 'ERROR') ORDER BY _timestamp DESC`, f.reqs[1].Query.SQL,
		"no *, and neither the secret-named nor the bookkeeping column")
	list := f.reqs[1]
	assert.False(t, list.UseCache)
	assert.Equal(t, 30, list.Timeout)
	assert.Equal(t, 1, list.Query.From)
	assert.Equal(t, 1, list.Query.Size)
	assert.Equal(t, time.Date(2026, 9, 25, 11, 0, 0, 0, time.UTC).UnixMicro(), list.Query.StartTime)
	assert.Equal(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC).UnixMicro(), list.Query.EndTime)

	assert.Equal(t, 3, page.Total)
	assert.True(t, page.HasMore, "offset 1 and one line reach line 2 of 3")
	assert.Equal(t, 2, page.NextOffset)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "boom", page.Items[0].Message)
	assert.Equal(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC), page.Until)
}

func TestReader_Read_Paging(t *testing.T) {
	hits := func(n int) string {
		rows := make([]string, n)
		for i := range rows {
			rows[i] = `{"_timestamp":1790000000000000,"body":"x"}`
		}
		return `{"hits":[` + strings.Join(rows, ",") + `]}`
	}
	tests := []struct {
		name           string
		total, offset  int
		limit, returns int
		hasMore        bool
		nextOffset     int
		lists          int
	}{
		{"more beyond the page", 5, 0, 2, 2, true, 2, 2},
		{"the last page", 5, 4, 2, 1, false, 0, 2},
		{"offset past the end reads no lines", 5, 5, 2, 0, false, 0, 1},
		{"nothing matches", 0, 0, 2, 0, false, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeStore{
				schema: reply(fullSchema),
				search: func(w http.ResponseWriter, sql string) {
					if strings.Contains(sql, "count(*)") {
						_, _ = w.Write([]byte(`{"hits":[{"c":` + strconv.Itoa(tt.total) + `}]}`))
						return
					}
					_, _ = w.Write([]byte(hits(tt.returns)))
				},
			}
			page, err := newStore(t, f).Read(t.Context(), mustQuery(t, Params{Limit: tt.limit, Offset: tt.offset}))
			require.NoError(t, err)
			assert.Len(t, f.reqs, tt.lists)
			assert.Equal(t, tt.total, page.Total)
			assert.Len(t, page.Items, tt.returns)
			assert.Equal(t, tt.hasMore, page.HasMore)
			assert.Equal(t, tt.nextOffset, page.NextOffset)
		})
	}
}

func TestReader_Read_NothingLoggedYetIsAnEmptyPage(t *testing.T) {
	tests := []struct {
		name string
		f    *fakeStore
	}{
		{"no stream", &fakeStore{schema: func(w http.ResponseWriter) { w.WriteHeader(http.StatusNotFound) }}},
		{"an empty schema", &fakeStore{schema: reply(`{"schema":[]}`)}},
		{"a search that finds no stream", &fakeStore{schema: reply(fullSchema), search: func(w http.ResponseWriter, _ string) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":20002,"message":"Search stream not found: nexul"}`))
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := newStore(t, tt.f).Read(t.Context(), mustQuery(t, Params{}))
			require.NoError(t, err)
			assert.Empty(t, page.Items)
			assert.NotNil(t, page.Items, "an empty list, not null")
			assert.Zero(t, page.Total)
		})
	}
}

func TestReader_Read_FilterOnAColumnTheStreamLacksSearchesNothing(t *testing.T) {
	f := &fakeStore{schema: reply(`{"schema":[{"name":"_timestamp"},{"name":"body"},{"name":"severity"}]}`)}
	page, err := newStore(t, f).Read(t.Context(), mustQuery(t, Params{Source: "web"}))
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.Empty(t, f.reqs, "no search was made")
}

func TestReader_Read_QuoteInjectionStaysInsideTheString(t *testing.T) {
	f := &fakeStore{
		schema: reply(fullSchema),
		search: func(w http.ResponseWriter, _ string) { _, _ = w.Write([]byte(`{"hits":[{"c":0}]}`)) },
	}
	_, err := newStore(t, f).Read(t.Context(), mustQuery(t, Params{TraceID: "x' OR '1'='1", Text: "'; DROP TABLE nexul; --"}))
	require.NoError(t, err)
	assert.Equal(t, `SELECT count(*) AS c FROM "nexul" WHERE severity IN ('INFO', 'WARN', 'ERROR') AND trace_id = 'x'' OR ''1''=''1' AND str_match_ignore_case(body, '''; DROP TABLE nexul; --')`, f.reqs[0].Query.SQL)
}

func TestReader_Read_TheStoreNotAnswering(t *testing.T) {
	tests := []struct {
		name string
		f    *fakeStore
	}{
		{"schema 500", &fakeStore{schema: func(w http.ResponseWriter) {
			http.Error(w, "SELECT secret FROM x failed", http.StatusInternalServerError)
		}}},
		{"schema 401", &fakeStore{schema: func(w http.ResponseWriter) { http.Error(w, "unauthorized", http.StatusUnauthorized) }}},
		{"schema not json", &fakeStore{schema: reply(`<html>`)}},
		{"search 500", &fakeStore{schema: reply(fullSchema), search: func(w http.ResponseWriter, _ string) {
			http.Error(w, "SELECT secret FROM x failed", http.StatusInternalServerError)
		}}},
		{"search not json", &fakeStore{schema: reply(fullSchema), search: func(w http.ResponseWriter, _ string) { _, _ = w.Write([]byte(`nope`)) }}},
		{"a count that is a string", &fakeStore{schema: reply(fullSchema), search: func(w http.ResponseWriter, _ string) { _, _ = w.Write([]byte(`{"hits":[{"c":"12"}]}`)) }}},
		{"a count that is missing", &fakeStore{schema: reply(fullSchema), search: func(w http.ResponseWriter, _ string) { _, _ = w.Write([]byte(`{"hits":[{"n":12}]}`)) }}},
		{"a count that is fractional", &fakeStore{schema: reply(fullSchema), search: func(w http.ResponseWriter, _ string) { _, _ = w.Write([]byte(`{"hits":[{"c":1.5}]}`)) }}},
		{"a list that fails after the count", &fakeStore{schema: reply(fullSchema), search: func(w http.ResponseWriter, sql string) {
			if strings.Contains(sql, "count(*)") {
				_, _ = w.Write([]byte(`{"hits":[{"c":2}]}`))
				return
			}
			http.Error(w, "boom", http.StatusBadGateway)
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := newStore(t, tt.f).Read(t.Context(), mustQuery(t, Params{}))
			require.ErrorIs(t, err, apperrs.ErrConflict)
			assert.Equal(t, "conflict: the log store (OpenObserve) did not answer; retry in a moment", err.Error(), "nothing the store said reaches the caller")
			assert.Empty(t, page.Items)
		})
	}
}

func TestReader_Read_TheStoreUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	srv.Close()

	_, err = New(Config{URL: u}).Read(t.Context(), mustQuery(t, Params{}))
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.ErrorContains(t, err, "retry")
}
