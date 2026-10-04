package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

const uDev = "u-dev"

// TestIntegration_Clarification walks a doc's clarification through the wired services: who sees, answers, posts,
// and closes it, and how a round's run holds the doc's lock.
func TestIntegration_Clarification(t *testing.T) {
	f := newPermFixture(t)
	restrictedClient(t, f)
	ctx := context.Background()
	now := time.Now()
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uDev, Provider: auth.ProviderGitHub, ProviderUserID: uDev, Login: uDev})
	require.NoError(t, err)
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-dev", WorkspaceID: wsDefault, Name: "Dev", Permissions: grant("docs:read", "docs:write", "plays:run"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uDev, WorkspaceID: wsDefault, RoleID: "role-dev", CreatedAt: now}))
	require.NoError(t, f.store.Access.Set(ctx, "doc", f.doc, uClient, grant("docs:read", "docs:write"), nil))

	s := f.svc.docsSvc
	tookLock, err := s.LockForPlay(ctx, f.doc)
	require.NoError(t, err)
	_, err = s.OpenRound(ctx, f.doc, uDev, "trail-1", tookLock)
	require.NoError(t, err)
	require.NoError(t, s.PostRound(as(uDev), f.doc, []docs.ClarificationQuestion{{Question: "Who signs in?"}, {Question: "Which devices?"}}, ""))
	c, err := s.Clarification(as(uDev), f.doc)
	require.NoError(t, err)
	qid := c.Rounds[0].Questions[0].ID

	t.Run("refusals", func(t *testing.T) {
		for name, tc := range map[string]struct {
			call func() error
			want string
		}{
			"a Restricted member without the doc's project does not see it": {func() error {
				_, err := s.Clarification(as(uClient), f.doc)
				return err
			}, notFound},
			"nor answers it, though the doc is shared to them": {func() error {
				_, err := s.AnswerQuestion(as(uClient), f.doc, qid, docs.Answer{Text: "Staff"})
				return err
			}, notFound},
			"a reader sees it": {func() error {
				_, err := s.Clarification(as(uReader), f.doc)
				return err
			}, ok},
			"but cannot answer": {func() error {
				_, err := s.AnswerQuestion(as(uReader), f.doc, qid, docs.Answer{Text: "Staff"})
				return err
			}, forbidden},
			"someone other than the round's starter cannot post into it": {func() error {
				return s.PostRound(as(uEditor), f.doc, nil, "A reply")
			}, forbidden},
			"nor write the doc through its lock": {func() error {
				_, err := s.WriteNoGaps(as(uEditor), f.doc, "Rewritten")
				return err
			}, forbidden},
			"an editor without plays:run cannot close": {func() error {
				_, err := s.CloseClarification(as(uEditor), f.doc)
				return err
			}, forbidden},
		} {
			assert.Equal(t, tc.want, outcome(tc.call()), name)
		}
	})

	t.Run("answers save on the run's locked doc, and the run's end unlocks it", func(t *testing.T) {
		_, err := s.AnswerQuestion(as(uEditor), f.doc, qid, docs.Answer{Selected: []string{"Staff"}})
		require.NoError(t, err)
		d, err := s.Get(as(uEditor), f.doc)
		require.NoError(t, err)
		assert.True(t, d.Locked)
		require.NoError(t, s.EndRound(ctx, f.doc, "trail-1"))
		d, err = s.Get(as(uEditor), f.doc)
		require.NoError(t, err)
		assert.False(t, d.Locked)
	})

	t.Run("the no-gaps body from the running round's starter goes through the lock", func(t *testing.T) {
		tookLock, err := s.LockForPlay(ctx, f.doc)
		require.NoError(t, err)
		_, err = s.OpenRound(ctx, f.doc, uDev, "trail-2", tookLock)
		require.NoError(t, err)
		_, err = s.Update(as(uDev), f.doc, "Spec", "Rewritten")
		assert.Equal(t, conflict, outcome(err), "a plain edit stays refused")
		written, err := s.WriteNoGaps(as(uDev), f.doc, "Rewritten")
		require.NoError(t, err)
		assert.True(t, written.Locked)
		require.NoError(t, s.EndRound(ctx, f.doc, "trail-2"))
	})

	t.Run("closing takes plays:run on the Clarify play, or workspace-wide while there is none", func(t *testing.T) {
		_, err := s.CloseClarification(as(uDev), f.doc)
		require.NoError(t, err)

		clarify := &plays.Play{ID: "play-clarify", WorkspaceID: wsDefault, Label: "Clarify via AI", Type: plays.TypeDoc, Enabled: true, BuiltinKey: clarifyPlayKey, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, f.store.Plays.Create(ctx, clarify))
		require.NoError(t, f.store.Access.Set(ctx, "play", clarify.ID, uDev, nil, grant("plays:run")))
		_, err = s.CloseClarification(as(uDev), f.doc)
		assert.Equal(t, forbidden, outcome(err), "excluded from the Clarify play")
		c, err := s.Clarification(as(uDev), f.doc)
		require.NoError(t, err)
		assert.False(t, c.CanClose)
		assert.Nil(t, c.Rounds[1].NoGapsAt, "nor sees the no-gaps signal")
		c, err = s.CloseClarification(as(uOwner), f.doc)
		require.NoError(t, err)
		assert.True(t, c.Closed)
		assert.NotNil(t, c.Rounds[1].NoGapsAt)
	})

	t.Run("live frames reach whoever reads the doc", func(t *testing.T) {
		a := liveAudience{access: f.svc.accessSvc}
		frame := docs.ClarificationAnswerEvent{Doc: docs.WatchedDoc{ID: f.doc, ProjectID: pGeneral}, Round: 1, QuestionID: qid, Question: "Who signs in?", AuthorID: uEditor}
		for user, want := range map[string]bool{uReader: true, uPlain: false, uClient: false, uOutsider: false} {
			assert.Equal(t, want, a.allows(as(user), docs.TopicClarificationAnswerSaved, frame), user)
		}
	})
}
