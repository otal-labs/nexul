package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ deploy.ServiceRepo = (*ServicesRepo)(nil)

// ServicesRepo backs deploy.ServiceRepo: observed deploy.Container rows in the `services` table, not stacks.
type ServicesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *ServicesRepo) Create(ctx context.Context, svc *deploy.Container) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return r.q.WithTx(tx).CreateContainer(ctx, toCreateContainerParams(svc))
	})
}

// Upsert replaces the row matching (stack_id, name), or inserts one if none exists; used by observation reports.
func (r *ServicesRepo) Upsert(ctx context.Context, svc *deploy.Container) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return r.q.WithTx(tx).UpsertContainer(ctx, sqlcgen.UpsertContainerParams(toCreateContainerParams(svc)))
	})
}

func (r *ServicesRepo) ListByStack(ctx context.Context, stackID string) ([]*deploy.Container, error) {
	rows, err := r.q.ListContainersByStack(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("list services for stack %s: %w", stackID, err)
	}
	return toContainers(rows)
}

// ListContainerNamesByMachine names every container a stack on the machine already tracks, managed or
// adopted; machine discovery hides these so the import door only offers what Nexul does not know yet.
func (r *ServicesRepo) ListContainerNamesByMachine(ctx context.Context, machine string) ([]string, error) {
	names, err := r.q.ListContainerNamesByMachine(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("list container names for machine %s: %w", machine, err)
	}
	return names, nil
}

func (r *ServicesRepo) Get(ctx context.Context, id string) (*deploy.Container, error) {
	row, err := r.q.GetContainer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get service %s: %w", id, notFoundIfNoRows(err))
	}
	return toContainer(row)
}

func (r *ServicesRepo) DeleteByStack(ctx context.Context, stackID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return r.q.WithTx(tx).DeleteContainersByStack(ctx, stackID)
	})
}

func toCreateContainerParams(svc *deploy.Container) sqlcgen.CreateContainerParams {
	var observedAt int64
	if !svc.ObservedAt.IsZero() {
		observedAt = svc.ObservedAt.Unix()
	}
	return sqlcgen.CreateContainerParams{
		ID: svc.ID, StackID: svc.StackID, Name: svc.Name, Declared: declaredJSON(svc.Declared),
		ContainerName: svc.ContainerName, Image: svc.Image, Status: string(svc.Status),
		Networks: networksJSON(svc.Networks), Ports: portsJSON(svc.Ports), ObservedAt: observedAt,
	}
}

func toContainer(row sqlcgen.Service) (*deploy.Container, error) {
	svc := &deploy.Container{
		ID: row.ID, StackID: row.StackID, Name: row.Name,
		ContainerName: row.ContainerName, Image: row.Image, Status: deploy.ServiceStatus(row.Status),
	}
	if row.ObservedAt > 0 {
		svc.ObservedAt = time.Unix(row.ObservedAt, 0).UTC()
	}
	if err := json.Unmarshal([]byte(row.Declared), &svc.Declared); err != nil {
		return nil, fmt.Errorf("unmarshal declared for service %s: %w", svc.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Networks), &svc.Networks); err != nil {
		return nil, fmt.Errorf("unmarshal networks for service %s: %w", svc.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Ports), &svc.Ports); err != nil {
		return nil, fmt.Errorf("unmarshal ports for service %s: %w", svc.ID, err)
	}
	return svc, nil
}

func toContainers(rows []sqlcgen.Service) ([]*deploy.Container, error) {
	var out []*deploy.Container
	for _, row := range rows {
		svc, err := toContainer(row)
		if err != nil {
			return nil, err
		}
		out = append(out, svc)
	}
	return out, nil
}

func declaredJSON(d deploy.Declared) string {
	b, err := json.Marshal(d)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func networksJSON(networks []deploy.Network) string {
	if len(networks) == 0 {
		return "[]"
	}
	b, err := json.Marshal(networks)
	if err != nil {
		return "[]"
	}
	return string(b)
}
