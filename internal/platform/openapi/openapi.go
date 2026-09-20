// Package openapi builds an OpenAPI 3.0.x document from gateway route registrations (ADR 0045).
package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Info is the OpenAPI top-level metadata block.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Spec accumulates route registrations and renders an OpenAPI 3.0.x document.
type Spec struct {
	info     Info
	paths    map[string]*PathItem
	tags     map[string]string // tag -> description
	security bool
}

// New starts an empty spec with the given info.
func New(info Info) *Spec {
	return &Spec{
		info:  info,
		paths: make(map[string]*PathItem),
		tags:  make(map[string]string),
	}
}

// AddSecuritySchemes declares that the spec carries the bearer auth scheme
// (used by both user sessions and integration tokens).
func (s *Spec) AddSecuritySchemes() {
	s.security = true
}

// Register adds one operation to the spec; a repeat method+path replaces the summary but keeps tag order stable.
func (s *Spec) Register(method, path, summary string, tags ...string) {
	method = strings.ToUpper(method)
	item, ok := s.paths[path]
	if !ok {
		item = &PathItem{}
		s.paths[path] = item
	}
	op := &Operation{Summary: summary, Tags: tags}
	if len(tags) == 0 {
		op.Tags = nil
	}
	switch method {
	case http.MethodGet:
		item.Get = op
	case http.MethodPost:
		item.Post = op
	case http.MethodPut:
		item.Put = op
	case http.MethodPatch:
		item.Patch = op
	case http.MethodDelete:
		item.Delete = op
	}
	for _, tag := range tags {
		if _, ok := s.tags[tag]; !ok {
			s.tags[tag] = ""
		}
	}
}

// RegisterMountedRoutes adds every method-bearing route collected from the HTTP gateway.
func (s *Spec) RegisterMountedRoutes(routes []httpx.Route) {
	for _, route := range routes {
		s.Register(route.Method, route.Path, "API operation", routeTag(route.Path))
	}
}

func routeTag(path string) string {
	path = strings.TrimPrefix(path, "/api/")
	if path == "" {
		return ""
	}
	if tag, _, ok := strings.Cut(path, "/"); ok {
		return tag
	}
	return path
}

// SetTagDescription annotates a tag shown in the Swagger UI grouping.
func (s *Spec) SetTagDescription(tag, description string) {
	s.tags[tag] = description
}

// PathItem holds the operations for one path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation is one endpoint's summary + tags.
type Operation struct {
	Summary   string              `json:"summary,omitempty"`
	Tags      []string            `json:"tags,omitempty"`
	Responses map[string]Response `json:"responses"`
}

// Response is a minimal default response entry; every operation advertises the standard JSON envelope.
type Response struct {
	Description string `json:"description"`
}

type tagEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// document is the rendered OpenAPI 3.0.3 structure.
type document struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Paths      map[string]any         `json:"paths"`
	Tags       []tagEntry             `json:"tags,omitempty"`
	Security   *[]map[string][]string `json:"security,omitempty"`
	Components map[string]any         `json:"components,omitempty"`
}

// JSON renders the OpenAPI document.
func (s *Spec) JSON() ([]byte, error) {
	paths := make(map[string]any, len(s.paths))
	for path, item := range s.paths {
		if !item.isEmpty() {
			paths[path] = item
		}
	}

	for _, item := range s.paths {
		for _, op := range item.operations() {
			if op.Responses == nil {
				op.Responses = defaultResponses()
			}
		}
	}

	tags := make([]tagEntry, 0, len(s.tags))
	for tag, desc := range s.tags {
		tags = append(tags, tagEntry{Name: tag, Description: desc})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })

	doc := document{
		OpenAPI: "3.0.3",
		Info:    s.info,
		Paths:   paths,
		Tags:    tags,
	}
	if s.security {
		scheme := "bearerAuth"
		doc.Security = &[]map[string][]string{{scheme: {}}}
		doc.Components = map[string]any{
			"securitySchemes": map[string]any{
				scheme: map[string]any{
					"type":        "http",
					"scheme":      "bearer",
					"description": "User session token, personal access token, or a scoped integration token.",
				},
			},
		}
	}
	return json.MarshalIndent(doc, "", "  ")
}

func defaultResponses() map[string]Response {
	return map[string]Response{
		"200": {Description: "OK"},
		"400": {Description: "Invalid request (error envelope)"},
		"401": {Description: "Unauthorized"},
		"500": {Description: "Internal error"},
	}
}

func (p *PathItem) isEmpty() bool {
	return p.Get == nil && p.Post == nil && p.Put == nil && p.Patch == nil && p.Delete == nil
}

func (p *PathItem) operations() []*Operation {
	var out []*Operation
	for _, op := range []*Operation{p.Get, p.Post, p.Put, p.Patch, p.Delete} {
		if op != nil {
			out = append(out, op)
		}
	}
	return out
}

// Handler serves /openapi.json and a Swagger UI page at /swagger.
func (s *Spec) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		doc, err := s.JSON()
		if err != nil {
			http.Error(w, fmt.Sprintf("render openapi: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(doc)
	})
	mux.HandleFunc("/swagger", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerUIHTML))
	})
	return mux
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Nexul API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" });
    };
  </script>
</body>
</html>`
