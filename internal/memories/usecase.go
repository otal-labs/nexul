package memories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// maxWhenToUseChars caps the one-line when-to-use field, mirroring docs' former memory field.
const maxWhenToUseChars = 200

// AccessGate is the slice of access memories consumes (ADR 0017): a memory is checked through its project
// (ADR 0099), the Interview template through its workspace. A call with no actor is the server's own and passes.
type AccessGate interface {
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// ProjectLookup resolves a project's workspace so a memory can denormalize workspace_id at creation (ADR 0017).
type ProjectLookup interface {
	WorkspaceForProject(ctx context.Context, projectID string) (string, error)
}

// AttachmentsCopier duplicates a memory's attachments under a clone's id (ADR 0017: memories never imports
// attachments). ListOwnerIDs runs before the clone row exists so its body can be rewritten to the new ids up
// front; CopyOwnerWithIDs then creates the copies under those pre-assigned ids once the clone row is there to
// satisfy the FK.
type AttachmentsCopier interface {
	ListOwnerIDs(ctx context.Context, memoryID string) ([]string, error)
	CopyOwnerWithIDs(ctx context.Context, fromMemoryID, toMemoryID string, idMap map[string]string) error
}

// Service is the memories use-case layer (ADR 0019); permission checks run here so every adapter inherits them.
type Service struct {
	repo        Repo
	access      AccessGate
	projects    ProjectLookup
	attachments AttachmentsCopier
	instance    InstanceTemplates
	now         func() time.Time
}

// InstanceTemplates reads the instance's text for a template kind and key, the code default until edited (ADR 0103).
type InstanceTemplates interface {
	Effective(ctx context.Context, kind, key string) (string, error)
}

// SetInstanceTemplates wires the instance layer an unedited workspace Interview template follows.
func (s *Service) SetInstanceTemplates(t InstanceTemplates) { s.instance = t }

// NewService wires the memories use-cases over the given repo, access gate, project lookup, and attachments
// copier (for Clone).
func NewService(repo Repo, access AccessGate, projects ProjectLookup, attachmentsCopier AttachmentsCopier) *Service {
	return &Service{repo: repo, access: access, projects: projects, attachments: attachmentsCopier, now: time.Now}
}

// Create validates and persists a new memory in projectID as version 1, enqueuing memory.created; a memory always
// names a project (ADR 0099). via is "mcp" when the call came through an MCP tool (ADR 0049), empty for the browser.
func (s *Service) Create(ctx context.Context, projectID, title, whenToUse, body string, alwaysIncluded bool, via string) (*Memory, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required; a memory belongs to one project", apperrs.ErrInvalid)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	workspaceID, err := s.projectForWrite(ctx, projectID, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	normalized, err := richtext.Normalize(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	now := s.now().UTC()
	m := &Memory{
		ID:             ids.New(),
		WorkspaceID:    workspaceID,
		ProjectID:      projectID,
		Title:          title,
		WhenToUse:      capWhenToUse(whenToUse),
		Body:           normalized,
		AlwaysIncluded: alwaysIncluded,
		Version:        1,
		CreatedBy:      actor.ID,
		CreatedAt:      now,
		UpdatedBy:      actor.ID,
		UpdatedAt:      now,
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCreated, Payload: CreatedEvent{Memory: toRef(m), AuthorID: actor.ID}}
	if err := s.repo.Create(ctx, m, via, evt); err != nil {
		return nil, fmt.Errorf("create memory: %w", err)
	}
	return m, nil
}

// Get returns a memory by id; requires memories:read on its project.
func (s *Service) Get(ctx context.Context, id string) (*Memory, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	return s.getChecked(ctx, id, permissions.MemoriesRead)
}

// getChecked loads a memory and checks action on its project, so every per-memory use-case asks the same way.
func (s *Service) getChecked(ctx context.Context, id string, action permissions.Action) (*Memory, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get memory %s: %w", id, err)
	}
	if err := s.requireProject(ctx, m.ProjectID, action); err != nil {
		return nil, err
	}
	return m, nil
}

// List returns the memories of a workspace's projects the caller may read, ordered by project then creation; the
// page groups them by project. A project the caller cannot read is left out rather than failing the list.
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Memory, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.Member); err != nil {
		return nil, err
	}
	ms, err := s.repo.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list memories for workspace %s: %w", workspaceID, err)
	}
	return permissions.Filter(ms, func(m *Memory) string { return m.ProjectID }, func(projectID string) error {
		return s.requireProject(ctx, projectID, permissions.MemoriesRead)
	})
}

// ListForProject returns the project's memories; requires memories:read on the project.
func (s *Service) ListForProject(ctx context.Context, projectID string) ([]*Memory, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireProject(ctx, projectID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	ms, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list memories for project %s: %w", projectID, err)
	}
	return ms, nil
}

// ListMemoryItems returns the project's memories for an agent turn to name, none for a turn with no project
// (ADR 0099). It carries no permission check: the turn pipeline reads it on the server's behalf.
func (s *Service) ListMemoryItems(ctx context.Context, projectID string) ([]MemoryItem, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, nil
	}
	ms, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list memory items for project %s: %w", projectID, err)
	}
	out := make([]MemoryItem, len(ms))
	for i, m := range ms {
		out[i] = MemoryItem{ID: m.ID, Title: m.Title, AlwaysIncluded: m.AlwaysIncluded, Kind: m.Kind}
	}
	return out, nil
}

// Update replaces title/when-to-use/body/always-included and appends a version; requires memories:write on the
// memory's project. via is "mcp" for an MCP tool call (ADR 0049), empty for the browser.
func (s *Service) Update(ctx context.Context, id, title, whenToUse, body string, alwaysIncluded bool, via string) (*Memory, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	current, err := s.getChecked(ctx, id, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	normalized, err := richtext.Normalize(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	// The decisions log is found with memory_list, never named every turn (ADR 0065).
	if current.Kind == KindDecisionsLog {
		alwaysIncluded = false
	}
	// The interview memory is never switched off and stays under its cap (ADR 0065).
	if current.Kind == KindInterview {
		alwaysIncluded = true
		if err := checkInterviewBody(normalized); err != nil {
			return nil, err
		}
	}
	current.Title = title
	current.WhenToUse = capWhenToUse(whenToUse)
	current.Body = normalized
	current.AlwaysIncluded = alwaysIncluded
	current.Version++
	current.UpdatedBy = actor.ID
	current.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Memory: toRef(current), AuthorID: actor.ID, AuthorVia: via}}
	if err := s.repo.Update(ctx, current, via, evt); err != nil {
		return nil, fmt.Errorf("update memory %s: %w", id, err)
	}
	return current, nil
}

// ListVersions returns a memory's version history, newest first; requires memories:read on its project.
func (s *Service) ListVersions(ctx context.Context, id string) ([]*MemoryVersion, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if _, err := s.getChecked(ctx, id, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	vs, err := s.repo.ListVersions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list memory versions %s: %w", id, err)
	}
	return vs, nil
}

// GetVersion returns one historical version of a memory; requires memories:read on its project.
func (s *Service) GetVersion(ctx context.Context, id string, version int) (*MemoryVersion, error) {
	m, v, err := s.getVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}
	if err := s.requireProject(ctx, m.ProjectID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	return v, nil
}

// getVersion is the permission-free lookup Revert builds on: reverting only needs memories:write (checked by
// the Update call it delegates to), not memories:read as well.
func (s *Service) getVersion(ctx context.Context, id string, version int) (*Memory, *MemoryVersion, error) {
	if strings.TrimSpace(id) == "" {
		return nil, nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if version < 1 {
		return nil, nil, fmt.Errorf("%w: version must be positive", apperrs.ErrInvalid)
	}
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("get memory %s version %d: %w", id, version, err)
	}
	v, err := s.repo.GetVersion(ctx, id, version)
	if err != nil {
		return nil, nil, fmt.Errorf("get memory %s version %d: %w", id, version, err)
	}
	return m, v, nil
}

// Revert copies a historical version's content back onto the memory as a new version; it never deletes
// history. Requires memories:write on the memory's project. via is "mcp" for an MCP tool call.
func (s *Service) Revert(ctx context.Context, id string, version int, via string) (*Memory, error) {
	_, target, err := s.getVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}
	return s.Update(ctx, id, target.Title, target.WhenToUse, target.Body, target.AlwaysIncluded, via)
}

// Delete removes a memory; requires memories:delete on its project.
func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	m, err := s.getChecked(ctx, id, permissions.MemoriesDelete)
	if err != nil {
		return err
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeleted, Payload: DeletedEvent{ID: m.ID, WorkspaceID: m.WorkspaceID, ProjectID: m.ProjectID, Title: m.Title, AuthorID: actor.ID}}
	if err := s.repo.Delete(ctx, id, evt); err != nil {
		return fmt.Errorf("delete memory %s: %w", id, err)
	}
	return nil
}

// ExportMarkdown renders a memory's body as markdown for LLM/MCP consumption; requires the read bit.
func (s *Service) ExportMarkdown(ctx context.Context, id string) (string, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	md, err := richtext.ToMarkdown(m.Body)
	if err != nil {
		return "", fmt.Errorf("export memory %s: %w", id, err)
	}
	return md, nil
}

// Clone copies a memory into a project as a new, independent memory with fresh version history; its attachments
// are copied as new rows and the body's references rewritten to them (ADR 0057, ticket 18). Requires
// memories:clone on the source's project and memories:write on the destination; the error names whichever is
// missing.
func (s *Service) Clone(ctx context.Context, id, destinationProjectID string) (*Memory, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	destinationProjectID = strings.TrimSpace(destinationProjectID)
	if destinationProjectID == "" {
		return nil, fmt.Errorf("%w: destination project id is required; a memory belongs to one project", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	source, err := s.getChecked(ctx, id, permissions.MemoriesClone)
	if err != nil {
		return nil, err
	}
	destWorkspaceID, err := s.projectForWrite(ctx, destinationProjectID, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	idMap, err := s.attachmentIDMap(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list attachments for memory %s: %w", id, err)
	}
	body, err := richtext.RewriteAttachmentRefs(source.Body, idMap)
	if err != nil {
		return nil, fmt.Errorf("rewrite attachment refs for memory %s: %w", id, err)
	}
	now := s.now().UTC()
	clone := &Memory{
		ID:             ids.New(),
		WorkspaceID:    destWorkspaceID,
		ProjectID:      destinationProjectID,
		Title:          source.Title,
		WhenToUse:      source.WhenToUse,
		Body:           body,
		AlwaysIncluded: source.AlwaysIncluded,
		Version:        1,
		CreatedBy:      actor.ID,
		CreatedAt:      now,
		UpdatedBy:      actor.ID,
		UpdatedAt:      now,
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCreated, Payload: CreatedEvent{Memory: toRef(clone), AuthorID: actor.ID}}
	if err := s.repo.Create(ctx, clone, "", evt); err != nil {
		return nil, fmt.Errorf("clone memory %s: %w", id, err)
	}
	if s.attachments != nil && len(idMap) > 0 {
		if err := s.attachments.CopyOwnerWithIDs(ctx, source.ID, clone.ID, idMap); err != nil {
			return nil, fmt.Errorf("copy attachments for memory clone %s: %w", id, err)
		}
	}
	return clone, nil
}

// projectForWrite checks action on projectID and returns its workspace, which a new memory denormalizes.
func (s *Service) projectForWrite(ctx context.Context, projectID string, action permissions.Action) (string, error) {
	if err := s.requireProject(ctx, projectID, action); err != nil {
		return "", err
	}
	workspaceID, err := s.projects.WorkspaceForProject(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
	}
	return workspaceID, nil
}

// attachmentIDMap pre-assigns a fresh id for each of a memory's attachments, so the clone's body can be
// rewritten to them before the clone row (and therefore the attachment FK target) exists.
func (s *Service) attachmentIDMap(ctx context.Context, memoryID string) (map[string]string, error) {
	if s.attachments == nil {
		return nil, nil
	}
	oldIDs, err := s.attachments.ListOwnerIDs(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	if len(oldIDs) == 0 {
		return nil, nil
	}
	idMap := make(map[string]string, len(oldIDs))
	for _, oldID := range oldIDs {
		idMap[oldID] = ids.New()
	}
	return idMap, nil
}

func capWhenToUse(whenToUse string) string {
	whenToUse = strings.TrimSpace(whenToUse)
	if len(whenToUse) > maxWhenToUseChars {
		return whenToUse[:maxWhenToUseChars]
	}
	return whenToUse
}

func (s *Service) require(ctx context.Context, workspaceID string, action permissions.Action) error {
	if s.access == nil {
		return permissions.Ungated(ctx)
	}
	return s.access.Require(ctx, workspaceID, action)
}

func (s *Service) requireProject(ctx context.Context, projectID string, action permissions.Action) error {
	if s.access == nil {
		return permissions.Ungated(ctx)
	}
	return s.access.RequireProject(ctx, projectID, action)
}
