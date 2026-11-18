package imports

import (
	"slices"
	"strings"
)

// memberFor is the login of the member a person maps to, by GitHub login, then email, then for a source with neither
// a member's login and then their name, the name only when one member alone has it; "" when nobody does.
func memberFor(members []Member, p Person) string {
	login, email := strings.ToLower(strings.TrimSpace(p.Login)), strings.ToLower(strings.TrimSpace(p.Email))
	for _, m := range members {
		if login != "" && slices.Contains(m.GitHubLogins, login) {
			return m.Login
		}
	}
	for _, m := range members {
		if email != "" && slices.Contains(m.Emails, email) {
			return m.Login
		}
	}
	username := strings.TrimSpace(p.Username)
	if username == "" || login != "" || email != "" {
		return ""
	}
	for _, m := range members {
		if strings.EqualFold(m.Login, username) {
			return m.Login
		}
	}
	return memberNamed(members, p.Name)
}

// memberNamed is the one member whose name is name, ignoring case; "" when none or several are.
func memberNamed(members []Member, name string) string {
	name = strings.TrimSpace(name)
	found := ""
	for _, m := range members {
		if name == "" || !slices.ContainsFunc(m.Names, func(n string) bool { return strings.EqualFold(strings.TrimSpace(n), name) }) {
			continue
		}
		if found != "" {
			return ""
		}
		found = m.Login
	}
	return found
}

// matchPeople maps every person once, in the order the source found them.
func matchPeople(members []Member, people []Person) []PersonMatch {
	seen := map[string]bool{}
	var out []PersonMatch
	for _, p := range people {
		if p.Zero() || seen[p.Key()] {
			continue
		}
		seen[p.Key()] = true
		out = append(out, PersonMatch{Person: p, Member: memberFor(members, p)})
	}
	return out
}

// columnFor is the column a status lands in: one of its name, ignoring case, else the first of its kind.
func columnFor(cols []Column, ref StatusRef) (Column, bool) {
	name := strings.TrimSpace(ref.Name)
	for _, c := range cols {
		if name != "" && strings.EqualFold(strings.TrimSpace(c.Name), name) {
			return c, true
		}
	}
	if name != "" {
		return Column{}, false
	}
	for _, c := range cols {
		if c.Kind == ref.Kind {
			return c, true
		}
	}
	return Column{}, false
}

// newColumnName is what a column the import adds is called: the status's own name, else its kind's usual one.
func newColumnName(ref StatusRef) string {
	if name := strings.TrimSpace(ref.Name); name != "" {
		return name
	}
	switch ref.Kind {
	case StatusProgress:
		return "In progress"
	case StatusReview:
		return "In review"
	case StatusTesting:
		return "Testing"
	case StatusDone:
		return "Done"
	}
	return "Backlog"
}

// planStatuses says where each status lands without adding a column, for the scan's summary.
func planStatuses(cols []Column, refs []StatusRef) []StatusMatch {
	var out []StatusMatch
	for _, ref := range refs {
		if slices.ContainsFunc(out, func(m StatusMatch) bool { return m.StatusRef == ref }) {
			continue
		}
		c, ok := columnFor(cols, ref)
		if ok {
			out = append(out, StatusMatch{StatusRef: ref, Column: c.Name})
			continue
		}
		out = append(out, StatusMatch{StatusRef: ref, Column: newColumnName(ref), New: true})
	}
	return out
}

// assignedLine is what an unmatched assignee leaves in the body, so who held it is not lost.
func assignedLine(title string, p Person) string {
	return "Assigned in " + title + " to " + strings.TrimSpace(personName(p))
}

func personName(p Person) string {
	for _, s := range []string{p.Name, p.Login, p.Username, p.Email} {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return "someone"
}

// listStatus is where one of a board's lists or a project's sections lands, i counting the open ones in order: the
// first is backlog, one named like done, complete or shipped is done, and the rest are in progress.
func listStatus(i int, name string) StatusRef {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "shipped") {
		return StatusRef{Name: name, Kind: StatusDone}
	}
	if i == 0 {
		return StatusRef{Name: name, Kind: StatusBacklog}
	}
	return StatusRef{Name: name, Kind: StatusProgress}
}

// taskList is a checklist as task-list markdown.
func taskList(title string, items []checkItem) string {
	var b strings.Builder
	b.WriteString("**" + strings.TrimSpace(title) + "**\n")
	for _, it := range items {
		mark := " "
		if it.Done {
			mark = "x"
		}
		b.WriteString("\n- [" + mark + "] " + strings.TrimSpace(it.Name))
	}
	return b.String()
}

type checkItem struct {
	Name string
	Done bool
}

// fileLinks is a body's list of a record's files, which the sink points at their copies.
func fileLinks(files []fileLink) string {
	var b strings.Builder
	b.WriteString("**Attachments**\n")
	for _, f := range files {
		b.WriteString("\n- [" + strings.NewReplacer("[", "", "]", "").Replace(strings.TrimSpace(f.Name)) + "](" + f.URL + ")")
	}
	return b.String()
}

type fileLink struct{ Name, URL string }
