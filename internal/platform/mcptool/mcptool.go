// Package mcptool is the MCP tool contract every domain declares its tools with; only internal/mcp speaks the protocol.
package mcptool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Hints are the tool annotations clients decide confirmations from; each zero value is the protocol's conservative default.
type Hints struct {
	// ReadOnly: the tool changes nothing.
	ReadOnly bool
	// Additive: the tool only adds, never overwrites or deletes. Ignored when ReadOnly.
	Additive bool
	// Idempotent: repeating the call with the same arguments changes nothing more. Ignored when ReadOnly.
	Idempotent bool
	// Local: the tool touches only Nexul's own database, never GitHub, Cloudflare, a harness, or a machine.
	Local bool
}

// Tool is one MCP tool: what tools/list shows and the handler tools/call runs.
type Tool struct {
	Name        string
	Title       string
	Description string
	Hints       Hints
	InputSchema *jsonschema.Schema
	// Call receives the raw arguments object; tools built with New validate and decode it before the handler runs.
	Call func(ctx context.Context, args json.RawMessage) (any, error)
}

// New builds a Tool whose input schema is inferred from In, and whose Call validates the arguments against it,
// decodes them into In, and runs h. It panics when In has no JSON Schema, which only a programming error causes.
func New[In any](name, title, description string, hints Hints, h func(ctx context.Context, in In) (any, error)) Tool {
	schema, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("mcptool: tool %s: %v", name, err))
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		panic(fmt.Sprintf("mcptool: tool %s: %v", name, err))
	}
	return Tool{
		Name:        name,
		Title:       title,
		Description: description,
		Hints:       hints,
		InputSchema: schema,
		Call: func(ctx context.Context, args json.RawMessage) (any, error) {
			in, err := decode[In](resolved, args)
			if err != nil {
				return nil, err
			}
			return h(ctx, in)
		},
	}
}

func decode[In any](resolved *jsonschema.Resolved, args json.RawMessage) (In, error) {
	var in In
	args = bytes.TrimSpace(args)
	if len(args) == 0 || bytes.Equal(args, []byte("null")) {
		args = []byte("{}")
	}
	var instance any
	if err := json.Unmarshal(args, &instance); err != nil {
		return in, fmt.Errorf("%w: arguments are not valid JSON: %v", apperrs.ErrInvalid, err)
	}
	if err := resolved.Validate(instance); err != nil {
		return in, fmt.Errorf("%w: the arguments do not match this tool's input schema: %v", apperrs.ErrInvalid, err)
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return in, fmt.Errorf("%w: %v", apperrs.ErrInvalid, err)
	}
	return in, nil
}

// DefaultLimit and MaxLimit bound every list a tool returns.
const (
	DefaultLimit = 50
	MaxLimit     = 100
)

// PageArgs is embedded in a list tool's input so every list takes the same limit and offset.
type PageArgs struct {
	Limit  int `json:"limit,omitzero" jsonschema:"How many items to return, 1 to 100. Defaults to 50."`
	Offset int `json:"offset,omitzero" jsonschema:"How many items to skip, from next_offset of the previous page. Defaults to 0."`
}

// Page is a bounded slice of a list plus what the caller needs to fetch the rest.
type Page[T any] struct {
	Items      []T  `json:"items"`
	Total      int  `json:"total"`
	HasMore    bool `json:"has_more"`
	NextOffset int  `json:"next_offset,omitzero"`
}

// Paginate cuts items to the requested page; out-of-range limits fall back to the defaults instead of failing.
func Paginate[T any](items []T, p PageArgs) Page[T] {
	limit := p.Limit
	if limit < 1 {
		limit = DefaultLimit
	}
	limit = min(limit, MaxLimit)
	offset := min(max(p.Offset, 0), len(items))
	end := min(offset+limit, len(items))
	page := Page[T]{Items: items[offset:end], Total: len(items), HasMore: end < len(items)}
	if page.Items == nil {
		page.Items = []T{}
	}
	if page.HasMore {
		page.NextOffset = end
	}
	return page
}

// Deleted is what every delete tool returns, so the model can confirm the effect.
type Deleted struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// Gone reports id as deleted.
func Gone(id string) Deleted {
	return Deleted{ID: id, Deleted: true}
}
