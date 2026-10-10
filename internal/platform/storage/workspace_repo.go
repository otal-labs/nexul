package storage

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/workspace"
)

var _ workspace.Repo = (*ProjectsRepo)(nil)

type ProjectsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *ProjectsRepo) Create(ctx context.Context, p *workspace.Project) error {
	steps, envKeys, err := setupJSON(p)
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.CreateProject(ctx, sqlcgen.CreateProjectParams{
			ID: p.ID, Name: p.Name, Prefix: p.Prefix, Position: int64(p.Position),
			WorkspaceID: p.WorkspaceID, Icon: string(p.Icon), SetupFinished: boolToInt(p.Setup.Finished), SetupSteps: steps, SetupStackID: p.Setup.StackID, SetupEnvKeys: envKeys,
			CreatedAt: p.CreatedAt.Unix(), UpdatedAt: p.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert project %s: %w", p.ID, classifyWriteErr(err))
		}
		types := p.SeedTicketTypes
		if len(types) == 0 {
			types = workspace.DefaultTicketTypes
		}
		return seedProjectDefaults(ctx, q, p.WorkspaceID, p.ID, types, p.CreatedAt.Unix())
	})
}

// seedProjectDefaults inserts defaults in the same transaction as the project, so a board is always usable and a
// new doc always has its default folder to land in.
func seedProjectDefaults(ctx context.Context, q *sqlcgen.Queries, workspaceID, projectID string, types []workspace.TicketType, at int64) error {
	statuses := []struct {
		name string
		kind string
	}{
		{"Backlog", "backlog"},
		{"In progress", "progress"},
		{"In review", "review"},
		{"Testing", "testing"},
		{"Done", "done"},
	}
	for i, st := range statuses {
		if err := q.SeedProjectStatus(ctx, sqlcgen.SeedProjectStatusParams{
			ID: uuid.NewString(), ProjectID: projectID, Name: st.name, Position: int64(i), Kind: st.kind, CreatedAt: at, UpdatedAt: at,
		}); err != nil {
			return fmt.Errorf("seed status %q for project %s: %w", st.name, projectID, err)
		}
	}
	for i, tt := range types {
		if err := q.SeedProjectTicketType(ctx, sqlcgen.SeedProjectTicketTypeParams{
			ID: uuid.NewString(), ProjectID: projectID, Name: tt.Name, Position: int64(i),
			BodyTemplate: tt.BodyTemplate, CreatedAt: at, UpdatedAt: at,
		}); err != nil {
			return fmt.Errorf("seed ticket type %q for project %s: %w", tt.Name, projectID, err)
		}
	}
	created := time.Unix(at, 0).UTC()
	if err := insertDocFolder(ctx, q, &docs.Folder{
		ID: uuid.NewString(), ProjectID: projectID, Name: mainFolderName, IsDefault: true, CreatedAt: created, UpdatedAt: created,
	}); err != nil {
		return fmt.Errorf("seed default doc folder for project %s: %w", projectID, err)
	}
	memoryID := uuid.NewString()
	if err := q.CreateMemory(ctx, sqlcgen.CreateMemoryParams{
		ID: memoryID, WorkspaceID: workspaceID, ProjectID: sql.NullString{String: projectID, Valid: true},
		Title: "Working in this project", WhenToUse: "Always included in this project's turns.",
		Body:           seedMemoryBody,
		AlwaysIncluded: 1, Version: 1, CreatedAt: at, UpdatedAt: at,
	}); err != nil {
		return fmt.Errorf("seed memory for project %s: %w", projectID, err)
	}
	// No author: this row seeds automatically, no user wrote it (mirrors 0139's seed for project-general).
	if err := q.InsertMemoryVersion(ctx, sqlcgen.InsertMemoryVersionParams{
		ID: uuid.NewString(), MemoryID: memoryID, Version: 1,
		Title: "Working in this project", WhenToUse: "Always included in this project's turns.",
		Body: seedMemoryBody, AlwaysIncluded: 1, CreatedAt: at,
	}); err != nil {
		return fmt.Errorf("seed memory version for project %s: %w", projectID, err)
	}
	return nil
}

// seedMemoryBody is legacy plain text, not structured Tiptap JSON — ADR 0026 lets a pre-conversion body pass
// through unchanged, and no user authored this row for richtext.Normalize to run against.
const seedMemoryBody = `This memory is Agent's shared context for this project — durable notes worth remembering across every chat turn.

Replace this starter text with anything the team wants Agent to always know: conventions, gotchas, or standing preferences.

Edit it any time; it stays always-included until someone turns that off.`

func (r *ProjectsRepo) Get(ctx context.Context, id string) (*workspace.Project, error) {
	row, err := r.q.GetProject(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", id, notFoundIfNoRows(err))
	}
	return toProject(row)
}

// List returns a workspace's projects ordered by position; a project always nests under exactly one workspace.
func (r *ProjectsRepo) List(ctx context.Context, workspaceID string) ([]*workspace.Project, error) {
	rows, err := r.q.ListProjectsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return toProjects(rows)
}

func (r *ProjectsRepo) Update(ctx context.Context, p *workspace.Project) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateProject(ctx, sqlcgen.UpdateProjectParams{
			Name: p.Name, Prefix: p.Prefix, Icon: string(p.Icon), TestsLocation: string(p.TestsLocation),
			UpdatedAt: p.UpdatedAt.Unix(), ID: p.ID,
		})
		if err != nil {
			return fmt.Errorf("update project %s: %w", p.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update project %s: %w", p.ID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *ProjectsRepo) SaveSetup(ctx context.Context, id string, apply func(*workspace.Project) []eventbus.OutboxEvent) (*workspace.Project, error) {
	var p *workspace.Project
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.GetProject(ctx, id)
		if err != nil {
			return fmt.Errorf("get setup of project %s: %w", id, notFoundIfNoRows(err))
		}
		p, err = toProject(row)
		if err != nil {
			return err
		}
		evts := apply(p)
		if len(evts) == 0 {
			return nil
		}
		steps, envKeys, err := setupJSON(p)
		if err != nil {
			return err
		}
		_, err = q.UpdateProjectSetup(ctx, sqlcgen.UpdateProjectSetupParams{
			SetupFinished: boolToInt(p.Setup.Finished), SetupSteps: steps, SetupStackID: p.Setup.StackID, SetupEnvKeys: envKeys, UpdatedAt: p.UpdatedAt.Unix(), ID: p.ID,
		})
		if err != nil {
			return fmt.Errorf("save setup of project %s: %w", p.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProjectsRepo) Delete(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.DeleteProject(ctx, id)
		if err != nil {
			return fmt.Errorf("delete project %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete project %s: %w", id, apperrs.ErrNotFound)
		}
		if err := q.DeleteOverwritesByResource(ctx, sqlcgen.DeleteOverwritesByResourceParams{ResourceType: projectResourceType, ResourceID: id}); err != nil {
			return fmt.Errorf("delete access to project %s: %w", id, err)
		}
		return nil
	})
}

// ListRestrictedAccess names each Restricted member by what People shows: display name, else account name, else login.
func (r *ProjectsRepo) ListRestrictedAccess(ctx context.Context, projectID string) ([]workspace.ProjectAccessEntry, error) {
	rows, err := r.q.ListProjectRestrictedAccess(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list access to project %s: %w", projectID, err)
	}
	out := make([]workspace.ProjectAccessEntry, 0, len(rows))
	for _, row := range rows {
		allow, err := parseSet(row.Allow)
		if err != nil {
			return nil, fmt.Errorf("decode access of %s to project %s: %w", row.UserID, projectID, err)
		}
		name := cmp.Or(row.DisplayName, row.Name, row.Login)
		out = append(out, workspace.ProjectAccessEntry{RestrictedMember: workspace.RestrictedMember{UserID: row.UserID, Name: name}, Actions: allow})
	}
	return out, nil
}

// Reorder assigns each project's position by its index in ids, in one transaction so a reorder is atomic.
func (r *ProjectsRepo) Reorder(ctx context.Context, ids []string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for i, id := range ids {
			if err := q.ReorderProjectPosition(ctx, sqlcgen.ReorderProjectPositionParams{
				Position: int64(i), UpdatedAt: time.Now().Unix(), ID: id,
			}); err != nil {
				return fmt.Errorf("reorder project %s: %w", id, err)
			}
		}
		return nil
	})
}

func (r *ProjectsRepo) CountTickets(ctx context.Context, projectID string) (int, error) {
	return r.count(ctx, projectID, "tickets")
}

func (r *ProjectsRepo) CountRepos(ctx context.Context, projectID string) (int, error) {
	return r.count(ctx, projectID, "project_repos")
}

// CountServices reports how many deploy stacks a project owns; deletion must be blocked while any exist.
// Counts the "stacks" table; the "services" table holds observed containers, which have no project_id of their own.
func (r *ProjectsRepo) CountServices(ctx context.Context, projectID string) (int, error) {
	return r.count(ctx, projectID, "stacks")
}

func (r *ProjectsRepo) ProjectForStack(ctx context.Context, stackID string) (string, error) {
	row, err := r.q.GetStack(ctx, stackID)
	if err != nil {
		return "", notFoundIfNoRows(err)
	}
	return row.ProjectID.String, nil
}

func (r *ProjectsRepo) count(ctx context.Context, projectID, table string) (int, error) {
	var n int
	// hand-written: sqlc cannot express a table name chosen at runtime
	if err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE project_id = ?`, table), projectID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count %s for project %s: %w", table, projectID, err)
	}
	return n, nil
}

func (r *ProjectsRepo) AddRepo(ctx context.Context, projectID string, ref workspace.RepoRef) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).AddProjectRepo(ctx, sqlcgen.AddProjectRepoParams{
			ProjectID: projectID, Owner: ref.Owner, Name: ref.Name, FullName: ref.FullName,
			ConnectorID: ref.ConnectorID, Role: string(ref.Role), AddedAt: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("add repo %s/%s to project %s: %w", ref.Owner, ref.Name, projectID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *ProjectsRepo) RemoveRepo(ctx context.Context, owner, name string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RemoveProjectRepo(ctx, sqlcgen.RemoveProjectRepoParams{Owner: owner, Name: name})
		if err != nil {
			return fmt.Errorf("remove repo %s/%s: %w", owner, name, err)
		}
		if n == 0 {
			return fmt.Errorf("remove repo %s/%s: %w", owner, name, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *ProjectsRepo) ListRepos(ctx context.Context, projectID string) ([]workspace.RepoRef, error) {
	rows, err := r.q.ListProjectRepos(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list repos for project %s: %w", projectID, err)
	}
	var out []workspace.RepoRef
	for _, row := range rows {
		out = append(out, workspace.RepoRef{Owner: row.Owner, Name: row.Name, FullName: row.FullName, ConnectorID: row.ConnectorID, Role: workspace.RepoRole(row.Role)})
	}
	return out, nil
}

// ListAllRepos lists every repository attached to any project.
func (r *ProjectsRepo) ListAllRepos(ctx context.Context) ([]workspace.RepoRef, error) {
	rows, err := r.q.ListAllProjectRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all project repos: %w", err)
	}
	out := make([]workspace.RepoRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, workspace.RepoRef{Owner: row.Owner, Name: row.Name, FullName: row.FullName, ConnectorID: row.ConnectorID, Role: workspace.RepoRole(row.Role)})
	}
	return out, nil
}

// GetRepoByFullName reverse-looks-up the RepoRef linked under owner/name, across all projects, for the git router.
func (r *ProjectsRepo) GetRepoByFullName(ctx context.Context, owner, name string) (workspace.RepoRef, error) {
	row, err := r.q.GetProjectRepoByOwnerAndName(ctx, sqlcgen.GetProjectRepoByOwnerAndNameParams{Owner: owner, Name: name})
	if err != nil {
		return workspace.RepoRef{}, fmt.Errorf("get repo %s/%s: %w", owner, name, notFoundIfNoRows(err))
	}
	return workspace.RepoRef{ProjectID: row.ProjectID, Owner: row.Owner, Name: row.Name, FullName: row.FullName, ConnectorID: row.ConnectorID, Role: workspace.RepoRole(row.Role)}, nil
}

// MoveTicket updates a ticket's project in place; the FK guarantees only an existing project can be referenced.
func (r *ProjectsRepo) MoveTicket(ctx context.Context, ticketID, projectID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MoveTicketProject(ctx, sqlcgen.MoveTicketProjectParams{
			ProjectID: nullString(projectID), ID: ticketID,
		})
		if err != nil {
			return fmt.Errorf("move ticket %s to project %s: %w", ticketID, projectID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("move ticket %s to project %s: %w", ticketID, projectID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func setupJSON(p *workspace.Project) (string, string, error) {
	steps := "{}"
	if len(p.Setup.Steps) > 0 {
		encoded, err := json.Marshal(p.Setup.Steps)
		if err != nil {
			return "", "", fmt.Errorf("encode setup of project %s: %w", p.ID, err)
		}
		steps = string(encoded)
	}
	envKeys, err := jsonx.Marshal(p.Setup.EnvKeys)
	if err != nil {
		return "", "", fmt.Errorf("encode setup environment of project %s: %w", p.ID, err)
	}
	return steps, string(envKeys), nil
}

func toProject(row sqlcgen.Project) (*workspace.Project, error) {
	steps := map[workspace.SetupStep]workspace.SetupMark{}
	if err := json.Unmarshal([]byte(row.SetupSteps), &steps); err != nil {
		return nil, fmt.Errorf("decode setup of project %s: %w", row.ID, err)
	}
	var envKeys []string
	if err := json.Unmarshal([]byte(row.SetupEnvKeys), &envKeys); err != nil {
		return nil, fmt.Errorf("decode setup environment of project %s: %w", row.ID, err)
	}
	if len(envKeys) == 0 {
		envKeys = nil
	}
	return &workspace.Project{
		ID:            row.ID,
		Name:          row.Name,
		Prefix:        row.Prefix,
		Position:      int(row.Position),
		WorkspaceID:   row.WorkspaceID,
		Icon:          workspace.ProjectIcon(row.Icon),
		TestsLocation: workspace.TestsLocation(row.TestsLocation),
		Setup:         workspace.ProjectSetup{Finished: row.SetupFinished != 0, Steps: steps, StackID: row.SetupStackID, EnvKeys: envKeys},
		CreatedAt:     time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:     time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

func toProjects(rows []sqlcgen.Project) ([]*workspace.Project, error) {
	out := make([]*workspace.Project, 0, len(rows))
	for _, row := range rows {
		p, err := toProject(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
