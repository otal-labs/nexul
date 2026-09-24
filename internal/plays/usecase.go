package plays

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Seeded instruction texts (spec "Prompt composition"), editable like any other play once created.
const (
	fixWithAIInstructions = "Confirm your Nexul access by reading the ticket with `ticket_get` and its links " +
		"with `ticket_get_links`. Work in the project checkout you are running in. " +
		"Create a branch named `<ticket key>-<short-slug>` from the default branch, implement the fix, run the " +
		"project's tests and linters, and commit. Push the branch, open a pull request whose title starts with " +
		"the ticket key, then link the branch and the PR to the ticket with `ticket_link_branch` and " +
		"`ticket_link_pr`. Do not merge unless a memory selected for this run explicitly permits merging; if one " +
		"does, merge once checks pass. Reply with what changed, how it was verified, and the PR link. If you " +
		"cannot complete the fix, say what blocked you instead of opening a partial PR."
	toTicketsInstructions = "Confirm your Nexul access by reading the doc with `doc_get`. " +
		"List the doc's project's columns with `status_list` and its ticket types. Search existing tickets with " +
		"`ticket_search` so you do not duplicate work already tracked. Split the doc into tickets a developer " +
		"could pick up independently: one outcome per ticket, a title under eighty characters, a body with " +
		"context, acceptance criteria, and a pointer to the doc section it came from. Create each with " +
		"`ticket_create`, passing the doc id so the ticket links back. Put them in the backlog column. Reply " +
		"with the list of tickets created and anything in the doc you deliberately did not turn into a ticket."
	interviewInstructions = "Run this project's interview, the conversation that records its rules for agents; the project and the answers it already records are named below. " +
		"Start by calling `memory_create_interview` with the project id: it returns the interview memory, created from the workspace's Interview template the first time. " +
		"If the memory already holds rules, this is a re-run: amend it, never start over. Ask first what has changed, keep every rule that still holds, and change only what the answers change. " +
		"Ask one question at a time with your question tool, never a batch, and give every question your recommended answer as its first option, labelled (Recommended), so the person can accept it or type their own. " +
		"Your first question asks whether to scan the codebase for answers first. If they say yes, read the checkout you are running in (manifests and lockfiles, CI config, linter and formatter config, tests, README, docs and decision records), draft an answer for each category, then grill them on it: one question per finding, saying what you found and where, until each is confirmed or corrected. Never record a finding they have not confirmed. " +
		"Work through the categories in the interview's headings in order. Do not ask again what the project already records, such as where its tests live, unless the person changes it; record a change there with `project_set_tests_location` as well. " +
		"Save the interview with `memory_update` after each category, passing its id, title, when-to-use, and the full markdown body, so progress survives a stop. " +
		"Write rules, not a transcript: short imperative lines under each heading, with no questions, answers, or narration. Replace each heading's prompt line with its rules, and leave out a heading the project has no rule for. " +
		"The body is capped at 8,000 characters of markdown and every agent turn in the project carries it in full, so keep it well under the cap: tighten wording and drop what a linter or the code already enforces. " +
		"Reply with a short summary of what the interview now says and what this run changed."
)

// Service is the plays use-case layer: workspace-scoped play definitions (ADR 0055).
type Service struct {
	repo Repo
	perm PermissionGate
	now  func() time.Time
}

// NewService wires the plays use-cases over the given repo and permission gate.
func NewService(repo Repo, perm PermissionGate) *Service {
	return &Service{repo: repo, perm: perm, now: time.Now}
}

// CreateInput is the owner-edited surface of a new play.
type CreateInput struct {
	Label              string
	Type               Type
	Description        string
	Instructions       string
	Enabled            bool
	ShowWhenStage      *Stage
	ExcludedProjectIDs []string
}

// UpdateInput is the owner-edited surface of an existing play; type is immutable after create (not included here).
type UpdateInput struct {
	Label              string
	Description        string
	Instructions       string
	Enabled            bool
	ShowWhenStage      *Stage
	ExcludedProjectIDs []string
}

// List returns workspaceID's plays sorted by label.
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Play, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.PlaysRead); err != nil {
		return nil, err
	}
	list, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list plays for workspace %s: %w", workspaceID, err)
	}
	return list, nil
}

// ListApplicable returns workspaceID's enabled plays of playType that apply to a run on projectID for userID:
// not excluded for that project, matching stage for a ticket play, and not denied plays:run (ticket 21). Unlike
// List, this skips the plays:read gate: any viewer of the target may see the buttons they hold plays:run for.
func (s *Service) ListApplicable(ctx context.Context, workspaceID, userID, projectID string, playType Type, stage *Stage) ([]*Play, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if !playType.valid() {
		return nil, fmt.Errorf("%w: type must be ticket, doc, or interview", apperrs.ErrInvalid)
	}
	list, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list plays for workspace %s: %w", workspaceID, err)
	}
	out := make([]*Play, 0, len(list))
	for _, p := range list {
		if !p.Enabled || p.Type != playType {
			continue
		}
		if projectID != "" && slices.Contains(p.ExcludedProjectIDs, projectID) {
			continue
		}
		if playType == TypeTicket && (stage == nil || p.ShowWhenStage == nil || *p.ShowWhenStage != *stage) {
			continue
		}
		if !s.perm.HasPermission(ctx, userID, workspaceID, permissions.PlaysRun, resourceTypePlay, p.ID) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// Get returns a single play scoped to workspaceID; a play from another workspace is reported not found.
func (s *Service) Get(ctx context.Context, workspaceID, id string) (*Play, error) {
	if err := s.require(ctx, workspaceID, permissions.PlaysRead); err != nil {
		return nil, err
	}
	return s.getInWorkspace(ctx, workspaceID, id)
}

// Create adds a play to workspaceID; actorID must hold plays:write.
func (s *Service) Create(ctx context.Context, workspaceID string, in CreateInput) (*Play, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.PlaysWrite); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	p := &Play{
		ID:                 ids.New(),
		WorkspaceID:        workspaceID,
		Label:              strings.TrimSpace(in.Label),
		Type:               in.Type,
		Description:        strings.TrimSpace(in.Description),
		Instructions:       in.Instructions,
		Enabled:            in.Enabled,
		ShowWhenStage:      in.ShowWhenStage,
		ExcludedProjectIDs: normalizeIDs(in.ExcludedProjectIDs),
		CreatedBy:          actorID(ctx),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p, s.event(TopicCreated, CreatedEvent{Play: *p})); err != nil {
		return nil, fmt.Errorf("create play in workspace %s: %w", workspaceID, err)
	}
	return p, nil
}

// Update replaces a play's owner-edited fields; the play's type never changes after create.
func (s *Service) Update(ctx context.Context, workspaceID, id string, in UpdateInput) (*Play, error) {
	if err := s.require(ctx, workspaceID, permissions.PlaysWrite); err != nil {
		return nil, err
	}
	p, err := s.getInWorkspace(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	p.Label = strings.TrimSpace(in.Label)
	p.Description = strings.TrimSpace(in.Description)
	p.Instructions = in.Instructions
	p.Enabled = in.Enabled
	p.ShowWhenStage = in.ShowWhenStage
	p.ExcludedProjectIDs = normalizeIDs(in.ExcludedProjectIDs)
	p.UpdatedAt = s.now().UTC()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p, s.event(TopicUpdated, UpdatedEvent{Play: *p})); err != nil {
		return nil, fmt.Errorf("update play %s: %w", id, err)
	}
	return p, nil
}

// Delete removes a play; actorID must hold plays:delete.
func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	if err := s.require(ctx, workspaceID, permissions.PlaysDelete); err != nil {
		return err
	}
	p, err := s.getInWorkspace(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id, s.event(TopicDeleted, DeletedEvent{ID: p.ID, Label: p.Label})); err != nil {
		return fmt.Errorf("delete play %s: %w", id, err)
	}
	return nil
}

// SeedDefaults creates the three out-of-the-box plays for a fresh workspace (ticket 02); no permission gate,
// the same way CreateOwnerRole seeds a workspace's first role: there is no member yet to hold plays:write.
// Idempotent: the default workspace already carries its plays from migrations by the time the Owner
// Wizard binds someone to it, so a workspace that already has plays is left alone.
func (s *Service) SeedDefaults(ctx context.Context, workspaceID string) error {
	existing, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("check existing plays for workspace %s: %w", workspaceID, err)
	}
	if len(existing) > 0 {
		return nil
	}
	now := s.now().UTC()
	progress := StageProgress
	seeds := []*Play{
		{
			ID: ids.New(), WorkspaceID: workspaceID, Label: "Fix with AI", Type: TypeTicket,
			Description:  "Reads the ticket, implements a fix on its own branch, and opens a pull request.",
			Instructions: fixWithAIInstructions, Enabled: true, ShowWhenStage: &progress,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: ids.New(), WorkspaceID: workspaceID, Label: "To tickets via AI", Type: TypeDoc,
			Description:  "Splits a doc into tickets a developer could pick up independently.",
			Instructions: toTicketsInstructions, Enabled: true,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: ids.New(), WorkspaceID: workspaceID, Label: "Interview", Type: TypeInterview,
			Description:  "Asks one question at a time to record this project's rules for agents, and amends them on a re-run.",
			Instructions: interviewInstructions, Enabled: true,
			CreatedAt: now, UpdatedAt: now,
		},
	}
	for _, p := range seeds {
		if err := p.Validate(); err != nil {
			return err
		}
		if err := s.repo.Create(ctx, p, s.event(TopicCreated, CreatedEvent{Play: *p})); err != nil {
			return fmt.Errorf("seed default play %q for workspace %s: %w", p.Label, workspaceID, err)
		}
	}
	return nil
}

func (s *Service) getInWorkspace(ctx context.Context, workspaceID, id string) (*Play, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	id = strings.TrimSpace(id)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if id == "" {
		return nil, fmt.Errorf("%w: play id is required", apperrs.ErrInvalid)
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get play %s: %w", id, err)
	}
	if p.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("get play %s: %w", id, apperrs.ErrNotFound)
	}
	return p, nil
}

func (s *Service) require(ctx context.Context, workspaceID string, action permissions.Action) error {
	if !s.perm.HasPermission(ctx, actorID(ctx), workspaceID, action, "", "") {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
	}
	return nil
}

func (s *Service) event(topic string, payload any) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: payload}
}

func actorID(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

// normalizeIDs trims, drops empties, dedupes, and sorts so equal sets always compare equal.
func normalizeIDs(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
