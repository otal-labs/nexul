package gitprovider

import "regexp"

var (
	// refIssueRe matches "fixes #123" / "closes #123" / "resolves #123" / "refs #123".
	refIssueRe = regexp.MustCompile(`(?i)(?:fixes|closes|resolves|refs)\s+#(\d+)`)
	// refTicketRe matches "refs TICKET-456" and any bare "ticket-456" mention.
	refTicketRe = regexp.MustCompile(`(?i)ticket[-_](\d+)`)
	// ticketBranchRe matches "ticket/123" and "ticket/123-slug".
	ticketBranchRe = regexp.MustCompile(`(?i)^(?:refs/heads/)?ticket/(\d+)(?:[-_].*)?$`)
)

// LinkedTicketIDs extracts ticket refs from a PR body ("fixes #N"/"TICKET-N") and head branch ("ticket/<id>").
func LinkedTicketIDs(body, headBranch string) []string {
	seen := make(map[string]bool)
	var ids []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, m := range refIssueRe.FindAllStringSubmatch(body, -1) {
		add(m[1])
	}
	for _, m := range refTicketRe.FindAllStringSubmatch(body, -1) {
		add(m[1])
	}
	if m := ticketBranchRe.FindStringSubmatch(headBranch); m != nil {
		add(m[1])
	}
	return ids
}
