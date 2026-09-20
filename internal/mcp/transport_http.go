package mcp

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
)

// maxBodyBytes bounds a single Streamable HTTP request body; MCP messages are small JSON-RPC envelopes.
const maxBodyBytes = 1 << 20

// ServeHTTP implements the MCP Streamable HTTP transport: a JSON object or SSE stream, per Accept header.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mt != "" && mt != "application/json" {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse(CodeParseError, "read body: %v", err))
		return
	}
	if len(body) > maxBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse(CodeParseError, "parse error: %v", err))
		return
	}
	if mismatch := headerMismatch(r, &req); mismatch != nil {
		mismatch.ID = req.ID
		writeJSON(w, http.StatusBadRequest, mismatch)
		return
	}
	resp := s.Dispatch(r.Context(), &req)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if acceptsEventStream(r) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: "))
		enc := json.NewEncoder(w)
		_ = enc.Encode(resp)
		_, _ = w.Write([]byte("\n"))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// headerMismatch validates Mcp-Method/Mcp-Name against the body; legacy clients omit them, which isn't a mismatch.
func headerMismatch(r *http.Request, req *Request) *Response {
	if m := r.Header.Get("Mcp-Method"); m != "" && m != req.Method {
		return errResponse(CodeHeaderMismatch, "Mcp-Method header %q does not match body method %q", m, req.Method)
	}
	name := r.Header.Get("Mcp-Name")
	if name == "" {
		return nil
	}
	if bodyName, ok := paramsName(req.Params); ok && name != bodyName {
		return errResponse(CodeHeaderMismatch, "Mcp-Name header %q does not match body value %q", name, bodyName)
	}
	return nil
}

// paramsName extracts the tool/prompt name or resource URI a request's params carries, whichever is present.
func paramsName(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var p struct {
		Name string `json:"name"`
		URI  string `json:"uri"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", false
	}
	if p.Name != "" {
		return p.Name, true
	}
	if p.URI != "" {
		return p.URI, true
	}
	return "", false
}

func acceptsEventStream(r *http.Request) bool {
	for _, accept := range r.Header.Values("Accept") {
		for _, part := range strings.Split(accept, ",") {
			if strings.Contains(strings.ToLower(strings.TrimSpace(part)), "text/event-stream") {
				return true
			}
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, resp *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
