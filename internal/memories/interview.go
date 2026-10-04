package memories

import (
	"context"
	"errors"
	"fmt"
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
	interviewTitle     = "Interview"
	interviewWhenToUse = "This project's rules: stack, paradigm, testing, principles, and vocabulary."
)

// TemplateKind names the Interview template among the instance templates (ADR 0103).
const TemplateKind = "interview"

// DefaultInterviewTemplate is the code default of the Interview template: what only a person knows, one ## per question.
const DefaultInterviewTemplate = `## What languages and frameworks does this project use?
Name versions only where they are pinned on purpose.

## How is the code organised?
- Layers (routes, logic, storage)
- Feature folders
- Entities and systems
- Functional core with a thin outer shell

## How do errors travel?
And what gets logged, at which level.
- Returned and wrapped
- Thrown and caught at the edge
- Result types

## When are tests written?
- Before the code
- With the change
- Only for bugs
- No tests yet

## Which tests does a change need?
And the coverage floor, if any.
- [ ] Unit
- [ ] Integration against real dependencies
- [ ] End-to-end in this repo
- [ ] End-to-end in a separate repo

## Which style rules matter most?
Skip what a linter already enforces.
- [ ] Early return, no else
- [ ] Small functions
- [ ] Comments only for why
- [ ] Strict types, no any

## When may a change add a dependency?
- Freely
- When it saves real code
- Only after asking
- Only with a written decision

## Where do secrets live?
And what must never be committed or logged.
- Environment variables
- A secrets manager
- An encrypted file in the repo

## How does a change reach the main branch?
And how commit messages are written.
- Pull request, squash merge
- Pull request, merge commit
- Straight to main

## Where are decisions written down?
- Decision records in the repo
- A docs folder
- A wiki outside the repo
- Nowhere yet

## Does this project have a user interface?
If yes, the design system, screen sizes, and accessibility rules.
- Web
- Mobile
- Both
- None

## Which words mean something specific here?
One per line, the term then what it means.
`

// CreateInterview returns the project's interview memory, creating it empty the first time.
func (s *Service) CreateInterview(ctx context.Context, projectID, via string) (*Memory, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	workspaceID, err := s.projectForWrite(ctx, projectID, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByProjectKind(ctx, projectID, KindInterview)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get interview for project %s: %w", projectID, err)
	}
	body, err := richtext.Normalize("")
	if err != nil {
		return nil, fmt.Errorf("normalize an empty interview body: %w", err)
	}
	now := s.now().UTC()
	m := &Memory{
		ID: ids.New(), WorkspaceID: workspaceID, ProjectID: projectID, Kind: KindInterview,
		Title: interviewTitle, WhenToUse: interviewWhenToUse, Body: body, AlwaysIncluded: true, Version: 1,
		CreatedBy: actor.ID, CreatedAt: now, UpdatedBy: actor.ID, UpdatedAt: now,
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCreated, Payload: CreatedEvent{Memory: toRef(m), AuthorID: actor.ID}}
	if err := s.repo.Create(ctx, m, via, evt); err != nil {
		return nil, fmt.Errorf("create interview for project %s: %w", projectID, err)
	}
	return m, nil
}

// InterviewQuestions returns the questions of the workspace's effective Interview template for an interview memory,
// nil for any other memory; the caller has already read the memory.
func (s *Service) InterviewQuestions(ctx context.Context, m *Memory) ([]Question, error) {
	if m.Kind != KindInterview {
		return nil, nil
	}
	t, err := s.loadTemplate(ctx, m.WorkspaceID)
	if err != nil {
		return nil, err
	}
	return t.Questions, nil
}

// InterviewTemplate returns the workspace's Interview template, the instance's until the workspace saves its own.
func (s *Service) InterviewTemplate(ctx context.Context, workspaceID string) (*InterviewTemplate, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	return s.loadTemplate(ctx, workspaceID)
}

// ProjectInterviewTemplate reads the template through a project, for a restricted member without memories:read on the workspace.
func (s *Service) ProjectInterviewTemplate(ctx context.Context, projectID string) (*InterviewTemplate, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	workspaceID, err := s.projectForWrite(ctx, projectID, permissions.MemoriesRead)
	if err != nil {
		return nil, err
	}
	return s.loadTemplate(ctx, workspaceID)
}

// SaveInterviewTemplate replaces the workspace's Interview template; existing interviews are never touched.
func (s *Service) SaveInterviewTemplate(ctx context.Context, workspaceID, body string) (*InterviewTemplate, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesWrite); err != nil {
		return nil, err
	}
	if err := CheckInterviewTemplate(body); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	t := &InterviewTemplate{WorkspaceID: workspaceID, Body: body, Edited: true, UpdatedBy: actor.ID, UpdatedAt: now}
	if err := s.repo.SaveInterviewTemplate(ctx, t, s.templateEvent(workspaceID, actor.ID, now)); err != nil {
		return nil, fmt.Errorf("save interview template for workspace %s: %w", workspaceID, err)
	}
	return s.loadTemplate(ctx, workspaceID)
}

// ResetInterviewTemplate drops the workspace's own Interview template, so it follows the instance's again.
func (s *Service) ResetInterviewTemplate(ctx context.Context, workspaceID string) (*InterviewTemplate, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesWrite); err != nil {
		return nil, err
	}
	if err := s.repo.DeleteInterviewTemplate(ctx, workspaceID, s.templateEvent(workspaceID, actor.ID, s.now().UTC())); err != nil {
		return nil, fmt.Errorf("reset interview template for workspace %s: %w", workspaceID, err)
	}
	return s.loadTemplate(ctx, workspaceID)
}

func (s *Service) templateEvent(workspaceID, authorID string, at time.Time) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicInterviewTemplateUpdated, Payload: InterviewTemplateUpdatedEvent{
		WorkspaceID: workspaceID, AuthorID: authorID, UpdatedAt: at,
	}}
}

func (s *Service) loadTemplate(ctx context.Context, workspaceID string) (*InterviewTemplate, error) {
	instance, err := s.instanceTemplate(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.repo.GetInterviewTemplate(ctx, workspaceID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return &InterviewTemplate{WorkspaceID: workspaceID, Body: instance, DefaultBody: instance, Questions: TemplateQuestions(instance)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get interview template for workspace %s: %w", workspaceID, err)
	}
	t.DefaultBody, t.Edited, t.Questions = instance, true, TemplateQuestions(t.Body)
	return t, nil
}

func (s *Service) instanceTemplate(ctx context.Context) (string, error) {
	if s.instance == nil {
		return DefaultInterviewTemplate, nil
	}
	body, err := s.instance.Effective(ctx, TemplateKind, "")
	if err != nil {
		return "", fmt.Errorf("get the instance interview template: %w", err)
	}
	return body, nil
}

// checkInterviewBody enforces MaxInterviewChars on a stored rich-text body, measured as the markdown a turn sends.
func checkInterviewBody(body string) error {
	md, err := richtext.ToMarkdown(body)
	if err != nil {
		return fmt.Errorf("export interview body: %w", err)
	}
	n := utf8.RuneCountInString(md)
	if n <= MaxInterviewChars {
		return nil
	}
	return fmt.Errorf("%w: the interview is %d characters, over the %d-character cap; keep it to rules, not a transcript", apperrs.ErrInvalid, n, MaxInterviewChars)
}
