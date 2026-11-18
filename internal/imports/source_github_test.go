package imports

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type pagedIssues struct {
	pages    [][]GitHubIssue
	comments map[int][]GitHubComment
	listed   []int
}

func (p *pagedIssues) ListIssues(_ context.Context, _, _ string, page int) ([]GitHubIssue, int, error) {
	p.listed = append(p.listed, page)
	next := page + 1
	if page == len(p.pages) {
		next = 0
	}
	return p.pages[page-1], next, nil
}

func (p *pagedIssues) ListComments(_ context.Context, _, _ string, number int) ([]GitHubComment, error) {
	return p.comments[number], nil
}

type syncedRepos map[string]bool

func (s syncedRepos) SyncOn(_ context.Context, owner, name string) (bool, error) {
	return s[owner+"/"+name], nil
}

type repoList []string

func (r repoList) Repos(context.Context, string) ([]string, error) { return r, nil }

type recordingSink struct {
	tickets     []Ticket
	docs        []Doc
	checkpoints []string
}

func (s *recordingSink) Doc(_ context.Context, d Doc) error {
	s.docs = append(s.docs, d)
	return nil
}

func (s *recordingSink) DocLink(sourceID string) string { return "/docs/doc-" + sourceID }

func (s *recordingSink) Ticket(_ context.Context, t Ticket) error {
	s.tickets = append(s.tickets, t)
	return nil
}

func (s *recordingSink) Checkpoint(_ context.Context, cursor string) error {
	s.checkpoints = append(s.checkpoints, cursor)
	return nil
}

func TestIssueTicket_MapsTheIssueItsCommentsAndTheirFiles(t *testing.T) {
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	issue := GitHubIssue{
		ID: 77, Number: 12, Title: "Crash on save", Body: "See ![x](https://github.com/user-attachments/assets/abc-1)", State: "closed",
		Labels: []string{"bug", "ui"}, Assignees: []string{"alice", "bob"}, Author: "carol", HTMLURL: "https://github.com/acme/app/issues/12",
	}
	comments := []GitHubComment{{ID: 5, Body: "Log: https://github.com/acme/app/files/9/log.txt", Author: "dave", AuthorAvatarURL: "https://avatars/d", CreatedAt: at}}
	got := IssueTicket("acme/app", issue, comments)
	assert.Equal(t, Ticket{
		SourceID: "77", Title: "Crash on save",
		Body:   "See ![x](https://github.com/user-attachments/assets/abc-1)\n\nImported from [acme/app#12](https://github.com/acme/app/issues/12), opened by carol.",
		Status: StatusRef{Kind: StatusDone}, Labels: []string{"bug", "ui"}, Assignee: Person{Login: "alice", Name: "alice"},
		Comments: []Comment{{SourceID: "5", Body: comments[0].Body, Author: Person{Login: "dave", Name: "dave"}, AvatarURL: "https://avatars/d", At: at}},
		Files:    []string{"https://github.com/user-attachments/assets/abc-1", "https://github.com/acme/app/files/9/log.txt"},
	}, got)
	assert.Equal(t, StatusRef{Kind: StatusBacklog}, IssueTicket("acme/app", GitHubIssue{State: "open"}, nil).Status)
}

func TestGitHubRepository_IsTheProjectsOwn(t *testing.T) {
	tests := []struct {
		name  string
		repos repoList
		job   Job
		want  string
	}{
		{"a credential", repoList{"acme/app"}, Job{Credential: "ghp"}, "takes no credential"},
		{"no repository", nil, Job{}, "has no repository"},
		{"two and none named", repoList{"acme/app", "acme/tests"}, Job{}, "name one in from (acme/app, acme/tests)"},
		{"one it does not have", repoList{"acme/app"}, Job{From: "evil/app"}, "not attached to the project"},
		{"one whose issues already sync", repoList{"acme/synced"}, Job{}, "issue sync is on for acme/synced"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGitHubSource(&pagedIssues{}, tt.repos, syncedRepos{"acme/synced": true}).Scan(t.Context(), tt.job)
			assert.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestGitHubSource_ScansEveryPageAndImportsFromTheCheckpoint(t *testing.T) {
	issues := &pagedIssues{
		pages: [][]GitHubIssue{
			{{ID: 1, Number: 1, State: "open", Assignees: []string{"alice"}, Comments: 2}, {ID: 2, Number: 2, State: "closed"}},
			{{ID: 3, Number: 3, State: "open", Comments: 1}},
		},
		comments: map[int][]GitHubComment{1: {{ID: 10}, {ID: 11}}, 3: {{ID: 30}}},
	}
	src := NewGitHubSource(issues, repoList{"Acme/App"}, syncedRepos{})
	found, err := src.Scan(t.Context(), Job{From: "acme/app"})
	require.NoError(t, err)
	assert.Equal(t, Scan{
		From: "Acme/App", Tickets: 3, Comments: 3, People: []Person{{Login: "alice", Name: "alice"}},
		Statuses: []StatusRef{{Kind: StatusBacklog}, {Kind: StatusDone}},
	}, found)

	sink := &recordingSink{}
	require.NoError(t, src.Import(t.Context(), Job{}, sink))
	assert.Len(t, sink.tickets, 3)
	assert.Len(t, sink.tickets[0].Comments, 2)
	assert.Equal(t, []string{"2"}, sink.checkpoints)

	issues.listed = nil
	resumed := &recordingSink{}
	require.NoError(t, src.Import(t.Context(), Job{Cursor: "2"}, resumed))
	assert.Equal(t, []int{2}, issues.listed, "a resumed import starts at the checkpointed page")
	assert.Equal(t, "3", resumed.tickets[0].SourceID)
}
