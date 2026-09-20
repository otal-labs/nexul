package mcp

import (
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// Tool is the adapter's name for the shared contract type every domain declares its tools with.
type Tool = mcptool.Tool

// toolDef is the wire shape of a tool in tools/list.
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func toolDefOf(t Tool) toolDef {
	return toolDef{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema}
}
