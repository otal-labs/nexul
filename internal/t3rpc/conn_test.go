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
