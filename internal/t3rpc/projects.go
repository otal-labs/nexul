package t3rpc

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// listProjectsTimeout bounds the wait for the shell snapshot; a slow answer means a broken server, not a big list.
const listProjectsTimeout = 15 * time.Second

// repositoryWait bounds the wait for git remotes T3 had not cached when it sent the registry.
const repositoryWait = 1500 * time.Millisecond

type shellItem struct {
	Kind     string `json:"kind"`
	Snapshot struct {
		Projects []struct {
			ID                 string  `json:"id"`
			Title              string  `json:"title"`
			WorkspaceRoot      string  `json:"workspaceRoot"`
			DeletedAt          *string `json:"deletedAt"`
			RepositoryIdentity *struct {
				CanonicalKey string `json:"canonicalKey"`
			} `json:"repositoryIdentity"`
		} `json:"projects"`
	} `json:"snapshot"`
	ResolvedRepositoryIdentityRoots []string `json:"resolvedRepositoryIdentityRoots"`
}

// ListProjects reads the registry off orchestration.subscribeShell's first snapshot item, since no list RPC exists,
// then the follow-up items that carry the git remotes T3 resolves after it.
func (c *Conn) ListProjects(ctx context.Context) ([]harness.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, listProjectsTimeout)
	defer cancel()

	stream, err := c.Stream(ctx, "orchestration.subscribeShell", nil)
	if err != nil {
		return nil, fmt.Errorf("subscribe shell: %w", err)
	}
	defer stream.Close()

	for {
		values, err := stream.Next(ctx)
		if errors.Is(err, ErrConnectionLost) {
			return nil, err
		}
		if ctx.Err() != nil {
			return nil, apperrs.Retryable(fmt.Errorf("list projects: %w", ctx.Err()))
		}
		if err != nil {
			return nil, apperrs.Retryable(fmt.Errorf("shell subscription ended before a snapshot arrived"))
		}
		for i, raw := range values {
			var item shellItem
			if err := json.Unmarshal(raw, &item); err != nil || item.Kind != "snapshot" {
				continue
			}
			projects := make([]harness.Project, 0, len(item.Snapshot.Projects))
			for _, p := range item.Snapshot.Projects {
				if p.DeletedAt != nil || p.ID == "" {
					continue
				}
				project := harness.Project{ID: p.ID, Title: p.Title, Path: p.WorkspaceRoot}
				if p.RepositoryIdentity != nil {
					project.Repository = p.RepositoryIdentity.CanonicalKey
				}
				projects = append(projects, project)
			}
			if c.Protocol() >= 2 {
				addRepositories(ctx, stream, projects, values[i+1:])
			}
			return projects, nil
		}
	}
}

// addRepositories fills each project's Repository from the items after the registry, until T3 has resolved every
// project's remote or repositoryWait passes; a project outside a git checkout resolves to none.
func addRepositories(ctx context.Context, stream *Stream, projects []harness.Project, pending []json.RawMessage) {
	unresolved := map[string]bool{}
	for _, p := range projects {
		if p.Repository == "" {
			unresolved[p.Path] = true
		}
	}
	ctx, cancel := context.WithTimeout(ctx, repositoryWait)
	defer cancel()
	for len(unresolved) > 0 {
		for _, raw := range pending {
			var item shellItem
			if err := json.Unmarshal(raw, &item); err != nil || item.Kind != "snapshot" || item.ResolvedRepositoryIdentityRoots == nil {
				continue
			}
			for _, p := range item.Snapshot.Projects {
				i := slices.IndexFunc(projects, func(project harness.Project) bool { return project.Path == p.WorkspaceRoot })
				if i >= 0 && p.RepositoryIdentity != nil {
					projects[i].Repository = p.RepositoryIdentity.CanonicalKey
				}
			}
			for _, root := range item.ResolvedRepositoryIdentityRoots {
				delete(unresolved, root)
			}
		}
		if len(unresolved) == 0 {
			return
		}
		var err error
		if pending, err = stream.Next(ctx); err != nil {
			return
		}
	}
}

// projectCloneStart clones a repository into a new project; it answers once the project exists and the clone runs.
const projectCloneStart = "projectClone.start"

type cloneStart struct {
	ProjectID       string `json:"projectId"`
	Title           string `json:"title"`
	CreatedAt       string `json:"createdAt"`
	RemoteURL       string `json:"remoteUrl"`
	DestinationPath string `json:"destinationPath"`
}

// CloneProject has T3 clone remoteURL into path as a new project titled title and waits for the clone to end; a clone
// that fails has its project removed again and comes back as a *harness.CloneFailedError.
func (c *Conn) CloneProject(ctx context.Context, title, remoteURL, path string) (harness.Project, error) {
	project, err := c.startClone(ctx, title, remoteURL, path)
	if err != nil {
		return harness.Project{}, err
	}
	detail, err := c.awaitClone(ctx, project.ID)
	if err != nil {
		return harness.Project{}, fmt.Errorf("wait for the clone of %s: %w", remoteURL, err)
	}
	if detail == "" {
		return project, nil
	}
	if err := c.deleteProject(ctx, project.ID); err != nil {
		c.log.Warn("t3rpc: remove the project of a failed clone", "project_id", project.ID, "error", err)
	}
	return harness.Project{}, &harness.CloneFailedError{Detail: detail}
}

func (c *Conn) startClone(ctx context.Context, title, remoteURL, path string) (harness.Project, error) {
	raw, err := c.Call(ctx, projectCloneStart, cloneStart{
		ProjectID: ids.New(), Title: title, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), RemoteURL: remoteURL, DestinationPath: path,
	})
	var exit *ExitError
	if errors.As(err, &exit) && exit.Failed("EnvironmentAuthorizationError") {
		return harness.Project{}, harness.ErrPairAgainToClone
	}
	if errors.As(err, &exit) && exit.Detail() != "" {
		return harness.Project{}, fmt.Errorf("%w: %s", apperrs.ErrInvalid, exit.Detail())
	}
	if err != nil {
		return harness.Project{}, err
	}
	var out struct {
		ProjectID string `json:"projectId"`
		Cwd       string `json:"cwd"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.ProjectID == "" {
		return harness.Project{}, fmt.Errorf("%s: unexpected result %s", projectCloneStart, Snippet(raw))
	}
	return harness.Project{ID: out.ProjectID, Title: title, Path: out.Cwd}, nil
}

// subscribeProjectClones streams the list of clones T3 tracks, once at the start and again on every change.
const subscribeProjectClones = "subscribeProjectClones"

type cloneSnapshot struct {
	ProjectID string `json:"projectId"`
	Phase     string `json:"phase"`
	Error     string `json:"error"`
}

// awaitClone follows projectID's clone until it ends and returns why it did not finish, empty when it did.
func (c *Conn) awaitClone(ctx context.Context, projectID string) (string, error) {
	stream, err := c.Stream(ctx, subscribeProjectClones, nil)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	for {
		values, err := stream.Next(ctx)
		if err != nil {
			return "", err
		}
		for _, raw := range values {
			var snapshots []cloneSnapshot
			if err := json.Unmarshal(raw, &snapshots); err != nil {
				continue
			}
			listed := false
			for _, s := range snapshots {
				if s.ProjectID != projectID {
					continue
				}
				listed = true
				switch s.Phase {
				case "done":
					return "", nil
				case "cancelled":
					return "The clone was cancelled.", nil
				case "failed":
					return cmp.Or(s.Error, "The repository could not be cloned."), nil
				}
			}
			// T3 drops a finished clone 30 seconds after it ends, and a failed one stays, so a list without it means it finished.
			if !listed {
				return "", nil
			}
		}
	}
}

// projectsMutate carries T3's project create, update and delete commands; create and delete need only orchestration:operate.
const projectsMutate = "projects.mutate"

type projectCreate struct {
	Type                         string `json:"type"`
	CommandID                    string `json:"commandId"`
	ProjectID                    string `json:"projectId"`
	Title                        string `json:"title"`
	WorkspaceRoot                string `json:"workspaceRoot"`
	CreateWorkspaceRootIfMissing bool   `json:"createWorkspaceRootIfMissing"`
}

// CreateProject registers path as a new project titled title; T3 makes the folder when it is missing.
func (c *Conn) CreateProject(ctx context.Context, title, path string) (harness.Project, error) {
	raw, err := c.Call(ctx, projectsMutate, projectCreate{
		Type: "project.create", CommandID: ids.New(), ProjectID: ids.New(), Title: title, WorkspaceRoot: path, CreateWorkspaceRootIfMissing: true,
	})
	var exit *ExitError
	if errors.As(err, &exit) && exit.Message("ProjectMutationError") != "" {
		return harness.Project{}, fmt.Errorf("%w: %s", apperrs.ErrInvalid, exit.Message("ProjectMutationError"))
	}
	if err != nil {
		return harness.Project{}, err
	}
	var out struct {
		ID            string `json:"id"`
		WorkspaceRoot string `json:"workspaceRoot"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.ID == "" {
		return harness.Project{}, fmt.Errorf("%s: unexpected result %s", projectsMutate, Snippet(raw))
	}
	return harness.Project{ID: out.ID, Title: title, Path: out.WorkspaceRoot}, nil
}

// deleteProject removes projectID; T3 discards the folder of a clone that did not finish.
func (c *Conn) deleteProject(ctx context.Context, projectID string) error {
	_, err := c.Call(ctx, projectsMutate, map[string]string{"type": "project.delete", "commandId": ids.New(), "projectId": projectID})
	return err
}
