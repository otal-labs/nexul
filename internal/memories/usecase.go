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

// PermissionGate is the consumer-side seam onto access's HasPermission (ADR 0017); no per-memory overwrite grid,
// only doc threads are fine-grained in this effort.
type PermissionGate interface {
	HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action) bool
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

// WorkspaceMembership checks membership independent of any single permission (ADR 0057: clone requires it
// alongside memories:write, since a non-member with the bit is still the wrong workspace).
type WorkspaceMembership interface {
	IsMember(ctx context.Context, userID, workspaceID string) (bool, error)
}

// Service is the memories use-case layer (ADR 0019); permission checks run here so every adapter inherits them.
type Service struct {
	repo        Repo
	access      PermissionGate
	projects    ProjectLookup
	attachments AttachmentsCopier
	membership  WorkspaceMembership
	now         func() time.Time
}

// NewService wires the memories use-cases over the given repo, permission gate, project lookup, attachments
// copier (for Clone), and workspace membership check (for Clone).
func NewService(repo Repo, access PermissionGate, projects ProjectLookup, attachmentsCopier AttachmentsCopier, membership WorkspaceMembership) *Service {
	return &Service{repo: repo, access: access, projects: projects, attachments: attachmentsCopier, membership: membership, now: time.Now}
}

// Create validates and persists a new memory as version 1, enqueuing memory.created. projectID empty makes the
// memory workspace-scoped (ADR 0059), in which case workspaceID is required; when projectID is given, the
// workspace is derived from it and workspaceID is ignored. via is "mcp" when the call came through an MCP
// tool (ADR 0049), empty for the browser.
func (s *Service) Create(ctx context.Context, projectID, workspaceID, title, whenToUse, body string, alwaysIncluded bool, via string) (*Memory, error) {
	projectID = strings.TrimSpace(projectID)
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	if projectID != "" {
		resolved, err := s.projects.WorkspaceForProject(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
		}
		workspaceID = resolved
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required when project id is empty", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesWrite); err != nil {
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

// Get returns a memory by id; requires memories:read on its workspace.
func (s *Service) Get(ctx context.Context, id string) (*Memory, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get memory %s: %w", id, err)
	}
	if err := s.require(ctx, m.WorkspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	return m, nil
}

// List returns every memory in a workspace, ordered by project then creation; the page groups them by project.
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Memory, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	ms, err := s.repo.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list memories for workspace %s: %w", workspaceID, err)
	}
	return ms, nil
}

// ListForProject returns the project's own memories plus its workspace's workspace-scoped ones,
// workspace-scoped first (ADR 0059); requires memories:read on the project's workspace.
func (s *Service) ListForProject(ctx context.Context, projectID string) ([]*Memory, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	workspaceID, err := s.projects.WorkspaceForProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	ms, err := s.repo.ListByProject(ctx, projectID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list memories for project %s: %w", projectID, err)
	}
	return ms, nil
}

// ListWorkspaceScoped returns only a workspace's workspace-scoped memories (ADR 0059); requires memories:read
// on the workspace. Used by MCP's memory_list when called with no project_id.
func (s *Service) ListWorkspaceScoped(ctx context.Context, workspaceID string) ([]*Memory, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	ms, err := s.repo.ListWorkspaceScoped(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace-scoped memories for workspace %s: %w", workspaceID, err)
	}
	return ms, nil
}

// ListMemoryItems returns the turn's memory index for the agent prompt: a workspace's workspace-scoped items
// (workspaceItems), and — when projectID is given — that project's own items too (projectItems). It carries no
// permission check: the turn pipeline runs with no acting-user context (it reacts to a bus event, not a
// request), mirroring docs.Service.ListMemories before this entity split.
func (s *Service) ListMemoryItems(ctx context.Context, workspaceID, projectID string) (workspaceItems, projectItems []MemoryItem, err error) {
	workspaceID = strings.TrimSpace(workspaceID)
	projectID = strings.TrimSpace(projectID)
	if workspaceID == "" {
		return nil, nil, nil
	}
	if projectID == "" {
		ms, err := s.repo.ListWorkspaceScoped(ctx, workspaceID)
		if err != nil {
			return nil, nil, fmt.Errorf("list workspace memory items for workspace %s: %w", workspaceID, err)
		}
		items, err := toMemoryItems(ms)
		if err != nil {
			return nil, nil, err
		}
		return items, nil, nil
	}
	ms, err := s.repo.ListByProject(ctx, projectID, workspaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("list memory items for project %s: %w", projectID, err)
	}
	for _, m := range ms {
		item, err := toMemoryItem(m)
		if err != nil {
			return nil, nil, err
		}
		if m.ProjectID == "" {
			workspaceItems = append(workspaceItems, item)
			continue
		}
		projectItems = append(projectItems, item)
	}
	return workspaceItems, projectItems, nil
}

func toMemoryItems(ms []*Memory) ([]MemoryItem, error) {
	out := make([]MemoryItem, len(ms))
	for i, m := range ms {
		item, err := toMemoryItem(m)
		if err != nil {
			return nil, err
		}
		out[i] = item
	}
	return out, nil
}

// toMemoryItem renders one memory as a lean index entry, except an always-included memory, which also carries
// its full markdown body since the turn inlines it (ticket 27).
func toMemoryItem(m *Memory) (MemoryItem, error) {
	item := MemoryItem{ID: m.ID, Title: m.Title, WhenToUse: m.WhenToUse, AlwaysIncluded: m.AlwaysIncluded, Kind: m.Kind}
	if !m.AlwaysIncluded {
		return item, nil
	}
	md, err := richtext.ToMarkdown(m.Body)
	if err != nil {
		return MemoryItem{}, fmt.Errorf("export always-included memory %s: %w", m.ID, err)
	}
	item.Body = md
	return item, nil
}

// Update replaces title/when-to-use/body/always-included and appends a version; requires memories:write on the
// memory's workspace. via is "mcp" for an MCP tool call (ADR 0049), empty for the browser.
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
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update memory %s: %w", id, err)
	}
	if err := s.require(ctx, current.WorkspaceID, permissions.MemoriesWrite); err != nil {
		return nil, err
	}
	normalized, err := richtext.Normalize(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	// The decisions log is pulled from the index, never sent every turn (ADR 0065).
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

// ListVersions returns a memory's version history, newest first; requires memories:read on its workspace.
func (s *Service) ListVersions(ctx context.Context, id string) ([]*MemoryVersion, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list memory versions %s: %w", id, err)
	}
	if err := s.require(ctx, m.WorkspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	vs, err := s.repo.ListVersions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list memory versions %s: %w", id, err)
	}
	return vs, nil
}

// GetVersion returns one historical version of a memory; requires memories:read on its workspace.
func (s *Service) GetVersion(ctx context.Context, id string, version int) (*MemoryVersion, error) {
	m, v, err := s.getVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}
	if err := s.require(ctx, m.WorkspaceID, permissions.MemoriesRead); err != nil {
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
// history. Requires memories:write on the memory's workspace. via is "mcp" for an MCP tool call.
func (s *Service) Revert(ctx context.Context, id string, version int, via string) (*Memory, error) {
	_, target, err := s.getVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}
	return s.Update(ctx, id, target.Title, target.WhenToUse, target.Body, target.AlwaysIncluded, via)
}

// Delete removes a memory; requires memories:delete on its workspace.
func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete memory %s: %w", id, err)
	}
	if err := s.require(ctx, m.WorkspaceID, permissions.MemoriesDelete); err != nil {
		return err
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeleted, Payload: DeletedEvent{ID: m.ID, Title: m.Title, AuthorID: actor.ID}}
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

// Clone copies a memory into another project, or into a workspace directly (destinationProjectID empty,
// ADR 0059), as a new, independent memory with fresh version history; its attachments are copied as new rows
// and the body's references rewritten to them (ADR 0057, ticket 18). destinationWorkspaceID is required when
// destinationProjectID is empty and ignored otherwise, same as Create. Requires memories:clone on the source
// workspace, membership of the destination workspace, and memories:write there; the error names whichever is
// missing.
func (s *Service) Clone(ctx context.Context, id, destinationProjectID, destinationWorkspaceID string) (*Memory, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	destinationProjectID = strings.TrimSpace(destinationProjectID)
	destinationWorkspaceID = strings.TrimSpace(destinationWorkspaceID)
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("clone memory %s: %w", id, err)
	}
	if err := s.require(ctx, source.WorkspaceID, permissions.MemoriesClone); err != nil {
		return nil, err
	}
	destWorkspaceID, err := s.resolveDestinationWorkspace(ctx, destinationProjectID, destinationWorkspaceID)
	if err != nil {
		return nil, err
	}
	if err := s.requireDestination(ctx, actor.ID, destWorkspaceID); err != nil {
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

// resolveDestinationWorkspace derives Clone's destination workspace from its project id when given, otherwise
// falls back to the explicit workspace id, matching Create's same-shaped rule.
func (s *Service) resolveDestinationWorkspace(ctx context.Context, destinationProjectID, destinationWorkspaceID string) (string, error) {
	if destinationProjectID == "" {
		if destinationWorkspaceID == "" {
			return "", fmt.Errorf("%w: destination workspace id is required when destination project id is empty", apperrs.ErrInvalid)
		}
		return destinationWorkspaceID, nil
	}
	resolved, err := s.projects.WorkspaceForProject(ctx, destinationProjectID)
	if err != nil {
		return "", fmt.Errorf("resolve destination workspace for project %s: %w", destinationProjectID, err)
	}
	return resolved, nil
}

// requireDestination checks Clone's two destination-side conditions: membership of the workspace, and
// memories:write in it. Split out of Clone to keep it under the complexity gate.
func (s *Service) requireDestination(ctx context.Context, userID, workspaceID string) error {
	if s.membership == nil {
		return fmt.Errorf("%w: not a member of the destination workspace %s", apperrs.ErrForbidden, workspaceID)
	}
	member, err := s.membership.IsMember(ctx, userID, workspaceID)
	if err != nil {
		return fmt.Errorf("check membership of workspace %s: %w", workspaceID, err)
	}
	if !member {
		return fmt.Errorf("%w: not a member of the destination workspace %s", apperrs.ErrForbidden, workspaceID)
	}
	return s.require(ctx, workspaceID, permissions.MemoriesWrite)
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
	if !s.can(ctx, workspaceID, action) {
		return fmt.Errorf("%w: no %s permission in workspace %s", apperrs.ErrForbidden, action, workspaceID)
	}
	return nil
}

func (s *Service) can(ctx context.Context, workspaceID string, action permissions.Action) bool {
	// Fail closed: a service without a wired access gate (misconfiguration) must deny, never silently allow.
	if s.access == nil {
		return false
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return false
	}
	return s.access.HasPermission(ctx, actor.ID, workspaceID, action)
}
