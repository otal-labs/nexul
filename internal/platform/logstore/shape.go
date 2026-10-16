package logstore

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/redact"
)

// maxChars cuts a message or an attribute value, so one line cannot fill a reply.
const maxChars = 2000

// Line is one log line as a reader sees it: the headline fields, and every other column in Attrs.
type Line struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Source  string         `json:"source"`
	Message string         `json:"message"`
	TraceID string         `json:"trace_id,omitempty"`
	Name    string         `json:"name,omitempty"`
	Machine string         `json:"machine,omitempty"`
	Version string         `json:"version,omitempty"`
	UserID  string         `json:"user_id,omitempty"`
	URL     string         `json:"url,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// shape turns a hit into a Line: a server line carries no source column, a secret-named attribute is dropped, and
// every string passes redact.Tokens and the length cut.
func shape(row map[string]any) Line {
	var l Line
	headline := map[string]*string{"body": &l.Message, "source": &l.Source, "trace_id": &l.TraceID, "name": &l.Name,
		"machine": &l.Machine, "version": &l.Version, "user_id": &l.UserID, "url": &l.URL}
	for col, v := range row {
		if v == nil || sensitive(col) {
			continue
		}
		if s, ok := v.(string); ok {
			v = clean(s)
		}
		if dst, ok := headline[col]; ok {
			*dst = text(v)
			continue
		}
		switch col {
		case "_timestamp":
			micros, _ := v.(json.Number).Int64()
			l.Time = time.UnixMicro(micros).UTC()
		case "severity":
			l.Level = strings.ToLower(text(v))
		default:
			if l.Attrs == nil {
				l.Attrs = map[string]any{}
			}
			l.Attrs[col] = v
		}
	}
	if l.Source == "" {
		l.Source = "server"
	}
	return l
}

func text(v any) string {
	s, _ := v.(string)
	return s
}

func clean(s string) string {
	s = redact.Tokens(s)
	if r := []rune(s); len(r) > maxChars {
		return string(r[:maxChars]) + "…"
	}
	return s
}
