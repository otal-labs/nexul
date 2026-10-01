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

// DefaultInterviewTemplate is the code default of the Interview template, one heading per category.
const DefaultInterviewTemplate = `## Stack and versions
Languages, frameworks, and the versions this project pins.

## Architecture
Paradigm (ECS, OOP, composition, or functional) and module boundaries.

## Error handling and logging
How errors travel and what gets logged, at which level.

## Testing
Unit or integration, e2e in this repo or a separate one, the coverage floor, and whether tests come first.

## Code style
Early return, naming, and comment density.

## Dependency policy
When a new dependency is allowed and which ones are settled.

## Security and secrets
Where secrets live and what must never be committed or logged.

## Performance budgets
The limits a change must stay within.

## CI gates
What must pass before a change merges.

## Branching, PRs, and commits
Branch names, PR rules, and commit message style.

## Docs and decision records
What gets documented and where decisions are recorded.

## UI
Design system, mobile-first, and accessibility.

## Vocabulary
The project's own terms and what they mean.
`

// CreateInterview returns the project's interview memory, copying the Interview template on first need.
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
	tmpl, err := s.loadTemplate(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	body, err := richtext.Normalize(tmpl.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: the Interview template is not valid document content", apperrs.ErrInvalid)
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

// CheckInterviewTemplate refuses an Interview template over the interview memory's cap, at any layer.
func CheckInterviewTemplate(body string) error {
	return checkInterviewLength("Interview template", body)
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
		return &InterviewTemplate{WorkspaceID: workspaceID, Body: instance, DefaultBody: instance}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get interview template for workspace %s: %w", workspaceID, err)
	}
	t.DefaultBody, t.Edited = instance, true
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
	return checkInterviewLength("interview", md)
}

func checkInterviewLength(what, markdown string) error {
	n := utf8.RuneCountInString(markdown)
	if n <= MaxInterviewChars {
		return nil
	}
	return fmt.Errorf("%w: the %s is %d characters, over the %d-character cap; keep it to rules, not a transcript", apperrs.ErrInvalid, what, n, MaxInterviewChars)
}
