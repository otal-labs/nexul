package imports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemberFor_MatchesByGitHubLoginThenEmailThenUsernameAndName(t *testing.T) {
	members := []Member{
		{Login: "alice", GitHubLogins: []string{"alice-gh"}},
		{Login: "bob", Emails: []string{"bob@example.com"}},
		{Login: "carol", GitHubLogins: []string{"carol"}, Emails: []string{"alice-gh@example.com"}},
		{Login: "dee", Names: []string{"Dee Doe"}},
		{Login: "sam1", Names: []string{"Sam Lee"}},
		{Login: "sam2", Names: []string{"Sam Lee"}},
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
		{"a username as a member's login", Person{Username: "BOB", Name: "Robert"}, "bob"},
		{"a username's name, when one member has it", Person{Username: "dee-trello", Name: "dee doe"}, "dee"},
		{"a name two members share", Person{Username: "x-trello", Name: "Sam Lee"}, ""},
		{"a name alone, without a username", Person{Name: "Dee Doe"}, ""},
		{"a name beside a GitHub login", Person{Login: "zed", Name: "Dee Doe"}, ""},
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

func TestListStatus_FirstIsBacklogDoneLikeIsDoneTheRestProgress(t *testing.T) {
	assert.Equal(t, StatusRef{Name: "Ideas", Kind: StatusBacklog}, listStatus(0, "Ideas"))
	assert.Equal(t, StatusRef{Name: "Doing", Kind: StatusProgress}, listStatus(1, "Doing"))
	for _, name := range []string{"Done", "Completed tasks", "SHIPPED"} {
		assert.Equal(t, StatusRef{Name: name, Kind: StatusDone}, listStatus(3, name))
	}
	assert.Equal(t, StatusDone, listStatus(0, "Done").Kind, "a done-like first list is still done")
}

func TestTaskListAndFileLinks_WriteMarkdown(t *testing.T) {
	assert.Equal(t, "**Launch**\n\n- [x] Copy\n- [ ] Ship", taskList(" Launch ", []checkItem{{Name: "Copy", Done: true}, {Name: "Ship"}}))
	assert.Equal(t, "**Attachments**\n\n- [shot.png](https://x/a)", fileLinks([]fileLink{{Name: "[shot.png]", URL: "https://x/a"}}))
}
