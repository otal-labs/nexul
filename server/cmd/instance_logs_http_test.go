package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/logstore"
	"github.com/otal-labs/nexul/internal/runner"
)

// fakeLogStore answers the two calls a read makes: the stream's schema, then a count or a page of lines.
func fakeLogStore(t *testing.T, status int) *url.URL {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			http.Error(w, "SELECT secret failed", status)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/schema") {
			_, _ = w.Write([]byte(`{"schema":[{"name":"_timestamp"},{"name":"body"},{"name":"severity"},{"name":"api_token"}]}`))
			return
		}
		var req struct {
			Query struct{ SQL string }
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		if strings.Contains(req.Query.SQL, "count(*)") {
			_, _ = w.Write([]byte(`{"hits":[{"c":1}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"hits":[{"_timestamp":1790000000000000,"severity":"ERROR","body":"deploy failed"}]}`))
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return u
}

func logsService(allow bool, store *url.URL) *runner.Service {
	s := runner.NewService(noopRunnerRepo{}, fakeUpgradeDispatch{}).WithGate(fakeUpgradeGate{allow: allow})
	if store != nil {
		s.WithLogs(logstore.New(logstore.Config{URL: store}))
	}
	return s
}

func TestInstanceLogsHandler(t *testing.T) {
	get := func(svc *runner.Service, query string) *httptest.ResponseRecorder {
		req := withActor(httptest.NewRequest(http.MethodGet, "/api/instance/logs?"+query, nil), "user-1")
		rec := httptest.NewRecorder()
		instanceLogsHandler(svc).ServeHTTP(rec, req)
		return rec
	}

	tests := []struct {
		name  string
		svc   *runner.Service
		query string
		code  int
		msg   string
	}{
		{"without logs:read", logsService(false, fakeLogStore(t, 200)), "", http.StatusForbidden, ""},
		{"a limit that is no number", logsService(true, fakeLogStore(t, 200)), "limit=many", http.StatusBadRequest, "limit must be a whole number"},
		{"an unknown source", logsService(true, fakeLogStore(t, 200)), "source=database", http.StatusBadRequest, "source must be one of"},
		{"a range past 7 days", logsService(true, fakeLogStore(t, 200)), "since=8d", http.StatusBadRequest, "at most 7 days"},
		{"an instance that keeps no logs", logsService(true, nil), "", http.StatusBadRequest, "this instance keeps no logs"},
		{"a log store that errors", logsService(true, fakeLogStore(t, 500)), "", http.StatusConflict, "log store (OpenObserve)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(tt.svc, tt.query)
			assert.Equal(t, tt.code, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.msg)
			assert.NotContains(t, rec.Body.String(), "SELECT secret", "what the store said stays out of the answer")
		})
	}

	t.Run("a holder reads the page and its pinned until", func(t *testing.T) {
		rec := get(logsService(true, fakeLogStore(t, 200)), "level=error&limit=10&until=2026-09-25T12:00:00Z")
		require.Equal(t, http.StatusOK, rec.Code)
		var page struct {
			Items []struct {
				Level, Source, Message string
			} `json:"items"`
			Total   int    `json:"total"`
			HasMore bool   `json:"has_more"`
			Until   string `json:"until"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		assert.Equal(t, "error", page.Items[0].Level)
		assert.Equal(t, "server", page.Items[0].Source)
		assert.Equal(t, "deploy failed", page.Items[0].Message)
		assert.Equal(t, 1, page.Total)
		assert.False(t, page.HasMore)
		assert.Equal(t, "2026-09-25T12:00:00Z", page.Until)
	})
}
