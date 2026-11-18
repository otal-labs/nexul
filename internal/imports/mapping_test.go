package imports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemberFor_MatchesByGitHubLoginThenEmail(t *testing.T) {
	members := []Member{
		{Login: "alice", GitHubLogins: []string{"alice-gh"}},
		{Login: "bob", Emails: []string{"bob@example.com"}},
		{Login: "carol", GitHubLogins: []string{"carol"}, Emails: []string{"alice-gh@example.com"}},
	}
	tests := []struct {
		name   string
		person Person
		want   string
	}{
		{"a GitHub login, ignoring case", Person{Login: "Alice-GH"}, "alice"},
		{"an email, ignoring case", Person{Email: "BOB@example.com"}, "bob"},
		{"the login wins over another member's email", Person{Login: "alice-gh", Email: "alice-gh@example.com"}, "alice"},
		{"nobody", Person{Login: "zed", Email: "zed@example.com"}, ""},
		{"no one named", Person{Name: "Someone"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, memberFor(members, tt.person))
		})
	}
}

func TestMatchPeople_ListsEachPersonOnceAndSkipsTheNameless(t *testing.T) {
	got := matchPeople([]Member{{Login: "alice", GitHubLogins: []string{"alice"}}},
		[]Person{{Login: "alice"}, {Login: "ALICE"}, {Name: "Nobody"}, {Login: "zed"}})
	assert.Equal(t, []PersonMatch{{Person: Person{Login: "alice"}, Member: "alice"}, {Person: Person{Login: "zed"}}}, got)
}

func TestPlanStatuses_UsesAColumnByNameElseTheFirstOfItsKindElseANewOne(t *testing.T) {
	cols := []Column{{ID: "c1", Name: "Todo", Kind: StatusBacklog}, {ID: "c2", Name: "Shipped", Kind: StatusDone}, {ID: "c3", Name: "Done", Kind: StatusDone}}
	got := planStatuses(cols, []StatusRef{
		{Name: "done", Kind: StatusDone},
		{Kind: StatusDone},
		{Kind: StatusBacklog},
		{Kind: StatusBacklog},
		{Name: "In QA", Kind: StatusTesting},
		{Kind: StatusProgress},
	})
	assert.Equal(t, []StatusMatch{
		{StatusRef: StatusRef{Name: "done", Kind: StatusDone}, Column: "Done"},
		{StatusRef: StatusRef{Kind: StatusDone}, Column: "Shipped"},
		{StatusRef: StatusRef{Kind: StatusBacklog}, Column: "Todo"},
		{StatusRef: StatusRef{Name: "In QA", Kind: StatusTesting}, Column: "In QA", New: true},
		{StatusRef: StatusRef{Kind: StatusProgress}, Column: "In progress", New: true},
	}, got)
}
