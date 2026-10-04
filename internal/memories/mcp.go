package memories

import (
	"context"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// viaMCP is the author_via of every version saved through an MCP tool call (ADR 0049).
const viaMCP = "mcp"

// memoryGetVersions bounds the version history memory_get returns; older versions stay readable by number.
const memoryGetVersions = 50

type memoryListIn struct {
	ProjectID string `json:"project_id" jsonschema:"The project whose memories to list, from project_list."`
	mcptool.PageArgs
}

type memoryGetIn struct {
	ID      string `json:"id" jsonschema:"The memory's id, from memory_list or the memories a turn names."`
	Version int    `json:"version,omitzero" jsonschema:"A version number from 1 up, to read that version's content instead of the current one."`
}

type memoryCreateIn struct {
	ProjectID      string `json:"project_id" jsonschema:"The project the memory belongs to, or the destination of a copy, from project_list."`
	Title          string `json:"title,omitempty" jsonschema:"The memory's title, for example Deploy quirks. Required for an ordinary memory."`
	WhenToUse      string `json:"when_to_use,omitempty" jsonschema:"One short line saying when the memory applies, for example use this if you are writing React code."`
	Body           string `json:"body,omitempty" jsonschema:"The memory's body as markdown."`
	AlwaysIncluded bool   `json:"always_included,omitzero" jsonschema:"true names the memory, to read first, in every agent turn in its project. Defaults to false."`
	Kind           string `json:"kind,omitempty" jsonschema:"Omit for an ordinary memory. decisions_log creates the project's decisions log; interview, sent with project_id alone, returns the project's interview memory, creating it empty the first time."`
	CloneFromID    string `json:"clone_from_id,omitempty" jsonschema:"The id of a memory to copy, from memory_list, with its attachments, into project_id instead of writing a new one."`
}

type memoryUpdateIn struct {
	ID              string  `json:"id" jsonschema:"The memory's id, from memory_list."`
	Title           *string `json:"title,omitempty" jsonschema:"New title. Omit to keep the current one."`
	WhenToUse       *string `json:"when_to_use,omitempty" jsonschema:"New when-to-use line; an empty string clears it. Omit to keep the current one."`
	Body            *string `json:"body,omitempty" jsonschema:"New body as markdown, replacing the whole body. Omit to keep the current body."`
	AlwaysIncluded  *bool   `json:"always_included,omitempty" jsonschema:"Whether every agent turn in its project names the memory to read first. Omit to keep the current setting."`
	Footer          *bool   `json:"footer,omitempty" jsonschema:"Whether the memory sits in the Footer folder: a play run names it last, to read once its work is done and conclude the run, for example which column the ticket belongs in. Ordinary memories only. Omit to keep the current setting."`
	RevertToVersion *int    `json:"revert_to_version,omitempty" jsonschema:"Restore this version's title, when-to-use, body, and flag as a new version. Send it without the other fields."`

	Answers []answerIn `json:"answers,omitempty" jsonschema:"Interview memory only: answers to the Interview template's questions, each replacing that question's stored answer. Follow-up rounds are recorded by the run, not here."`

	AddSources    []sourceIn       `json:"add_sources,omitempty" jsonschema:"Interview memory only: material to point the interview at, each with its stance. A doc, memory, or project must be one you can read in this workspace; at most 50 sources per project."`
	UpdateSources []sourceChangeIn `json:"update_sources,omitempty" jsonschema:"Interview memory only: sources whose stance, or pasted text's label, to change, by id from memory_get."`
	RemoveSources []string         `json:"remove_sources,omitempty" jsonschema:"Interview memory only: ids of sources to take off the interview, from memory_get. Drafts made from them stay."`
	Drafts        []draftIn        `json:"drafts,omitempty" jsonschema:"Interview memory only: draft answers to the template's questions, each replacing that question's earlier draft. A draft never counts as an answer; one equal to the stored answer is dropped."`
	DismissDrafts []string         `json:"dismiss_drafts,omitempty" jsonschema:"Interview memory only: ids of drafts to delete, from memory_get."`
}

type sourceIn struct {
	Kind   string `json:"kind" jsonschema:"path, doc, memory, project, or text."`
	Ref    string `json:"ref,omitempty" jsonschema:"For path, a file or folder relative to the project's checkout, for example practices/testing.md; for doc, memory, or project, its id. Omit for text."`
	Label  string `json:"label,omitempty" jsonschema:"Text only: the name the pasted text shows under, for example Team wiki notes."`
	Body   string `json:"body,omitempty" jsonschema:"Text only: the pasted text, at most 32,000 characters."`
	Stance string `json:"stance" jsonschema:"follow to draft answers from it, or question for how something was done but should not be copied, only asked about."`
}

type sourceChangeIn struct {
	ID     string  `json:"id" jsonschema:"The source's id, from memory_get."`
	Stance *string `json:"stance,omitempty" jsonschema:"follow or question. Omit to keep the current stance."`
	Label  *string `json:"label,omitempty" jsonschema:"Text sources only: a new label. Omit to keep the current one."`
}

type draftIn struct {
	Question  string   `json:"question" jsonschema:"The template question's text exactly as the interview memory's questions list it."`
	Selected  []string `json:"selected,omitempty" jsonschema:"The picked options' labels."`
	Text      string   `json:"text,omitempty" jsonschema:"A free-text answer, alone or beside the picked options."`
	SourceIDs []string `json:"source_ids" jsonschema:"The ids of the follow sources the draft came from, from memory_get."`
	Where     string   `json:"where,omitempty" jsonschema:"One line, at most 500 characters, saying where in the sources it came from, for example practices/testing.md, Test error paths first."`
	TrailID   string   `json:"trail_id,omitempty" jsonschema:"The id of the play run's trail writing the draft, when the run names it."`
}

type answerIn struct {
	Question string   `json:"question" jsonschema:"The question's text exactly as the interview memory's questions list it, for example Testing."`
	Selected []string `json:"selected,omitempty" jsonschema:"The picked options' labels."`
	Text     string   `json:"text,omitempty" jsonschema:"A free-text answer, alone or beside the picked options."`
	Skip     bool     `json:"skip,omitzero" jsonschema:"true skips the question instead of answering it; omit selected and text."`
}

type memoryDeleteIn struct {
	ID string `json:"id" jsonschema:"The memory's id, from memory_list."`
}

// memoryListItem is a memory's index entry: enough to decide relevance, never the body.
type memoryListItem struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	Kind           string `json:"kind,omitempty"`
	Title          string `json:"title"`
	WhenToUse      string `json:"when_to_use"`
	AlwaysIncluded bool   `json:"always_included"`
	Footer         bool   `json:"footer"`
}

// memoryResult is a memory with its body as markdown; CurrentVersion is set only when an older version is shown.
type memoryResult struct {
	ID             string              `json:"id"`
	WorkspaceID    string              `json:"workspace_id"`
	ProjectID      string              `json:"project_id"`
	Kind           string              `json:"kind,omitempty"`
	Title          string              `json:"title"`
	WhenToUse      string              `json:"when_to_use"`
	Body           string              `json:"body"`
	AlwaysIncluded bool                `json:"always_included"`
	Footer         bool                `json:"footer"`
	Version        int                 `json:"version"`
	CurrentVersion int                 `json:"current_version,omitempty"`
	UpdatedBy      string              `json:"updated_by"`
	UpdatedAt      time.Time           `json:"updated_at"`
	Versions       []memoryVersionInfo `json:"versions,omitempty"`
	Questions      []Question          `json:"questions,omitempty"`
	Answers        []answerResult      `json:"answers,omitzero"`
	Sources        []sourceResult      `json:"sources,omitzero"`
	Drafts         []draftResult       `json:"drafts,omitzero"`
}

// sourceResult is one interview source; label is empty and not_visible set when the caller cannot read its ref.
type sourceResult struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`
	Ref          string     `json:"ref,omitempty"`
	Label        string     `json:"label,omitempty"`
	Body         string     `json:"body,omitempty"`
	Stance       string     `json:"stance"`
	NotVisible   bool       `json:"not_visible,omitempty"`
	Gone         bool       `json:"gone,omitempty"`
	RefUpdatedAt *time.Time `json:"ref_updated_at,omitempty"`
	AddedBy      string     `json:"added_by"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// draftResult is one draft answer a drafting run wrote.
type draftResult struct {
	ID        string    `json:"id"`
	Question  string    `json:"question"`
	Selected  []string  `json:"selected"`
	Text      string    `json:"text"`
	SourceIDs []string  `json:"source_ids"`
	Where     string    `json:"where,omitempty"`
	TrailID   string    `json:"trail_id,omitempty"`
	DraftedAt time.Time `json:"drafted_at"`
}

// answerResult is one stored interview answer; options and why are set on follow-ups only.
type answerResult struct {
	Round       int            `json:"round"`
	Question    string         `json:"question"`
	Options     []AnswerOption `json:"options,omitempty"`
	MultiSelect bool           `json:"multi_select,omitempty"`
	Why         string         `json:"why,omitempty"`
	Selected    []string       `json:"selected"`
	Text        string         `json:"text"`
	Skipped     bool           `json:"skipped"`
	AnsweredBy  string         `json:"answered_by"`
	AnsweredAt  time.Time      `json:"answered_at"`
}

type memoryVersionInfo struct {
	Version   int       `json:"version"`
	Title     string    `json:"title"`
	AuthorID  string    `json:"author_id"`
	AuthorVia string    `json:"author_via,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MCPTools returns the memories tools; the Interview template is read and changed through template_get and template_update.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		memoryListTool(s), memoryGetTool(s), memoryCreateTool(s), memoryUpdateTool(s), memoryDeleteTool(s),
	}
}

func memoryListTool(s *Service) mcptool.Tool {
	return mcptool.New("memory_list", "List memories",
		"Lists a project's memories by title and when-to-use line only, no body. "+
			"Every memory belongs to one project, so project_id is required. "+
			"Pick the ones whose when-to-use matches your task and read them with memory_get.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in memoryListIn) (any, error) {
			ms, err := s.ListForProject(ctx, in.ProjectID)
			if err != nil {
				return nil, err
			}
			items := make([]memoryListItem, 0, len(ms))
			for _, m := range ms {
				items = append(items, memoryListItem{
					ID: m.ID, ProjectID: m.ProjectID, Kind: m.Kind, Title: m.Title, WhenToUse: m.WhenToUse, AlwaysIncluded: m.AlwaysIncluded, Footer: m.Footer,
				})
			}
			return mcptool.Paginate(items, in.PageArgs), nil
		})
}

func memoryGetTool(s *Service) mcptool.Tool {
	return mcptool.New("memory_get", "Get memory",
		"Returns one memory with its full body as markdown and its newest 50 versions (number, title, author, and time); the interview memory also carries the questions of the workspace's Interview template. "+
			"With version it returns that version's title, when-to-use, body, and flag instead, and current_version says which is live. "+
			"For the interview memory, answers holds the project's stored answers, round 0 for the template's questions and 1 and up for follow-up rounds; "+
			"sources lists what the interview is pointed at with its stance, pasted text bodies included, and drafts the draft answers a drafting run wrote. "+
			"Use memory_list to find ids, and memory_update with revert_to_version to restore an old version.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in memoryGetIn) (any, error) {
			m, err := s.Get(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			vs, err := s.ListVersions(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := memoryOut(ctx, s, m)
			if err != nil {
				return nil, err
			}
			out.Versions = versionInfos(vs)
			if in.Version == 0 {
				return out, nil
			}
			v, err := s.GetVersion(ctx, in.ID, in.Version)
			if err != nil {
				return nil, err
			}
			return withVersion(out, v)
		})
}

func memoryCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("memory_create", "Create memory",
		"Saves a note for agents in a project, or copies one into it with clone_from_id; every memory belongs to one project. "+
			"Save a durable fact worth remembering; if a memory already covers the ground, change it with memory_update instead. "+
			"A memory holds reusable technical guidance for agents, such as research, architecture, coding standards, and library choices, while requirements, scope, and business processes people read go in doc_create. "+
			"kind decisions_log creates the project's decisions log (one per project, never sent every turn), and kind interview returns the project's interview memory, "+
			"creating it empty the first time, with the questions of the workspace's Interview template. "+
			"Copying needs memories:clone on the source and memories:write at the destination. "+
			"Returns the memory with its body as markdown.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in memoryCreateIn) (any, error) {
			m, err := createMemory(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return memoryOut(ctx, s, m)
		})
}

func createMemory(ctx context.Context, s *Service, in memoryCreateIn) (*Memory, error) {
	if in.CloneFromID != "" {
		if in.hasContent() {
			return nil, fmt.Errorf("%w: clone_from_id copies the source's content; omit kind, title, when_to_use, body, and always_included, then change the copy with memory_update", apperrs.ErrInvalid)
		}
		return s.Clone(ctx, in.CloneFromID, in.ProjectID)
	}
	if in.Kind == KindInterview {
		if in.hasText() {
			return nil, fmt.Errorf("%w: kind interview returns the project's interview memory as it stands; omit title, when_to_use, body, and always_included, then change it with memory_update", apperrs.ErrInvalid)
		}
		return s.CreateInterview(ctx, in.ProjectID, viaMCP)
	}
	return s.CreateWithKind(ctx, in.Kind, in.ProjectID, in.Title, in.WhenToUse, in.Body, in.AlwaysIncluded, viaMCP)
}

func (in memoryCreateIn) hasContent() bool {
	return in.Kind != "" || in.hasText()
}

func (in memoryCreateIn) hasText() bool {
	return in.Title != "" || in.WhenToUse != "" || in.Body != "" || in.AlwaysIncluded
}

func memoryUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("memory_update", "Update memory",
		"Changes a memory's title, when-to-use line, body, or always-included flag; only the fields you send change, and each save adds a version. "+
			"revert_to_version restores an earlier version as a new one; memory_get lists the versions. "+
			"The interview memory stays always included and its body is capped at 8,000 characters of markdown; the decisions log is never always included. "+
			"answers saves or skips the interview's answers to the template's questions without adding a version; "+
			"add_sources, update_sources, and remove_sources change what the interview is pointed at, and drafts and dismiss_drafts write or delete draft answers, which never count as answers. "+
			"Returns the memory as it now stands.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in memoryUpdateIn) (any, error) {
			m, err := updateMemory(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return memoryOut(ctx, s, m)
		})
}

func updateMemory(ctx context.Context, s *Service, in memoryUpdateIn) (*Memory, error) {
	if in.RevertToVersion != nil {
		if in.Title != nil || in.WhenToUse != nil || in.Body != nil || in.AlwaysIncluded != nil || in.Footer != nil || in.hasInterview() {
			return nil, fmt.Errorf("%w: revert_to_version restores that version's whole content; send it alone, then update again", apperrs.ErrInvalid)
		}
		return s.Revert(ctx, in.ID, *in.RevertToVersion, viaMCP)
	}
	m, err := s.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := applyInterview(ctx, s, m, in); err != nil {
		return nil, err
	}
	if in.Title == nil && in.WhenToUse == nil && in.Body == nil && in.AlwaysIncluded == nil && in.Footer == nil {
		return m, nil
	}
	return s.UpdateWithFooter(ctx, in.ID, deref(in.Title, m.Title), deref(in.WhenToUse, m.WhenToUse), deref(in.Body, m.Body),
		deref(in.AlwaysIncluded, m.AlwaysIncluded), in.Footer, viaMCP)
}

func memoryDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("memory_delete", "Delete memory",
		"Deletes a memory and its version history for good. "+
			"Prefer memory_update to correct a wrong or stale memory, so its history survives. "+
			"Returns the deleted id.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in memoryDeleteIn) (any, error) {
			if err := s.Delete(ctx, in.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(in.ID), nil
		})
}

func (in memoryUpdateIn) hasInterview() bool {
	return in.Answers != nil || in.AddSources != nil || in.UpdateSources != nil || in.RemoveSources != nil || in.Drafts != nil || in.DismissDrafts != nil
}

// applyInterview applies the interview fields in order, stopping at the first refusal; each step says what took effect.
func applyInterview(ctx context.Context, s *Service, m *Memory, in memoryUpdateIn) error {
	if !in.hasInterview() {
		return nil
	}
	if m.Kind != KindInterview {
		return fmt.Errorf("%w: answers, sources, and drafts belong to the interview memory; memory_list shows which memory has kind interview", apperrs.ErrInvalid)
	}
	if err := saveAnswers(ctx, s, m, in.Answers); err != nil {
		return err
	}
	if err := changeSources(ctx, s, m, in); err != nil {
		return err
	}
	if len(in.Drafts) > 0 {
		drafts := make([]InterviewDraft, 0, len(in.Drafts))
		for _, d := range in.Drafts {
			drafts = append(drafts, InterviewDraft{Question: d.Question, Selected: d.Selected, Text: d.Text, SourceIDs: d.SourceIDs, Where: d.Where, TrailID: d.TrailID})
		}
		if _, err := s.SaveDrafts(ctx, m.ProjectID, drafts); err != nil {
			return fmt.Errorf("drafts, after the answers and sources were saved: %w", err)
		}
	}
	for i, id := range in.DismissDrafts {
		if err := s.DismissDraft(ctx, id); err != nil {
			return fmt.Errorf("dismissing draft %d of %d (%s), the ones before it are gone: %w", i+1, len(in.DismissDrafts), id, err)
		}
	}
	return nil
}

// changeSources removes, then changes, then adds sources, so a full list can be swapped in one call.
func changeSources(ctx context.Context, s *Service, m *Memory, in memoryUpdateIn) error {
	for i, id := range in.RemoveSources {
		if err := s.RemoveSource(ctx, id); err != nil {
			return fmt.Errorf("removing source %d of %d (%s), the ones before it are removed: %w", i+1, len(in.RemoveSources), id, err)
		}
	}
	for i, c := range in.UpdateSources {
		if _, err := s.UpdateSource(ctx, c.ID, c.Stance, c.Label); err != nil {
			return fmt.Errorf("changing source %d of %d (%s), the ones before it are changed: %w", i+1, len(in.UpdateSources), c.ID, err)
		}
	}
	for i, a := range in.AddSources {
		_, err := s.AddSource(ctx, InterviewSource{ProjectID: m.ProjectID, Kind: a.Kind, Ref: a.Ref, Label: a.Label, Body: a.Body, Stance: a.Stance})
		if err != nil {
			return fmt.Errorf("adding source %d of %d (%s %s), the ones before it are added: %w", i+1, len(in.AddSources), a.Kind, a.Ref, err)
		}
	}
	return nil
}

// saveAnswers stores an agent's answers to the template's questions, stopping at the first refused one.
func saveAnswers(ctx context.Context, s *Service, m *Memory, answers []answerIn) error {
	for i, a := range answers {
		_, err := s.SaveAnswer(ctx, InterviewAnswer{ProjectID: m.ProjectID, Question: a.Question, Selected: a.Selected, Text: a.Text, Skipped: a.Skip})
		if err != nil {
			return fmt.Errorf("answer %d of %d (%q), the ones before it are saved: %w", i+1, len(answers), a.Question, err)
		}
	}
	return nil
}

// memoryOut shapes a memory for a tool result, with the questions and stored answers when it is the interview memory.
func memoryOut(ctx context.Context, s *Service, m *Memory) (memoryResult, error) {
	out, err := toMemoryResult(m)
	if err != nil || m.Kind != KindInterview {
		return out, err
	}
	if out.Questions, err = s.InterviewQuestions(ctx, m); err != nil {
		return memoryResult{}, err
	}
	as, err := s.ListAnswers(ctx, m.ProjectID)
	if err != nil {
		return memoryResult{}, err
	}
	out.Answers = make([]answerResult, 0, len(as))
	for _, a := range as {
		out.Answers = append(out.Answers, answerResult{
			Round: a.Round, Question: a.Question, Options: a.Options, MultiSelect: a.MultiSelect, Why: a.Why,
			Selected: a.Selected, Text: a.Text, Skipped: a.Skipped, AnsweredBy: a.AnsweredBy, AnsweredAt: a.AnsweredAt,
		})
	}
	if out.Sources, err = sourceResults(ctx, s, m.ProjectID); err != nil {
		return memoryResult{}, err
	}
	ds, err := s.ListDrafts(ctx, m.ProjectID)
	if err != nil {
		return memoryResult{}, err
	}
	out.Drafts = make([]draftResult, 0, len(ds))
	for _, d := range ds {
		out.Drafts = append(out.Drafts, draftResult{
			ID: d.ID, Question: d.Question, Selected: d.Selected, Text: d.Text, SourceIDs: d.SourceIDs, Where: d.Where, TrailID: d.TrailID, DraftedAt: d.DraftedAt,
		})
	}
	return out, nil
}

func sourceResults(ctx context.Context, s *Service, projectID string) ([]sourceResult, error) {
	srcs, err := s.ListSources(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]sourceResult, 0, len(srcs))
	for _, src := range srcs {
		out = append(out, sourceResult{
			ID: src.ID, Kind: src.Kind, Ref: src.Ref, Label: src.Label, Body: src.Body, Stance: src.Stance,
			NotVisible: src.NotVisible, Gone: src.Gone, RefUpdatedAt: src.RefUpdatedAt, AddedBy: src.AddedBy, UpdatedAt: src.UpdatedAt,
		})
	}
	return out, nil
}

func deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

func toMemoryResult(m *Memory) (memoryResult, error) {
	md, err := richtext.ToMarkdown(m.Body)
	if err != nil {
		return memoryResult{}, fmt.Errorf("render memory %s: %w", m.ID, err)
	}
	return memoryResult{
		ID: m.ID, WorkspaceID: m.WorkspaceID, ProjectID: m.ProjectID, Kind: m.Kind, Title: m.Title, WhenToUse: m.WhenToUse,
		Body: md, AlwaysIncluded: m.AlwaysIncluded, Footer: m.Footer, Version: m.Version, UpdatedBy: m.UpdatedBy, UpdatedAt: m.UpdatedAt,
	}, nil
}

// withVersion shows an older version's content in place of the current one.
func withVersion(out memoryResult, v *MemoryVersion) (memoryResult, error) {
	md, err := richtext.ToMarkdown(v.Body)
	if err != nil {
		return memoryResult{}, fmt.Errorf("render memory %s version %d: %w", out.ID, v.Version, err)
	}
	out.CurrentVersion = out.Version
	out.Version, out.Title, out.WhenToUse, out.Body, out.AlwaysIncluded = v.Version, v.Title, v.WhenToUse, md, v.AlwaysIncluded
	out.UpdatedBy, out.UpdatedAt = v.AuthorID, v.CreatedAt
	return out, nil
}

// versionInfos keeps the newest versions; the repo returns them newest first.
func versionInfos(vs []*MemoryVersion) []memoryVersionInfo {
	out := make([]memoryVersionInfo, 0, min(len(vs), memoryGetVersions))
	for _, v := range vs[:min(len(vs), memoryGetVersions)] {
		out = append(out, memoryVersionInfo{Version: v.Version, Title: v.Title, AuthorID: v.AuthorID, AuthorVia: v.AuthorVia, CreatedAt: v.CreatedAt})
	}
	return out
}
