package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ deploy.StackRepo = (*StacksRepo)(nil)

// StacksRepo persists stacks.
type StacksRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *StacksRepo) Create(ctx context.Context, stack *deploy.Stack, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := insertStack(ctx, r.q.WithTx(tx), stack); err != nil {
			return err
		}
		return enqueueServicesOutbox(ctx, tx, evts)
	})
}

func (r *StacksRepo) GetByID(ctx context.Context, id string) (*deploy.Stack, error) {
	row, err := r.q.GetStack(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get stack %s: %w", id, notFoundIfNoRows(err))
	}
	return toStack(row)
}

func (r *StacksRepo) GetBySlugAndMachine(ctx context.Context, slug, machine string) (*deploy.Stack, error) {
	row, err := r.q.GetStackBySlugAndMachine(ctx, sqlcgen.GetStackBySlugAndMachineParams{Slug: slug, Machine: machine})
	if err != nil {
		return nil, fmt.Errorf("get stack %s on %s: %w", slug, machine, notFoundIfNoRows(err))
	}
	return toStack(row)
}

// GetByName returns the first stack with this name (names are unique only per machine+slug).
// ponytail: first match, not machine-disambiguated — fine for one machine per workspace; add a lookup if that changes.
func (r *StacksRepo) GetByName(ctx context.Context, name string) (*deploy.Stack, error) {
	row, err := r.q.GetStackByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get stack %s: %w", name, notFoundIfNoRows(err))
	}
	return toStack(row)
}

func (r *StacksRepo) ListByProject(ctx context.Context, projectID string) ([]*deploy.Stack, error) {
	rows, err := r.q.ListStacksByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list stacks for project %s: %w", projectID, err)
	}
	return toStacks(rows)
}

// ListByBuildRepo returns every base stack whose build source references this repository, for the push consumer.
func (r *StacksRepo) ListByBuildRepo(ctx context.Context, owner, name string) ([]*deploy.Stack, error) {
	rows, err := r.q.ListStacksByBuildRepo(ctx, sqlcgen.ListStacksByBuildRepoParams{BuildRepoOwner: owner, BuildRepoName: name})
	if err != nil {
		return nil, fmt.Errorf("list stacks for repo %s/%s: %w", owner, name, err)
	}
	return toStacks(rows)
}

// ListByDerivedFrom returns a base stack's branch deployments, oldest first.
func (r *StacksRepo) ListByDerivedFrom(ctx context.Context, baseStackID string) ([]*deploy.Stack, error) {
	rows, err := r.q.ListStacksByDerivedFrom(ctx, baseStackID)
	if err != nil {
		return nil, fmt.Errorf("list branch deployments for %s: %w", baseStackID, err)
	}
	return toStacks(rows)
}

func (r *StacksRepo) Update(ctx context.Context, stack *deploy.Stack, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateStack(ctx, sqlcgen.UpdateStackParams{
			ProjectID: stack.ProjectID, Name: stack.Name, Slug: stack.Slug, Machine: stack.Machine,
			Strategy: string(stack.Strategy), ComposePath: stack.ComposePath,
			Env: envJSON(stack.Env), DockerNetwork: stack.DockerNetwork, Ports: portsJSON(stack.Ports), Mounts: portsJSON(stack.Mounts), Command: portsJSON(stack.Command),
			BuildRepoOwner: buildOwner(stack), BuildRepoName: buildName(stack), BuildBranch: buildBranch(stack), BuildDockerfile: buildDockerfile(stack), BuildComposePath: buildComposePath(stack),
			BranchDeployRules: branchDeployRulesJSON(stack.BranchDeployRules), DerivedFrom: stack.DerivedFrom, Branch: stack.Branch,
			Managed: boolToInt(stack.Managed), UpdatedAt: stack.UpdatedAt.Unix(), ID: stack.ID,
		})
		if err != nil {
			return fmt.Errorf("update stack %s: %w", stack.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update stack %s: %w", stack.ID, apperrs.ErrNotFound)
		}
		return enqueueServicesOutbox(ctx, tx, evts)
	})
}

func (r *StacksRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteStack(ctx, id)
		if err != nil {
			return fmt.Errorf("delete stack %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete stack %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueServicesOutbox(ctx, tx, evts)
	})
}

func insertStack(ctx context.Context, q *sqlcgen.Queries, stack *deploy.Stack) error {
	err := q.CreateStack(ctx, sqlcgen.CreateStackParams{
		ID: stack.ID, ProjectID: stack.ProjectID, Name: stack.Name, Slug: stack.Slug, Machine: stack.Machine,
		Strategy: string(stack.Strategy), ComposePath: stack.ComposePath,
		Env: envJSON(stack.Env), DockerNetwork: stack.DockerNetwork, Ports: portsJSON(stack.Ports), Mounts: portsJSON(stack.Mounts), Command: portsJSON(stack.Command),
		BuildRepoOwner: buildOwner(stack), BuildRepoName: buildName(stack), BuildBranch: buildBranch(stack), BuildDockerfile: buildDockerfile(stack), BuildComposePath: buildComposePath(stack),
		BranchDeployRules: branchDeployRulesJSON(stack.BranchDeployRules), DerivedFrom: stack.DerivedFrom, Branch: stack.Branch,
		Managed: boolToInt(stack.Managed), CreatedAt: stack.CreatedAt.Unix(), UpdatedAt: stack.UpdatedAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert stack %s: %w", stack.Name, classifyWriteErr(err))
	}
	return nil
}

func toStack(row sqlcgen.Stack) (*deploy.Stack, error) {
	stack := &deploy.Stack{
		ID:            row.ID,
		ProjectID:     row.ProjectID,
		Name:          row.Name,
		Slug:          row.Slug,
		Machine:       row.Machine,
		Strategy:      deploy.Strategy(row.Strategy),
		ComposePath:   row.ComposePath,
		DockerNetwork: row.DockerNetwork,
		DerivedFrom:   row.DerivedFrom,
		Branch:        row.Branch,
		Managed:       row.Managed != 0,
		CreatedAt:     time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:     time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if err := json.Unmarshal([]byte(row.Env), &stack.Env); err != nil {
		return nil, fmt.Errorf("unmarshal env for stack %s: %w", stack.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Ports), &stack.Ports); err != nil {
		return nil, fmt.Errorf("unmarshal ports for stack %s: %w", stack.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Mounts), &stack.Mounts); err != nil {
		return nil, fmt.Errorf("unmarshal mounts for stack %s: %w", stack.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Command), &stack.Command); err != nil {
		return nil, fmt.Errorf("unmarshal command for stack %s: %w", stack.ID, err)
	}
	if err := json.Unmarshal([]byte(row.BranchDeployRules), &stack.BranchDeployRules); err != nil {
		return nil, fmt.Errorf("unmarshal branch deploy rules for stack %s: %w", stack.ID, err)
	}
	if row.BuildRepoOwner != "" || row.BuildRepoName != "" {
		stack.BuildSource = &deploy.BuildSource{
			RepoOwner: row.BuildRepoOwner, RepoName: row.BuildRepoName, Branch: row.BuildBranch,
			Dockerfile: row.BuildDockerfile, ComposePath: row.BuildComposePath,
		}
	}
	return stack, nil
}

func toStacks(rows []sqlcgen.Stack) ([]*deploy.Stack, error) {
	var out []*deploy.Stack
	for _, row := range rows {
		stack, err := toStack(row)
		if err != nil {
			return nil, err
		}
		out = append(out, stack)
	}
	return out, nil
}

func envJSON(env map[string]string) string {
	b, err := json.Marshal(env)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func branchDeployRulesJSON(rules []deploy.BranchDeployRule) string {
	if len(rules) == 0 {
		return "[]"
	}
	b, err := json.Marshal(rules)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func portsJSON(ports []string) string {
	if len(ports) == 0 {
		return "[]"
	}
	b, err := json.Marshal(ports)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func buildOwner(stack *deploy.Stack) string {
	if stack.BuildSource != nil {
		return stack.BuildSource.RepoOwner
	}
	return ""
}

func buildName(stack *deploy.Stack) string {
	if stack.BuildSource != nil {
		return stack.BuildSource.RepoName
	}
	return ""
}

func buildBranch(stack *deploy.Stack) string {
	if stack.BuildSource != nil {
		return stack.BuildSource.Branch
	}
	return ""
}

func buildDockerfile(stack *deploy.Stack) string {
	if stack.BuildSource != nil {
		return stack.BuildSource.Dockerfile
	}
	return ""
}

func buildComposePath(stack *deploy.Stack) string {
	if stack.BuildSource != nil {
		return stack.BuildSource.ComposePath
	}
	return ""
}

// boolToInt stores a Go bool in a SQLite INTEGER column (0/1); there is no native boolean type.
func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// enqueueServicesOutbox writes the mutation's outbox rows in the same transaction.
func enqueueServicesOutbox(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}
