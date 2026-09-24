package tickets

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Stages the testing step reads; they mirror workspace's fixed board stages, matched by stage and never by column name.
const (
	StageProgress = "progress"
	StageDone     = "done"
)

// Stages reads a project's status columns by stage, without importing workspace (ADR 0017).
type Stages interface {
	// StageOf returns a status column's stage, or ErrNotFound when the status is not a column.
	StageOf(ctx context.Context, statusID string) (string, error)
	// FirstOfStage returns a project's first column of a stage in board order, or ErrNotFound when it has none.
	FirstOfStage(ctx context.Context, projectID, stage string) (Status, error)
}

// TicketThreads posts into a ticket's own chat thread, without importing chat (ADR 0017).
type TicketThreads interface {
	PostToTicketThread(ctx context.Context, t *Ticket, authorID, body string) error
}

// TestTargets resolves where a ticket's branches can be tested, without importing deploy (ADR 0017).
type TestTargets interface {
	ResolveTestTarget(ctx context.Context, projectID string, branches []BranchLink) (TestTarget, error)
}

// Testing is what the testing step needs from other domains, wired in the composition root.
type Testing struct {
	Stages  Stages
	Threads TicketThreads
	Targets TestTargets
}

// SetTesting wires the testing step's lookups; the testing use-cases need all three.
func (s *Service) SetTesting(t Testing) { s.testing = t }

// TestTarget is where a tester checks a ticket; Kind is preview or shared, and an empty URL means nothing safe exists.
type TestTarget struct {
	URL    string `json:"url"`
	Kind   string `json:"kind,omitempty"`
	Branch string `json:"branch,omitempty"`
}

// TestReport is a failed test's notes in the bug template's sections; Screenshots are ticket attachment ids.
type TestReport struct {
	Steps       string   `json:"steps"`
	Expected    string   `json:"expected"`
	Actual      string   `json:"actual"`
	Screenshots []string `json:"screenshots"`
}

var attachmentID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// markdown renders the report as the thread message; empty sections are left out, the actual result never is.
func (r TestReport) markdown() (string, error) {
	if strings.TrimSpace(r.Actual) == "" {
		return "", fmt.Errorf("%w: actual result is required: say what went wrong", apperrs.ErrInvalid)
	}
	parts := []string{"Test failed"}
	for _, sec := range [][2]string{{"Steps to reproduce", r.Steps}, {"Expected result", r.Expected}, {"Actual result", r.Actual}} {
		if text := strings.TrimSpace(sec[1]); text != "" {
			parts = append(parts, "## "+sec[0]+"\n"+text)
		}
	}
	if len(r.Screenshots) == 0 {
		return strings.Join(parts, "\n\n"), nil
	}
	lines := []string{"## Screenshot"}
	for _, id := range r.Screenshots {
		if !attachmentID.MatchString(id) {
			return "", fmt.Errorf("%w: screenshot %q is not an attachment id", apperrs.ErrInvalid, id)
		}
		lines = append(lines, "![screenshot](/api/attachments/"+id+")")
	}
	return strings.Join(append(parts, strings.Join(lines, "\n")), "\n\n"), nil
}

// TestTarget resolves where to test a ticket from its linked branches; production is never offered.
func (s *Service) TestTarget(ctx context.Context, id string) (TestTarget, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return TestTarget{}, err
	}
	branches, err := s.repo.ListBranchLinks(ctx, t.ID)
	if err != nil {
		return TestTarget{}, fmt.Errorf("test target for ticket %s: %w", id, err)
	}
	target, err := s.testing.Targets.ResolveTestTarget(ctx, t.ProjectID, branches)
	if err != nil {
		return TestTarget{}, fmt.Errorf("test target for ticket %s: %w", id, err)
	}
	return target, nil
}

// TestPass moves the ticket to the first done-stage column, fills an empty Tester, and posts who passed it to its thread.
func (s *Service) TestPass(ctx context.Context, id string) (*Ticket, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	done, err := s.firstOfStage(ctx, t.ProjectID, StageDone)
	if err != nil {
		return nil, err
	}
	tester, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	target, err := s.TestTarget(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Tester == "" {
		if _, err := s.SetPerson(ctx, id, RoleTester, tester); err != nil {
			return nil, err
		}
	}
	moved, err := s.transition(ctx, id, done, statusActor(ctx), "", testedEvent(TopicTestPassed, tester, ""))
	if err != nil {
		return nil, err
	}
	actor, _ := identity.ActorFromCtx(ctx)
	if err := s.testing.Threads.PostToTicketThread(ctx, moved, actor.ID, passedNote(tester, target.URL)); err != nil {
		return nil, fmt.Errorf("post test result to ticket %s: %w", id, err)
	}
	return moved, nil
}

func passedNote(tester, url string) string {
	if url == "" {
		return "Passed by " + tester
	}
	return "Passed by " + tester + " on " + url
}

// TestFail moves the ticket back to progress and posts the report to its thread; a done ticket is never reopened (ADR 0064).
func (s *Service) TestFail(ctx context.Context, id string, report TestReport) (*Ticket, error) {
	body, err := report.markdown()
	if err != nil {
		return nil, err
	}
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.refuseDone(ctx, t); err != nil {
		return nil, err
	}
	progress, err := s.firstOfStage(ctx, t.ProjectID, StageProgress)
	if err != nil {
		return nil, err
	}
	tester, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	actor, _ := identity.ActorFromCtx(ctx)
	moved, err := s.transition(ctx, id, progress, statusActor(ctx), "", testedEvent(TopicTestFailed, tester, body))
	if err != nil {
		return nil, err
	}
	if err := s.testing.Threads.PostToTicketThread(ctx, moved, actor.ID, body); err != nil {
		return nil, fmt.Errorf("post test result to ticket %s: %w", id, err)
	}
	return moved, nil
}

func (s *Service) refuseDone(ctx context.Context, t *Ticket) error {
	stage, err := s.testing.Stages.StageOf(ctx, string(t.Status))
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stage of ticket %s: %w", t.ID, err)
	}
	if stage == StageDone {
		return fmt.Errorf("%w: ticket %s is done and is never reopened; report a bug found in it instead", apperrs.ErrInvalid, t.ID)
	}
	return nil
}

func (s *Service) firstOfStage(ctx context.Context, projectID, stage string) (Status, error) {
	status, err := s.testing.Stages.FirstOfStage(ctx, projectID, stage)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", fmt.Errorf("%w: the project has no %s-stage column; add one on the board", apperrs.ErrInvalid, stage)
	}
	if err != nil {
		return "", fmt.Errorf("first %s column of project %s: %w", stage, projectID, err)
	}
	return status, nil
}

// caller is the acting person's login; a test result always names who tested.
func (s *Service) caller(ctx context.Context) (string, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return "", fmt.Errorf("%w: a test result needs a signed-in tester", apperrs.ErrUnauthorized)
	}
	return s.login(ctx, actor.ID), nil
}

func testedEvent(topic, tester, report string) func(Ticket) eventbus.OutboxEvent {
	return func(t Ticket) eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: TestedEvent{Ticket: t, Tester: tester, Report: report}}
	}
}
