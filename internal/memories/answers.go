package memories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	maxQuestionChars = 1_000
	maxSelected      = 50
)

// ListAnswers returns a project's interview answers by round, then in the order each was first answered; requires
// memories:read on the project.
func (s *Service) ListAnswers(ctx context.Context, projectID string) ([]*InterviewAnswer, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireProject(ctx, projectID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	as, err := s.repo.ListAnswers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list interview answers for project %s: %w", projectID, err)
	}
	return as, nil
}

// SaveAnswer stores a.Selected and a.Text, or a skip when a.Skipped, as the answer to a.Question in a.ProjectID,
// replacing any earlier one; a follow-up (round 1 and up) must already be stored. Requires memories:write.
func (s *Service) SaveAnswer(ctx context.Context, a InterviewAnswer) (*InterviewAnswer, error) {
	if err := normalizeAnswer(&a); err != nil {
		return nil, err
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	workspaceID, err := s.projectForWrite(ctx, a.ProjectID, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	a.ID, a.WorkspaceID, a.AnsweredBy, a.AnsweredAt = ids.New(), workspaceID, actor.ID, s.now().UTC()
	evt := answerEvent(TopicAnswerSaved, &a)
	if a.Round == 0 {
		saved, err := s.repo.UpsertAnswer(ctx, &a, evt)
		if err != nil {
			return nil, fmt.Errorf("save interview answer for project %s: %w", a.ProjectID, err)
		}
		return saved, nil
	}
	saved, err := s.repo.UpdateAnswer(ctx, &a, evt)
	if err != nil {
		return nil, fmt.Errorf("save follow-up answer for project %s round %d: %w", a.ProjectID, a.Round, err)
	}
	return saved, nil
}

// ClearAnswer makes a question unanswered: a template question's row goes, a follow-up keeps its question bare.
// Requires memories:write; clearing an unanswered template question does nothing.
func (s *Service) ClearAnswer(ctx context.Context, projectID string, round int, question string) error {
	a := InterviewAnswer{ProjectID: projectID, Round: round, Question: question, Skipped: true}
	if err := normalizeAnswer(&a); err != nil {
		return err
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	workspaceID, err := s.projectForWrite(ctx, a.ProjectID, permissions.MemoriesWrite)
	if err != nil {
		return err
	}
	a.WorkspaceID, a.Skipped, a.AnsweredBy, a.AnsweredAt = workspaceID, false, actor.ID, s.now().UTC()
	evt := answerEvent(TopicAnswerCleared, &a)
	if a.Round > 0 {
		if _, err := s.repo.UpdateAnswer(ctx, &a, evt); err != nil {
			return fmt.Errorf("clear follow-up answer for project %s round %d: %w", a.ProjectID, a.Round, err)
		}
		return nil
	}
	err = s.repo.DeleteAnswer(ctx, a.ProjectID, 0, a.Question, evt)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("clear interview answer for project %s: %w", a.ProjectID, err)
	}
	return nil
}

// RecordRound stores the follow-ups a run asked and their answers as the project's next round, in one transaction.
// No permission check: the play runner calls it on the server's behalf once the answer has reached the run.
func (s *Service) RecordRound(ctx context.Context, projectID, answeredBy string, followUps []InterviewAnswer) (int, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return 0, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if len(followUps) == 0 {
		return 0, fmt.Errorf("%w: a round needs at least one follow-up", apperrs.ErrInvalid)
	}
	workspaceID, err := s.projects.WorkspaceForProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
	}
	last, err := s.repo.LastRound(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("last interview round for project %s: %w", projectID, err)
	}
	round, now := last+1, s.now().UTC()
	seen := map[string]bool{}
	rows := make([]*InterviewAnswer, 0, len(followUps))
	evts := make([]eventbus.OutboxEvent, 0, len(followUps))
	for _, f := range followUps {
		f.ProjectID, f.Round = projectID, round
		f.Skipped = f.Skipped || (len(f.Selected) == 0 && strings.TrimSpace(f.Text) == "")
		if err := normalizeAnswer(&f); err != nil {
			return 0, err
		}
		if seen[f.Question] {
			return 0, fmt.Errorf("%w: the follow-up %q is asked twice in one round", apperrs.ErrInvalid, f.Question)
		}
		seen[f.Question] = true
		f.ID, f.WorkspaceID, f.AnsweredBy, f.AnsweredAt = ids.New(), workspaceID, answeredBy, now
		rows = append(rows, &f)
		evts = append(evts, answerEvent(TopicAnswerSaved, &f))
	}
	if err := s.repo.InsertRound(ctx, rows, evts...); err != nil {
		return 0, fmt.Errorf("record interview round %d for project %s: %w", round, projectID, err)
	}
	return round, nil
}

// normalizeAnswer trims and checks an answer as given; a skip carries no picks and no text.
func normalizeAnswer(a *InterviewAnswer) error {
	a.ProjectID = strings.TrimSpace(a.ProjectID)
	if a.ProjectID == "" {
		return fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if a.Round < 0 {
		return fmt.Errorf("%w: round is 0 for the template's questions or 1 and up for a follow-up round, not %d", apperrs.ErrInvalid, a.Round)
	}
	a.Question = strings.TrimSpace(a.Question)
	if a.Question == "" {
		return fmt.Errorf("%w: question is required", apperrs.ErrInvalid)
	}
	if utf8.RuneCountInString(a.Question) > maxQuestionChars {
		return fmt.Errorf("%w: the question is over %d characters", apperrs.ErrInvalid, maxQuestionChars)
	}
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
	if len(a.Selected) > maxSelected {
		return fmt.Errorf("%w: at most %d picked options", apperrs.ErrInvalid, maxSelected)
	}
	if n := utf8.RuneCountInString(a.Text); n > MaxInterviewChars {
		return fmt.Errorf("%w: the answer is %d characters, over the %d-character cap", apperrs.ErrInvalid, n, MaxInterviewChars)
	}
	return nil
}

func answerEvent(topic string, a *InterviewAnswer) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: AnswerEvent{
		WorkspaceID: a.WorkspaceID, ProjectID: a.ProjectID, Round: a.Round, Question: a.Question, AuthorID: a.AnsweredBy, At: a.AnsweredAt,
	}}
}
