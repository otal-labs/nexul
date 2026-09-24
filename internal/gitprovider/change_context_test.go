package gitprovider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeChangeReader serves tickets per PR number and records which refs each project's decisions were asked for.
type fakeChangeReader struct {
	tickets     map[int][]ChangeTicket
	entries     map[string][]string
	ticketsErr  error
	entriesErr  error
	askedRefs   map[string][]string
	askedNumber int
}

func (f *fakeChangeReader) TicketsForPR(_ context.Context, _, _ string, number int) ([]ChangeTicket, error) {
	f.askedNumber = number
	return f.tickets[number], f.ticketsErr
}

func (f *fakeChangeReader) DecisionEntries(_ context.Context, projectID string, refs []string) ([]string, error) {
	if f.askedRefs == nil {
		f.askedRefs = map[string][]string{}
	}
	f.askedRefs[projectID] = refs
	return f.entries[projectID], f.entriesErr
}

func changeFixture() (*fakeProvider, *fakeChangeReader) {
	p := &fakeProvider{pr: &PR{Number: 7, Title: "Drop the cache"}, prs: []*PR{{Number: 7, Title: "Drop the cache"}}}
	r := &fakeChangeReader{
		tickets: map[int][]ChangeTicket{7: {
			{ID: "t-1", Key: "NEX-1", ProjectID: "p-1", Done: true, Doc: &ChangeDoc{ID: "d-1", Title: "Caching"}, BugsFound: []ChangeBug{{ID: "t-9", Key: "NEX-9"}}},
			{ID: "t-2", Key: "NEX-2", ProjectID: "p-1"},
			{ID: "t-3", Key: "OPS-3", ProjectID: "p-2"},
		}},
		entries: map[string][]string{"p-1": {"2026-09-24 — no cache"}, "p-2": {"2026-09-24 — ops"}},
	}
	return p, r
}

func TestGetChangeContext_FromAPRNumber_WalksTheChain(t *testing.T) {
	p, r := changeFixture()
	got, err := GetChangeContext(context.Background(), p, r, ChangeRef{Owner: "acme", Repo: "app", Number: 7})
	require.NoError(t, err)
	assert.Equal(t, 7, got.PR.Number)
	require.Len(t, got.Tickets, 3)
	assert.Equal(t, "Caching", got.Tickets[0].Doc.Title)
	assert.Equal(t, "NEX-9", got.Tickets[0].BugsFound[0].Key)
	assert.Equal(t, []string{"2026-09-24 — no cache", "2026-09-24 — ops"}, got.Decisions)
	assert.Equal(t, []string{"NEX-1", "/tickets/t-1", "NEX-2", "/tickets/t-2"}, r.askedRefs["p-1"])
}

func TestGetChangeContext_FromACommit_ResolvesItsPR(t *testing.T) {
	p, r := changeFixture()
	got, err := GetChangeContext(context.Background(), p, r, ChangeRef{Owner: "acme", Repo: "app", Commit: " abc123 "})
	require.NoError(t, err)
	assert.Equal(t, 7, got.PR.Number)
	assert.Equal(t, 7, r.askedNumber)
}

func TestGetChangeContext_PRWithNoTickets_ReturnsEmptyLists(t *testing.T) {
	p, _ := changeFixture()
	got, err := GetChangeContext(context.Background(), p, &fakeChangeReader{}, ChangeRef{Owner: "acme", Repo: "app", Number: 7})
	require.NoError(t, err)
	assert.NotNil(t, got.Tickets)
	assert.Empty(t, got.Tickets)
	assert.NotNil(t, got.Decisions)
}

func TestGetChangeContext_Errors(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(p *fakeProvider, r *fakeChangeReader)
		ref     ChangeRef
		want    error
	}{
		{"no repo", nil, ChangeRef{Owner: "acme", Number: 7}, apperrs.ErrInvalid},
		{"neither number nor commit", nil, ChangeRef{Owner: "acme", Repo: "app"}, apperrs.ErrInvalid},
		{"commit in no PR", func(p *fakeProvider, _ *fakeChangeReader) { p.prs = nil }, ChangeRef{Owner: "acme", Repo: "app", Commit: "abc"}, apperrs.ErrNotFound},
		{"provider fails on commit", func(p *fakeProvider, _ *fakeChangeReader) { p.err = apperrs.ErrUnauthorized }, ChangeRef{Owner: "acme", Repo: "app", Commit: "abc"}, apperrs.ErrUnauthorized},
		{"provider fails on PR", func(p *fakeProvider, _ *fakeChangeReader) { p.err = apperrs.ErrNotFound }, ChangeRef{Owner: "acme", Repo: "app", Number: 7}, apperrs.ErrNotFound},
		{"tickets unreadable", func(_ *fakeProvider, r *fakeChangeReader) { r.ticketsErr = errors.New("disk") }, ChangeRef{Owner: "acme", Repo: "app", Number: 7}, nil},
		{"decisions unreadable", func(_ *fakeProvider, r *fakeChangeReader) { r.entriesErr = errors.New("disk") }, ChangeRef{Owner: "acme", Repo: "app", Number: 7}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, r := changeFixture()
			if tt.arrange != nil {
				tt.arrange(p, r)
			}
			_, err := GetChangeContext(context.Background(), p, r, tt.ref)
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestGitHandler_ChangeContext(t *testing.T) {
	p, r := changeFixture()
	h := NewHandler(p).WithChangeContext(r).Routes()

	rec := serve(t, h, http.MethodGet, "/api/repos/acme/app/change-context?pr=7")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got ChangeContext
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Len(t, got.Tickets, 3)

	assert.Equal(t, http.StatusOK, serve(t, h, http.MethodGet, "/api/repos/acme/app/change-context?commit=abc").Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodGet, "/api/repos/acme/app/change-context?pr=x").Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodGet, "/api/repos/acme/app/change-context").Code)
	assert.Equal(t, http.StatusNotFound, serve(t, NewHandler(p).Routes(), http.MethodGet, "/api/repos/acme/app/change-context?pr=7").Code, "no reader wired, no route")
}

func TestChangeContextTools(t *testing.T) {
	p, r := changeFixture()
	tools := ChangeContextTools(p, r)
	require.Len(t, tools, 1)
	assert.Equal(t, "git_get_change_context", tools[0].Name)

	got, err := tools[0].Call(context.Background(), map[string]any{"owner": "acme", "repo": "app", "number": float64(7)})
	require.NoError(t, err)
	assert.Len(t, got.(*ChangeContext).Tickets, 3)

	_, err = tools[0].Call(context.Background(), map[string]any{"owner": "acme", "repo": "app", "commit": "abc"})
	require.NoError(t, err)
	_, err = tools[0].Call(context.Background(), map[string]any{"owner": "acme"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
