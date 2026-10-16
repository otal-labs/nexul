package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logstore"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type fakeLogReader struct {
	got  []logstore.Query
	page logstore.Page
	err  error
}

func (f *fakeLogReader) Read(_ context.Context, q logstore.Query) (logstore.Page, error) {
	f.got = append(f.got, q)
	return f.page, f.err
}

// instanceReaderGate holds instance:read and everything else except logs:read, as a role that has the instance
// area but was never handed the Logs one.
type instanceReaderGate struct{ ownerGate }

func (g instanceReaderGate) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	if action == permissions.LogsRead {
		return fmt.Errorf("%w: logs:read required", apperrs.ErrForbidden)
	}
	return g.ownerGate.RequireAnywhere(ctx, action)
}

var logsNow = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

func logsService(t *testing.T, reader LogReader) *Service {
	t.Helper()
	withVersion(t, "dev")
	s := newUpgradeService(fakeGitHub(t, "v0.2.0", "x").URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{}).WithLogs(reader)
	s.now = func() time.Time { return logsNow }
	return s
}

func TestInstanceLogs_Refusals(t *testing.T) {
	failing := &fakeLogReader{err: fmt.Errorf("%w: the log store (OpenObserve) did not answer; retry in a moment", apperrs.ErrConflict)}
	tests := []struct {
		name   string
		ctx    context.Context
		reader LogReader
		gate   Gate
		params logstore.Params
		want   error
		msg    string
	}{
		{"someone without logs:read", asMember(), &fakeLogReader{}, nil, logstore.Params{}, apperrs.ErrForbidden, "logs:read"},
		{"a holder of the instance but not the Logs area", asAdmin(), &fakeLogReader{}, instanceReaderGate{}, logstore.Params{}, apperrs.ErrForbidden, "logs:read"},
		{"an instance that keeps no logs", asAdmin(), nil, nil, logstore.Params{}, apperrs.ErrInvalid, "this instance keeps no logs"},
		{"a bad filter", asAdmin(), &fakeLogReader{}, nil, logstore.Params{Source: "database"}, apperrs.ErrInvalid, "source must be one of"},
		{"a store that does not answer is not an empty list", asAdmin(), failing, nil, logstore.Params{}, apperrs.ErrConflict, "retry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := logsService(t, tt.reader)
			if tt.reader == nil {
				s.WithLogs(nil)
			}
			if tt.gate != nil {
				s.WithGate(tt.gate)
			}
			_, err := s.InstanceLogs(tt.ctx, tt.params)
			require.ErrorIs(t, err, tt.want)
			assert.Contains(t, err.Error(), tt.msg)
			if f, ok := tt.reader.(*fakeLogReader); ok && tt.want != apperrs.ErrConflict {
				assert.Empty(t, f.got, "nothing was read")
			}
		})
	}
}

func TestInstanceLogs_PinsUntilToNow(t *testing.T) {
	reader := &fakeLogReader{page: logstore.Page{Items: []logstore.Line{{Message: "up"}}, Total: 1, Until: logsNow}}
	s := logsService(t, reader)

	page, err := s.InstanceLogs(asAdmin(), logstore.Params{Level: "warn", Limit: 5})
	require.NoError(t, err)
	assert.Equal(t, reader.page, page)
	require.Len(t, reader.got, 1)
	assert.Equal(t, logsNow, reader.got[0].Until)
	assert.Equal(t, "warn", reader.got[0].Level)
	assert.Equal(t, 5, reader.got[0].Window.Limit)
}

func TestInstanceGet_Logs(t *testing.T) {
	reader := &fakeLogReader{page: logstore.Page{Items: []logstore.Line{{Level: "error", Source: "server", Message: "boom"}}, Total: 1, Until: logsNow}}

	t.Run("adds the logs beside the upgrade facts", func(t *testing.T) {
		got, err := callTool(asAdmin(), t, logsService(t, reader), "instance_get", `{"logs":{"level":"error","text":"boom"}}`)
		require.NoError(t, err)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		var out struct {
			Version string `json:"version"`
			Logs    struct {
				Items []struct {
					Message string `json:"message"`
				} `json:"items"`
				Total   int    `json:"total"`
				HasMore bool   `json:"has_more"`
				Until   string `json:"until"`
			} `json:"logs"`
		}
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.Equal(t, "boom", out.Logs.Items[0].Message)
		assert.Equal(t, 1, out.Logs.Total)
		assert.Equal(t, "2026-09-25T12:00:00Z", out.Logs.Until)
		assert.Equal(t, "boom", reader.got[0].Text)
	})

	t.Run("an empty logs object reads the defaults", func(t *testing.T) {
		before := len(reader.got)
		_, err := callTool(asAdmin(), t, logsService(t, reader), "instance_get", `{"logs":{}}`)
		require.NoError(t, err)
		require.Len(t, reader.got, before+1)
		assert.Equal(t, "info", reader.got[before].Level)
	})

	t.Run("without logs the answer has none and the store is not asked", func(t *testing.T) {
		before := len(reader.got)
		got, err := callTool(asAdmin(), t, logsService(t, reader), "instance_get", `{}`)
		require.NoError(t, err)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), `"logs"`)
		assert.Len(t, reader.got, before)
	})

	t.Run("logs needs logs:read besides instance:read", func(t *testing.T) {
		s := logsService(t, reader).WithGate(instanceReaderGate{})
		_, err := callTool(asAdmin(), t, s, "instance_get", `{}`)
		require.NoError(t, err, "the upgrade facts still read")
		_, err = callTool(asAdmin(), t, s, "instance_get", `{"logs":{}}`)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("an unknown filter is refused", func(t *testing.T) {
		_, err := callTool(asAdmin(), t, logsService(t, reader), "instance_get", `{"logs":{"query":"select"}}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
}
