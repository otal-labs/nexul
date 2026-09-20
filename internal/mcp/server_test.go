package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// newTestServer builds a Server with hand-rolled tools/resources/prompts so
// dispatch and transport tests exercise the adapter, not the registry.
func newTestServer() *Server {
	s := &Server{}
	s.tools = []Tool{
		{
			Name:        "echo",
			Description: "echo a string back",
			InputSchema: objectSchema(map[string]any{"value": map[string]any{"type": "string"}}, "value"),
			Call: func(_ context.Context, args map[string]any) (any, error) {
				return args["value"], nil
			},
		},
		{
			Name:        "boom",
			Description: "always returns not found",
			InputSchema: objectSchema(map[string]any{}),
			Call: func(_ context.Context, _ map[string]any) (any, error) {
				return nil, apperrs.ErrNotFound
			},
		},
		{
			Name:        "invalid",
			Description: "always returns invalid",
			InputSchema: objectSchema(map[string]any{}),
			Call: func(_ context.Context, _ map[string]any) (any, error) {
				return nil, apperrs.ErrInvalid
			},
		},
	}
	s.resources = []Resource{
		{URI: "static://thing", Name: "static", MIMEType: "text/plain", Read: func(context.Context) (string, error) { return "hi", nil }},
	}
	s.templates = []ResourceTemplate{
		{URITemplate: "docs://{id}", Name: "doc", MIMEType: "text/markdown", Read: func(_ context.Context, vars map[string]string) (string, error) {
			return "doc:" + vars["id"], nil
		}},
	}
	s.prompts = []Prompt{
		{
			Name:        "greet",
			Description: "greet someone",
			Arguments:   []PromptArgument{{Name: "name", Description: "who", Required: true}},
			Messages: func(args map[string]string) []PromptMessage {
				return []PromptMessage{{Role: "user", Content: interpolate("hi {{name}}", args)}}
			},
		},
	}
	return s
}

func dispatch(t *testing.T, s *Server, id int, method string, params any) *Response {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		require.NoError(t, err)
		raw = b
	}
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(fmt.Sprintf("%d", id)), Method: method, Params: raw}
	return s.Dispatch(context.Background(), req)
}

// decodeResult unmarshals a response Result into a typed struct after a
// JSON round-trip (transports) or for consistency in direct dispatch tests.
func decodeResult[T any](t *testing.T, resp *Response) T {
	t.Helper()
	b, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

func TestDispatch_Initialize(t *testing.T) {
	resp := dispatch(t, newTestServer(), 1, MethodInitialize, map[string]any{"protocolVersion": "2025-03-26"})
	require.Nil(t, resp.Error)
	assert.Equal(t, "1", string(resp.ID))
	result := decodeResult[initializeResult](t, resp)
	assert.Equal(t, ProtocolVersion, result.ProtocolVersion)
	assert.Equal(t, ServerName, result.ServerInfo["name"])
	assert.Contains(t, result.Capabilities, "tools")
	assert.Contains(t, result.Capabilities, "resources")
	assert.Contains(t, result.Capabilities, "prompts")
}

func TestDispatch_Ping(t *testing.T) {
	resp := dispatch(t, newTestServer(), 2, MethodPing, nil)
	require.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestDispatch_ToolsList(t *testing.T) {
	resp := dispatch(t, newTestServer(), 3, MethodToolsList, nil)
	require.Nil(t, resp.Error)
	result := decodeResult[toolsListResult](t, resp)
	require.Len(t, result.Tools, 3)
	assert.Equal(t, "echo", result.Tools[0].Name)
	assert.NotEmpty(t, result.Tools[0].Description)
	assert.NotNil(t, result.Tools[0].InputSchema)
	assert.Equal(t, "complete", result.ResultType)
	assert.Equal(t, int64(3600000), result.TTLMs)
	assert.Equal(t, "private", result.CacheScope)
}

func TestDispatch_ServerDiscover(t *testing.T) {
	resp := dispatch(t, newTestServer(), 3, MethodServerDiscover, nil)
	require.Nil(t, resp.Error)
	result := decodeResult[discoverResult](t, resp)
	assert.Equal(t, "complete", result.ResultType)
	assert.Equal(t, SupportedVersions, result.SupportedVersions)
	assert.Contains(t, result.Capabilities, "tools")
	assert.Contains(t, result.Capabilities, "resources")
	assert.Contains(t, result.Capabilities, "prompts")
	serverInfo, ok := result.Meta[MetaKeyServerInfo].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, ServerName, serverInfo["name"])
}

func TestDispatch_ProtocolVersionMeta(t *testing.T) {
	t.Run("supported version succeeds", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 3, MethodToolsList, map[string]any{
			"_meta": map[string]any{MetaKeyProtocolVersion: ProtocolVersion2026},
		})
		require.Nil(t, resp.Error)
	})
	t.Run("unsupported version is rejected", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 3, MethodToolsList, map[string]any{
			"_meta": map[string]any{MetaKeyProtocolVersion: "1999-01-01"},
		})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeUnsupportedProtocolVersion, resp.Error.Code)
		data, ok := resp.Error.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "1999-01-01", data["requested"])
		assert.ElementsMatch(t, SupportedVersions, data["supported"])
	})
}

func TestDispatch_ToolsCall(t *testing.T) {
	t.Run("happy path returns content", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 4, MethodToolsCall, map[string]any{"name": "echo", "arguments": map[string]any{"value": "hello"}})
		require.Nil(t, resp.Error)
		result := decodeResult[toolCallResult](t, resp)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "hello", result.Content[0].Text)
	})
	t.Run("no arguments becomes empty map", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 4, MethodToolsCall, map[string]any{"name": "echo"})
		require.Nil(t, resp.Error)
		result := decodeResult[toolCallResult](t, resp)
		assert.Equal(t, "null", result.Content[0].Text)
	})
	t.Run("unknown tool is invalid params", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 4, MethodToolsCall, map[string]any{"name": "nope"})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidParams, resp.Error.Code)
	})
	t.Run("not found maps to -32002", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 4, MethodToolsCall, map[string]any{"name": "boom"})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeNotFound, resp.Error.Code)
	})
	t.Run("invalid maps to -32602", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 4, MethodToolsCall, map[string]any{"name": "invalid"})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidParams, resp.Error.Code)
	})
	t.Run("malformed params is invalid params", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 4, MethodToolsCall, map[string]any{"name": 42})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidParams, resp.Error.Code)
	})
	t.Run("result that is not a string is json text", func(t *testing.T) {
		s := &Server{}
		s.tools = []Tool{{
			Name: "obj", Call: func(context.Context, map[string]any) (any, error) {
				return map[string]int{"n": 1}, nil
			},
		}}
		resp := dispatch(t, s, 4, MethodToolsCall, map[string]any{"name": "obj"})
		require.Nil(t, resp.Error)
		result := decodeResult[toolCallResult](t, resp)
		assert.JSONEq(t, `{"n":1}`, result.Content[0].Text)
	})
}

func TestDispatch_ResourcesList(t *testing.T) {
	resp := dispatch(t, newTestServer(), 5, MethodResourcesList, nil)
	require.Nil(t, resp.Error)
	result := decodeResult[resourcesListResult](t, resp)
	require.Len(t, result.Resources, 1)
	assert.Equal(t, "static://thing", result.Resources[0].URI)
	require.Len(t, result.ResourceTemplates, 1)
	assert.Equal(t, "docs://{id}", result.ResourceTemplates[0].URITemplate)
}

func TestDispatch_ResourcesRead(t *testing.T) {
	t.Run("static resource", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 6, MethodResourcesRead, map[string]any{"uri": "static://thing"})
		require.Nil(t, resp.Error)
		result := decodeResult[resourceReadResult](t, resp)
		assert.Equal(t, "hi", result.Contents[0].Text)
	})
	t.Run("template resource", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 6, MethodResourcesRead, map[string]any{"uri": "docs://abc"})
		require.Nil(t, resp.Error)
		result := decodeResult[resourceReadResult](t, resp)
		assert.Equal(t, "doc:abc", result.Contents[0].Text)
		assert.Equal(t, "docs://abc", result.Contents[0].URI)
	})
	t.Run("unknown uri is invalid params", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 6, MethodResourcesRead, map[string]any{"uri": "nope://x"})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidParams, resp.Error.Code)
	})
	t.Run("missing uri is invalid params", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 6, MethodResourcesRead, map[string]any{})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidParams, resp.Error.Code)
	})
}

func TestDispatch_PromptsList(t *testing.T) {
	resp := dispatch(t, newTestServer(), 7, MethodPromptsList, nil)
	require.Nil(t, resp.Error)
	result := decodeResult[promptsListResult](t, resp)
	require.Len(t, result.Prompts, 1)
	assert.Equal(t, "greet", result.Prompts[0].Name)
	assert.True(t, result.Prompts[0].Arguments[0].Required)
}

func TestDispatch_PromptsGet(t *testing.T) {
	t.Run("renders with arguments", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 8, MethodPromptsGet, map[string]any{"name": "greet", "arguments": map[string]string{"name": "alice"}})
		require.Nil(t, resp.Error)
		result := decodeResult[promptGetResult](t, resp)
		require.Len(t, result.Messages, 1)
		assert.Equal(t, "user", result.Messages[0].Role)
		assert.Equal(t, "hi alice", result.Messages[0].Content.Text)
	})
	t.Run("unknown prompt is invalid params", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 8, MethodPromptsGet, map[string]any{"name": "nope"})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidParams, resp.Error.Code)
	})
	t.Run("non-string argument value is invalid params", func(t *testing.T) {
		resp := dispatch(t, newTestServer(), 8, MethodPromptsGet, map[string]any{"name": "greet", "arguments": map[string]any{"name": 1}})
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidParams, resp.Error.Code)
	})
}

func TestDispatch_Notifications(t *testing.T) {
	t.Run("initialized notification yields no response", func(t *testing.T) {
		req := &Request{JSONRPC: "2.0", Method: MethodNotifyInitialized, Params: json.RawMessage(`{}`)}
		assert.Nil(t, newTestServer().Dispatch(context.Background(), req))
	})
	t.Run("cancelled notification yields no response", func(t *testing.T) {
		req := &Request{JSONRPC: "2.0", Method: MethodNotifyCancelled, Params: json.RawMessage(`{}`)}
		assert.Nil(t, newTestServer().Dispatch(context.Background(), req))
	})
	t.Run("request method with no id is treated as a notification", func(t *testing.T) {
		req := &Request{JSONRPC: "2.0", Method: MethodToolsList}
		assert.Nil(t, newTestServer().Dispatch(context.Background(), req))
	})
}

func TestDispatch_UnknownMethod(t *testing.T) {
	resp := dispatch(t, newTestServer(), 9, "wat/not/a/method", nil)
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
}

func TestDispatch_InvalidRequest(t *testing.T) {
	t.Run("bad jsonrpc version", func(t *testing.T) {
		req := &Request{JSONRPC: "1.0", ID: json.RawMessage("7"), Method: MethodPing}
		resp := newTestServer().Dispatch(context.Background(), req)
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeInvalidRequest, resp.Error.Code)
		assert.Equal(t, "7", string(resp.ID))
	})
	t.Run("nil request", func(t *testing.T) {
		assert.Nil(t, newTestServer().Dispatch(context.Background(), nil))
	})
}
