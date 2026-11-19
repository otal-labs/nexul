package docs

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// ListFolders returns a project's folders, the default first; a viewer without docs:write sees only the folders
// holding a doc they can open, which folder rules decide with the doc's own, and the folders those sit inside.
func (s *Service) ListFolders(ctx context.Context, projectID string) ([]*Folder, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireProject(ctx, projectID, permissions.Member); err != nil {
		return nil, err
	}
	folders, err := s.repo.ListFolders(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list folders for project %s: %w", projectID, err)
	}
	if s.requireProject(ctx, projectID, permissions.DocsWrite) == nil {
		return folders, nil
	}
	ds, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list docs for project %s: %w", projectID, err)
	}
	reads := s.canEach(ctx, ds, permissions.DocsRead)
	parents := make(map[string]string, len(folders))
	for _, f := range folders {
		parents[f.ID] = f.ParentID
	}
	holding := map[string]bool{}
	for _, d := range ds {
		for id := d.FolderID; reads[d.ID] && id != "" && !holding[id]; id = parents[id] {
			holding[id] = true
		}
	}
	return slices.DeleteFunc(folders, func(f *Folder) bool { return !holding[f.ID] }), nil
}

// CreateFolder adds a folder to a project (docs:write), inside parentID or at the top when it is empty; a name another
// folder of that parent has, ignoring case, conflicts.
func (s *Service) CreateFolder(ctx context.Context, projectID, parentID, name string) (*Folder, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	name, err := folderName(name)
	if err != nil {
		return nil, err
	}
	if err := s.requireProject(ctx, projectID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	parentID, err = s.parentIn(ctx, projectID, parentID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	f := &Folder{ID: ids.New(), ProjectID: projectID, ParentID: parentID, Name: name, CreatedAt: now, UpdatedAt: now}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFolderCreated, Payload: FolderEvent{Folder: *f, ActorID: actorID(ctx)}}
	if err := s.repo.CreateFolder(ctx, f, evt); err != nil {
		return nil, folderWriteErr(name, err)
	}
	return f, nil
}

// RenameFolder renames a folder (docs:write), the default one included.
func (s *Service) RenameFolder(ctx context.Context, id, name string) (*Folder, error) {
	name, err := folderName(name)
	if err != nil {
		return nil, err
	}
	f, err := s.writableFolder(ctx, id)
	if err != nil {
		return nil, err
	}
	renamed := *f
	renamed.Name = name
	return s.updateFolder(ctx, f, &renamed)
}

// MoveFolder puts a folder inside parentID, or at the top when it is empty (docs:write), with every folder and doc in
// it; the default folder stays at the top, and no folder goes inside itself.
func (s *Service) MoveFolder(ctx context.Context, id, parentID string) (*Folder, error) {
	f, err := s.writableFolder(ctx, id)
	if err != nil {
		return nil, err
	}
	parentID, err = s.parentIn(ctx, f.ProjectID, parentID)
	if err != nil {
		return nil, err
	}
	if f.ParentID == parentID {
		return f, nil
	}
	if f.IsDefault {
		return nil, fmt.Errorf("%w: %s is the project's default folder, which stays at the top level", apperrs.ErrInvalid, f.Name)
	}
	moved := *f
	moved.ParentID = parentID
	return s.updateFolder(ctx, f, &moved)
}

func (s *Service) updateFolder(ctx context.Context, from, to *Folder) (*Folder, error) {
	to.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFolderUpdated, Payload: FolderUpdatedEvent{
		Folder: *to, PreviousName: from.Name, PreviousParentID: from.ParentID, ActorID: actorID(ctx),
	}}
	if err := s.repo.UpdateFolder(ctx, to, evt); err != nil {
		return nil, folderWriteErr(to.Name, err)
	}
	return to, nil
}

// DeleteFolder deletes a folder and every folder inside it (docs:write) and moves all their docs to its parent, or to
// the project's default folder, which is never deleted, when it is top-level.
func (s *Service) DeleteFolder(ctx context.Context, id string) error {
	f, err := s.writableFolder(ctx, id)
	if err != nil {
		return err
	}
	if f.IsDefault {
		return fmt.Errorf("%w: %s is the project's default folder, which can be renamed but not deleted", apperrs.ErrInvalid, f.Name)
	}
	to, err := s.folderIn(ctx, f.ProjectID, f.ParentID)
	if err != nil {
		return err
	}
	evts, err := s.folderDeletedEvents(ctx, f, to)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteFolder(ctx, f.ID, to, evts...); err != nil {
		return fmt.Errorf("delete folder %s: %w", id, err)
	}
	return nil
}

// folderDeletedEvents is one doc.folder.deleted per folder of f's subtree, f first.
func (s *Service) folderDeletedEvents(ctx context.Context, f *Folder, to string) ([]eventbus.OutboxEvent, error) {
	subtree, err := s.repo.FolderSubtree(ctx, f.ID)
	if err != nil {
		return nil, fmt.Errorf("folders inside %s: %w", f.ID, err)
	}
	folders, err := s.repo.ListFolders(ctx, f.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("list folders for project %s: %w", f.ProjectID, err)
	}
	evts := []eventbus.OutboxEvent{{ID: ids.New(), Topic: TopicFolderDeleted, Payload: FolderDeletedEvent{Folder: *f, MovedToFolderID: to, ActorID: actorID(ctx)}}}
	for _, inner := range folders {
		if inner.ID == f.ID || !slices.Contains(subtree, inner.ID) {
			continue
		}
		evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicFolderDeleted, Payload: FolderDeletedEvent{Folder: *inner, MovedToFolderID: to, ActorID: actorID(ctx)}})
	}
	return evts, nil
}

// MoveToFolder moves a doc into another folder of its project, needing docs:write on the doc; a locked doc moves too.
func (s *Service) MoveToFolder(ctx context.Context, docID, folderID string) (*Doc, error) {
	if strings.TrimSpace(docID) == "" || strings.TrimSpace(folderID) == "" {
		return nil, fmt.Errorf("%w: doc id and folder id are required", apperrs.ErrInvalid)
	}
	d, err := s.repo.GetByID(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("move doc %s: %w", docID, err)
	}
	if err := s.require(ctx, d.ID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	to, err := s.folderIn(ctx, d.ProjectID, folderID)
	if err != nil {
		return nil, err
	}
	if d.FolderID == to {
		return d, nil
	}
	moved := *d
	moved.FolderID = to
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMoved, Payload: MovedEvent{Doc: moved, FromFolderID: d.FolderID, ActorID: actorID(ctx)}}
	if err := s.repo.SetDocFolder(ctx, d.ID, to, evt); err != nil {
		return nil, fmt.Errorf("move doc %s: %w", docID, err)
	}
	return &moved, nil
}

// folderIn resolves where a doc in projectID goes: folderID when it is one of the project's, its default when empty.
func (s *Service) folderIn(ctx context.Context, projectID, folderID string) (string, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		main, err := s.repo.DefaultFolder(ctx, projectID)
		if err != nil {
			return "", fmt.Errorf("default folder of project %s: %w", projectID, err)
		}
		return main.ID, nil
	}
	f, err := s.repo.GetFolder(ctx, folderID)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && f.ProjectID != projectID) {
		return "", fmt.Errorf("%w: folder %s is not a folder of this project", apperrs.ErrInvalid, folderID)
	}
	if err != nil {
		return "", fmt.Errorf("get folder %s: %w", folderID, err)
	}
	return f.ID, nil
}

// parentIn resolves where a new or moved folder of projectID sits: inside parentID when it is one of the project's
// folders, at the top when it is empty.
func (s *Service) parentIn(ctx context.Context, projectID, parentID string) (string, error) {
	if strings.TrimSpace(parentID) == "" {
		return "", nil
	}
	return s.folderIn(ctx, projectID, parentID)
}

func (s *Service) writableFolder(ctx context.Context, id string) (*Folder, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: folder id is required", apperrs.ErrInvalid)
	}
	f, err := s.repo.GetFolder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get folder %s: %w", id, err)
	}
	if err := s.requireProject(ctx, f.ProjectID, permissions.DocsWrite); err != nil {
		return nil, err
	}
	return f, nil
}

func folderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: folder name is required", apperrs.ErrInvalid)
	}
	return name, nil
}

func folderWriteErr(name string, err error) error {
	if errors.Is(err, apperrs.ErrConflict) {
		return fmt.Errorf("%w: a folder named %s is already there", apperrs.ErrConflict, name)
	}
	if errors.Is(err, apperrs.ErrInvalid) {
		return err
	}
	return fmt.Errorf("save folder %s: %w", name, err)
}
