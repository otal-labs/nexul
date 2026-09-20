package mcp

import (
	"strings"
)

// Prompt is a reusable prompt template exposed to clients; Messages renders from caller-supplied arguments.
type Prompt struct {
	Name        string
	Description string
	Arguments   []PromptArgument
	Messages    func(args map[string]string) []PromptMessage
}

// PromptArgument declares one parameter the prompt accepts.
type PromptArgument struct {
	Name        string
	Description string
	Required    bool
}

// PromptMessage is a single message in a rendered prompt. Role is "user" or "assistant"; Content carries the text.
type PromptMessage struct {
	Role    string
	Content string
}

type promptArgDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type promptDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Arguments   []promptArgDef `json:"arguments"`
}

func (p Prompt) def() promptDef {
	def := promptDef{Name: p.Name, Description: p.Description}
	for _, a := range p.Arguments {
		def.Arguments = append(def.Arguments, promptArgDef(a))
	}
	return def
}

type promptMessageDef struct {
	Role    string       `json:"role"`
	Content contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// interpolate substitutes {{name}} placeholders with argument values.
func interpolate(text string, args map[string]string) string {
	for k, v := range args {
		text = strings.ReplaceAll(text, "{{"+k+"}}", v)
	}
	return text
}
