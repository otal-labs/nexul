package richtext

import (
	"regexp"
	"strings"
)

var (
	// bareMention is an @login not inside a word, an address, a path or a link's text.
	bareMention = regexp.MustCompile(`(^|[^\w.\[/])@([\w-]+)`)
	// personLink is a person mention as ToMarkdown writes it.
	personLink = regexp.MustCompile(`\[@[^\]\n]*\]\(/people/([^)\s]+)\)`)
	// instanceLink is a link or image whose target is a path on the instance, such as /tickets/<id> or /api/attachments/<id>.
	instanceLink = regexp.MustCompile(`\]\((/[^)\s]*)\)`)
)

// Outsider is how a mentioned person shows to a reader outside Nexul: by their login there, else by name.
type Outsider struct {
	Login string
	Name  string
}

// People finds a mentioned person by user id or by Nexul login, false when nobody in Nexul is that person.
type People func(userID, login string) (Outsider, bool)

// ForOutside rewrites markdown for a reader who is not signed in at base: a link or image to an instance path points
// at base, and a person mention becomes @<their login there>, or their name when they have none, so a Nexul login,
// which can be an email, never shows and a stranger who shares it is never pinged. With bare, a chat message's plain
// @login is a mention too; an @handle nobody in Nexul holds stays as it is. Code is left alone.
func ForOutside(md, base string, people People, bare bool) string {
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
		lines[i] = outsideLine(line, base, people, bare)
	}
	return strings.Join(lines, "\n")
}

// outsideLine rewrites the parts of line outside its code spans.
func outsideLine(line, base string, people People, bare bool) string {
	var b strings.Builder
	for len(line) > 0 {
		text, code, rest := nextCodeSpan(line)
		b.WriteString(outsideText(text, base, people, bare))
		b.WriteString(code)
		line = rest
	}
	return b.String()
}

// nextCodeSpan splits line at its first code span: the text before it, the span itself, and what follows. A span
// opens on a run of backticks no backslash escapes and closes on the next run of the same length; an unclosed run is
// text.
func nextCodeSpan(line string) (text, code, rest string) {
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' {
			i++
			continue
		}
		if line[i] != '`' {
			continue
		}
		n := runLength(line, i)
		for j := i + n; j < len(line); j++ {
			if line[j] != '`' {
				continue
			}
			m := runLength(line, j)
			if m == n {
				return line[:i], line[i : j+m], line[j+m:]
			}
			j += m - 1
		}
		i += n - 1
	}
	return line, "", ""
}

func runLength(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}

func outsideText(text, base string, people People, bare bool) string {
	if !bare {
		return outsideLinks(text, base, people)
	}
	text = bareMention.ReplaceAllStringFunc(text, func(m string) string {
		sub := bareMention.FindStringSubmatch(m)
		who, ok := people("", sub[2])
		if !ok {
			return m
		}
		return sub[1] + shown(who, "")
	})
	return outsideLinks(text, base, people)
}

func outsideLinks(text, base string, people People) string {
	text = personLink.ReplaceAllStringFunc(text, func(m string) string {
		id := personLink.FindStringSubmatch(m)[1]
		who, _ := people(id, "")
		return shown(who, base+"/people/"+id)
	})
	return instanceLink.ReplaceAllString(text, "]("+base+"$1)")
}

// shown is who as @login there, else their name, linked to page when it is set.
func shown(who Outsider, page string) string {
	if who.Login != "" {
		return "@" + who.Login
	}
	name := escapeMarkdownText(who.Name)
	if name == "" {
		name = "a Nexul member"
	}
	if page == "" {
		return "**" + name + "**"
	}
	return "[" + name + "](" + page + ")"
}
