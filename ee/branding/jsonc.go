package branding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// parseJSONC reads a VS Code theme file: JSON with comments and trailing commas, as the editor itself accepts.
func parseJSONC(source string) (map[string]any, error) {
	clean := []byte(stripJSONC(strings.TrimPrefix(source, "\xef\xbb\xbf")))
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(clean))
	if err := dec.Decode(&out); err != nil || out == nil {
		return nil, fmt.Errorf("%w: this is not a VS Code color theme file: it does not hold a JSON object", apperrs.ErrInvalid)
	}
	return out, nil
}

// stripJSONC drops // and /* */ comments and commas before a closing bracket, leaving strings untouched.
func stripJSONC(src string) string {
	var out strings.Builder
	out.Grow(len(src))
	inString, escaped := false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			out.WriteByte(c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch {
		case c == '"':
			inString = true
			out.WriteByte(c)
		case c == '/' && i+1 < len(src) && (src[i+1] == '/' || src[i+1] == '*'):
			next, ok := skipComment(src, i)
			if !ok {
				return out.String()
			}
			i = next
			out.WriteByte(' ')
		default:
			out.WriteByte(c)
		}
	}
	return dropTrailingCommas(out.String())
}

// skipComment returns the index of the last byte of the comment opening at i; false when a block comment never closes.
func skipComment(src string, i int) (int, bool) {
	if src[i+1] == '/' {
		end := strings.IndexByte(src[i:], '\n')
		if end < 0 {
			return len(src) - 1, true
		}
		return i + end - 1, true
	}
	end := strings.Index(src[i+2:], "*/")
	if end < 0 {
		return 0, false
	}
	return i + end + 3, true
}

func dropTrailingCommas(src string) string {
	var out strings.Builder
	out.Grow(len(src))
	inString, escaped := false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			out.WriteByte(c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
		}
		if c == ',' && closesNext(src[i+1:]) {
			continue
		}
		out.WriteByte(c)
	}
	return out.String()
}

func closesNext(rest string) bool {
	rest = strings.TrimLeft(rest, " \t\r\n")
	return rest != "" && (rest[0] == '}' || rest[0] == ']')
}
