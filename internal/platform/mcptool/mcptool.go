// Package mcptool is the single MCP tool contract shared by every domain and the internal/mcp adapter.
package mcptool

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Tool maps to a single use-case function. Name is
// <domain>_<action>; InputSchema is the JSON Schema for the arguments object.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Call        func(ctx context.Context, args map[string]any) (any, error)
}

// ObjectSchema is the JSON Schema for a tool's arguments object; nil properties become {}, since clients reject a null.
func ObjectSchema(properties map[string]any, required ...string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// RequiredString returns args[key] as a non-empty string, or an ErrInvalid
// error naming the key when it is missing, not a string, or empty.
func RequiredString(args map[string]any, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("%w: %s is required", apperrs.ErrInvalid, key)
	}
	return v, nil
}

// RequiredStrings returns args[name] for each name, short-circuiting on the first missing/invalid one.
func RequiredStrings(args map[string]any, names ...string) ([]string, error) {
	out := make([]string, len(names))
	for i, name := range names {
		v, err := RequiredString(args, name)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// OptionalString returns v as a string, or "" when v holds any other type.
func OptionalString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
