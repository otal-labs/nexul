package docs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func (f *fakeRepo) ListClarificationRounds(_ context.Context, docID string) ([]*ClarificationRound, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.clarifyErr != nil {
		return nil, f.clarifyErr
	}
	out := []*ClarificationRound{}
	for _, r := range f.rounds {
		if r.DocID == docID {
			cp := *r
			out = append(out, &cp)
		}
	}
	slices.SortFunc(out, func(a, b *ClarificationRound) int { return a.Round - b.Round })
	return out, nil
}

func (f *fakeRepo) ListClarificationQuestions(_ context.Context, docID string) ([]*ClarificationQuestion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*ClarificationQuestion{}
	for _, q := range f.questions {
		if q.DocID == docID {
			cp := *q
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetClarificationQuestion(_ context.Context, id string) (*ClarificationQuestion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, q := range f.questions {
		if q.ID == id {
			cp := *q
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) CreateClarificationRound(_ context.Context, r *ClarificationRound, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if slices.ContainsFunc(f.rounds, func(o *ClarificationRound) bool { return o.DocID == r.DocID && o.Round == r.Round }) {
		return apperrs.ErrConflict
	}
	cp := *r
	f.rounds = append(f.rounds, &cp)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) SaveClarification(_ context.Context, rounds []*ClarificationRound, questions []*ClarificationQuestion, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.clarifyErr != nil {
		return f.clarifyErr
	}
	for _, r := range rounds {
		i := slices.IndexFunc(f.rounds, func(o *ClarificationRound) bool { return o.DocID == r.DocID && o.Round == r.Round })
		if i < 0 {
			return apperrs.ErrNotFound
		}
		cp := *r
		cp.Questions = nil
		f.rounds[i] = &cp
	}
	for _, q := range questions {
		cp := *q
		f.questions = append(f.questions, &cp)
	}
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) DeleteClarificationRound(_ context.Context, docID string, round int, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rounds = slices.DeleteFunc(f.rounds, func(r *ClarificationRound) bool { return r.DocID == docID && r.Round == round })
	f.questions = slices.DeleteFunc(f.questions, func(q *ClarificationQuestion) bool { return q.DocID == docID && q.Round == round })
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) SaveClarificationAnswer(_ context.Context, q *ClarificationQuestion, roundAnswered *eventbus.OutboxEvent, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.clarifyErr != nil {
		return f.clarifyErr
	}
	i := slices.IndexFunc(f.questions, func(o *ClarificationQuestion) bool { return o.ID == q.ID })
	if i < 0 {
		return apperrs.ErrNotFound
	}
	cp := *q
	f.questions[i] = &cp
	f.events = append(f.events, evts...)
	pending := slices.ContainsFunc(f.questions, func(o *ClarificationQuestion) bool {
		return o.DocID == q.DocID && o.Round == q.Round && o.Pending()
	})
	if roundAnswered != nil && !pending {
		f.events = append(f.events, *roundAnswered)
	}
	return nil
}

const (
	starter = "user-1"
	client  = "user-2"
)

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

type fakeClarifyGate bool

func (g fakeClarifyGate) CanRunClarify(context.Context, string, string) bool { return bool(g) }

var (
	readOnly  = []permissions.Action{permissions.DocsRead}
	readWrite = []permissions.Action{permissions.DocsRead, permissions.DocsWrite}
)

// clarifyService answers for one person holding exactly allow on every doc, with or without the Clarify play.
func clarifyService(repo *fakeRepo, runsClarify bool, allow ...permissions.Action) *Service {
	s := NewService(repo, fakeAccess{allow: allow}, nil)
	s.now = func() time.Time { return fixedNow }
	s.SetClarifyGate(fakeClarifyGate(runsClarify))
	return s
}

// postedRound opens a round as starter and posts two questions, returning the doc and the questions' ids.
func postedRound(t *testing.T, repo *fakeRepo, s *Service) (*Doc, []string) {
	t.Helper()
	d := mustDoc(t, newTestService(repo), "project-1", "Spec", "Client wants a portal.")
	_, err := s.OpenRound(context.Background(), d.ID, starter, "trail-1", false)
	require.NoError(t, err)
	require.NoError(t, s.PostRound(as(starter), d.ID, []ClarificationQuestion{
		{Question: "Who signs in?", Why: "Decides the login.", Options: []QuestionOption{{Label: "Staff"}, {Label: "Customers"}}},
		{Question: "Which devices?"},
	}, ""))
	require.NoError(t, s.EndRound(context.Background(), d.ID, "trail-1"))
	var qids []string
	for _, q := range repo.questions {
		qids = append(qids, q.ID)
	}
	return d, qids
}

func TestClarification_Refusals(t *testing.T) {
	repo := newFakeRepo()
	writer := clarifyService(repo, false, readWrite...)
	d, qids := postedRound(t, repo, writer)
	reader := clarifyService(repo, true, readOnly...)
	outsider := NewService(repo, fakeAccess{projectErr: apperrs.ErrNotFound}, nil)

	tests := []struct {
		name string
		call func() error
		want error
	}{
		{"someone who cannot open the doc's project does not see the questions", func() error {
			_, err := outsider.Clarification(as(client), d.ID)
			return err
		}, apperrs.ErrNotFound},
		{"nor answers them", func() error {
			_, err := outsider.AnswerQuestion(as(client), d.ID, qids[0], Answer{Text: "Staff"})
			return err
		}, apperrs.ErrNotFound},
		{"a reader cannot answer", func() error {
			_, err := reader.AnswerQuestion(as(client), d.ID, qids[0], Answer{Text: "Staff"})
			return err
		}, apperrs.ErrForbidden},
		{"nor clear an answer", func() error {
			_, err := reader.ClearAnswer(as(client), d.ID, qids[0])
			return err
		}, apperrs.ErrForbidden},
		{"nor write Anything else?", func() error {
			_, err := reader.SaveAnythingElse(as(client), d.ID, 1, "Also billing")
			return err
		}, apperrs.ErrForbidden},
		{"nor close, even running plays", func() error {
			_, err := reader.CloseClarification(as(client), d.ID)
			return err
		}, apperrs.ErrForbidden},
		{"a writer who cannot run the Clarify play cannot close", func() error {
			_, err := writer.CloseClarification(as(client), d.ID)
			return err
		}, apperrs.ErrForbidden},
		{"posting with no round running is refused", func() error {
			return writer.PostRound(as(starter), d.ID, []ClarificationQuestion{{Question: "More?"}}, "")
		}, apperrs.ErrConflict},
		{"a no-gaps rewrite with no round running is refused", func() error {
			_, err := writer.WriteNoGaps(as(starter), d.ID, "New body")
			return err
		}, apperrs.ErrConflict},
		{"an answer needs a signed-in person", func() error {
			_, err := writer.AnswerQuestion(context.Background(), d.ID, qids[0], Answer{Text: "Staff"})
			return err
		}, apperrs.ErrUnauthorized},
		{"a question of another doc is not found", func() error {
			other := mustDoc(t, newTestService(repo), "project-1", "Other", "")
			_, err := writer.AnswerQuestion(as(client), other.ID, qids[0], Answer{Text: "Staff"})
			return err
		}, apperrs.ErrNotFound},
		{"an empty answer is invalid", func() error {
			_, err := writer.AnswerQuestion(as(client), d.ID, qids[0], Answer{Text: "  "})
			return err
		}, apperrs.ErrInvalid},
		{"a missing round's Anything else? is not found", func() error {
			_, err := writer.SaveAnythingElse(as(client), d.ID, 7, "x")
			return err
		}, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, tt.call(), tt.want)
		})
	}
}

func TestPostRound_OnlyTheRunningRoundsStarterPostsItOnce(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, true, readWrite...)
	d := mustDoc(t, newTestService(repo), "project-1", "Spec", "body")
	round, err := s.OpenRound(context.Background(), d.ID, starter, "trail-1", false)
	require.NoError(t, err)
	assert.Equal(t, 1, round)

	err = s.PostRound(as(client), d.ID, []ClarificationQuestion{{Question: "Who signs in?"}}, "")
	require.ErrorIs(t, err, apperrs.ErrForbidden, "someone else posting into the running round")
	for name, qs := range map[string][]ClarificationQuestion{
		"a blank question":        {{Question: " "}},
		"a question asked twice":  {{Question: "Who?"}, {Question: "Who?"}},
		"an option with no label": {{Question: "Who?", Options: []QuestionOption{{Label: " "}}}},
		"nothing at all":          nil,
	} {
		require.ErrorIs(t, s.PostRound(as(starter), d.ID, qs, ""), apperrs.ErrInvalid, name)
	}
	require.ErrorIs(t, s.PostRound(as(starter), d.ID, nil, "Yes"), apperrs.ErrInvalid, "round 1 has no Anything else? before it")

	require.NoError(t, s.PostRound(as(starter), d.ID, []ClarificationQuestion{{Question: " Who signs in? ", Options: []QuestionOption{{Label: "Staff"}}}}, ""))
	posted := repo.eventsFor(TopicClarificationRoundPosted)
	require.Len(t, posted, 1)
	assert.Equal(t, 1, posted[0].Payload.(ClarificationPostedEvent).QuestionCount)
	require.ErrorIs(t, s.PostRound(as(starter), d.ID, []ClarificationQuestion{{Question: "More?"}}, ""), apperrs.ErrConflict, "a round posts once")

	c, err := s.Clarification(as(client), d.ID)
	require.NoError(t, err)
	require.Len(t, c.Rounds, 1)
	assert.True(t, c.Running)
	assert.Equal(t, "Who signs in?", c.Rounds[0].Questions[0].Question)
	assert.True(t, c.Rounds[0].Questions[0].Pending())
}

func TestAnswers_SaveOnALockedDocAndTellTheStarterOnce(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, false, readWrite...)
	d, qids := postedRound(t, repo, s)
	repo.docs[d.ID].Locked = true

	saved, err := s.AnswerQuestion(as(client), d.ID, qids[0], Answer{Selected: []string{" Staff ", ""}, Text: " and admins "})
	require.NoError(t, err, "a locked doc still takes answers")
	assert.Equal(t, []string{"Staff"}, saved.Selected)
	assert.Equal(t, "and admins", saved.Text)
	assert.Equal(t, client, saved.AnsweredBy)
	assert.Empty(t, repo.eventsFor(TopicClarificationRoundAnswered), "one question is still pending")

	_, err = s.AnswerQuestion(as(client), d.ID, qids[1], Answer{Skipped: true, Text: "ignored"})
	require.NoError(t, err)
	answered := repo.eventsFor(TopicClarificationRoundAnswered)
	require.Len(t, answered, 1)
	assert.Equal(t, starter, answered[0].Payload.(ClarificationRoundEvent).StartedBy, "names whom the last answer is for")

	_, err = s.AnswerQuestion(as(client), d.ID, qids[1], Answer{Text: "Phones"})
	require.NoError(t, err)
	assert.Len(t, repo.eventsFor(TopicClarificationRoundAnswered), 1, "changing an answer is not the last answer again")

	cleared, err := s.ClearAnswer(as(client), d.ID, qids[1])
	require.NoError(t, err)
	assert.True(t, cleared.Pending())
	assert.Empty(t, cleared.AnsweredBy)
	saves := repo.eventsFor(TopicClarificationAnswerSaved)
	require.Len(t, saves, 3)
	assert.Equal(t, "Who signs in?", saves[0].Payload.(ClarificationAnswerEvent).Question)
	assert.Len(t, repo.eventsFor(TopicClarificationAnswerCleared), 1)
	_, err = s.AnswerQuestion(as(client), d.ID, qids[1], Answer{Text: "Phones"})
	require.NoError(t, err)
	assert.Len(t, repo.eventsFor(TopicClarificationRoundAnswered), 2, "answering it again after a clear is the last answer again")
}

func TestWriteNoGaps_GoesThroughTheRunsLockForItsStarterOnly(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, true, readWrite...)
	d := mustDoc(t, newTestService(repo), "project-1", "Spec", "Client wants a portal.")
	tookLock, err := s.LockForPlay(context.Background(), d.ID)
	require.NoError(t, err)
	_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-1", tookLock)
	require.NoError(t, err)

	_, err = s.WriteNoGaps(as(client), d.ID, "Rewritten")
	require.ErrorIs(t, err, apperrs.ErrForbidden, "someone else's rewrite stays refused")
	_, err = s.Update(as(starter), d.ID, "Spec", "Rewritten")
	require.ErrorIs(t, err, apperrs.ErrConflict, "a plain edit stays refused, even from the starter")
	_, err = clarifyService(repo, true, readOnly...).WriteNoGaps(as(starter), d.ID, "Rewritten")
	require.ErrorIs(t, err, apperrs.ErrForbidden, "the rewrite is an edit, so it takes docs:write")

	written, err := s.WriteNoGaps(as(starter), d.ID, "Rewritten")
	require.NoError(t, err)
	assert.Equal(t, 2, written.Version)
	assert.True(t, repo.docs[d.ID].Locked, "the lock stays until the run ends")
	posted := repo.eventsFor(TopicClarificationRoundPosted)
	require.Len(t, posted, 1)
	assert.True(t, posted[0].Payload.(ClarificationPostedEvent).NoGaps)
	_, err = s.WriteNoGaps(as(starter), d.ID, "Again")
	require.ErrorIs(t, err, apperrs.ErrConflict, "the verdict comes once")
	require.ErrorIs(t, s.PostRound(as(starter), d.ID, []ClarificationQuestion{{Question: "More?"}}, ""), apperrs.ErrConflict)

	require.NoError(t, s.EndRound(context.Background(), d.ID, "trail-1"))
	assert.False(t, repo.docs[d.ID].Locked, "the run's own lock comes off at its end")
	c, err := s.Clarification(as(starter), d.ID)
	require.NoError(t, err)
	require.Len(t, c.Rounds, 1, "a round that found no gaps stays")
	assert.NotNil(t, c.Rounds[0].NoGapsAt)
	c, err = clarifyService(repo, false, readWrite...).Clarification(as(client), d.ID)
	require.NoError(t, err)
	assert.Nil(t, c.Rounds[0].NoGapsAt, "the no-gaps signal shows only to people who may close")
}

func TestOpenAndEndRound(t *testing.T) {
	t.Run("a doc locked before the run stays locked, and an empty round is removed", func(t *testing.T) {
		repo := newFakeRepo()
		s := clarifyService(repo, true, readWrite...)
		d := mustDoc(t, newTestService(repo), "project-1", "Spec", "body")
		repo.docs[d.ID].Locked = true
		_, err := s.OpenRound(context.Background(), d.ID, starter, "trail-1", false)
		require.NoError(t, err)
		_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-2", false)
		require.ErrorIs(t, err, apperrs.ErrConflict, "one round runs at a time")

		require.NoError(t, s.EndRound(context.Background(), d.ID, "trail-1"))
		assert.True(t, repo.docs[d.ID].Locked)
		assert.Empty(t, repo.rounds, "a failed run leaves no empty round behind")
		ended := repo.eventsFor(TopicClarificationRoundEnded)
		require.Len(t, ended, 1)
		assert.True(t, ended[0].Payload.(ClarificationRoundEvent).Removed)
		require.NoError(t, s.EndRound(context.Background(), d.ID, "trail-1"), "ending again does nothing")
		assert.Len(t, repo.eventsFor(TopicClarificationRoundEnded), 1)
	})
	t.Run("a posted round stays, stops running, and the next one is numbered after it", func(t *testing.T) {
		repo := newFakeRepo()
		s := clarifyService(repo, true, readWrite...)
		d, _ := postedRound(t, repo, s)
		require.Len(t, repo.rounds, 1)
		assert.False(t, repo.rounds[0].Running)
		round, err := s.OpenRound(context.Background(), d.ID, starter, "trail-2", false)
		require.NoError(t, err)
		assert.Equal(t, 2, round)
	})
	t.Run("a missing doc is not found", func(t *testing.T) {
		s := clarifyService(newFakeRepo(), true, readWrite...)
		_, err := s.OpenRound(context.Background(), "nope", starter, "trail-1", false)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		require.ErrorIs(t, s.EndRound(context.Background(), "nope", "trail-1"), apperrs.ErrNotFound)
	})
}

func TestAnythingElse_AndItsReplyFromTheNextRound(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, true, readWrite...)
	d, _ := postedRound(t, repo, s)

	_, err := s.SaveAnythingElse(as(client), d.ID, 1, strings.Repeat("x", maxClarifyTextChars+1))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	saved, err := s.SaveAnythingElse(as(client), d.ID, 1, " Can it do billing? ")
	require.NoError(t, err)
	assert.Equal(t, "Can it do billing?", saved.AnythingElse)
	assert.Equal(t, client, saved.AnythingElseBy)
	assert.Len(t, repo.eventsFor(TopicClarificationAnythingElseSaved), 1)

	_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-2", false)
	require.NoError(t, err)
	_, err = s.SaveAnythingElse(as(client), d.ID, 2, "x")
	require.ErrorIs(t, err, apperrs.ErrConflict, "a round still being written has no box yet")
	require.NoError(t, s.PostRound(as(starter), d.ID, nil, "Billing is out of scope for now."))
	require.ErrorIs(t, s.PostRound(as(starter), d.ID, nil, "Again"), apperrs.ErrConflict, "the reply comes once")
	c, err := s.Clarification(as(client), d.ID)
	require.NoError(t, err)
	assert.Equal(t, "Billing is out of scope for now.", c.Rounds[0].AnythingElseReply)

	cleared, err := s.SaveAnythingElse(as(client), d.ID, 1, "")
	require.NoError(t, err)
	assert.Empty(t, cleared.AnythingElseBy)
	assert.Nil(t, cleared.AnythingElseAt)
}

func TestCloseClarification(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, true, readWrite...)
	empty := mustDoc(t, newTestService(repo), "project-1", "Empty", "")
	_, err := s.CloseClarification(as(starter), empty.ID)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "nothing to close")

	d, _ := postedRound(t, repo, s)
	_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-2", false)
	require.NoError(t, err)
	_, err = s.CloseClarification(as(starter), d.ID)
	require.ErrorIs(t, err, apperrs.ErrConflict, "not while a round runs")
	require.NoError(t, s.EndRound(context.Background(), d.ID, "trail-2"))

	c, err := s.CloseClarification(as(starter), d.ID)
	require.NoError(t, err)
	assert.True(t, c.Closed)
	assert.True(t, c.CanClose)
	_, err = s.CloseClarification(as(starter), d.ID)
	require.NoError(t, err)
	assert.Len(t, repo.eventsFor(TopicClarificationClosed), 1, "closing a closed clarification changes nothing")

	_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-3", false)
	require.NoError(t, err)
	c, err = s.Clarification(as(starter), d.ID)
	require.NoError(t, err)
	assert.False(t, c.Closed, "a new round reopens it")
}

func TestClarification_RepoFailuresSurface(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, true, readWrite...)
	d, qids := postedRound(t, repo, s)
	repo.clarifyErr = errors.New("disk full")

	_, err := s.Clarification(as(client), d.ID)
	require.ErrorIs(t, err, repo.clarifyErr)
	_, err = s.ClearAnswer(as(client), d.ID, qids[0])
	require.ErrorIs(t, err, repo.clarifyErr)
	_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-2", false)
	require.ErrorIs(t, err, repo.clarifyErr)
}

func TestDocTools_Clarification(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, true, readWrite...)
	d := mustDoc(t, newTestService(repo), "project-1", "Spec", "Client wants a portal.")
	tookLock, err := s.LockForPlay(context.Background(), d.ID)
	require.NoError(t, err)
	_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-1", tookLock)
	require.NoError(t, err)
	call := func(ctx context.Context, args string) (docResult, error) {
		out, err := callTool(ctx, t, s, "doc_update", `{"id":"`+d.ID+`",`+args+`}`)
		if err != nil {
			return docResult{}, err
		}
		return out.(docResult), nil
	}

	_, err = call(as(client), `"questions":[{"question":"Who signs in?"}]`)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "only the running round's starter posts it")
	_, err = call(as(starter), `"no_gaps":true`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "no_gaps comes with the body")
	_, err = call(as(starter), `"no_gaps":true,"body":"x","title":"y"`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	got, err := call(as(starter), `"questions":[{"question":"Who signs in?","why":"Decides the login.","options":[{"label":"Staff"},{"label":"Customers"}],"multi_select":true}]`)
	require.NoError(t, err)
	require.Len(t, got.Clarification.Rounds, 1)
	q := got.Clarification.Rounds[0].Questions[0]
	assert.Equal(t, []QuestionOption{{Label: "Staff"}, {Label: "Customers"}}, q.Options)
	assert.True(t, q.MultiSelect)
	require.NoError(t, s.EndRound(context.Background(), d.ID, "trail-1"))

	got, err = call(as(client), `"answers":[{"question_id":"`+q.ID+`","selected":["Staff"]}]`)
	require.NoError(t, err)
	assert.Equal(t, []string{"Staff"}, got.Clarification.Rounds[0].Questions[0].Selected)
	got, err = call(as(client), `"answers":[{"question_id":"`+q.ID+`","clear":true}]`)
	require.NoError(t, err)
	assert.True(t, got.Clarification.Rounds[0].Questions[0].Pending())
	_, err = call(as(client), `"answers":[{"question_id":"nope","text":"x"}]`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = call(as(client), `"title":"Renamed","answers":[{"question_id":"nope","text":"x"}]`)
	var partial *mcptool.PartialError
	require.ErrorAs(t, err, &partial, "a refused answer says the title was saved")

	_, err = call(as(starter), `"clarification_closed":false`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "reopening is a new round")
	got, err = call(as(starter), `"clarification_closed":true`)
	require.NoError(t, err)
	assert.True(t, got.Clarification.Closed)

	_, err = s.OpenRound(context.Background(), d.ID, starter, "trail-2", false)
	require.NoError(t, err)
	_, err = s.SaveAnythingElse(as(client), d.ID, 1, "Billing?")
	require.NoError(t, err)
	_, err = s.LockForPlay(context.Background(), d.ID)
	require.NoError(t, err)
	got, err = call(as(starter), `"no_gaps":true,"body":"# Portal\n\nStaff sign in.","anything_else_reply":"Billing is out of scope."`)
	require.NoError(t, err, "the no-gaps body goes through the lock")
	assert.Equal(t, "# Portal\n\nStaff sign in.", got.Body)
	assert.True(t, got.Locked)
	assert.Equal(t, "Billing is out of scope.", got.Clarification.Rounds[0].AnythingElseReply)
	assert.NotNil(t, got.Clarification.Rounds[1].NoGapsAt)
}

func TestDocsHandler_Clarification(t *testing.T) {
	repo := newFakeRepo()
	s := clarifyService(repo, true, readWrite...)
	d, qids := postedRound(t, repo, s)
	h := NewHandler(s).Routes()
	base := "/api/docs/" + d.ID + "/clarification"

	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPut, base+"/questions/"+qids[0], `{"text":" "}`).Code)
	assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodPut, base+"/questions/nope", `{"text":"x"}`).Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPut, base+"/rounds/zero/anything-else", `{"text":"x"}`).Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPut, base+"/rounds/1/anything-else", `{`).Code)
	assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodDelete, base+"/questions/nope", "").Code)

	require.Equal(t, http.StatusOK, serve(t, h, http.MethodPut, base+"/questions/"+qids[0], `{"selected":["Staff"]}`).Code)
	require.Equal(t, http.StatusOK, serve(t, h, http.MethodDelete, base+"/questions/"+qids[0], "").Code)
	require.Equal(t, http.StatusOK, serve(t, h, http.MethodPut, base+"/rounds/1/anything-else", `{"text":"Billing?"}`).Code)
	require.Equal(t, http.StatusOK, serve(t, h, http.MethodPost, base+"/close", "").Code)
	assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodPost, "/api/docs/nope/clarification/close", "").Code)

	rec := serve(t, h, http.MethodGet, base, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var c Clarification
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &c))
	assert.True(t, c.Closed)
	assert.Equal(t, "Billing?", c.Rounds[0].AnythingElse)
	assert.True(t, c.Rounds[0].Questions[0].Pending())
	assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodGet, "/api/docs/nope/clarification", "").Code)
}
