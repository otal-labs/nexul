package mcp

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Server holds the tool/resource/prompt registries and dispatches JSON-RPC; transport-agnostic.
type Server struct {
	tools     []Tool
	resources []Resource
	templates []ResourceTemplate
	prompts   []Prompt
	actor     func(context.Context) identity.Actor
}

// Dispatch handles one JSON-RPC message, returning nil for a notification (no ID), which never gets a response.
func (s *Server) Dispatch(ctx context.Context, req *Request) *Response {
	if req == nil || req.JSONRPC != "2.0" {
		if req == nil || len(req.ID) == 0 {
			return nil
		}
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: CodeInvalidRequest, Message: "invalid request"}}
	}
	isNotification := len(req.ID) == 0 || bytes.Equal(bytes.TrimSpace(req.ID), []byte("null"))
	resp := s.handle(ctx, req)
	if resp == nil || isNotification {
		return nil
	}
	resp.ID = req.ID
	return resp
}

func (s *Server) handle(ctx context.Context, req *Request) *Response {
	if v, ok := requestProtocolVersion(req.Params); ok && !supportsVersion(v) {
		return unsupportedVersionResponse(v)
	}
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize()
	case MethodPing:
		return &Response{JSONRPC: "2.0", Result: map[string]any{}}
	case MethodServerDiscover:
		return s.handleServerDiscover()
	case MethodToolsList:
		return s.handleToolsList()
	case MethodToolsCall:
		return s.handleToolsCall(ctx, req.Params)
	case MethodResourcesList:
		return s.handleResourcesList()
	case MethodResourcesRead:
		return s.handleResourcesRead(ctx, req.Params)
	case MethodPromptsList:
		return s.handlePromptsList()
	case MethodPromptsGet:
		return s.handlePromptsGet(ctx, req.Params)
	case MethodNotifyInitialized, MethodNotifyCancelled:
		return nil
	default:
		return errResponse(CodeMethodNotFound, "method not found: %s", req.Method)
	}
}

func (s *Server) handleInitialize() *Response {
	return &Response{
		JSONRPC: "2.0",
		Result: initializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities:    serverCapabilities(),
			ServerInfo:      map[string]string{"name": ServerName, "version": ServerVersion},
		},
	}
}

type initializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]any    `json:"capabilities"`
	ServerInfo      map[string]string `json:"serverInfo"`
}

// requestProtocolVersion reads params._meta's protocol version; absent (legacy clients) is not an error.
func requestProtocolVersion(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var p struct {
		Meta map[string]any `json:"_meta"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", false
	}
	v, ok := p.Meta[MetaKeyProtocolVersion].(string)
	return v, ok
}

func supportsVersion(v string) bool {
	for _, sv := range SupportedVersions {
		if sv == v {
			return true
		}
	}
	return false
}

func unsupportedVersionResponse(requested string) *Response {
	return &Response{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    CodeUnsupportedProtocolVersion,
			Message: "Unsupported protocol version",
			Data:    map[string]any{"supported": SupportedVersions, "requested": requested},
		},
	}
}

// serverCapabilities is shared by initialize and server/discover; both describe the same server.
func serverCapabilities() map[string]any {
	return map[string]any{
		"tools":     map[string]any{"listChanged": false},
		"resources": map[string]any{},
		"prompts":   map[string]any{"listChanged": false},
	}
}

// resultMeta carries resultType and serverInfo on every result; legacy clients ignore the extra fields.
type resultMeta struct {
	ResultType string         `json:"resultType"`
	Meta       map[string]any `json:"_meta"`
}

func newResultMeta() resultMeta {
	return resultMeta{
		ResultType: "complete",
		Meta:       map[string]any{MetaKeyServerInfo: map[string]string{"name": ServerName, "version": ServerVersion}},
	}
}

// CacheableResult is embedded in list/read results; registries are static per process, so a long TTL is always safe.
type CacheableResult struct {
	TTLMs      int64  `json:"ttlMs"`
	CacheScope string `json:"cacheScope"`
}

func newCacheableResult() CacheableResult {
	return CacheableResult{TTLMs: 3600000, CacheScope: "private"}
}

func (s *Server) handleServerDiscover() *Response {
	return &Response{
		JSONRPC: "2.0",
		Result: discoverResult{
			resultMeta:        newResultMeta(),
			SupportedVersions: SupportedVersions,
			Capabilities:      serverCapabilities(),
		},
	}
}

type discoverResult struct {
	resultMeta
	SupportedVersions []string       `json:"supportedVersions"`
	Capabilities      map[string]any `json:"capabilities"`
}

func (s *Server) handleToolsList() *Response {
	out := make([]toolDef, 0, len(s.tools))
	for _, t := range s.tools {
		out = append(out, toolDefOf(t))
	}
	return &Response{JSONRPC: "2.0", Result: toolsListResult{Tools: out, resultMeta: newResultMeta(), CacheableResult: newCacheableResult()}}
}

type toolsListResult struct {
	Tools []toolDef `json:"tools"`
	resultMeta
	CacheableResult
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type toolCallResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
	resultMeta
}

func (s *Server) handleToolsCall(ctx context.Context, raw json.RawMessage) *Response {
	var p toolCallParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return errResponse(CodeInvalidParams, "invalid tools/call params: %v", err)
		}
	}
	t, ok := s.findTool(p.Name)
	if !ok {
		return errResponse(CodeInvalidParams, "unknown tool %q", p.Name)
	}
	if p.Arguments == nil {
		p.Arguments = map[string]any{}
	}
	res, err := t.Call(s.actorCtx(ctx), p.Arguments)
	if err != nil {
		return &Response{JSONRPC: "2.0", Error: mapError(err)}
	}
	return &Response{
		JSONRPC: "2.0",
		Result:  toolCallResult{Content: []contentBlock{{Type: "text", Text: textify(res)}}, resultMeta: newResultMeta()},
	}
}

// actorCtx attaches the acting user when the server has an actor resolver, so permission checks see who's calling.
func (s *Server) actorCtx(ctx context.Context) context.Context {
	if s.actor == nil {
		return ctx
	}
	return identity.WithActor(ctx, s.actor(ctx))
}

type resourcesListResult struct {
	Resources         []resourceDef         `json:"resources"`
	ResourceTemplates []resourceTemplateDef `json:"resourceTemplates"`
	resultMeta
	CacheableResult
}

func (s *Server) handleResourcesList() *Response {
	resources := make([]resourceDef, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, resourceDef{URI: r.URI, Name: r.Name, Description: r.Description, MIMEType: r.MIMEType})
	}
	templates := make([]resourceTemplateDef, 0, len(s.templates))
	for _, t := range s.templates {
		templates = append(templates, resourceTemplateDef{URITemplate: t.URITemplate, Name: t.Name, Description: t.Description, MIMEType: t.MIMEType})
	}
	return &Response{JSONRPC: "2.0", Result: resourcesListResult{
		Resources:         resources,
		ResourceTemplates: templates,
		resultMeta:        newResultMeta(),
		CacheableResult:   newCacheableResult(),
	}}
}

func (s *Server) handleResourcesRead(ctx context.Context, raw json.RawMessage) *Response {
	var p struct {
		URI string `json:"uri"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return errResponse(CodeInvalidParams, "invalid resources/read params: %v", err)
		}
	}
	if p.URI == "" {
		return errResponse(CodeInvalidParams, "uri is required")
	}
	for _, r := range s.resources {
		if r.URI == p.URI {
			text, err := r.Read(s.actorCtx(ctx))
			return s.resourceResult(p.URI, r.MIMEType, text, err)
		}
	}
	for _, t := range s.templates {
		if vars, ok := matchTemplate(p.URI, t); ok {
			text, err := t.Read(s.actorCtx(ctx), vars)
			return s.resourceResult(p.URI, t.MIMEType, text, err)
		}
	}
	// Invalid Params, not Not Found: aligns the unknown-resource case with JSON-RPC's own error taxonomy.
	return errResponse(CodeInvalidParams, "unknown resource %q", p.URI)
}

func (s *Server) resourceResult(uri, mimeType, text string, err error) *Response {
	if err != nil {
		return &Response{JSONRPC: "2.0", Error: mapError(err)}
	}
	return &Response{
		JSONRPC: "2.0",
		Result: resourceReadResult{
			Contents:        []resourceContent{{URI: uri, MIMEType: mimeType, Text: text}},
			resultMeta:      newResultMeta(),
			CacheableResult: newCacheableResult(),
		},
	}
}

type resourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType"`
	Text     string `json:"text"`
}

type resourceReadResult struct {
	Contents []resourceContent `json:"contents"`
	resultMeta
	CacheableResult
}

type promptsListResult struct {
	Prompts []promptDef `json:"prompts"`
	resultMeta
	CacheableResult
}

func (s *Server) handlePromptsList() *Response {
	out := make([]promptDef, 0, len(s.prompts))
	for _, p := range s.prompts {
		out = append(out, p.def())
	}
	return &Response{JSONRPC: "2.0", Result: promptsListResult{Prompts: out, resultMeta: newResultMeta(), CacheableResult: newCacheableResult()}}
}

type promptGetResult struct {
	Description string             `json:"description"`
	Messages    []promptMessageDef `json:"messages"`
	resultMeta
}

func (s *Server) handlePromptsGet(_ context.Context, raw json.RawMessage) *Response {
	var p struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return errResponse(CodeInvalidParams, "invalid prompts/get params: %v", err)
		}
	}
	prompt, ok := s.findPrompt(p.Name)
	if !ok {
		return errResponse(CodeInvalidParams, "unknown prompt %q", p.Name)
	}
	if p.Arguments == nil {
		p.Arguments = map[string]string{}
	}
	msgs := prompt.Messages(p.Arguments)
	out := make([]promptMessageDef, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, promptMessageDef{Role: m.Role, Content: contentBlock{Type: "text", Text: m.Content}})
	}
	return &Response{JSONRPC: "2.0", Result: promptGetResult{Description: prompt.Description, Messages: out, resultMeta: newResultMeta()}}
}

func (s *Server) findTool(name string) (Tool, bool) {
	for _, t := range s.tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

func (s *Server) findPrompt(name string) (Prompt, bool) {
	for _, p := range s.prompts {
		if p.Name == name {
			return p, true
		}
	}
	return Prompt{}, false
}

// textify renders a tool result as the text of a content block: strings pass through, everything else is JSON.
func textify(res any) string {
	if s, ok := res.(string); ok {
		return s
	}
	b, err := json.Marshal(res)
	if err != nil {
		return ""
	}
	return string(b)
}
