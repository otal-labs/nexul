package t3rpc

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

func testCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func connectFake(t *testing.T, f *t3rpctest.Server) *Conn {
	c, err := Connect(testCtx(t), f.Session(), Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestConnect_HandshakeMintsTicketAndCallsGetConfig(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	connectFake(t, f)
	assert.Equal(t, int32(1), f.ConfigCalls.Load())
}

func TestConnect_AcceptsAlternateTicketFieldName(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	f.TicketField = "wsTicket"
	connectFake(t, f)
	assert.Equal(t, int32(1), f.ConfigCalls.Load())
}

func TestConnect_RejectedBearerIsUnauthorized(t *testing.T) {
	t.Parallel()
	f := t3rpctest.New(t)
	computer := f.Session()
	computer.BearerToken = "wrong"
	_, err := Connect(testCtx(t), computer, Options{HTTPClient: f.Client()})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestProtocol_DescriptorAndHandshakeAgree(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		served   int    // 0 leaves the field out, as T3 before protocol negotiation does
		dialWith string // the orchestrationProtocol query value, empty for none
		want     int
		refused  bool
	}{
		{"absent is protocol 1", 0, "", 1, false},
		{"protocol 1", 1, "", 1, false},
		{"protocol 2", 2, "2", 2, false},
		{"protocol 2 refuses a dial without its query", 2, "", 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			f.Protocol = tt.served

			d, err := Describe(testCtx(t), f.Client(), f.URL)
			require.NoError(t, err)
			assert.Equal(t, Descriptor{ServerVersion: "0.0.34", Protocol: tt.want}, d)

			opts := Options{HTTPClient: f.Client()}
			if tt.dialWith != "" {
				opts.Query = url.Values{"orchestrationProtocol": {tt.dialWith}}
			}
			c, err := Connect(testCtx(t), f.Session(), opts)
			if tt.refused {
				var mismatch *ProtocolMismatchError
				require.ErrorAs(t, err, &mismatch)
				assert.Equal(t, ProtocolMismatchError{Version: tt.want}, *mismatch)
				assert.NotErrorIs(t, err, apperrs.ErrRetryable, "redialing cannot change the protocol T3 speaks")
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() { _ = c.Close() })
			assert.Equal(t, tt.want, c.Protocol())
		})
	}
}

// T3 holds each stream's next chunk until the client acks the last one, so a Next that stops acking stalls the stream.
func TestStream_AcksEachChunkBeforeWaitingAndNamesHowItEnded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		end  func(f *t3rpctest.Server, id string)
		want error
	}{
		{"clean exit", func(f *t3rpctest.Server, id string) { f.Write(t3rpctest.ExitSuccess(id, nil)) }, io.EOF},
		{"failed exit", func(f *t3rpctest.Server, id string) {
			f.Write(map[string]any{"_tag": "Exit", "requestId": id, "exit": map[string]any{"_tag": "Failure", "cause": []any{map[string]any{"_tag": "Die"}}}})
		}, apperrs.ErrInvalid},
		{"dropped connection", func(f *t3rpctest.Server, _ string) { f.Drop() }, ErrConnectionLost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			c := connectFake(t, f)
			stream, err := c.Stream(testCtx(t), "orchestration.subscribeThread", map[string]any{"threadId": "th-1"})
			require.NoError(t, err)
			t.Cleanup(stream.Close)
			id := t3rpctest.WaitFor(t, f.Subscribed, "subscription")

			ctx := testCtx(t)
			f.Write(t3rpctest.Chunk(id, "one", "two"))
			values, err := stream.Next(ctx)
			require.NoError(t, err)
			assert.Equal(t, []json.RawMessage{json.RawMessage(`"one"`), json.RawMessage(`"two"`)}, values)

			go func() {
				select {
				case <-f.Acks:
					tt.end(f, id)
				case <-ctx.Done():
				}
			}()
			_, err = stream.Next(ctx)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

// A missing thread and a stream T3 gave up on both fail the subscription; only the cause tells them apart.
func TestStream_FailedExit_NamesItsCauses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cause     any
		wantTags  []string
		missing   bool
		wantInErr string
	}{
		{"a typed failure", []any{map[string]any{"_tag": "Fail", "error": map[string]any{
			"_tag": "OrchestrationV2GetThreadProjectionError", "threadId": "th-1", "message": "Failed to load orchestration V2 thread th-1"}}},
			[]string{"Fail"}, true, "Failed to load orchestration V2 thread th-1"},
		{"a defect, as LiveStreamBufferError arrives", []any{map[string]any{"_tag": "Die", "defect": "LiveStreamBufferError"}},
			[]string{"Die"}, false, "LiveStreamBufferError"},
		{"an undecodable cause still fails", "not a cause", nil, false, "not a cause"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			c := connectFake(t, f)
			stream, err := c.Stream(testCtx(t), "orchestration.subscribeThread", map[string]any{"threadId": "th-1"})
			require.NoError(t, err)
			t.Cleanup(stream.Close)
			id := t3rpctest.WaitFor(t, f.Subscribed, "subscription")
			f.Write(map[string]any{"_tag": "Exit", "requestId": id, "exit": map[string]any{"_tag": "Failure", "cause": tt.cause}})

			_, err = stream.Next(testCtx(t))
			var exit *ExitError
			require.ErrorAs(t, err, &exit)
			assert.ErrorIs(t, err, apperrs.ErrInvalid)
			var tags []string
			for _, c := range exit.Causes {
				tags = append(tags, c.Tag)
			}
			assert.Equal(t, tt.wantTags, tags)
			assert.Equal(t, tt.missing, exit.Failed("OrchestrationV2GetThreadProjectionError"))
			assert.Contains(t, err.Error(), tt.wantInErr)
		})
	}
}

// A tunnel cuts a socket that stays silent too long without a close frame, so a session pings to stay up through a
// long quiet turn, and a ping left unanswered ends it.
func TestConn_Keepalive_PingsThroughAnIdleCutAndEndsWhenUnanswered(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		ignorePings bool
		wantErr     string
	}{
		{"a tunnel that cuts idle sockets keeps a pinging session", false, ""},
		{"a server that stops answering ends the session", true, "stopped answering pings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			f.IdleDrop = 500 * time.Millisecond
			f.IgnorePings = tt.ignorePings
			c, err := Connect(testCtx(t), f.Session(), Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second, PingEvery: 50 * time.Millisecond})
			require.NoError(t, err)
			t.Cleanup(func() { _ = c.Close() })

			if tt.wantErr != "" {
				t3rpctest.WaitFor(t, c.Done(), "the session to end")
				assert.ErrorContains(t, c.Err(), tt.wantErr)
				return
			}
			for range 12 {
				t3rpctest.WaitFor(t, f.Pings, "a ping")
			}
			_, err = c.Call(testCtx(t), "server.getConfig", nil)
			require.NoError(t, err, "still up after more than its idle cut")
		})
	}
}
