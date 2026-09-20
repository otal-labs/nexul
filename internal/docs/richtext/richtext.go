// Package richtext converts between the docs domain's canonical structured rich text (Tiptap JSON, ADR 0026) and CommonMark.
package richtext

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Doc is the root of a Tiptap document, the exact shape the web editor produces.
type Doc struct {
	Type    string `json:"type"`
	Content []Node `json:"content,omitempty"`
}

// Node is a single Tiptap node: a block, an inline text run, or a leaf.
type Node struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Marks   []Mark         `json:"marks,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []Node         `json:"content,omitempty"`
}

// Mark is a Tiptap mark applied to a text run (bold, italic, strike, ...).
type Mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// MarshalJSON serializes the document in its canonical compact form.
func MarshalJSON(doc *Doc) (string, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal richtext doc: %w", err)
	}
	return string(b), nil
}

// UnmarshalJSON parses a canonical document, tolerating either the {"type":"doc",...} form or a bare top-level array.
func UnmarshalJSON(body string) (*Doc, error) {
	var doc Doc
	if err := json.Unmarshal([]byte(body), &doc); err == nil && doc.Type == "doc" {
		return &doc, nil
	}
	var content []Node
	if err := json.Unmarshal([]byte(body), &content); err != nil {
		return nil, errors.New("not a Tiptap document")
	}
	return &Doc{Type: "doc", Content: content}, nil
}

// IsStructured reports whether body is already a canonical Tiptap document; a legacy body fails and is round-tripped.
func IsStructured(body string) bool {
	_, err := UnmarshalJSON(body)
	return err == nil
}

// Normalize returns the canonical JSON for a structured or legacy body, so storage always holds structured content.
func Normalize(body string) (string, error) {
	if IsStructured(body) {
		return body, nil
	}
	doc, err := MarkdownToDoc(body)
	if err != nil {
		return "", err
	}
	return MarshalJSON(doc)
}

// ToMarkdown renders a stored body as markdown; a structured body converts, legacy markdown passes through unchanged.
func ToMarkdown(body string) (string, error) {
	if !IsStructured(body) {
		return body, nil
	}
	return JSONToMarkdown(body)
}

// SearchText returns the markdown rendering of a body for FTS indexing; legacy markdown passes through.
func SearchText(body string) string {
	md, err := ToMarkdown(body)
	if err != nil {
		return body
	}
	return md
}

const attachmentPathPrefix = "/api/attachments/"

// RewriteAttachmentRefs points every /api/attachments/<id> src/href in a structured body at its mapped id,
// so a cloned body keeps referring to its own copied attachments rather than the source's; a legacy or
// unstructured body passes through unchanged.
func RewriteAttachmentRefs(body string, idMap map[string]string) (string, error) {
	if len(idMap) == 0 || !IsStructured(body) {
		return body, nil
	}
	doc, err := UnmarshalJSON(body)
	if err != nil {
		return "", err
	}
	rewriteAttachmentNodes(doc.Content, idMap)
	return MarshalJSON(doc)
}

func rewriteAttachmentNodes(nodes []Node, idMap map[string]string) {
	for i := range nodes {
		rewriteAttachmentAttrs(nodes[i].Attrs, idMap)
		for j := range nodes[i].Marks {
			rewriteAttachmentAttrs(nodes[i].Marks[j].Attrs, idMap)
		}
		rewriteAttachmentNodes(nodes[i].Content, idMap)
	}
}

func rewriteAttachmentAttrs(attrs map[string]any, idMap map[string]string) {
	for _, key := range [...]string{"src", "href"} {
		v, ok := attrs[key].(string)
		if !ok || !strings.HasPrefix(v, attachmentPathPrefix) {
			continue
		}
		if newID, ok := idMap[strings.TrimPrefix(v, attachmentPathPrefix)]; ok {
			attrs[key] = attachmentPathPrefix + newID
		}
	}
}
