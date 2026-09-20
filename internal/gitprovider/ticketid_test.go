package gitprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkedTicketIDs(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		headBranch string
		want       []string
	}{
		{"issue keyword in body", "Fixes #123", "", []string{"123"}},
		{"lowercase keyword", "closes #7", "", []string{"7"}},
		{"multiple keywords", "fixes #1\nresolves #2", "", []string{"1", "2"}},
		{"ticket id in body", "Refs TICKET-456", "", []string{"456"}},
		{"bare ticket mention", "ticket-9 is blocking", "", []string{"9"}},
		{"keyword and branch", "Fixes #1", "ticket/123-fix-login", []string{"1", "123"}},
		{"branch only", "", "ticket/42", []string{"42"}},
		{"branch with slug", "", "ticket/42-slug-name", []string{"42"}},
		{"branch with underscore", "", "ticket/42_slug", []string{"42"}},
		{"refs/heads prefixed branch", "", "refs/heads/ticket/7-fix", []string{"7"}},
		{"feature branch", "", "feature/foo", nil},
		{"no refs", "just a body", "main", nil},
		{"empty everything", "", "", nil},
		{"deduplicates across body and branch", "Fixes #42", "ticket/42-fix", []string{"42"}},
		{"deduplicates within body", "Fixes #1\nfixes #1", "", []string{"1"}},
		{"ignores non-referencing issue numbers", "see issue #999", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LinkedTicketIDs(tt.body, tt.headBranch)
			assert.Equal(t, tt.want, got)
		})
	}
}
