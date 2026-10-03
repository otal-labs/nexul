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

// fakeSourceDocs knows the readable doc ids; any other id fails with err, or not found.
type fakeSourceDocs struct {
	readable map[string]bool
	err      error
}

func (f fakeSourceDocs) RequireReadable(_ context.Context, docID string) error {
	if f.readable[docID] {
		return nil
	}
	if f.err != nil {
		return f.err
	}
	return apperrs.ErrNotFound
}

func newSourceService(repo *fakeRepo) *Service {
	s := newTestService(repo)
	s.SetSourceDocs(fakeSourceDocs{readable: map[string]bool{"doc-1": true, "doc-2": true}})
	return s
}

func TestSetSource(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		_, err := newSourceService(newFakeRepo()).SetSource(t.Context(), " ", "doc-1")
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		_, err := newSourceService(newFakeRepo()).SetSource(t.Context(), "nope", "doc-1")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("a doc the caller cannot read is refused and nothing changes", func(t *testing.T) {
		for _, docErr := range []error{apperrs.ErrNotFound, apperrs.ErrForbidden} {
			repo := newFakeRepo()
			s := newTestService(repo)
			s.SetSourceDocs(fakeSourceDocs{err: docErr})
			created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
			require.NoError(t, err)

			_, err = s.SetSource(t.Context(), created.ID, "doc-9")
			assert.ErrorIs(t, err, docErr)
			assert.Empty(t, repo.tickets[created.ID].DocID)
			assert.Empty(t, repo.eventsFor(TopicUpdated))
		}
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		s := newSourceService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		repo.updateErr = errors.New("db down")
		_, err = s.SetSource(t.Context(), created.ID, "doc-1")
		assert.ErrorIs(t, err, repo.updateErr)
	})
	t.Run("sets, replaces, and clears with a ticket.updated each time", func(t *testing.T) {
		repo := newFakeRepo()
		s := newSourceService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)

		got, err := s.SetSource(t.Context(), created.ID, " doc-1 ")
		require.NoError(t, err)
		assert.Equal(t, "doc-1", got.DocID)
		assert.Equal(t, "doc-1", repo.tickets[created.ID].DocID)

		got, err = s.SetSource(t.Context(), created.ID, "doc-2")
		require.NoError(t, err)
		assert.Equal(t, "doc-2", got.DocID)

		got, err = s.SetSource(t.Context(), created.ID, "")
		require.NoError(t, err)
		assert.Empty(t, got.DocID)
		evts := repo.eventsFor(TopicUpdated)
		require.Len(t, evts, 3)
		assert.Empty(t, evts[2].Payload.(UpdatedEvent).Ticket.DocID)
	})
	t.Run("the same doc is a no-op without an event", func(t *testing.T) {
		repo := newFakeRepo()
		s := newSourceService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "doc-1", "")
		require.NoError(t, err)
		_, err = s.SetSource(t.Context(), created.ID, "doc-1")
		require.NoError(t, err)
		assert.Empty(t, repo.eventsFor(TopicUpdated))
	})
}

func TestCreate_WithUnreadableSourceDoc_IsRefused(t *testing.T) {
	repo := newFakeRepo()
	_, err := newSourceService(repo).Create(t.Context(), "p-1", "ticket", "", "doc-9", "")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Empty(t, repo.tickets)
}

func TestTicketsHandler_SetSource(t *testing.T) {
	repo := newFakeRepo()
	s := newSourceService(repo)
	created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
	require.NoError(t, err)
	h := NewHandler(s).Routes()

	rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/source", `{"doc_id":"doc-9"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/source", `{"doc_id":"doc-1"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "doc-1", decodeTicket(t, rec).DocID)

	rec = serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/source", `{"doc_id":""}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, decodeTicket(t, rec).DocID)
}
