package gitprovider

import (
	"context"
	"regexp"
	"strings"
)

// ticketKeyRe matches a ticket key, PREFIX-NUMBER (ADR 0004), as a whole word: "NEXUL-112", "feature/ref-42-login".
var ticketKeyRe = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9]{1,4}-[0-9]+)\b`)

// TicketKeys returns the ticket keys named in a PR's title, body, and head branch, uppercased, in order, once each.
func TicketKeys(texts ...string) []string {
	seen := make(map[string]bool)
	var keys []string
	for _, text := range texts {
		for _, m := range ticketKeyRe.FindAllStringSubmatch(text, -1) {
			key := strings.ToUpper(m[1])
			if seen[key] {
				continue
			}
			seen[key] = true
			keys = append(keys, key)
		}
	}
	return keys
}

// TicketResolver turns the keys a repository's PR names into ticket ids, dropping keys that name no ticket.
type TicketResolver interface {
	TicketIDs(ctx context.Context, owner, repo string, keys []string) ([]string, error)
}
