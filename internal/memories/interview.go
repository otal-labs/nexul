package memories

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// DefaultInterviewTemplate is the Interview template a workspace starts with, one heading per category.
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
	workspaceID, err := s.projects.WorkspaceForProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesWrite); err != nil {
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

// InterviewTemplate returns the workspace's Interview template, the default until one is saved.
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
	if err := checkInterviewLength("Interview template", body); err != nil {
		return nil, err
	}
	t := &InterviewTemplate{WorkspaceID: workspaceID, Body: body, DefaultBody: DefaultInterviewTemplate, UpdatedBy: actor.ID, UpdatedAt: s.now().UTC()}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicInterviewTemplateUpdated, Payload: InterviewTemplateUpdatedEvent{
		WorkspaceID: workspaceID, AuthorID: actor.ID, UpdatedAt: t.UpdatedAt,
	}}
	if err := s.repo.SaveInterviewTemplate(ctx, t, evt); err != nil {
		return nil, fmt.Errorf("save interview template for workspace %s: %w", workspaceID, err)
	}
	return t, nil
}

func (s *Service) loadTemplate(ctx context.Context, workspaceID string) (*InterviewTemplate, error) {
	t, err := s.repo.GetInterviewTemplate(ctx, workspaceID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return &InterviewTemplate{WorkspaceID: workspaceID, Body: DefaultInterviewTemplate, DefaultBody: DefaultInterviewTemplate}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get interview template for workspace %s: %w", workspaceID, err)
	}
	t.DefaultBody = DefaultInterviewTemplate
	return t, nil
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
