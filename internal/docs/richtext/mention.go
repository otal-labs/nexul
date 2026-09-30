package richtext

import (
	"slices"
	"strings"
)

// MentionType values mirror the web editor's mention node attrs.
const (
	MentionTypeTicket = "ticket"
	MentionTypeDoc    = "doc"
	// MentionTypePerson's id is the user id; its label is the login, written as @login in markdown.
	MentionTypePerson = "person"
)

// mentionPaths maps each mention kind to its canonical URL prefix; person mentions point at People.
var mentionPaths = []struct{ kind, prefix string }{
	{MentionTypeTicket, "/tickets/"},
	{MentionTypeDoc, "/docs/"},
	{MentionTypePerson, "/people/"},
}

// MentionHref returns the canonical internal URL for a reference: the identifier, not the title.
func MentionHref(kind, id string) string {
	for _, p := range mentionPaths {
		if p.kind == kind {
			return p.prefix + id
		}
	}
	return ""
}

// ParseMentionHref recognizes a canonical mention URL, returning its kind and target id; false for any other URL.
func ParseMentionHref(href string) (kind, id string, ok bool) {
	for _, p := range mentionPaths {
		if strings.HasPrefix(href, p.prefix) && len(href) > len(p.prefix) {
			return p.kind, href[len(p.prefix):], true
		}
	}
	return "", "", false
}

// mentionMarkdownLabel is the link text a mention renders with: a person's login as @login, anything else its title.
func mentionMarkdownLabel(kind, label string) string {
	if kind == MentionTypePerson {
		return "@" + label
	}
	return label
}

// mentionNodeLabel reverses mentionMarkdownLabel, so a person's stored label is the bare login.
func mentionNodeLabel(kind, text string) string {
	if kind == MentionTypePerson {
		return strings.TrimPrefix(text, "@")
	}
	return text
}

// PersonMentions returns the user ids a body @-mentions, in order of first appearance; a legacy markdown body is parsed.
func PersonMentions(body string) []string {
	doc, err := UnmarshalJSON(body)
	if err != nil {
		doc, err = MarkdownToDoc(body)
	}
	if err != nil {
		return nil
	}
	var ids []string
	collectPersonMentions(doc.Content, &ids)
	return ids
}

func collectPersonMentions(nodes []Node, ids *[]string) {
	for _, n := range nodes {
		if n.Type == "mention" && n.Attrs["type"] == MentionTypePerson {
			if id, _ := n.Attrs["id"].(string); id != "" && !slices.Contains(*ids, id) {
				*ids = append(*ids, id)
			}
		}
		collectPersonMentions(n.Content, ids)
	}
}

// AddedPersonMentions returns the people after mentions that before did not, so a re-save never repeats a mention.
func AddedPersonMentions(before, after string) []string {
	previous := PersonMentions(before)
	var added []string
	for _, id := range PersonMentions(after) {
		if !slices.Contains(previous, id) {
			added = append(added, id)
		}
	}
	return added
}
