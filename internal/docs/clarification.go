package docs

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	maxClarifyLineChars = 1_000
	maxClarifyTextChars = 8_000
	maxRoundQuestions   = 50
	maxQuestionOptions  = 20
	maxPickedOptions    = 50
)

// ClarifyGate says whether a person may run the Clarify play of a project's workspace, the bit closing a
// clarification takes beside docs:write; docs never imports plays (ADR 0017).
type ClarifyGate interface {
	CanRunClarify(ctx context.Context, userID, projectID string) bool
}

// SetClarifyGate wires the Clarify play check; without it nobody may close a clarification.
func (s *Service) SetClarifyGate(g ClarifyGate) {
	s.clarify = g
}

// Clarification returns a doc's rounds with their questions and answers; requires docs:read.
func (s *Service) Clarification(ctx context.Context, docID string) (*Clarification, error) {
	d, err := s.Get(ctx, docID)
	if err != nil {
		return nil, err
	}
	return s.clarificationOf(ctx, d)
}

func (s *Service) clarificationOf(ctx context.Context, d *Doc) (*Clarification, error) {
	rounds, err := s.loadRounds(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	c := &Clarification{Rounds: rounds, CanClose: s.canClose(ctx, d)}
	for _, r := range rounds {
		c.Running = c.Running || r.Running
		if !c.CanClose {
			r.NoGapsAt = nil
		}
	}
	if n := len(rounds); n > 0 {
		c.Closed = rounds[n-1].ClosedAt != nil
	}
	return c, nil
}

// loadRounds reads the doc's rounds, oldest first, each holding its questions.
func (s *Service) loadRounds(ctx context.Context, docID string) ([]*ClarificationRound, error) {
	rounds, err := s.repo.ListClarificationRounds(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list clarification rounds of doc %s: %w", docID, err)
	}
	qs, err := s.repo.ListClarificationQuestions(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list clarification questions of doc %s: %w", docID, err)
	}
	byRound := make(map[int]*ClarificationRound, len(rounds))
	for _, r := range rounds {
		r.Questions = []*ClarificationQuestion{}
		byRound[r.Round] = r
	}
	for _, q := range qs {
		if r, ok := byRound[q.Round]; ok {
			r.Questions = append(r.Questions, q)
		}
	}
	return rounds, nil
}

func (s *Service) canClose(ctx context.Context, d *Doc) bool {
	if s.clarify == nil || !s.can(ctx, d.ID, permissions.DocsWrite) {
		return false
	}
	return s.clarify.CanRunClarify(ctx, actorID(ctx), d.ProjectID)
}

// AnswerQuestion saves picked options and text, or a skip, as the answer to one of the doc's questions, replacing any
// earlier one. Requires docs:write; a locked doc still takes answers, which are not edits to it.
func (s *Service) AnswerQuestion(ctx context.Context, docID, questionID string, a Answer) (*ClarificationQuestion, error) {
	if err := normalizeAnswer(&a); err != nil {
		return nil, err
	}
	return s.setAnswer(ctx, docID, questionID, a, TopicClarificationAnswerSaved)
}

// ClearAnswer makes one of the doc's questions pending again; requires docs:write.
func (s *Service) ClearAnswer(ctx context.Context, docID, questionID string) (*ClarificationQuestion, error) {
	return s.setAnswer(ctx, docID, questionID, Answer{Selected: []string{}}, TopicClarificationAnswerCleared)
}

func (s *Service) setAnswer(ctx context.Context, docID, questionID string, a Answer, topic string) (*ClarificationQuestion, error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.docFor(ctx, docID, permissions.DocsWrite)
	if err != nil {
		return nil, err
	}
	q, err := s.repo.GetClarificationQuestion(ctx, strings.TrimSpace(questionID))
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && q.DocID != d.ID) {
		return nil, fmt.Errorf("%w: question %s is not one of doc %s's; doc_get lists its questions", apperrs.ErrNotFound, questionID, d.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("get question %s: %w", questionID, err)
	}
	wasPending := q.Pending()
	q.Selected, q.Text, q.Skipped, q.AnsweredBy, q.AnsweredAt = a.Selected, a.Text, a.Skipped, actor, s.stamp()
	if q.Pending() {
		q.AnsweredBy, q.AnsweredAt = "", nil
	}
	ref := docRef(d)
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: ClarificationAnswerEvent{
		Doc: ref, Round: q.Round, QuestionID: q.ID, Question: q.Question, AuthorID: actor, At: s.now().UTC(),
	}}
	var answered *eventbus.OutboxEvent
	if wasPending && !q.Pending() {
		answered, err = s.roundAnsweredEvent(ctx, d, q.Round, actor)
		if err != nil {
			return nil, err
		}
	}
	if err := s.repo.SaveClarificationAnswer(ctx, q, answered, evt); err != nil {
		return nil, fmt.Errorf("save answer to question %s: %w", q.ID, err)
	}
	return q, nil
}

// roundAnsweredEvent names the round's starter, whom its last answer is for.
func (s *Service) roundAnsweredEvent(ctx context.Context, d *Doc, round int, actor string) (*eventbus.OutboxEvent, error) {
	r, err := s.roundOf(ctx, d.ID, round)
	if err != nil {
		return nil, err
	}
	return &eventbus.OutboxEvent{ID: ids.New(), Topic: TopicClarificationRoundAnswered, Payload: ClarificationRoundEvent{
		Doc: docRef(d), Round: r.Round, StartedBy: r.StartedBy, ActorID: actor,
	}}, nil
}

// SaveAnythingElse stores the text of a round's "Anything else?" box, or clears it when empty; requires docs:write.
func (s *Service) SaveAnythingElse(ctx context.Context, docID string, round int, text string) (*ClarificationRound, error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); n > maxClarifyTextChars {
		return nil, fmt.Errorf("%w: the text is %d characters, over the %d-character cap", apperrs.ErrInvalid, n, maxClarifyTextChars)
	}
	d, err := s.docFor(ctx, docID, permissions.DocsWrite)
	if err != nil {
		return nil, err
	}
	r, err := s.roundOf(ctx, d.ID, round)
	if err != nil {
		return nil, err
	}
	if r.Running {
		return nil, fmt.Errorf("%w: round %d is still being written; answer it once its questions are posted", apperrs.ErrConflict, round)
	}
	r.AnythingElse, r.AnythingElseBy, r.AnythingElseAt = text, actor, s.stamp()
	if text == "" {
		r.AnythingElseBy, r.AnythingElseAt = "", nil
	}
	evt := roundEvent(TopicClarificationAnythingElseSaved, d, r, actor)
	if err := s.repo.SaveClarification(ctx, []*ClarificationRound{r}, nil, evt); err != nil {
		return nil, fmt.Errorf("save anything else of doc %s round %d: %w", d.ID, round, err)
	}
	return r, nil
}

// CloseClarification closes the doc's clarification on its newest round; another round reopens it. Requires
// docs:write and plays:run on the Clarify play, and refuses while a round is running.
func (s *Service) CloseClarification(ctx context.Context, docID string) (*Clarification, error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.docFor(ctx, docID, permissions.DocsWrite)
	if err != nil {
		return nil, err
	}
	if !s.canClose(ctx, d) {
		return nil, fmt.Errorf("%w: closing a clarification takes %s on the Clarify via AI play", apperrs.ErrForbidden, permissions.PlaysRun)
	}
	rounds, err := s.repo.ListClarificationRounds(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("list clarification rounds of doc %s: %w", d.ID, err)
	}
	if len(rounds) == 0 {
		return nil, fmt.Errorf("%w: doc %s has no clarification to close; start one with Clarify via AI", apperrs.ErrInvalid, d.ID)
	}
	newest := rounds[len(rounds)-1]
	if newest.Running {
		return nil, fmt.Errorf("%w: round %d is still running; close the clarification once it ends", apperrs.ErrConflict, newest.Round)
	}
	if newest.ClosedAt == nil {
		newest.ClosedBy, newest.ClosedAt = actor, s.stamp()
		evt := roundEvent(TopicClarificationClosed, d, newest, actor)
		if err := s.repo.SaveClarification(ctx, []*ClarificationRound{newest}, nil, evt); err != nil {
			return nil, fmt.Errorf("close clarification of doc %s: %w", d.ID, err)
		}
	}
	return s.clarificationOf(ctx, d)
}

// PostRound stores the running round's questions, and the reply to the round before's "Anything else?", from the
// round's starter while it runs: the run's own agent acts as them. Either part may come alone, each once.
func (s *Service) PostRound(ctx context.Context, docID string, questions []ClarificationQuestion, reply string) error {
	reply = strings.TrimSpace(reply)
	if len(questions) == 0 && reply == "" {
		return fmt.Errorf("%w: post at least one question or an anything_else_reply", apperrs.ErrInvalid)
	}
	d, r, err := s.runningRoundOfCaller(ctx, docID)
	if err != nil {
		return err
	}
	saves, err := s.replyTo(ctx, d.ID, r, reply)
	if err != nil {
		return err
	}
	qs, err := s.newQuestions(r, questions)
	if err != nil {
		return err
	}
	var evts []eventbus.OutboxEvent
	if len(qs) > 0 {
		evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicClarificationRoundPosted, Payload: ClarificationPostedEvent{
			ClarificationRoundEvent: ClarificationRoundEvent{Doc: docRef(d), Round: r.Round, StartedBy: r.StartedBy, ActorID: r.StartedBy},
			QuestionCount:           len(qs),
		}})
	}
	if err := s.repo.SaveClarification(ctx, saves, qs, evts...); err != nil {
		return fmt.Errorf("post round %d of doc %s: %w", r.Round, d.ID, err)
	}
	return nil
}

// replyTo sets the reply on the round before r, which must have had an "Anything else?" and no reply yet.
func (s *Service) replyTo(ctx context.Context, docID string, r *ClarificationRound, reply string) ([]*ClarificationRound, error) {
	if reply == "" {
		return nil, nil
	}
	if n := utf8.RuneCountInString(reply); n > maxClarifyLineChars {
		return nil, fmt.Errorf("%w: anything_else_reply is %d characters, over the %d-character cap", apperrs.ErrInvalid, n, maxClarifyLineChars)
	}
	prev, err := s.roundOf(ctx, docID, r.Round-1)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && prev.AnythingElse == "") {
		return nil, fmt.Errorf("%w: the round before this one has no Anything else? to reply to", apperrs.ErrInvalid)
	}
	if err != nil {
		return nil, err
	}
	if prev.AnythingElseReply != "" {
		return nil, fmt.Errorf("%w: round %d's Anything else? already has its reply", apperrs.ErrConflict, prev.Round)
	}
	prev.AnythingElseReply = reply
	return []*ClarificationRound{prev}, nil
}

func (s *Service) newQuestions(r *ClarificationRound, questions []ClarificationQuestion) ([]*ClarificationQuestion, error) {
	if len(questions) == 0 {
		return nil, nil
	}
	if len(r.Questions) > 0 || r.NoGapsAt != nil {
		return nil, fmt.Errorf("%w: round %d is already posted; end the turn", apperrs.ErrConflict, r.Round)
	}
	if len(questions) > maxRoundQuestions {
		return nil, fmt.Errorf("%w: a round asks at most %d questions", apperrs.ErrInvalid, maxRoundQuestions)
	}
	out := make([]*ClarificationQuestion, 0, len(questions))
	for i, q := range questions {
		if err := normalizeQuestion(&q); err != nil {
			return nil, fmt.Errorf("question %d: %w", i+1, err)
		}
		if slices.ContainsFunc(out, func(o *ClarificationQuestion) bool { return o.Question == q.Question }) {
			return nil, fmt.Errorf("%w: question %d, %q, is asked twice", apperrs.ErrInvalid, i+1, q.Question)
		}
		q.ID, q.DocID, q.Round, q.Position = ids.New(), r.DocID, r.Round, i+1
		q.Selected, q.Text, q.Skipped, q.AnsweredBy, q.AnsweredAt = []string{}, "", false, "", nil
		out = append(out, &q)
	}
	return out, nil
}

// WriteNoGaps replaces the doc's body with the running round's no-gaps rewrite, through the lock its run holds
// (ADR 0121); only the round's starter may send it, holding docs:write, and only instead of questions.
func (s *Service) WriteNoGaps(ctx context.Context, docID, body string) (*Doc, error) {
	body, err := richtext.Normalize(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	d, r, err := s.runningRoundOfCaller(ctx, docID)
	if err != nil {
		return nil, err
	}
	if err := s.require(ctx, d.ID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	if len(r.Questions) > 0 || r.NoGapsAt != nil {
		return nil, fmt.Errorf("%w: round %d is already posted; end the turn", apperrs.ErrConflict, r.Round)
	}
	mentioned := richtext.AddedPersonMentions(d.Body, body)
	bodyChanged := body != d.Body
	d.Body = body
	d.Version++
	d.UpdatedAt = s.now().UTC()
	if err := s.persistUpdate(ctx, d, bodyChanged, mentioned); err != nil {
		return nil, fmt.Errorf("write doc %s: %w", d.ID, err)
	}
	r.NoGapsAt = s.stamp()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicClarificationRoundPosted, Payload: ClarificationPostedEvent{
		ClarificationRoundEvent: ClarificationRoundEvent{Doc: docRef(d), Round: r.Round, StartedBy: r.StartedBy, ActorID: r.StartedBy},
		NoGaps:                  true,
	}}
	if err := s.repo.SaveClarification(ctx, []*ClarificationRound{r}, nil, evt); err != nil {
		return nil, fmt.Errorf("record no gaps on doc %s round %d: %w", d.ID, r.Round, err)
	}
	return d, nil
}

// runningRoundOfCaller returns the doc, which the caller must read, and its running round, which they must have started.
func (s *Service) runningRoundOfCaller(ctx context.Context, docID string) (*Doc, *ClarificationRound, error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, nil, err
	}
	d, err := s.Get(ctx, docID)
	if err != nil {
		return nil, nil, err
	}
	rounds, err := s.loadRounds(ctx, d.ID)
	if err != nil {
		return nil, nil, err
	}
	i := slices.IndexFunc(rounds, func(r *ClarificationRound) bool { return r.Running })
	if i < 0 {
		return nil, nil, fmt.Errorf("%w: no round is running on doc %s; questions come only from a Clarify via AI run", apperrs.ErrConflict, d.ID)
	}
	if rounds[i].StartedBy != actor {
		return nil, nil, fmt.Errorf("%w: only the person who started round %d posts it", apperrs.ErrForbidden, rounds[i].Round)
	}
	return d, rounds[i], nil
}

// OpenRound opens the doc's next round as a Clarify run starts, recording its starter, its trail, and whether the
// run took the doc's lock. No permission check: the run's start already checked the starter, as LockForPlay. A round
// still running is stale, since the runner allows one live run per doc, so it is ended first; a lock it took passes
// to the new round, which the runner's own lock attempt found already taken.
func (s *Service) OpenRound(ctx context.Context, docID, starterID, trailID string, tookLock bool) (int, error) {
	d, err := s.repo.GetByID(ctx, docID)
	if err != nil {
		return 0, fmt.Errorf("get doc %s: %w", docID, err)
	}
	rounds, err := s.loadRounds(ctx, d.ID)
	if err != nil {
		return 0, err
	}
	if i := slices.IndexFunc(rounds, func(r *ClarificationRound) bool { return r.Running }); i >= 0 {
		stale := rounds[i]
		handOver := stale.TookLock && d.Locked
		if err := s.endRound(ctx, d, stale, handOver); err != nil {
			return 0, fmt.Errorf("end stale round %d of doc %s: %w", stale.Round, d.ID, err)
		}
		tookLock = tookLock || handOver
		if len(stale.Questions) == 0 && stale.NoGapsAt == nil {
			rounds = slices.Delete(rounds, i, i+1)
		}
	}
	next := 1
	if n := len(rounds); n > 0 {
		next = rounds[n-1].Round + 1
	}
	r := &ClarificationRound{
		DocID: d.ID, Round: next, StartedBy: starterID, TrailID: trailID, StartedAt: s.now().UTC(), Running: true, TookLock: tookLock,
	}
	if err := s.repo.CreateClarificationRound(ctx, r, roundEvent(TopicClarificationRoundStarted, d, r, starterID)); err != nil {
		return 0, fmt.Errorf("open round %d of doc %s: %w", r.Round, d.ID, err)
	}
	return r.Round, nil
}

// EndRound ends the round trailID opened, at any end of its run: it unlocks the doc if the run locked it, and removes
// a round that asked nothing and found no gaps. A round already ended is left as it is.
func (s *Service) EndRound(ctx context.Context, docID, trailID string) error {
	d, err := s.repo.GetByID(ctx, docID)
	if err != nil {
		return fmt.Errorf("get doc %s: %w", docID, err)
	}
	rounds, err := s.loadRounds(ctx, d.ID)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(rounds, func(r *ClarificationRound) bool { return r.Running && r.TrailID == trailID })
	if i < 0 {
		return nil
	}
	return s.endRound(ctx, d, rounds[i], false)
}

// endRound stops r running, unlocking the doc if its run took the lock unless keepLock, and removes it when empty.
func (s *Service) endRound(ctx context.Context, d *Doc, r *ClarificationRound, keepLock bool) error {
	if r.TookLock && d.Locked && !keepLock {
		if err := s.persistLocked(ctx, d, false); err != nil {
			return err
		}
	}
	r.Running = false
	evt := roundEvent(TopicClarificationRoundEnded, d, r, r.StartedBy)
	if len(r.Questions) == 0 && r.NoGapsAt == nil {
		evt.Payload = ClarificationRoundEvent{Doc: docRef(d), Round: r.Round, StartedBy: r.StartedBy, ActorID: r.StartedBy, Removed: true}
		if err := s.repo.DeleteClarificationRound(ctx, d.ID, r.Round, evt); err != nil {
			return fmt.Errorf("remove empty round %d of doc %s: %w", r.Round, d.ID, err)
		}
		return nil
	}
	if err := s.repo.SaveClarification(ctx, []*ClarificationRound{r}, nil, evt); err != nil {
		return fmt.Errorf("end round %d of doc %s: %w", r.Round, d.ID, err)
	}
	return nil
}

// docFor loads a doc and checks the caller holds action on it.
func (s *Service) docFor(ctx context.Context, docID string, action permissions.Action) (*Doc, error) {
	if strings.TrimSpace(docID) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	d, err := s.repo.GetByID(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("get doc %s: %w", docID, err)
	}
	if err := s.require(ctx, d.ID, action); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) roundOf(ctx context.Context, docID string, round int) (*ClarificationRound, error) {
	rounds, err := s.loadRounds(ctx, docID)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(rounds, func(r *ClarificationRound) bool { return r.Round == round })
	if i < 0 {
		return nil, fmt.Errorf("%w: doc %s has no round %d", apperrs.ErrNotFound, docID, round)
	}
	return rounds[i], nil
}

func (s *Service) stamp() *time.Time {
	t := s.now().UTC()
	return &t
}

func requireActor(ctx context.Context) (string, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return "", fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	return actor.ID, nil
}

func docRef(d *Doc) WatchedDoc {
	return WatchedDoc{ID: d.ID, ProjectID: d.ProjectID, Title: d.Title}
}

func roundEvent(topic string, d *Doc, r *ClarificationRound, actor string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: ClarificationRoundEvent{
		Doc: docRef(d), Round: r.Round, StartedBy: r.StartedBy, ActorID: actor,
	}}
}

// normalizeAnswer trims and checks an answer as given; a skip carries no picks and no text.
func normalizeAnswer(a *Answer) error {
	if a.Skipped {
		a.Selected, a.Text = []string{}, ""
		return nil
	}
	a.Text = strings.TrimSpace(a.Text)
	selected := make([]string, 0, len(a.Selected))
	for _, v := range a.Selected {
		if v = strings.TrimSpace(v); v != "" {
			selected = append(selected, v)
		}
	}
	a.Selected = selected
	if len(a.Selected) == 0 && a.Text == "" {
		return fmt.Errorf("%w: pick an option or write an answer, or skip the question", apperrs.ErrInvalid)
	}
	if len(a.Selected) > maxPickedOptions {
		return fmt.Errorf("%w: at most %d picked options", apperrs.ErrInvalid, maxPickedOptions)
	}
	if n := utf8.RuneCountInString(a.Text); n > maxClarifyTextChars {
		return fmt.Errorf("%w: the answer is %d characters, over the %d-character cap", apperrs.ErrInvalid, n, maxClarifyTextChars)
	}
	return nil
}

// normalizeQuestion trims and checks a question a run posts: its text, why-line, and options.
func normalizeQuestion(q *ClarificationQuestion) error {
	q.Question, q.Why = strings.TrimSpace(q.Question), strings.TrimSpace(q.Why)
	if q.Question == "" {
		return fmt.Errorf("%w: question is required", apperrs.ErrInvalid)
	}
	if utf8.RuneCountInString(q.Question) > maxClarifyLineChars || utf8.RuneCountInString(q.Why) > maxClarifyLineChars {
		return fmt.Errorf("%w: a question and its why are at most %d characters each", apperrs.ErrInvalid, maxClarifyLineChars)
	}
	if len(q.Options) > maxQuestionOptions {
		return fmt.Errorf("%w: at most %d options", apperrs.ErrInvalid, maxQuestionOptions)
	}
	options := make([]QuestionOption, 0, len(q.Options))
	for _, o := range q.Options {
		o.Label, o.Description = strings.TrimSpace(o.Label), strings.TrimSpace(o.Description)
		if o.Label == "" {
			return fmt.Errorf("%w: every option needs a label", apperrs.ErrInvalid)
		}
		options = append(options, o)
	}
	q.Options = options
	return nil
}
