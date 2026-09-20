package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StdioProxy forwards newline-delimited JSON-RPC from stdio to a running server's MCP HTTP transport; it proxies rather than opening the database because SQLite has one writer (ADR 0011).
type StdioProxy struct {
	endpoint string
	pat      string
	hc       *http.Client
}

// NewStdioProxy wires the proxy; an empty hc yields a generous timeout since MCP tool calls can be slow.
func NewStdioProxy(endpoint, pat string, hc *http.Client) *StdioProxy {
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &StdioProxy{endpoint: endpoint, pat: pat, hc: hc}
}

// Serve mirrors the in-process stdio transport's framing; notifications (no id) produce no output.
func (p *StdioProxy) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64*1024), maxStdioLineBytes)
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return nil
		}
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var parsed Request
		notification := json.Unmarshal(line, &parsed) == nil && len(parsed.ID) == 0
		resp, err := p.call(ctx, line, &parsed)
		if err != nil {
			if notification {
				continue
			}
			if err := writeStdioLine(out, errResponse(CodeInternalError, "proxy error: %v", err)); err != nil {
				return err
			}
			continue
		}
		if resp == nil || notification {
			continue
		}
		if err := writeStdioLine(out, resp); err != nil {
			return err
		}
	}
	return sc.Err()
}

// call POSTs one line upstream with Mcp-Method/Mcp-Name headers; non-JSON-RPC failures wrap into a JSON-RPC error.
func (p *StdioProxy) call(ctx context.Context, line []byte, parsed *Request) (result *Response, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(line))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if parsed.Method != "" {
		req.Header.Set("Mcp-Method", parsed.Method)
	}
	if name, ok := mcpCallName(parsed); ok {
		req.Header.Set("Mcp-Name", name)
	}
	if p.pat != "" {
		req.Header.Set("Authorization", "Bearer "+p.pat)
	}
	httpResp, err := p.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, httpResp.Body.Close())
	}()
	if httpResp.StatusCode == http.StatusAccepted {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, err
	}
	var out Response
	if err := json.Unmarshal(body, &out); err == nil && out.JSONRPC == "2.0" {
		return &out, nil
	}
	code := CodeInternalError
	if httpResp.StatusCode == http.StatusUnauthorized {
		code = CodeUnauthorized
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		text = httpResp.Status
	}
	return &Response{JSONRPC: "2.0", Error: &RPCError{Code: code, Message: text}}, nil
}

// mcpCallName extracts params.name for the two methods the Mcp-Name header applies to: tools/call and prompts/get.
func mcpCallName(req *Request) (string, bool) {
	if req.Method != MethodToolsCall && req.Method != MethodPromptsGet {
		return "", false
	}
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.Name == "" {
		return "", false
	}
	return p.Name, true
}
