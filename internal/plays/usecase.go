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
	fixWithAIInstructions = "Confirm your Nexul access by reading the ticket with `ticket_get`, which also lists its " +
		"linked pull requests and branches. Work in the project checkout you are running in. " +
		"Create a branch named `<ticket key>-<short-slug>` from the default branch, implement the fix, run the " +
		"project's tests and linters, and commit. Push the branch, open a pull request whose title starts with " +
		"the ticket key, then link both to the ticket with `ticket_update`, passing `link_branch` and " +
		"`link_pr`. Do not merge unless a memory selected for this run explicitly permits merging; if one " +
		"does, merge once checks pass. Reply with what changed, how it was verified, and the PR link. If you " +
		"cannot complete the fix, say what blocked you instead of opening a partial PR."
	toTicketsInstructions = "Confirm your Nexul access by reading the doc with `doc_get`. " +
		"Read the doc's project with `project_get`, which lists its status columns and ticket types. Search existing " +
		"tickets with `ticket_list` and a `query` so you do not duplicate work already tracked. Split the doc into tickets a developer " +
		"could pick up independently: one outcome per ticket, a title under eighty characters, a body with " +
		"context, acceptance criteria, and a pointer to the doc section it came from. Create each with " +
		"`ticket_create`, passing the doc id so the ticket links back. Put them in the backlog column. Reply " +
		"with the list of tickets created and anything in the doc you deliberately did not turn into a ticket."
	interviewInstructions = "Run this project's interview follow-ups: the person has answered the workspace's Interview template on the project's Interview page, and you ask about what is still open, then write the interview memory; the project, the answers it already records, and the checkouts of its project sources under question are named below. " +
		"Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory, whose body is the project's current rules (empty the first time), its `questions` (the Interview template's questions), its `answers` (the stored answers: round 0 answers the template's questions, round 1 and up are earlier follow-ups), and its `sources`. An answer to a question no longer in `questions` answers an earlier wording of one; use it too. " +
		"Then read the checkout you are running in: manifests and lockfiles, CI config, linter and formatter config, tests, README, docs and decision records. " +
		"Read every source whose stance is question as well: how something was done before, such as an earlier phase of this project. A source is a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text from its body, or another project through its memories and interview answers with `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source selectively: its table of contents or headings first, then only the passages a question needs. Read a source whose stance is follow only as context for the gaps; another run drafts answers from it, so write no drafts. " +
		"Ask about every skipped or unanswered question, every gap the answers leave that the code cannot settle, anything the code contradicts, and what a question source did that the answers do not settle. Ask with your own question tool, never through another agent, one round at a time: all of a round's questions in one call, each with your recommended answer as its first option, labelled (Recommended), and a header of at most 12 characters naming its topic, such as Testing. Write each question's text as the question followed by one sentence on why you are asking that names what you found and where, such as: When are tests written? You skipped this, and most commits in the last month add a test file beside the code they change. For a question source, name the source and the file, such as: Keep booking state on the server for phase 2? Phase 1 keeps it in the mobile app, in src/state/booking.ts of its checkout. Keep asking rounds until nothing is left open. Never record something from the code or a source the person did not confirm, and never ask whether to scan the codebase. " +
		"Do not ask again what the project already records, such as where its tests live, unless the person changes it; record a change there with `project_update` and `tests_location` as well. " +
		"Then write the memory with `memory_update`, passing its id and the full markdown body: rules, not a transcript, as short imperative lines under headings you choose, with no questions, answers, or narration. Keep every existing rule no answer contradicts, and change only what the answers change. " +
		"The body is capped at 8,000 characters and every agent turn in the project reads it first, so keep it well under the cap: tighten wording and drop what a linter or the code already enforces. " +
		"Reply with a short summary of what the memory now says and what this run changed."
	draftInterviewInstructions = "Draft answers to this project's Interview template from its follow sources, for the person to confirm on the project's Interview page; the project, this run's trail id, and the checkouts of its project sources are named below. " +
		"Start by calling `memory_create` with the project id and `kind` `interview`: it returns the interview memory with its `questions` (the Interview template's questions), its `answers` (round 0 answers the template's questions, a skip is no answer), its `sources`, and its `drafts`. " +
		"Read every source whose stance is follow: a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text from its body, and another project through its memories and interview answers with `memory_list` and `memory_get`, plus its checkout when one is named below. " +
		"Read a large source selectively: its table of contents or headings first, then only the passages a question needs. Never read a source whose stance is question and never draft from one; another run asks about those. " +
		"Draft every template question with no answer or a skip that a follow source speaks to. Draft a question that already has an answer only where the sources now say something different from that answer, and leave every other answered question alone. A question no follow source speaks to gets no draft. " +
		"Shape each draft like an answer: `selected` with the labels of the options it picks, `text` for what the options do not say, `source_ids` with the ids of the follow sources it came from, and `where`, one line of at most 500 characters saying where in them, such as practices/testing.md, Test error paths first. " +
		"Save drafts with `memory_update` on the interview memory, in `drafts` with this run's `trail_id`, as you find them rather than all at the end, so the page shows them while you work; a draft replaces its question's earlier one. " +
		"Never ask the person anything and never use a question tool: the person confirms drafts on the page. Do not change the memory's body, its answers, or its sources. " +
		"End with a one-line summary of how many questions you drafted and from which sources."
	testWithAIInstructions = "Test this ticket the way a tester would, then pass or fail it. Read it with `ticket_get`, which also carries its links and where to test; its acceptance criteria are what you test against. " +
		"Follow the testing strategy in this project's interview memory, which comes with this run. With no interview, check each criterion on the live URL and run the tests the project already has, and add none. " +
		"Where to test is the `test_target` in that result. Test nowhere else: never production, and never anything that shares production's services. If its url is empty, the only place to test is production: stop without passing or failing the ticket, and reply that it needs a deploy branch on its own network. " +
		"Open the url and check it against each acceptance criterion in turn, noting what you did and what you saw. " +
		"Run the project's tests. List its repositories with `project_get`: when one has the role tests, run them from that repository, cloning it if the checkout you are running in is not it; otherwise run them in this checkout on the ticket's linked branch. " +
		"Where the interview calls for an automated end-to-end suite, add or extend a test covering the ticket's acceptance criteria, run it against the url, commit it, and push: to the ticket's linked branch, or in a tests repository to a branch named `<ticket key>-<short-slug>` with a pull request whose title starts with the ticket key. " +
		"Then record the result exactly as a person would. If every criterion holds and the tests pass, call `ticket_test_report` with `outcome` `pass`. Otherwise call it with `outcome` `fail` and the bug template filled: the steps to reproduce, the expected result the criterion promises, and the actual result you saw. It posts them to the ticket's thread and moves the ticket back to progress. " +
		"Reply with each criterion and whether it held, the tests you ran and added, and the result you recorded. If you cannot reach the url or run the tests, say what blocked you instead of passing or failing the ticket."
)

// Service is the plays use-case layer: workspace-scoped play definitions (ADR 0055).
type Service struct {
	repo     Repo
	perm     PermissionGate
	instance InstanceTemplates
	now      func() time.Time
}

// InstanceTemplates reads the instance's text for a template kind and key, the code default until edited (ADR 0103).
type InstanceTemplates interface {
	Effective(ctx context.Context, kind, key string) (string, error)
}

// SetInstanceTemplates wires the instance layer a new workspace's built-in plays take their instructions from.
func (s *Service) SetInstanceTemplates(t InstanceTemplates) { s.instance = t }

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
	p.ExcludedProjectIDs = normalizeIDs(slices.Concat(in.ExcludedProjectIDs, s.unseenExclusions(ctx, p)))
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
	if err := s.repo.Delete(ctx, id, s.event(TopicDeleted, DeletedEvent{ID: p.ID, Label: p.Label, WorkspaceID: p.WorkspaceID})); err != nil {
		return fmt.Errorf("delete play %s: %w", id, err)
	}
	return nil
}

// TemplateKind names built-in play instructions among the instance templates (ADR 0103).
const TemplateKind = "play_instructions"

// DraftInterviewKey is the built-in drafting play's key: an interview run that drafts answers instead of asking (ADR 0122).
const DraftInterviewKey = "interview-draft"

// Builtin is one seeded play: the stable key a clone matches it by, and what a new workspace gets.
type Builtin struct {
	Key           string
	Label         string
	Type          Type
	Description   string
	Instructions  string
	ShowWhenStage *Stage
}

// Builtins lists the seeded plays in seeding order; their instructions are the instance templates' code defaults.
func Builtins() []Builtin {
	progress, testingStage := StageProgress, StageTesting
	return []Builtin{
		{Key: "fix-with-ai", Label: "Fix with AI", Type: TypeTicket, ShowWhenStage: &progress,
			Description:  "Reads the ticket, implements a fix on its own branch, and opens a pull request.",
			Instructions: fixWithAIInstructions},
		{Key: "to-tickets-via-ai", Label: "To tickets via AI", Type: TypeDoc,
			Description:  "Splits a doc into tickets a developer could pick up independently.",
			Instructions: toTicketsInstructions},
		{Key: "interview", Label: "Interview", Type: TypeInterview,
			Description:  "Asks follow-ups about what the Interview answers and the code leave open, then writes this project's rules for agents.",
			Instructions: interviewInstructions},
		{Key: "test-with-ai", Label: "Test with AI", Type: TypeTicket, ShowWhenStage: &testingStage,
			Description:  "Tests the ticket on its test environment against its acceptance criteria, then passes or fails it.",
			Instructions: testWithAIInstructions},
		{Key: DraftInterviewKey, Label: "Draft interview", Type: TypeInterview,
			Description:  "Drafts answers to the Interview questions from this project's follow sources, for a person to confirm.",
			Instructions: draftInterviewInstructions},
	}
}

// SeedDefaults creates the out-of-the-box plays for a fresh workspace (ticket 02), each with the instance's
// instructions for it; no permission gate, the same way CreateOwnerRole seeds a workspace's first role: there is no
// member yet to hold plays:write.
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
	for _, b := range Builtins() {
		instructions, err := s.instanceInstructions(ctx, b)
		if err != nil {
			return err
		}
		p := &Play{
			ID: ids.New(), WorkspaceID: workspaceID, Label: b.Label, Type: b.Type, Description: b.Description,
			Instructions: instructions, Enabled: true, ShowWhenStage: b.ShowWhenStage, BuiltinKey: b.Key,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := p.Validate(); err != nil {
			return err
		}
		if err := s.repo.Create(ctx, p, s.event(TopicCreated, CreatedEvent{Play: *p})); err != nil {
			return fmt.Errorf("seed default play %q for workspace %s: %w", p.Label, workspaceID, err)
		}
	}
	return nil
}

func (s *Service) instanceInstructions(ctx context.Context, b Builtin) (string, error) {
	if s.instance == nil {
		return b.Instructions, nil
	}
	body, err := s.instance.Effective(ctx, TemplateKind, b.Key)
	if err != nil {
		return "", fmt.Errorf("get the instance instructions for %s: %w", b.Key, err)
	}
	return body, nil
}

// BuiltinPlay returns workspaceID's copy of the built-in play key (plays:read); a deleted one is not found.
func (s *Service) BuiltinPlay(ctx context.Context, workspaceID, key string) (*Play, error) {
	if err := s.require(ctx, workspaceID, permissions.PlaysRead); err != nil {
		return nil, err
	}
	return s.builtin(ctx, workspaceID, key)
}

// SetBuiltinInstructions replaces the instructions of workspaceID's copy of the built-in play key (plays:write).
func (s *Service) SetBuiltinInstructions(ctx context.Context, workspaceID, key, instructions string) (*Play, error) {
	if err := s.require(ctx, workspaceID, permissions.PlaysWrite); err != nil {
		return nil, err
	}
	p, err := s.builtin(ctx, workspaceID, key)
	if err != nil {
		return nil, err
	}
	p.Instructions = instructions
	p.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, p, s.event(TopicUpdated, UpdatedEvent{Play: *p})); err != nil {
		return nil, fmt.Errorf("update play %s: %w", p.ID, err)
	}
	return p, nil
}

func (s *Service) builtin(ctx context.Context, workspaceID, key string) (*Play, error) {
	list, err := s.repo.List(ctx, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list plays for workspace %s: %w", workspaceID, err)
	}
	key = strings.TrimSpace(key)
	for _, p := range list {
		if p.BuiltinKey != "" && strings.EqualFold(p.BuiltinKey, key) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("%w: workspace %s has no built-in play %q; it was deleted, or the key is not one of fix-with-ai, to-tickets-via-ai, interview, test-with-ai, interview-draft", apperrs.ErrNotFound, workspaceID, key)
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

// unseenExclusions are the play's excluded projects its editor cannot open: they never saw them, so saving keeps them.
func (s *Service) unseenExclusions(ctx context.Context, p *Play) []string {
	var out []string
	for _, projectID := range p.ExcludedProjectIDs {
		if !s.perm.HasPermission(ctx, actorID(ctx), p.WorkspaceID, permissions.Member, resourceTypeProject, projectID) {
			out = append(out, projectID)
		}
	}
	return out
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
