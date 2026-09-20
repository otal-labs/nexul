package richtext

import "strings"

// MentionType values mirror the web editor's mention node attrs.
const (
	MentionTypeTicket = "ticket"
	MentionTypeDoc    = "doc"
)

// MentionHref returns the canonical internal URL for a reference: the identifier, not the title.
func MentionHref(kind, id string) string {
	switch kind {
	case MentionTypeTicket:
		return "/tickets/" + id
	case MentionTypeDoc:
		return "/docs/" + id
	}
	return ""
}

// ParseMentionHref recognizes a canonical mention URL, returning its kind and target id; false for any other URL.
func ParseMentionHref(href string) (kind, id string, ok bool) {
	for _, k := range []string{MentionTypeTicket, MentionTypeDoc} {
		prefix := "/" + k + "s/"
		if strings.HasPrefix(href, prefix) && len(href) > len(prefix) {
			return k, href[len(prefix):], true
		}
	}
	return "", "", false
}
