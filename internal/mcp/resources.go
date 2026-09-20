package mcp

import (
	"context"
	"strings"
)

// Resource is a static URI whose Read returns its text content, e.g. topology://current.
type Resource struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
	Read        func(ctx context.Context) (string, error)
}

// ResourceTemplate is a parameterized resource (e.g. docs://{id}); Read receives the captured values.
type ResourceTemplate struct {
	URITemplate string
	Name        string
	Description string
	MIMEType    string
	Read        func(ctx context.Context, vars map[string]string) (string, error)
}

type resourceDef struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type resourceTemplateDef struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType,omitempty"`
}

// matchTemplate matches a single {param} placeholder; a template with no placeholder requires exact equality.
func matchTemplate(uri string, t ResourceTemplate) (map[string]string, bool) {
	open, close := strings.Index(t.URITemplate, "{"), strings.Index(t.URITemplate, "}")
	if open < 0 || close < 0 || close < open {
		return nil, uri == t.URITemplate
	}
	param := t.URITemplate[open+1 : close]
	prefix, suffix := t.URITemplate[:open], t.URITemplate[close+1:]
	if !strings.HasPrefix(uri, prefix) || !strings.HasSuffix(uri, suffix) {
		return nil, false
	}
	value := uri[len(prefix) : len(uri)-len(suffix)]
	if value == "" {
		return nil, false
	}
	return map[string]string{param: value}, true
}
