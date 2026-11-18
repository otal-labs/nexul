package richtext

import (
	"regexp"
	"strings"
)

var (
	// bareMention is an @login not inside a word, an address, a code span or a link's text.
	bareMention = regexp.MustCompile("(^|[^\\w.\\[`/])@([\\w-]+)")
	// personLink is a person mention as ToMarkdown writes it.
	personLink = regexp.MustCompile(`\[@[^\]\n]*\]\(/people/([^)\s]+)\)`)
	// instanceLink is a link or image whose target is a path on the instance, such as /tickets/<id> or /api/attachments/<id>.
	instanceLink = regexp.MustCompile(`\]\((/[^)\s]*)\)`)
)

// Logins names a mentioned person's login on the outside host, by their user id or their Nexul login, empty for none.
type Logins func(userID, login string) string

// ForOutside rewrites markdown for a reader who is not signed in at base: a link or image to an instance path points
// at base, and a person mention becomes @<their login there>; one with no login there stays a link to their page or,
// bare, a code span, so it never pings a stranger who shares the name. Fenced code is left as it is.
func ForOutside(md, base string, logins Logins) string {
	base = strings.TrimRight(base, "/")
	lines := strings.Split(md, "\n")
	fenced := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		lines[i] = outsideLine(line, base, logins)
	}
	return strings.Join(lines, "\n")
}

func outsideLine(line, base string, logins Logins) string {
	line = bareMention.ReplaceAllStringFunc(line, func(m string) string {
		parts := bareMention.FindStringSubmatch(m)
		if theirs := logins("", parts[2]); theirs != "" {
			return parts[1] + "@" + theirs
		}
		return parts[1] + "`@" + parts[2] + "`"
	})
	line = personLink.ReplaceAllStringFunc(line, func(m string) string {
		if theirs := logins(personLink.FindStringSubmatch(m)[1], ""); theirs != "" {
			return "@" + theirs
		}
		return m
	})
	return instanceLink.ReplaceAllString(line, "]("+base+"$1)")
}
