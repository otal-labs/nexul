package plays

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/textedit"
)

// Seeded instruction texts (spec "Prompt composition"), editable like any other play once created.
const (
	fixWithAIInstructions = "Build this ticket and carry it to the end.\n" +
		"\n" +
		"1. Read it with `ticket_get`, which lists its blockers and its linked branches and pull requests, and with `message_list` its thread and the thread of any blocker whose body opens with \"Needs a person\". Stop and say why if this ticket's own body opens with \"Needs a person\", or if a blocker is neither merged nor in a done column. If this ticket's pull request merged and no failed test report came after, build nothing and go to step 6.\n" +
		"2. Work in the project checkout you are running in. If the ticket sits in a backlog column, move it to the first progress column with `ticket_update` before you start. Continue a linked branch of this repository whose pull request is absent or open; otherwise start a new branch `<ticket key>-<short-slug>` from the default branch. Fix what the latest failed test report names first.\n" +
		"3. Build what the acceptance criteria ask (for a bug, its expected result), and nothing else.\n" +
		"4. Prove it: run the project's tests and linters, where it has any, then watch each criterion hold in the running app or with a command.\n" +
		"5. When every criterion holds, commit and push, open a pull request whose title starts with the ticket key unless one is already open, and link both to the ticket with `ticket_update`, passing `link_branch` and `link_pr`.\n" +
		"6. Conclude as the memories this prompt names for the end of the run say. With none: a merged ticket goes to the first testing column and keeps its branch only when it has a tester or a `test_target` url and the board has a testing column; any other merged ticket goes to the first done column and its branch is deleted. An open pull request stays open and its ticket goes to the first review column, or stays where it is when the board has none.\n" +
		"\n" +
		"Reply with what changed, the proof for each criterion, and the pull request. If you could not finish, say what blocked you instead of opening a partial pull request."
	toTicketsInstructions = "Turn this doc into tickets that can each be built, proven and shipped on their own.\n" +
		"\n" +
		"1. Read the doc with `doc_get`, its thread with `message_list`, and the project with `project_get`: its columns, ticket types and their body templates. Find what is already tracked with `ticket_list`, once by `doc_id` and once with a `query` for the doc's subject. On a re-run, create only what is missing.\n" +
		"2. Ask nothing, and leave technical choices to the builder. A product decision the doc leaves open, or a choice the project's rules reserve for a person, becomes its own ticket: its body opens with \"Needs a person: <the decision>. Settle it on this thread, then move the ticket to done.\", and step 5 makes it the blocker of the tickets that wait on it.\n" +
		"3. Slice vertically: each ticket is one outcome someone can see, through every layer it needs, small enough for one pull request. The first is the thinnest end-to-end path.\n" +
		"4. Create them with `ticket_create`, the `doc_id` and `developer` `me`, each with the type that fits; `tickets` creates several of one type in one call. Fill the type's body template: every acceptance criterion is a result someone can observe, and \"How to verify\" names the proof for each; for a bug, the expected result is the criterion and the steps to reproduce are the proof, and `ticket_create` takes `origin_id`, the ticket it was found in, or `origin_unknown` true.\n" +
		"5. Wire the order with `ticket_update` `add_blocker_ids`: a blocker only where a ticket cannot start before another lands. Leave them in the backlog.\n" +
		"6. Check coverage both ways: every requirement in the doc is in a ticket's acceptance criteria or in your list of what you left out, and every ticket traces back to the doc.\n" +
		"\n" +
		"Reply with the tickets in build order with their blockers, which can start now, the decisions left open, and what you left out and why."
	interviewInstructions = "Finish this project's interview: the person answered the Interview questions on the project's Interview page, and you settle what is still open, then write the project's rules. The project, the answers it already records, and the checkouts of its project sources under question are named below.\n" +
		"\n" +
		"1. Call `memory_create` with the project id and `kind` `interview`, and nothing else. It returns the interview memory: its body is the current rules (empty the first time), with its `questions`, its `answers` (round 0 answers the questions, later rounds are earlier follow-ups, and an answer to a question no longer listed still counts) and its `sources`.\n" +
		"2. Find the facts yourself. Read the checkout you are running in: manifests, CI, linter and formatter config, tests, README and decision records. Read every source whose stance is question: a path in the checkout, a doc with `doc_get`, a memory with `memory_get`, pasted text, or another project through `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source by its headings first. A source whose stance is follow is context only; write no drafts.\n" +
		"3. Ask only for decisions: a question the person skipped whose answer neither the code nor its hint settles, a gap that changes how work is done here, an answer the code contradicts, and what a question source did that the answers leave open. Never ask what you can read or what the project already records. An answer stands: do not ask it again in other words.\n" +
		"4. Ask with your own question tool, never through another agent, a round at a time: all of a round's questions in one call, four at most, each with a header of at most 12 characters naming its topic, your recommended answer first and labelled (Recommended), and its question text followed by one sentence on why you ask, naming what you found and where. Stop after two rounds in this run, and sooner when the code or the recorded answers settle the rest.\n" +
		"5. Write the memory with `memory_update`, passing its id and the full markdown body: rules, not a transcript, as short imperative lines under headings you choose. Record a fact only where an agent would get it wrong without it, and a rule only when the person gave or confirmed it, except what step 6 takes from the code. Keep every rule no answer contradicts.\n" +
		"6. Always record how a change reaches the main branch (the answer, or the code when that question was skipped), which tests a change needs (those ticked, with any coverage floor; none when skipped), and what an agent may do without asking, which is only what the person ticked: skipped, it asks first, merging its own pull request included.\n" +
		"7. The body is capped at 8,000 characters and every agent turn reads it first, so leave out what a linter or the code already enforces.\n" +
		"\n" +
		"Reply with what the memory now says and what this run changed."
	draftInterviewInstructions = "Draft answers to this project's Interview questions from its follow sources, for the person to confirm on the project's Interview page. The project, this run's trail id, and the checkouts of its project sources are named below.\n" +
		"\n" +
		"1. Call `memory_create` with the project id and `kind` `interview`, and nothing else. It returns the interview memory with its `questions`, its `answers` (round 0 answers the questions; a skip is no answer), its `sources` and its `drafts`.\n" +
		"2. Read every source whose stance is follow: a path in the checkout you are running in, a doc with `doc_get`, a memory with `memory_get`, pasted text, or another project through `memory_list` and `memory_get`, plus its checkout when one is named below. Read a large source by its headings first. Never read or draft from a source whose stance is question.\n" +
		"3. Draft each unanswered or skipped question a follow source speaks to. Draft an answered question only where the sources now disagree with its answer. No source, no draft.\n" +
		"4. Shape a draft like an answer: `question` with the question's exact text, `selected` with the option labels it picks, `text` for what the options do not say, `source_ids`, and `where`, one line of at most 500 characters saying where inside the source, such as a heading or a quoted phrase, without repeating the source's name.\n" +
		"5. Save drafts with `memory_update` on the interview memory, in `drafts` with this run's `trail_id`, as you find them, so the page shows them while you work. A draft replaces its question's earlier one.\n" +
		"\n" +
		"Ask nothing and use no question tool. Change nothing else: not the memory's body, its answers or its sources. End with one line saying how many questions you drafted and from which sources."
	auditInstructions = "Audit code against this project's interview memory and write the findings as one doc. The project, today's date, this run's trail id, and the code to audit are named below.\n" +
		"\n" +
		"1. Call `memory_create` with the project id and `kind` `interview`, and nothing else. Its body is the project's rules, the yardstick, and it returns the `answers` and `sources` too. With an empty body, stop and say the interview is not written yet.\n" +
		"2. Read the code named below: each project source under question through its checkout on this computer, or through its memories and interview answers with `memory_list` and `memory_get` when no checkout is named. With no source under question, audit this project's own code in the checkout you are running in. Read first what the answers say each source under question did; then structure, manifests and config; then the code each rule speaks to.\n" +
		"3. Write one new doc with `doc_create` in this project, with no `folder_id`, titled Audit of <what you audited>, <today's date>. Open it with one line naming what you audited and this run's trail id. Group the findings under the memory's headings, in the memory's order, one line each: the verdict, where it is, as `path:line` when it has one, and the rule it meets or breaks. A heading with nothing found says so in one line.\n" +
		"4. Auditing a source under question, the verdicts are Carry over (it meets the rule), Rebuild (this project needs it, but it breaks the rule) and Avoid (the rules rule it out). Auditing this project's own code, they are Keeps (it meets the rule) and Fix (it breaks it).\n" +
		"\n" +
		"Ask nothing, file no tickets, and change no earlier audit, the memory, its answers or its sources. End with one line naming the doc by its title and counting its findings per verdict."
	testWithAIInstructions = "Test this ticket the way a tester would, then pass or fail it.\n" +
		"\n" +
		"1. Read it with `ticket_get` and its thread with `message_list`. Test its acceptance criteria (for a bug, its expected result). Never test on production. With a `test_target` url, test there. With none, test this checkout on the ticket's linked branch while its pull request is open, else the default branch; if a criterion needs the running app, stop without a result and reply: No test environment yet. Testing never uses production, so add a branch deploy rule with its own network.\n" +
		"2. Check each criterion in turn, the way \"How to verify\" says where the ticket has it. Note what you did and what you saw.\n" +
		"3. Run the project's tests, where it has any: in the repository with the role tests when `project_get` lists one, cloned if needed, on its default branch; else in this checkout, on the ticket's linked branch while its pull request is open, else the default branch.\n" +
		"4. Only where the testing rules in the project's interview memory call for an end-to-end suite and the ticket has a `test_target` url, add or extend a test for the criteria and run it there. Push it in the repository the tests live in: to the ticket's linked branch there while its pull request is open, otherwise to a new branch `<ticket key>-tests` with a pull request whose title starts with the ticket key, linked to the ticket with `ticket_update`. Merge it only as the project's rules allow, after a review by someone who did not write it.\n" +
		"5. Record the result with `ticket_test_report`: `outcome` `pass` when every criterion held and the tests you ran passed, otherwise `fail` with the steps to reproduce, the expected result and the actual one. A pass moves the ticket to done; a fail posts the report to its thread and moves it back to progress. After a pass, delete each linked branch whose pull request merged, which takes its preview down; for one still open, keep it and post a note on the ticket that it passed and still needs merging.\n" +
		"\n" +
		"Reply with each criterion and whether it held, the tests you ran and added, and the result you recorded. If you could not reach the url or run the tests, say what blocked you and record nothing."
	clarifyInstructions = "Find what this doc leaves open and ask its authors. Read it with `doc_get`: its body, its clarification and every earlier round. The instructions for this run, if any, win over everything below.\n" +
		"\n" +
		"A gap is something the developer who ran this would have to guess to build it: who uses it and what each of them does, the steps of a flow, a rule and its exceptions, the data kept, what is in and out of scope, what finished looks like, and how well it must work in the authors' terms, such as how many people, how fast, who may see what, which devices, by when, and what matters most. Never ask how to build it; put anything technical left unclear in your reply to the developer.\n" +
		"\n" +
		"If the previous round has \"Anything else?\" text and no reply yet, send `anything_else_reply` on the `doc_update` this run sends, or alone when it posts no questions: one plain line that answers it, names which new question follows it up, or says where the doc now covers it, with no round number.\n" +
		"\n" +
		"If gaps remain, post one round with `doc_update` `questions`: one to six, the biggest unknown first, one gap per question. Write each in plain words for someone non-technical, with two to four options, the one most projects pick listed first and labelled \"(Suggested)\", multi-select only where options combine, and a one-line why that says why it matters to them. Never repeat a question that was answered or is still waiting; ask a skipped one once more only if it still matters; a gap left out of an earlier round for space still needs asking. Never use your question tool: post that update and end your turn.\n" +
		"\n" +
		"If a question is still waiting and nothing else is open, post no questions and say so in your reply. If none is waiting and the developer could turn the doc into tickets without asking its authors anything more, ask nothing: send `doc_update` with `no_gaps` and the whole new body. Keep the authors' headings and their own sentences wherever they still hold, put each answer and each \"Anything else?\" text in the section it belongs to, adding a section only where nothing fits, and add only what an answer says, with no invented detail, figure or requirement. Questions asked and skipped twice that still matter go under a short \"Open points\" section. Never mention questions or rounds in the doc.\n" +
		"\n" +
		"Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what you asked and why, or that the doc is complete and what changed, plus any technical questions for them."
	updateDocInstructions = "Write the answers to this doc's questions into the doc. Read it with `doc_get`: its body, its clarification and every round. The instructions for this run, if any, win over everything below.\n" +
		"\n" +
		"Ask nothing: send no `questions` and use no question tool. If no question has an answer and no round has \"Anything else?\" text, change nothing and reply that there is nothing to write yet.\n" +
		"\n" +
		"Otherwise send `doc_update` with `no_gaps` and the whole new body. Keep the authors' headings and their own sentences wherever they still hold, put each answer and each \"Anything else?\" text in the section it belongs to, adding a section only where nothing fits, and change only what a newer or changed answer says. Add only what an answer says, with no invented detail, figure or requirement. Because this run asks nothing more, every question skipped or still unanswered that still matters goes under a short \"Open points\" section. Never mention questions or rounds in the doc. If the previous round has \"Anything else?\" text with no reply, send `anything_else_reply` with it: one plain line saying where the doc now covers it, with no round number.\n" +
		"\n" +
		"Never mention AI, an agent, or yourself in anything the authors see. Reply to the developer with what changed in the doc, which points are still open, and any technical questions for them."
)

// ClarifyKey is the built-in key of the Clarify via AI play, whose run opens and ends a round of the doc's
// clarification (ADR 0121).
const ClarifyKey = "clarify"

// UpdateDocKey is the built-in key of the Update doc via AI play, whose run opens a round like Clarify's but only
// writes the answers into the doc.
const UpdateDocKey = "update-doc"

// Service is the plays use-case layer: workspace-scoped play definitions (ADR 0055).
type Service struct {
	repo     Repo
	perm     PermissionGate
	instance InstanceTemplates
	// computers is optional; without it no auto play can be pinned to a computer.
	now func() time.Time
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
	// Repository is app when empty.
	Repository Repository
	Needs      []string
}

// UpdateInput is the owner-edited surface of an existing play; type is immutable after create (not included here).
type UpdateInput struct {
	Label              string
	Description        string
	Instructions       string
	Enabled            bool
	ShowWhenStage      *Stage
	ExcludedProjectIDs []string
	// Repository keeps the play's when empty.
	Repository Repository
	// Needs replaces the play's when set; nil keeps them.
	Needs *[]string
}

// List returns the plays of workspaceID the caller can see, sorted by label, each saying whether they can run it and
// what its runs say about it.
func (s *Service) List(ctx context.Context, workspaceID string) ([]Listed, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	list, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list plays for workspace %s: %w", workspaceID, err)
	}
	out := s.listed(ctx, actorID(ctx), workspaceID, list)
	return out, s.withHealth(ctx, out)
}

// ListApplicable returns workspaceID's enabled plays of playType that apply to a run on projectID for userID:
// not excluded for that project, matching stage for a ticket play, and ones they can see and run.
func (s *Service) ListApplicable(ctx context.Context, workspaceID, userID, projectID string, playType Type, stage *Stage) ([]Listed, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if !playType.valid() {
		return nil, fmt.Errorf("%w: type must be ticket, doc, interview, or stack", apperrs.ErrInvalid)
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
		out = append(out, p)
	}
	return slices.DeleteFunc(s.listed(ctx, userID, workspaceID, out), func(l Listed) bool { return !l.CanRun }), nil
}

// listed keeps the plays of list userID can see, each with whether they can run it, from one read of the rules on them.
func (s *Service) listed(ctx context.Context, userID, workspaceID string, list []*Play) []Listed {
	ids := make([]string, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	sees := s.perm.PlaysWith(ctx, userID, workspaceID, permissions.PlaysRead, ids)
	runs := s.perm.PlaysWith(ctx, userID, workspaceID, permissions.PlaysRun, ids)
	out := make([]Listed, 0, len(list))
	for _, p := range list {
		if sees[p.ID] {
			out = append(out, Listed{Play: *p, CanRun: runs[p.ID]})
		}
	}
	return out
}

// Get returns a single play scoped to workspaceID; a play from another workspace, or one the caller can't see, is
// reported not found.
func (s *Service) Get(ctx context.Context, workspaceID, id string) (*Play, error) {
	return s.seen(ctx, workspaceID, id)
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
		Repository:         cmp.Or(in.Repository, RepositoryApp),
		Needs:              in.Needs,
		CreatedBy:          actorID(ctx),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p, creation(ctx, p), s.event(TopicCreated, CreatedEvent{Play: *p})); err != nil {
		return nil, labelTaken(p, fmt.Errorf("create play in workspace %s: %w", workspaceID, err))
	}
	return p, nil
}

// ApplyEdits is the instructions play_list shows for a play with edits applied, ready for Update; it needs plays:write.
// ponytail: the read and the save are two steps, so instructions saved between them are overwritten; a repo-level apply closes it
func (s *Service) ApplyEdits(ctx context.Context, workspaceID, id string, edits []textedit.Edit) (string, error) {
	p, err := s.seen(ctx, workspaceID, id)
	if err != nil {
		return "", err
	}
	if err := s.requireOn(ctx, p, permissions.PlaysWrite); err != nil {
		return "", err
	}
	text, err := textedit.Apply(p.Instructions, edits)
	if err != nil {
		return "", fmt.Errorf("%w: %w", apperrs.ErrInvalid, err)
	}
	return text, nil
}

// Update replaces a play's owner-edited fields (plays:write on it); the play's type never changes after create.
func (s *Service) Update(ctx context.Context, workspaceID, id string, in UpdateInput) (*Play, error) {
	return s.update(ctx, workspaceID, id, in, 0)
}

// update is Update that records the version a restore copied the instructions from, zero for an ordinary save.
func (s *Service) update(ctx context.Context, workspaceID, id string, in UpdateInput, restoredFrom int) (*Play, error) {
	p, err := s.seen(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireOn(ctx, p, permissions.PlaysWrite); err != nil {
		return nil, err
	}
	before := *p
	p.Label = strings.TrimSpace(in.Label)
	p.Description = strings.TrimSpace(in.Description)
	p.Instructions = in.Instructions
	p.Enabled = in.Enabled
	p.ShowWhenStage = in.ShowWhenStage
	p.ExcludedProjectIDs = normalizeIDs(slices.Concat(in.ExcludedProjectIDs, s.unseenExclusions(ctx, p)))
	p.Repository = cmp.Or(in.Repository, p.Repository)
	if in.Needs != nil {
		p.Needs = *in.Needs
	}
	p.UpdatedAt = s.now().UTC()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p, revision(ctx, &before, p, restoredFrom), s.event(TopicUpdated, UpdatedEvent{Play: *p})); err != nil {
		return nil, labelTaken(p, fmt.Errorf("update play %s: %w", id, err))
	}
	return p, nil
}

// RenameImpact is what renaming a play would break: the automations whose code still calls it by its old label.
type RenameImpact struct {
	Count int `json:"count"`
	// Automations names them for a caller holding automations:read; without it only Count says how many.
	Automations []string `json:"automations"`
}

// RenameImpact reports the automations whose active or pending code names play id's current label in quotes, which
// runPlay stops finding once the label becomes label; none when only the case changes, since runPlay ignores it.
func (s *Service) RenameImpact(ctx context.Context, workspaceID, id, label string) (RenameImpact, error) {
	p, err := s.seen(ctx, workspaceID, id)
	if err != nil {
		return RenameImpact{}, err
	}
	if strings.EqualFold(strings.TrimSpace(label), p.Label) {
		return RenameImpact{}, nil
	}
	names, err := s.repo.AutomationNamesByLabel(ctx, workspaceID, p.Label)
	if err != nil {
		return RenameImpact{}, fmt.Errorf("find automations calling play %s: %w", id, err)
	}
	impact := RenameImpact{Count: len(names)}
	if s.perm.HasPermission(ctx, actorID(ctx), workspaceID, permissions.AutomationsRead, "", "") {
		impact.Automations = names
	}
	return impact, nil
}

// labelTaken words a write the unique label index refused, since a label names the play to automations (ADR 0132).
func labelTaken(p *Play, err error) error {
	if !errors.Is(err, apperrs.ErrConflict) {
		return err
	}
	return fmt.Errorf("%w: a play named %q already exists in this workspace", apperrs.ErrConflict, p.Label)
}

// Delete removes a play; the caller must hold plays:delete on it.
func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	p, err := s.seen(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	if err := s.requireOn(ctx, p, permissions.PlaysDelete); err != nil {
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

// AuditKey is the built-in audit play's key: an interview run that measures code against the interview memory and writes a doc.
const AuditKey = "audit"

// Builtin is one seeded play: the stable key a clone matches it by, and what a new workspace gets.
type Builtin struct {
	Key           string
	Label         string
	Type          Type
	Description   string
	Instructions  string
	ShowWhenStage *Stage
	// AutoPlays are seeded on the play switched off, so a workspace turns them on rather than composing them.
	AutoPlays []AutoPlayInput
}

// Builtins lists the seeded plays in seeding order; their instructions are the instance templates' code defaults.
func Builtins() []Builtin {
	progress, testingStage, done := StageProgress, StageTesting, StageDone
	return []Builtin{
		{Key: "fix-with-ai", Label: "Fix with AI", Type: TypeTicket, ShowWhenStage: &progress,
			Description:  "Builds the ticket on its own branch, proves it, and opens a pull request; the project's memories say how it finishes.",
			Instructions: fixWithAIInstructions, AutoPlays: fixWithAIAutoPlays()},
		{Key: "to-tickets-via-ai", Label: "To tickets via AI", Type: TypeDoc,
			Description:  "Splits a doc into thin, ordered tickets, each with acceptance criteria, how to verify it, and its blockers.",
			Instructions: toTicketsInstructions},
		{Key: "interview", Label: "Interview", Type: TypeInterview,
			Description:  "Asks follow-ups about what the Interview answers and the code leave open, then writes this project's rules for agents.",
			Instructions: interviewInstructions},
		{Key: "test-with-ai", Label: "Test with AI", Type: TypeTicket, ShowWhenStage: &testingStage,
			Description:  "Tests the ticket on its test environment against its acceptance criteria, then passes or fails it.",
			Instructions: testWithAIInstructions, AutoPlays: testWithAIAutoPlays()},
		{Key: DraftInterviewKey, Label: "Draft interview", Type: TypeInterview,
			Description:  "Drafts answers to the Interview questions from this project's follow sources, for a person to confirm.",
			Instructions: draftInterviewInstructions},
		{Key: ClarifyKey, Label: "Clarify via AI", Type: TypeDoc,
			Description:  "Asks the doc's authors about the gaps in what they need, a round at a time, then writes the answers into the doc.",
			Instructions: clarifyInstructions},
		{Key: UpdateDocKey, Label: "Update doc via AI", Type: TypeDoc,
			Description:  "Writes the answers to the doc's questions into the doc, in its authors' words, without asking more.",
			Instructions: updateDocInstructions},
		{Key: AuditKey, Label: "Audit via AI", Type: TypeInterview,
			Description:  "Audits a predecessor's code, or this project's own, against the interview memory and writes the findings as a doc.",
			Instructions: auditInstructions},
		{Key: DecisionsCheckKey, Label: decisionsCheckLabel, Type: TypeTicket, ShowWhenStage: &done,
			Description: decisionsCheckDescription, Instructions: decisionsCheckInstructions, AutoPlays: []AutoPlayInput{decisionsCheckAutoPlay()}},
		{Key: InvestigateCrashKey, Label: investigateCrashLabel, Type: TypeStack,
			Description: investigateCrashDescription, Instructions: investigateCrashInstructions, AutoPlays: []AutoPlayInput{investigateCrashAutoPlay()}},
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
			Instructions: instructions, Enabled: true, ShowWhenStage: b.ShowWhenStage, BuiltinKey: b.Key, Repository: RepositoryApp,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := p.Validate(); err != nil {
			return err
		}
		if err := s.repo.Create(ctx, p, creation(ctx, p), s.event(TopicCreated, CreatedEvent{Play: *p})); err != nil {
			return fmt.Errorf("seed default play %q for workspace %s: %w", p.Label, workspaceID, err)
		}
		for _, in := range b.AutoPlays {
			if err := s.seedAutoPlay(ctx, p, in); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) seedAutoPlay(ctx context.Context, p *Play, in AutoPlayInput) error {
	a := &AutoPlay{ID: ids.New(), PlayID: p.ID, WorkspaceID: p.WorkspaceID, CreatedAt: p.CreatedAt, UpdatedAt: p.CreatedAt}
	a.apply(in)
	if err := s.repo.CreateAutoPlay(ctx, a, s.event(TopicAutoPlayCreated, AutoPlayEvent{AutoPlay: *a})); err != nil {
		return fmt.Errorf("seed the auto play of play %q for workspace %s: %w", p.Label, p.WorkspaceID, err)
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

// BuiltinPlay returns workspaceID's copy of the built-in play key; a deleted one, or one the caller can't see, is not
// found.
func (s *Service) BuiltinPlay(ctx context.Context, workspaceID, key string) (*Play, error) {
	p, err := s.builtin(ctx, workspaceID, key)
	if err != nil {
		return nil, err
	}
	if !s.sees(ctx, p) {
		return nil, fmt.Errorf("%w: workspace %s has no built-in play %q", apperrs.ErrNotFound, workspaceID, key)
	}
	return p, nil
}

// SetBuiltinInstructions replaces the instructions of workspaceID's copy of the built-in play key (plays:write on it).
func (s *Service) SetBuiltinInstructions(ctx context.Context, workspaceID, key, instructions string) (*Play, error) {
	p, err := s.BuiltinPlay(ctx, workspaceID, key)
	if err != nil {
		return nil, err
	}
	if err := s.requireOn(ctx, p, permissions.PlaysWrite); err != nil {
		return nil, err
	}
	before := *p
	p.Instructions = instructions
	p.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, p, revision(ctx, &before, p, 0), s.event(TopicUpdated, UpdatedEvent{Play: *p})); err != nil {
		return nil, fmt.Errorf("update play %s: %w", p.ID, err)
	}
	return p, nil
}

func (s *Service) builtin(ctx context.Context, workspaceID, key string) (*Play, error) {
	list, err := s.repo.List(ctx, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list plays for workspace %s: %w", workspaceID, err)
	}
	if p := findBuiltin(list, key); p != nil {
		return p, nil
	}
	keys := make([]string, 0, len(Builtins()))
	for _, b := range Builtins() {
		keys = append(keys, b.Key)
	}
	return nil, fmt.Errorf("%w: workspace %s has no built-in play %q; it was deleted, or the key is not one of %s", apperrs.ErrNotFound, workspaceID, key, strings.Join(keys, ", "))
}

// findBuiltin is the play in list seeded as key, nil when it was deleted.
func findBuiltin(list []*Play, key string) *Play {
	key = strings.TrimSpace(key)
	for _, p := range list {
		if p.BuiltinKey != "" && strings.EqualFold(p.BuiltinKey, key) {
			return p
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

// seen is workspaceID's play id, reported not found when the caller can't see it, as when it isn't there.
func (s *Service) seen(ctx context.Context, workspaceID, id string) (*Play, error) {
	p, err := s.getInWorkspace(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if !s.sees(ctx, p) {
		return nil, fmt.Errorf("get play %s: %w", p.ID, apperrs.ErrNotFound)
	}
	return p, nil
}

// sees is whether the caller holds plays:read on p, its rules included.
func (s *Service) sees(ctx context.Context, p *Play) bool {
	return s.perm.HasPermission(ctx, actorID(ctx), p.WorkspaceID, permissions.PlaysRead, resourceTypePlay, p.ID)
}

// requireOn refuses a caller who can see p but doesn't hold action on it.
func (s *Service) requireOn(ctx context.Context, p *Play, action permissions.Action) error {
	if !s.perm.HasPermission(ctx, actorID(ctx), p.WorkspaceID, action, resourceTypePlay, p.ID) {
		return fmt.Errorf("%w: %s required on play %q", apperrs.ErrForbidden, action, p.Label)
	}
	return nil
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
