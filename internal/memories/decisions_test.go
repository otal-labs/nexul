package memories

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const logBody = "## Decisions\n\n" +
	"2026-09-01 — SQLite is the only store (superseded by [NEX-12](/tickets/t-12))  \nWhy: one file to back up.  \nTicket: [NEX-1](/tickets/t-1)\n\n" +
	"2026-09-20 — Logs go to the log store over OTLP  \nWhy: one sink.  \nTicket: [NEX-12](/tickets/t-12)\n\n" +
	"- 2026-09-22 — Plays run on the starter's harness, NEX-3\n" +
	"- 2026-09-23 — NEX-31 changed nothing here\n"

func TestCreateWithKind_DecisionsLog_DefaultsAndNeverAlwaysIncluded(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)

	m, err := s.CreateWithKind(testCtx(), KindDecisionsLog, "project-1", "", " ", "", "2026-09-24 — first", true, "mcp")
	require.NoError(t, err)
	assert.Equal(t, KindDecisionsLog, m.Kind)
	assert.Equal(t, decisionsLogTitle, m.Title)
	assert.Equal(t, decisionsLogWhenToUse, m.WhenToUse)
	assert.False(t, m.AlwaysIncluded)
	assert.Equal(t, "workspace-1", m.WorkspaceID)
	require.Len(t, repo.eventsFor(TopicCreated), 1)

	updated, err := s.Update(testCtx(), m.ID, m.Title, m.WhenToUse, "2026-09-24 — second", true, "mcp")
	require.NoError(t, err)
	assert.False(t, updated.AlwaysIncluded)
}

func TestCreateWithKind_Ordinary_CreatesAPlainMemory(t *testing.T) {
	s := newTestService(newFakeRepo())
	m, err := s.CreateWithKind(testCtx(), "", "project-1", "", "Notes", "", "body", true, "")
	require.NoError(t, err)
	assert.Empty(t, m.Kind)
	assert.True(t, m.AlwaysIncluded)
}

func TestCreateWithKind_Errors(t *testing.T) {
	existing := func() *Service {
		s := newTestService(newFakeRepo())
		_, err := s.CreateWithKind(testCtx(), KindDecisionsLog, "project-1", "", "", "", "x", false, "")
		if err != nil {
			panic(err)
		}
		return s
	}
	failing := func(set func(*fakeRepo)) func() *Service {
		return func() *Service {
			repo := newFakeRepo()
			set(repo)
			return newTestService(repo)
		}
	}
	tests := []struct {
		name    string
		svc     func() *Service
		ctx     context.Context
		kind    string
		project string
		want    error
	}{
		{"unknown kind", func() *Service { return newTestService(newFakeRepo()) }, testCtx(), KindInterview, "project-1", apperrs.ErrInvalid},
		{"no project", func() *Service { return newTestService(newFakeRepo()) }, testCtx(), KindDecisionsLog, " ", apperrs.ErrInvalid},
		{"no actor", func() *Service { return newTestService(newFakeRepo()) }, context.Background(), KindDecisionsLog, "project-1", apperrs.ErrUnauthorized},
		{"unknown project", func() *Service { return newTestService(newFakeRepo()) }, testCtx(), KindDecisionsLog, "nope", apperrs.ErrNotFound},
		{"no write permission", func() *Service { return newDenyService(newFakeRepo()) }, testCtx(), KindDecisionsLog, "project-1", apperrs.ErrForbidden},
		{"second log", existing, testCtx(), KindDecisionsLog, "project-1", apperrs.ErrConflict},
		{"lookup fails", failing(func(r *fakeRepo) { r.getErr = errors.New("disk") }), testCtx(), KindDecisionsLog, "project-1", nil},
		{"create fails", failing(func(r *fakeRepo) { r.createErr = errors.New("disk") }), testCtx(), KindDecisionsLog, "project-1", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc().CreateWithKind(tt.ctx, tt.kind, tt.project, "", "", "", "body", false, "")
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestDecisionEntriesCiting_ReturnsOnlyEntriesNamingTheTickets(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.CreateWithKind(testCtx(), KindDecisionsLog, "project-1", "", "", "", logBody, false, "")
	require.NoError(t, err)

	byLink, err := s.DecisionEntriesCiting(testCtx(), "project-1", []string{"/tickets/t-12"})
	require.NoError(t, err)
	require.Len(t, byLink, 2, "the superseded entry cites its replacement too")
	assert.Contains(t, byLink[0], "superseded by")
	assert.Contains(t, byLink[1], "OTLP")

	byKey, err := s.DecisionEntriesCiting(testCtx(), "project-1", []string{"NEX-3"})
	require.NoError(t, err)
	require.Len(t, byKey, 1, "NEX-3 never matches NEX-31")
	assert.Contains(t, byKey[0], "starter's harness")

	none, err := s.DecisionEntriesCiting(testCtx(), "project-1", []string{"", "NEX-99"})
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestDecisionEntriesCiting_NoLogYet_ReturnsNone(t *testing.T) {
	got, err := newTestService(newFakeRepo()).DecisionEntriesCiting(testCtx(), "project-1", []string{"NEX-1"})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NotNil(t, got)
}

func TestDecisionEntriesCiting_Errors(t *testing.T) {
	broken := newFakeRepo()
	broken.getErr = errors.New("disk")
	tests := []struct {
		name    string
		svc     *Service
		project string
		want    error
	}{
		{"no project", newTestService(newFakeRepo()), " ", apperrs.ErrInvalid},
		{"unknown project", newTestService(newFakeRepo()), "nope", apperrs.ErrNotFound},
		{"no read permission", newDenyService(newFakeRepo()), "project-1", apperrs.ErrForbidden},
		{"lookup fails", newTestService(broken), "project-1", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.DecisionEntriesCiting(testCtx(), tt.project, []string{"NEX-1"})
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestDecisionEntries_SplitsParagraphsAndListItems(t *testing.T) {
	got := decisionEntries("# Log\n\nfirst line\nsecond line\n\n- item one\n  continued\n1. item two\n")
	assert.Equal(t, []string{"first line\nsecond line", "- item one\ncontinued", "1. item two"}, got)
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		s, w string
		want bool
	}{
		{"see NEX-1.", "NEX-1", true},
		{"see NEX-12", "NEX-1", false},
		{"XNEX-1 then NEX-1", "NEX-1", true},
		{"[NEX-1](/tickets/abc)", "/tickets/abc", true},
		{"/tickets/abcd", "/tickets/abc", false},
		{"", "NEX-1", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			assert.Equal(t, tt.want, containsWord(tt.s, tt.w))
		})
	}
}
