package gitprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTicketKeys(t *testing.T) {
	tests := []struct {
		name                string
		title, body, branch string
		want                []string
	}{
		{"key leading the title", "NEXUL-112 Show each auto play", "", "", []string{"NEXUL-112"}},
		{"lowercase key in the body is uppercased", "", "fixes ref-7", "", []string{"REF-7"}},
		{"key inside a branch path", "", "", "feature/REF-42-fix-login", []string{"REF-42"}},
		{"prefix with a digit", "", "", "V2API-3", []string{"V2API-3"}},
		{"keys across fields keep their order", "REF-1", "and WEB-2", "ref-3-x", []string{"REF-1", "WEB-2", "REF-3"}},
		{"one key named twice is listed once", "REF-42 fix", "Closes REF-42", "ref-42", []string{"REF-42"}},
		{"a prefix longer than five characters is no key", "", "TICKETS-9", "", nil},
		{"an issue number is no key", "", "Fixes #42", "ticket/42", nil},
		{"nothing named", "Fix login", "", "main", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TicketKeys(tt.title, tt.body, tt.branch))
		})
	}
}
