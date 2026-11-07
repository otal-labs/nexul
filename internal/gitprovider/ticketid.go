package gitprovider

import (
	"context"
	"regexp"
	"strings"
)

// ticketKeyRe matches a ticket key, PREFIX-NUMBER (ADR 0004), as a whole word: "NEXUL-112", "feature/ref-42-login".
var ticketKeyRe = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9]{1,4}-[0-9]+)\b`)

// closingKeyRe matches a key after a closing word, "Fixes REF-42" or "closes: ref-7": a body that only mentions a key does not link it.
var closingKeyRe = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?):?\s+([A-Za-z][A-Za-z0-9]{1,4}-[0-9]+)\b`)

// TicketKeys returns the ticket keys a PR names in its title, after a closing word in its body, and in its head branch, uppercased, in order, once each.
func TicketKeys(title, body, branch string) []string {
	matches := ticketKeyRe.FindAllStringSubmatch(title, -1)
	matches = append(matches, closingKeyRe.FindAllStringSubmatch(body, -1)...)
	matches = append(matches, ticketKeyRe.FindAllStringSubmatch(branch, -1)...)
	seen := make(map[string]bool)
	var keys []string
	for _, m := range matches {
		key := strings.ToUpper(m[1])
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	return keys
}

// TicketResolver turns the keys a repository's PR names into ticket ids, dropping keys that name no ticket.
type TicketResolver interface {
	TicketIDs(ctx context.Context, owner, repo string, keys []string) ([]string, error)
}
