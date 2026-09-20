package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
)

// maxStdioLineBytes bounds a single stdio message; the default bufio.Scanner cap (64KB) is too small for large bodies.
const maxStdioLineBytes = 4 << 20

// ServeStdio implements the MCP stdio transport: newline-delimited JSON-RPC in, responses out.
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
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
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			if err := writeStdioLine(out, errResponse(CodeParseError, "parse error: %v", err)); err != nil {
				return err
			}
			continue
		}
		resp := s.Dispatch(ctx, &req)
		if resp == nil {
			continue
		}
		if err := writeStdioLine(out, resp); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

func writeStdioLine(out io.Writer, resp *Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = out.Write(b)
	return err
}
