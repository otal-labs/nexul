package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeStdio_RoundTrip(t *testing.T) {
	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n")
	in.WriteString(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"value":"stdio"}}}` + "\n")

	err := newTestServer().ServeStdio(context.Background(), &in, &out)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 2)

	var init Response
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &init))
	assert.Nil(t, init.Error)
	assert.Equal(t, "1", string(init.ID))

	var call Response
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &call))
	assert.Nil(t, call.Error)
	result := decodeResult[toolCallResult](t, &call)
	assert.Equal(t, "stdio", result.Content[0].Text)
}

func TestServeStdio_NotificationProducedNoOutput(t *testing.T) {
	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")

	err := newTestServer().ServeStdio(context.Background(), &in, &out)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 1)
	var resp Response
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &resp))
	assert.Equal(t, "1", string(resp.ID))
}

func TestServeStdio_ParseError(t *testing.T) {
	var in, out bytes.Buffer
	in.WriteString(`{this is not json` + "\n")

	err := newTestServer().ServeStdio(context.Background(), &in, &out)
	require.NoError(t, err)
	var resp Response
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeParseError, resp.Error.Code)
}

func TestServeStdio_CancelledContextStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	err := newTestServer().ServeStdio(ctx, &in, &out)
	require.NoError(t, err)
	assert.Empty(t, out.String())
}
