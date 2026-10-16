package containerlog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeFeed hands out its batches one per Next, then ends with end; closed records the viewer leaving.
type fakeFeed struct {
	batches [][]Line
	end     error
	closed  chan struct{}
}

func (f *fakeFeed) Next(ctx context.Context) ([]Line, error) {
	if len(f.batches) > 0 {
		out := f.batches[0]
		f.batches = f.batches[1:]
		return out, nil
	}
	if f.end != nil {
		return nil, f.end
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (f *fakeFeed) Close() { close(f.closed) }

func opener(feed *fakeFeed, err error) Open {
	return func(context.Context, int, bool) (Feed, error) {
		if err != nil {
			return nil, err
		}
		return feed, nil
	}
}

var line = Line{TS: "2026-10-11T10:00:00Z", Stream: "stdout", Line: "listening"}

func TestSnapshot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		feed    *fakeFeed
		openErr error
		want    []Line
		wantErr error
	}{
		{"reads until the stream ends", &fakeFeed{batches: [][]Line{{line}, {line}}, end: io.EOF}, nil, []Line{line, line}, nil},
		{"an empty log is an empty list", &fakeFeed{end: io.EOF}, nil, []Line{}, nil},
		{"a refusal to open passes through", nil, apperrs.ErrNotFound, nil, apperrs.ErrNotFound},
		{"a stream that fails says whose", &fakeFeed{end: apperrs.ErrConflict}, nil, nil, apperrs.ErrConflict},
		{"a runner that never answers is retryable", &fakeFeed{end: context.DeadlineExceeded}, nil, nil, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.feed != nil {
				tt.feed.closed = make(chan struct{})
			}
			got, err := Snapshot(t.Context(), opener(tt.feed, tt.openErr), 5, "shop-web-1")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			<-tt.feed.closed
		})
	}
}

func TestTail(t *testing.T) {
	t.Parallel()
	for raw, want := range map[string]int{"": DefaultTail, "?tail=5": 5} {
		got, err := Tail(httptest.NewRequest(http.MethodGet, "/logs"+raw, nil))
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := Tail(httptest.NewRequest(http.MethodGet, "/logs?tail=all", nil))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestCutForAgent_CutsOnlyALongLine(t *testing.T) {
	t.Parallel()
	lines := []Line{{Line: strings.Repeat("é", agentLineChars+1)}, {Line: "short"}}
	CutForAgent(lines)
	assert.Equal(t, strings.Repeat("é", agentLineChars)+"…", lines[0].Line)
	assert.Equal(t, "short", lines[1].Line)
}

func TestServeSnapshot(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	ServeSnapshot(rec, httptest.NewRequest(http.MethodGet, "/logs?tail=x", nil), opener(nil, nil), "web")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	ServeSnapshot(rec, httptest.NewRequest(http.MethodGet, "/logs", nil), opener(nil, apperrs.ErrNotFound), "web")
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = httptest.NewRecorder()
	ServeSnapshot(rec, httptest.NewRequest(http.MethodGet, "/logs", nil), opener(&fakeFeed{batches: [][]Line{{line}}, end: io.EOF, closed: make(chan struct{})}, nil), "web")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"lines":[{"ts":"2026-10-11T10:00:00Z","stream":"stdout","line":"listening"}]}`, rec.Body.String())
}

func TestServeSocket(t *testing.T) {
	t.Parallel()
	serve := func(t *testing.T, open Open) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ServeSocket(w, r, open) }))
		t.Cleanup(srv.Close)
		return "ws" + strings.TrimPrefix(srv.URL, "http")
	}

	t.Run("a refusal answers before the upgrade", func(t *testing.T) {
		t.Parallel()
		_, resp, err := websocket.Dial(t.Context(), serve(t, opener(nil, apperrs.ErrNotFound)), nil)
		require.Error(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		_, resp, err = websocket.Dial(t.Context(), serve(t, opener(nil, nil))+"?tail=x", nil)
		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	for name, tt := range map[string]struct {
		end  error
		code websocket.StatusCode
	}{
		"a stream that ends closes normally":     {io.EOF, websocket.StatusNormalClosure},
		"a stream that fails says why, to retry": {errors.New(strings.Repeat("the runner disconnected ", 10)), websocket.StatusTryAgainLater},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			feed := &fakeFeed{batches: [][]Line{{line}}, end: tt.end, closed: make(chan struct{})}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, serve(t, opener(feed, nil)), nil)
			require.NoError(t, err)
			var got Lines
			require.NoError(t, wsjson.Read(ctx, conn, &got))
			assert.Equal(t, []Line{line}, got.Lines)
			_, _, err = conn.Read(ctx)
			assert.Equal(t, tt.code, websocket.CloseStatus(err))
			<-feed.closed
		})
	}

	t.Run("a viewer that leaves closes the stream", func(t *testing.T) {
		t.Parallel()
		feed := &fakeFeed{closed: make(chan struct{})}
		conn, _, err := websocket.Dial(t.Context(), serve(t, opener(feed, nil)), nil)
		require.NoError(t, err)
		require.NoError(t, conn.Close(websocket.StatusNormalClosure, ""))
		<-feed.closed
	})
}

func TestEnvMask_HidesValuesOfSixCharactersOrMoreLongestFirst(t *testing.T) {
	mask := EnvMask(map[string]string{"TOKEN": "abcdef", "LONGER": "abcdef-and-more", "SHORT": "abcde", "EMPTY": ""})
	assert.Equal(t, "got •••• and •••• and abcde", mask.Replace("got abcdef-and-more and abcdef and abcde"))
	assert.Equal(t, "nothing here", EnvMask(nil).Replace("nothing here"))
}
