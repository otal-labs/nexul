package tickets

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newBugService(t *testing.T) (*Service, *fakeRepo, *Ticket) {
	t.Helper()
	repo := newFakeRepo()
	s := newTestService(repo)
	s.SetTicketTypes(fakeTypeTemplates{names: map[string]string{"tt-bug": " Bug ", "tt-task": "task", "tt-defect": "defect"}})
	origin, err := s.Create(t.Context(), "p-1", "Login page", "", "", "")
	require.NoError(t, err)
	return s, repo, origin
}

func TestIsBugType(t *testing.T) {
	assert.True(t, IsBugType("bug"))
	assert.True(t, IsBugType(" BUG "))
	assert.False(t, IsBugType("bugs"))
	assert.False(t, IsBugType("defect"))
	assert.False(t, IsBugType(""))
}

func TestCreate_FoundInRule(t *testing.T) {
	tests := []struct {
		name    string
		opt     func(originID string) CreateOptions
		wantErr error
	}{
		{"bug without an origin is refused", func(string) CreateOptions { return CreateOptions{TypeID: "tt-bug"} }, apperrs.ErrInvalid},
		{"bug with origin and unknown is refused", func(o string) CreateOptions {
			return CreateOptions{TypeID: "tt-bug", OriginID: o, OriginUnknown: true}
		}, apperrs.ErrInvalid},
		{"bug with a missing origin is refused as invalid", func(string) CreateOptions {
			return CreateOptions{TypeID: "tt-bug", OriginID: "nope"}
		}, apperrs.ErrInvalid},
		{"unknown type is refused as invalid", func(string) CreateOptions { return CreateOptions{TypeID: "tt-gone"} }, apperrs.ErrInvalid},
		{"bug with an origin is filed", func(o string) CreateOptions { return CreateOptions{TypeID: "tt-bug", OriginID: o} }, nil},
		{"bug with origin unknown is filed", func(string) CreateOptions { return CreateOptions{TypeID: "tt-bug", OriginUnknown: true} }, nil},
		{"a task needs no origin", func(string) CreateOptions { return CreateOptions{TypeID: "tt-task"} }, nil},
		{"a renamed bug type is an ordinary type", func(string) CreateOptions { return CreateOptions{TypeID: "tt-defect"} }, nil},
		{"no type needs no origin", func(string) CreateOptions { return CreateOptions{} }, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _, origin := newBugService(t)
			_, err := s.Create(t.Context(), "p-1", "Crash", "", "", "", tt.opt(origin.ID))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCreate_BugWithOrigin_WritesFoundInLinkAndEvent(t *testing.T) {
	s, repo, origin := newBugService(t)
	bug, err := s.Create(t.Context(), "p-1", "Crash", "", "", "", CreateOptions{TypeID: "tt-bug", OriginID: " " + origin.ID + " "})
	require.NoError(t, err)

	links, err := s.Links(t.Context(), bug.ID)
	require.NoError(t, err)
	require.NotNil(t, links.FoundIn)
	assert.Equal(t, origin.ID, links.FoundIn.ID)

	originLinks, err := s.Links(t.Context(), origin.ID)
	require.NoError(t, err)
	require.Len(t, originLinks.BugsFound, 1)
	assert.Equal(t, bug.ID, originLinks.BugsFound[0].ID)

	topics := []string{}
	for _, e := range repo.events {
		topics = append(topics, e.Topic)
	}
	assert.Contains(t, topics, TopicCreated)
	assert.Contains(t, topics, TopicLinkCreated)
}

func TestCreate_BugOriginUnknown_RecordsMarker(t *testing.T) {
	s, _, _ := newBugService(t)
	bug, err := s.Create(t.Context(), "p-1", "Crash", "", "", "", CreateOptions{TypeID: "tt-bug", OriginUnknown: true})
	require.NoError(t, err)
	links, err := s.Links(t.Context(), bug.ID)
	require.NoError(t, err)
	assert.True(t, links.OriginUnknown)
	assert.Nil(t, links.FoundIn)
}

func TestCreate_FoundInWriteFails_NothingReturned(t *testing.T) {
	s, repo, origin := newBugService(t)
	repo.ticketLinkErr = errors.New("disk full")
	_, err := s.Create(t.Context(), "p-1", "Crash", "", "", "", CreateOptions{TypeID: "tt-bug", OriginID: origin.ID})
	require.Error(t, err)
}

func TestCreate_TypeLookupFails_Propagates(t *testing.T) {
	s := newTestService(newFakeRepo())
	boom := errors.New("db down")
	s.SetTicketTypes(fakeTypeTemplates{err: boom})
	_, err := s.Create(context.Background(), "p-1", "Crash", "", "", "", CreateOptions{TypeID: "tt-bug"})
	require.ErrorIs(t, err, boom)
}

func TestTicketsHandler_Create_BugNeedsOrigin(t *testing.T) {
	s, _, origin := newBugService(t)
	h := NewHandler(s).Routes()

	rec := serve(t, h, http.MethodPost, "/api/tickets", `{"title":"Crash","project_id":"p-1","type_id":"tt-bug"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, h, http.MethodPost, "/api/tickets", `{"title":"Crash","project_id":"p-1","type_id":"tt-bug","origin_id":"`+origin.ID+`"}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = serve(t, h, http.MethodPost, "/api/tickets", `{"title":"Crash","project_id":"p-1","type_id":"tt-bug","origin_unknown":true}`)
	require.Equal(t, http.StatusCreated, rec.Code)
}
