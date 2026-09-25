package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

const testInstanceURL = "https://nexul.example"

type whoamiIn struct {
	Fail string `json:"fail,omitempty" jsonschema:"invalid, internal, or empty."`
}

func fakeServer() server {
	return server{
		tools: []mcptool.Tool{
			mcptool.New("thing_get", "Get thing", "Returns the caller's id. Fails on request. Used by the adapter tests.",
				mcptool.Hints{ReadOnly: true, Local: true},
				func(ctx context.Context, in whoamiIn) (any, error) {
					switch in.Fail {
					case "invalid":
						return nil, fmt.Errorf("%w: status_id is not a status of project REF; project_get lists them", apperrs.ErrInvalid)
					case "internal":
						return nil, errors.New("sqlite: disk I/O error at /data/nexul.db")
					}
					a, _ := identity.ActorFromCtx(ctx)
					return map[string]string{"actor": a.ID}, nil
				}),
			mcptool.New("thing_delete", "Delete thing", "Deletes a thing. Used by the adapter tests. Always succeeds.",
				mcptool.Hints{Idempotent: true},
				func(context.Context, struct{}) (any, error) { return mcptool.Gone("t-1"), nil }),
		},
		resources: []resource{{
			uri: "things://{id}", name: "thing", title: "Thing", mimeType: "text/plain",
			description: "A thing, for the adapter tests.",
			read: func(_ context.Context, id string) (string, error) {
				if id != "t-1" {
					return "", apperrs.ErrNotFound
				}
				return "thing one", nil
			},
		}},
		prompts:      workflowPrompts(),
		instructions: instructions,
	}
}

// asCaller stands in for RequireAuth + withIdentity: every request acts as user-1.
func asCaller(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: "user-1"})))
	})
}

func newTestEndpoint(t *testing.T) *httptest.Server {
	t.Helper()
	h := fakeServer().handler(func(context.Context) (string, error) { return testInstanceURL, nil })
	srv := httptest.NewServer(asCaller(h))
	t.Cleanup(srv.Close)
	return srv
}

func connect(t *testing.T, url, revision string) *sdk.ClientSession {
	t.Helper()
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: url, DisableStandaloneSSE: true},
		&sdk.ClientSessionOptions{ProtocolVersion: revision})
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func textOf(t *testing.T, res *sdk.CallToolResult) string {
	t.Helper()
	require.Len(t, res.Content, 1)
	text, ok := res.Content[0].(*sdk.TextContent)
	require.True(t, ok)
	return text.Text
}

// Both eras must see the same surface: the modern stateless revision and the legacy handshake.
var revisions = []string{"2026-07-28", "2025-11-25"}

func TestAdapter_ToolFailuresAreResultsTheModelCanRead(t *testing.T) {
	for _, rev := range revisions {
		t.Run(rev, func(t *testing.T) {
			session := connect(t, newTestEndpoint(t).URL+"/mcp", rev)
			tests := []struct {
				name     string
				args     map[string]any
				contains string
				hides    string
			}{
				{"a domain error passes its message", map[string]any{"fail": "invalid"}, "project_get lists them", ""},
				{"an internal error is hidden behind a trace id", map[string]any{"fail": "internal"}, "internal error (trace ", "sqlite"},
				{"an argument the schema rejects", map[string]any{"fail": 7}, "invalid", ""},
				{"an unknown argument", map[string]any{"user_id": "u2"}, "invalid", ""},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					res, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "thing_get", Arguments: tt.args})
					require.NoError(t, err, "a tool failure is a result, not a protocol error")
					assert.True(t, res.IsError)
					text := textOf(t, res)
					assert.Contains(t, text, tt.contains)
					if tt.hides != "" {
						assert.NotContains(t, text, tt.hides)
					}
				})
			}
		})
	}
}

func TestAdapter_UnknownToolIsAProtocolError(t *testing.T) {
	session := connect(t, newTestEndpoint(t).URL+"/mcp", revisions[0])
	_, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "thing_explode"})
	require.Error(t, err)
}

func TestAdapter_ToolsRunAsTheCaller(t *testing.T) {
	session := connect(t, newTestEndpoint(t).URL+"/mcp", revisions[0])
	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "thing_get", Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.JSONEq(t, `{"actor":"user-1"}`, textOf(t, res))
}

func TestAdapter_ToolsCarryTitlesAndHints(t *testing.T) {
	session := connect(t, newTestEndpoint(t).URL+"/mcp", revisions[0])
	res, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	byName := map[string]*sdk.Tool{}
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
	}
	get, del := byName["thing_get"], byName["thing_delete"]
	require.NotNil(t, get)
	require.NotNil(t, del)
	assert.Equal(t, "Get thing", get.Title)
	assert.True(t, get.Annotations.ReadOnlyHint)
	assert.False(t, *get.Annotations.DestructiveHint, "a read is never destructive")
	assert.True(t, get.Annotations.IdempotentHint, "a read is safe to repeat")
	assert.False(t, *get.Annotations.OpenWorldHint, "a Local tool stays in Nexul's database")
	assert.False(t, del.Annotations.ReadOnlyHint)
	assert.True(t, *del.Annotations.DestructiveHint, "the zero Hints are the conservative default")
	assert.True(t, del.Annotations.IdempotentHint)
	assert.Equal(t, "public", res.CacheScope)
	assert.Positive(t, res.TTLMs)
}

func TestAdapter_AdvertisesOnlyWhatItImplements(t *testing.T) {
	for _, rev := range revisions {
		t.Run(rev, func(t *testing.T) {
			session := connect(t, newTestEndpoint(t).URL+"/mcp", rev)
			init := session.InitializeResult()
			require.NotNil(t, init)
			caps := init.Capabilities
			require.NotNil(t, caps.Tools)
			assert.False(t, caps.Tools.ListChanged)
			wire, err := json.Marshal(caps)
			require.NoError(t, err)
			assert.NotContains(t, string(wire), `"logging"`, "logging is deprecated and not implemented")
			assert.Nil(t, caps.Completions)
			assert.Equal(t, instructions, init.Instructions)
		})
	}
}

func TestAdapter_Resources(t *testing.T) {
	session := connect(t, newTestEndpoint(t).URL+"/mcp", revisions[0])

	res, err := session.ReadResource(t.Context(), &sdk.ReadResourceParams{URI: "things://t-1"})
	require.NoError(t, err)
	require.Len(t, res.Contents, 1)
	assert.Equal(t, "thing one", res.Contents[0].Text)
	assert.Equal(t, 0, res.TTLMs, "live content is never cached")
	assert.Equal(t, "private", res.CacheScope)

	_, err = session.ReadResource(t.Context(), &sdk.ReadResourceParams{URI: "things://missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Resource not found")

	templates, err := session.ListResourceTemplates(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, templates.ResourceTemplates, 1)
	assert.Equal(t, "things://{id}", templates.ResourceTemplates[0].URITemplate)
}

func TestAdapter_PromptsEnforceRequiredArguments(t *testing.T) {
	session := connect(t, newTestEndpoint(t).URL+"/mcp", revisions[0])
	_, err := session.GetPrompt(t.Context(), &sdk.GetPromptParams{Name: "create_ticket_from_doc"})
	require.Error(t, err)

	res, err := session.GetPrompt(t.Context(), &sdk.GetPromptParams{Name: "create_ticket_from_doc", Arguments: map[string]string{"doc_id": "d-1"}})
	require.NoError(t, err)
	require.Len(t, res.Messages, 1)
	text, ok := res.Messages[0].Content.(*sdk.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "d-1")
}

func TestAdapter_ShipRepositoryOnlyExposesWhenAHostnameIsGiven(t *testing.T) {
	var ship prompt
	for _, p := range workflowPrompts() {
		if p.name == "ship_repository" {
			ship = p
		}
	}
	args := map[string]string{"owner": "o", "repo": "r", "project_id": "p", "machine": "m"}
	assert.NotContains(t, ship.render(args), "exposure_create")
	args["hostname"] = "app.example.com"
	assert.Contains(t, ship.render(args), "app.example.com")
}

func TestAdapter_RefusesForeignOrigins(t *testing.T) {
	srv := newTestEndpoint(t)
	tests := []struct {
		name   string
		origin string
		want   int
	}{
		{"a foreign origin is refused", "https://evil.example", http.StatusForbidden},
		{"the instance's own origin passes", testInstanceURL, http.StatusOK},
		{"no origin (a non-browser client) passes", "", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/mcp", strings.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			_ = resp.Body.Close()
			assert.Equal(t, tt.want, resp.StatusCode)
		})
	}
}

func TestAdapter_OriginCheckWithoutAnInstanceURLRefusesBrowsers(t *testing.T) {
	assert.False(t, allowedOrigin(t.Context(), nil, testInstanceURL))
	assert.False(t, allowedOrigin(t.Context(), func(context.Context) (string, error) { return "", nil }, testInstanceURL))
	assert.False(t, allowedOrigin(t.Context(), func(context.Context) (string, error) { return "", assert.AnError }, testInstanceURL))
	assert.True(t, allowedOrigin(t.Context(), func(context.Context) (string, error) { return testInstanceURL + "/", nil }, testInstanceURL))
}

func TestAdapter_RateLimitsPerActor(t *testing.T) {
	l := &limiters{byActor: map[string]*rate.Limiter{}}
	for range callBurst {
		require.Zero(t, l.wait("user-1"))
	}
	assert.Positive(t, l.wait("user-1"), "past the burst, the caller waits")
	assert.Zero(t, l.wait("user-2"), "each actor has its own budget")
}

func TestToolErrorMessage_PartialUpdatesSayWhatApplied(t *testing.T) {
	internalFailure := &mcptool.PartialError{Applied: []string{"title", "status_id"}, Err: errors.New("sqlite: locked")}
	message, internal := toolErrorMessage(internalFailure, "trace-1")
	assert.True(t, internal)
	assert.Equal(t, "internal error (trace trace-1); already applied: title, status_id", message)

	domainFailure := &mcptool.PartialError{Applied: []string{"title"}, Err: fmt.Errorf("type_id: %w: no such type", apperrs.ErrInvalid)}
	message, internal = toolErrorMessage(domainFailure, "trace-2")
	assert.False(t, internal)
	assert.Equal(t, "type_id: invalid: no such type; already applied: title", message)
}

func TestResultText(t *testing.T) {
	text, err := resultText("already text")
	require.NoError(t, err)
	assert.Equal(t, "already text", text)

	text, err = resultText(mcptool.Gone("x"))
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"x","deleted":true}`, text)

	_, err = resultText(map[string]any{"bad": make(chan int)})
	require.Error(t, err)
	var syntax *json.UnsupportedTypeError
	assert.ErrorAs(t, err, &syntax)
}
