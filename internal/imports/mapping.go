package imports

import (
	"slices"
	"strings"
)

// memberFor is the login of the member a person maps to, by GitHub login first and then email; "" when nobody does.
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
	return ""
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
	for _, s := range []string{p.Name, p.Login, p.Email} {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return "someone"
}
