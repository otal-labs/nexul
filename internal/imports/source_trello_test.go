package imports

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const trelloCred = "k3y:t0ken"

// sampleTrello is a board with four lists, one archived, and five cards over three pages of two, two of them archived.
func sampleTrello(t *testing.T) (*TrelloSource, *stubAPI, *sleeps) {
	t.Helper()
	lists := []map[string]any{
		{"id": "L1", "name": "To do"}, {"id": "L2", "name": "Doing"}, {"id": "L3", "name": "Done"}, {"id": "L4", "name": "Old", "closed": true},
	}
	upload := "https://trello.com/1/cards/c5/attachments/a1/download/shot.png"
	cards := map[string][]map[string]any{
		"": {
			{
				"id": "c5", "name": "Ship the login", "desc": "Needs a test.", "idList": "L1", "idMembers": []string{"m1"},
				"due": "2026-12-01T23:30:00.000Z", "shortUrl": "https://trello.com/c/c5",
				"labels": []map[string]string{{"name": "bug", "color": "red"}, {"name": "", "color": "green"}},
				"badges": map[string]int{"comments": 1},
				"checklists": []map[string]any{{"name": "Steps", "pos": 1, "checkItems": []map[string]any{
					{"name": "Write", "state": "complete", "pos": 2}, {"name": "Plan", "state": "incomplete", "pos": 1},
				}}},
				"attachments": []map[string]any{
					{"name": "shot.png", "url": upload, "isUpload": true},
					{"name": "Spec", "url": "https://docs.example.com/spec", "isUpload": false},
				},
			},
			{"id": "c4", "name": "Archived card", "idList": "L1", "closed": true},
		},
		"c4": {{"id": "c3", "name": "On an archived list", "idList": "L4"}, {"id": "c2", "name": "Finished", "idList": "L3"}},
		"c2": {{"id": "c1", "name": "Started", "idList": "L2"}},
	}
	rateLimited := false
	stub := newStubAPI(t, func(r *http.Request) (int, http.Header, any) {
		switch r.URL.Path {
		case "/boards/abc":
			return 200, nil, map[string]any{"id": "b1", "name": "Roadmap", "shortLink": "abc"}
		case "/boards/b1/lists":
			return 200, http.Header{"X-Rate-Limit-Api-Token-Remaining": {"0"}, "X-Rate-Limit-Api-Token-Interval-Ms": {"10000"}}, lists
		case "/boards/b1/members":
			if !rateLimited {
				rateLimited = true
				return http.StatusTooManyRequests, http.Header{"Retry-After": {"2"}}, nil
			}
			return 200, nil, []map[string]string{{"id": "m1", "username": "alice", "fullName": "Alice Doe", "avatarUrl": "https://trello-members.s3.amazonaws.com/m1/x"}}
		case "/boards/b1/cards/all":
			return 200, nil, cards[r.URL.Query().Get("before")]
		case "/cards/c5/actions":
			return 200, nil, []map[string]any{{
				"id": "act1", "date": "2026-11-02T10:00:00Z", "data": map[string]string{"text": "Looks good"},
				"memberCreator": map[string]string{"username": "bob", "fullName": "Bob Roe"},
			}}
		}
		return 200, nil, nil
	})
	src := NewTrelloSource(stub.srv.URL, stub.srv.Client())
	src.page = 2
	sl := &sleeps{}
	src.api.sleep = sl.sleep
	return src, stub, sl
}

func TestTrelloSource_ScansOpenCardsAndCountsTheArchived(t *testing.T) {
	src, stub, sl := sampleTrello(t)
	found, err := src.Scan(t.Context(), Job{From: "https://trello.com/b/abc/roadmap", Credential: trelloCred})
	require.NoError(t, err)
	assert.Equal(t, Scan{
		From: "abc", Tickets: 3, Comments: 1, Archived: 3, People: []Person{{Username: "alice", Name: "Alice Doe"}},
		Statuses: []StatusRef{{Name: "To do", Kind: StatusBacklog}, {Name: "Doing", Kind: StatusProgress}, {Name: "Done", Kind: StatusDone}},
	}, found)
	for _, r := range stub.requests {
		assert.Equal(t, `OAuth oauth_consumer_key="k3y", oauth_token="t0ken"`, r.Header.Get("Authorization"))
		assert.NotContains(t, r.URL.RawQuery, "t0ken", "the token never goes in the URL")
	}
	assert.Contains(t, sl.got, 2*time.Second, "a 429 waits out its Retry-After")
	assert.GreaterOrEqual(t, sl.longest(), 9*time.Second, "a spent token waits out its interval")
}

func TestTrelloSource_ImportsEachOpenCardAndResumesFromItsCheckpoint(t *testing.T) {
	src, _, _ := sampleTrello(t)
	sink := &recordingSink{}
	require.NoError(t, src.Import(t.Context(), Job{From: "abc", Credential: trelloCred}, sink))
	require.Len(t, sink.tickets, 3)
	assert.Equal(t, []string{"c4", "c2"}, sink.checkpoints)
	upload := "https://trello.com/1/cards/c5/attachments/a1/download/shot.png"
	assert.Equal(t, Ticket{
		SourceID: "c5", Title: "Ship the login",
		Body: "Needs a test.\n\n**Steps**\n\n- [ ] Plan\n- [x] Write\n\n**Attachments**\n\n- [shot.png](" + upload + ")\n- [Spec](https://docs.example.com/spec)" +
			"\n\nImported from [Trello](https://trello.com/c/c5), board Roadmap.",
		Status: StatusRef{Name: "To do", Kind: StatusBacklog}, Labels: []string{"bug", "green"},
		Assignee: Person{Username: "alice", Name: "Alice Doe"}, DueOn: "2026-12-01",
		Comments: []Comment{{SourceID: "act1", Body: "Looks good", Author: Person{Username: "bob", Name: "Bob Roe"}, At: time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)}},
		Files:    []string{upload},
	}, sink.tickets[0])
	assert.Equal(t, StatusRef{Name: "Done", Kind: StatusDone}, sink.tickets[1].Status)
	assert.Equal(t, StatusRef{Name: "Doing", Kind: StatusProgress}, sink.tickets[2].Status)

	resumed := &recordingSink{}
	require.NoError(t, src.Import(t.Context(), Job{From: "abc", Credential: trelloCred, Cursor: "c2"}, resumed))
	require.Len(t, resumed.tickets, 1)
	assert.Equal(t, "c1", resumed.tickets[0].SourceID)
}

func TestTrelloSource_SignsDownloadsFromTrelloAlone(t *testing.T) {
	sign := NewTrelloSource(TrelloAPI, nil).Info().FileHeader
	u, _ := url.Parse("https://trello.com/1/cards/c/attachments/a/download/f.png")
	assert.Equal(t, `OAuth oauth_consumer_key="k3y", oauth_token="t0ken"`, sign(trelloCred, u).Get("Authorization"))
	other, _ := url.Parse("https://trello-attachments.s3.amazonaws.com/f.png")
	assert.Nil(t, sign(trelloCred, other))
}

func TestTrelloSource_Refusals(t *testing.T) {
	boards := []map[string]string{{"id": "b1", "name": "One", "shortLink": "aaa"}, {"id": "b2", "name": "Two", "shortLink": "bbb"}}
	tests := []struct {
		name   string
		cred   string
		answer func(r *http.Request) (int, http.Header, any)
		want   error
		says   string
	}{
		{"no token", "just-a-key", nil, apperrs.ErrInvalid, "key:token"},
		{"a refused credential", trelloCred, func(*http.Request) (int, http.Header, any) { return 401, nil, map[string]string{} }, apperrs.ErrUnauthorized, "refused the credential"},
		{"several boards and none named", trelloCred, func(*http.Request) (int, http.Header, any) { return 200, nil, boards }, apperrs.ErrInvalid, "One (aaa), Two (bbb)"},
		{"a limit that keeps refusing", trelloCred, func(*http.Request) (int, http.Header, any) { return 429, nil, nil }, apperrs.ErrRetryable, "rate limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer := tt.answer
			if answer == nil {
				answer = func(*http.Request) (int, http.Header, any) { return 200, nil, nil }
			}
			stub := newStubAPI(t, answer)
			src := NewTrelloSource(stub.srv.URL, stub.srv.Client())
			src.api.sleep = (&sleeps{}).sleep
			_, err := src.Scan(t.Context(), Job{Credential: tt.cred})
			assert.ErrorIs(t, err, tt.want)
			assert.ErrorContains(t, err, tt.says)
			assert.False(t, strings.Contains(err.Error(), "t0ken"), "no refusal repeats the credential")
		})
	}
}
