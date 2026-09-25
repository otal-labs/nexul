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

// Users' installed nexul-memory skill copies name these memory tools and arguments and are never rewritten: keep both.
type memoryListIn struct {
	ProjectID   string `json:"project_id,omitempty" jsonschema:"A project's id: lists its workspace memories first, then its own."`
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"A workspace's id: lists only its workspace memories. Required when project_id is omitted."`
	mcptool.PageArgs
}

type memoryGetIn struct {
	ID      string `json:"id" jsonschema:"The memory's id, from memory_list or the turn's memories index."`
	Version int    `json:"version,omitzero" jsonschema:"A version number from 1 up, to read that version's content instead of the current one."`
}

type memoryCreateIn struct {
	ProjectID      string `json:"project_id,omitempty" jsonschema:"The project the memory belongs to. Omit to save at workspace scope, where workspace_id is then required."`
	WorkspaceID    string `json:"workspace_id,omitempty" jsonschema:"The workspace for a workspace-scoped memory; ignored when project_id is set."`
	Title          string `json:"title,omitempty" jsonschema:"The memory's title, for example Deploy quirks. Required for an ordinary memory."`
	WhenToUse      string `json:"when_to_use,omitempty" jsonschema:"One short line saying when the memory applies, for example use this if you are writing React code."`
	Body           string `json:"body,omitempty" jsonschema:"The memory's body as markdown."`
	AlwaysIncluded bool   `json:"always_included,omitzero" jsonschema:"true inlines the memory in full in every agent turn it reaches. Defaults to false."`
	Kind           string `json:"kind,omitempty" jsonschema:"Omit for an ordinary memory. decisions_log creates the project's decisions log; interview, sent with project_id alone, returns the project's interview memory, creating it from the Interview template the first time."`
	CloneFromID    string `json:"clone_from_id,omitempty" jsonschema:"The id of a memory to copy, from memory_list, with its attachments, into project_id or workspace_id instead of writing a new one."`
}

type memoryUpdateIn struct {
	ID              string  `json:"id" jsonschema:"The memory's id, from memory_list."`
	Title           *string `json:"title,omitempty" jsonschema:"New title. Omit to keep the current one."`
	WhenToUse       *string `json:"when_to_use,omitempty" jsonschema:"New when-to-use line; an empty string clears it. Omit to keep the current one."`
	Body            *string `json:"body,omitempty" jsonschema:"New body as markdown, replacing the whole body. Omit to keep the current body."`
	AlwaysIncluded  *bool   `json:"always_included,omitempty" jsonschema:"Whether every agent turn inlines the memory in full. Omit to keep the current setting."`
	RevertToVersion *int    `json:"revert_to_version,omitempty" jsonschema:"Restore this version's title, when-to-use, body, and flag as a new version. Send it without the other fields."`
}

type memoryDeleteIn struct {
	ID string `json:"id" jsonschema:"The memory's id, from memory_list."`
}

type templateGetIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace whose Interview template to read."`
}

type templateUpdateIn struct {
	WorkspaceID string  `json:"workspace_id" jsonschema:"The workspace whose Interview template to change."`
	Body        *string `json:"body,omitempty" jsonschema:"The new template as markdown, at most 8,000 characters. Omit to keep the current template."`
}

// memoryListItem is a memory's index entry: enough to decide relevance, never the body.
type memoryListItem struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id,omitempty"`
	Kind           string `json:"kind,omitempty"`
	Title          string `json:"title"`
	WhenToUse      string `json:"when_to_use"`
	AlwaysIncluded bool   `json:"always_included"`
}

// memoryResult is a memory with its body as markdown; CurrentVersion is set only when an older version is shown.
type memoryResult struct {
	ID             string              `json:"id"`
	WorkspaceID    string              `json:"workspace_id"`
	ProjectID      string              `json:"project_id,omitempty"`
	Kind           string              `json:"kind,omitempty"`
	Title          string              `json:"title"`
	WhenToUse      string              `json:"when_to_use"`
	Body           string              `json:"body"`
	AlwaysIncluded bool                `json:"always_included"`
	Version        int                 `json:"version"`
	CurrentVersion int                 `json:"current_version,omitempty"`
	UpdatedBy      string              `json:"updated_by"`
	UpdatedAt      time.Time           `json:"updated_at"`
	Versions       []memoryVersionInfo `json:"versions,omitempty"`
}

type memoryVersionInfo struct {
	Version   int       `json:"version"`
	Title     string    `json:"title"`
	AuthorID  string    `json:"author_id"`
	AuthorVia string    `json:"author_via,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MCPTools returns the memories and Interview template tools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		memoryListTool(s), memoryGetTool(s), memoryCreateTool(s), memoryUpdateTool(s), memoryDeleteTool(s),
		templateGetTool(s), templateUpdateTool(s),
	}
}

func memoryListTool(s *Service) mcptool.Tool {
	return mcptool.New("memory_list", "List memories",
		"Lists memories by title and when-to-use line only, no body. "+
			"With project_id it returns the project's workspace memories first, then its own; with workspace_id alone, only the workspace's. "+
			"Pick the ones whose when-to-use matches your task and read them with memory_get.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in memoryListIn) (any, error) {
			ms, err := listMemories(ctx, s, in.ProjectID, in.WorkspaceID)
			if err != nil {
				return nil, err
			}
			items := make([]memoryListItem, 0, len(ms))
			for _, m := range ms {
				items = append(items, memoryListItem{
					ID: m.ID, ProjectID: m.ProjectID, Kind: m.Kind, Title: m.Title, WhenToUse: m.WhenToUse, AlwaysIncluded: m.AlwaysIncluded,
				})
			}
			return mcptool.Paginate(items, in.PageArgs), nil
		})
}

func listMemories(ctx context.Context, s *Service, projectID, workspaceID string) ([]*Memory, error) {
	if projectID != "" {
		return s.ListForProject(ctx, projectID)
	}
	return s.ListWorkspaceScoped(ctx, workspaceID)
}

func memoryGetTool(s *Service) mcptool.Tool {
	return mcptool.New("memory_get", "Get memory",
		"Returns one memory with its full body as markdown and its newest 50 versions (number, title, author, and time). "+
			"With version it returns that version's title, when-to-use, body, and flag instead, and current_version says which is live. "+
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
			out, err := toMemoryResult(m)
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
		"Saves a note for agents at project scope (project_id) or workspace scope (workspace_id), or copies one with clone_from_id. "+
			"Save a durable fact worth remembering; if a memory already covers the ground, change it with memory_update instead. "+
			"kind decisions_log creates the project's decisions log (one per project, never sent every turn), and kind interview returns the project's interview memory, "+
			"creating it from the Interview template the first time; both need project_id. "+
			"Returns the memory with its body as markdown.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in memoryCreateIn) (any, error) {
			m, err := createMemory(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return toMemoryResult(m)
		})
}

func createMemory(ctx context.Context, s *Service, in memoryCreateIn) (*Memory, error) {
	if in.CloneFromID != "" {
		if in.hasContent() {
			return nil, fmt.Errorf("%w: clone_from_id copies the source's content; omit kind, title, when_to_use, body, and always_included, then change the copy with memory_update", apperrs.ErrInvalid)
		}
		return s.Clone(ctx, in.CloneFromID, in.ProjectID, in.WorkspaceID)
	}
	if in.Kind == KindInterview {
		if in.hasText() {
			return nil, fmt.Errorf("%w: kind interview returns the project's interview memory as it stands; omit title, when_to_use, body, and always_included, then change it with memory_update", apperrs.ErrInvalid)
		}
		return s.CreateInterview(ctx, in.ProjectID, viaMCP)
	}
	return s.CreateWithKind(ctx, in.Kind, in.ProjectID, in.WorkspaceID, in.Title, in.WhenToUse, in.Body, in.AlwaysIncluded, viaMCP)
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
			"Returns the memory as it now stands.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in memoryUpdateIn) (any, error) {
			m, err := updateMemory(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return toMemoryResult(m)
		})
}

func updateMemory(ctx context.Context, s *Service, in memoryUpdateIn) (*Memory, error) {
	if in.RevertToVersion != nil {
		if in.Title != nil || in.WhenToUse != nil || in.Body != nil || in.AlwaysIncluded != nil {
			return nil, fmt.Errorf("%w: revert_to_version restores that version's whole content; send it alone, then update again", apperrs.ErrInvalid)
		}
		return s.Revert(ctx, in.ID, *in.RevertToVersion, viaMCP)
	}
	m, err := s.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if in.Title == nil && in.WhenToUse == nil && in.Body == nil && in.AlwaysIncluded == nil {
		return m, nil
	}
	return s.Update(ctx, in.ID, deref(in.Title, m.Title), deref(in.WhenToUse, m.WhenToUse), deref(in.Body, m.Body),
		deref(in.AlwaysIncluded, m.AlwaysIncluded), viaMCP)
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

func templateGetTool(s *Service) mcptool.Tool {
	return mcptool.New("interview_template_get", "Get Interview template",
		"Returns the workspace's Interview template as markdown, the headings each new project's interview memory starts from, plus default_body, the seeded template. "+
			"Use it to see what an interview will ask before running one. "+
			"Change it with interview_template_update.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in templateGetIn) (any, error) {
			return s.InterviewTemplate(ctx, in.WorkspaceID)
		})
}

func templateUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("interview_template_update", "Update Interview template",
		"Replaces the workspace's Interview template with a markdown body of at most 8,000 characters. "+
			"Only projects whose interview memory is created afterwards start from it; existing interview memories are not changed, so edit those with memory_update. "+
			"Returns the template as it now stands.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in templateUpdateIn) (any, error) {
			if in.Body == nil {
				return s.InterviewTemplate(ctx, in.WorkspaceID)
			}
			return s.SaveInterviewTemplate(ctx, in.WorkspaceID, *in.Body)
		})
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
		Body: md, AlwaysIncluded: m.AlwaysIncluded, Version: m.Version, UpdatedBy: m.UpdatedBy, UpdatedAt: m.UpdatedAt,
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
