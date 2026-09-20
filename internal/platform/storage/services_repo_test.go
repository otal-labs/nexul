package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestContainer(id, stackID string) *deploy.Container {
	return &deploy.Container{
		ID:      id,
		StackID: stackID,
		Name:    "api",
		Declared: deploy.Declared{
			Image: "nginx", Ports: []string{"8080:80"}, EnvKeys: []string{"PORT"},
		},
		Status: deploy.ServiceStatusPending,
	}
}

// seedStackForContainers seeds a stack whose slug is derived from id, so seeding several in one test never
// collides on the (machine, slug) unique index the way a shared hardcoded "api" name/slug would.
func seedStackForContainers(t *testing.T, s *Store, id string) {
	t.Helper()
	stack := newTestStack(id)
	stack.Name, stack.Slug = id, id
	require.NoError(t, s.Stacks.Create(context.Background(), stack))
}

func TestServicesRepo_Create_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedStackForContainers(t, s, "stack-1")
	want := newTestContainer("svc-1", "stack-1")
	require.NoError(t, s.Services.Create(context.Background(), want))

	got, err := s.Services.Get(context.Background(), "svc-1")
	require.NoError(t, err)
	assert.Equal(t, "api", got.Name)
	assert.Equal(t, "stack-1", got.StackID)
	assert.Equal(t, "nginx", got.Declared.Image)
	assert.Equal(t, []string{"8080:80"}, got.Declared.Ports)
	assert.Equal(t, deploy.ServiceStatusPending, got.Status)
}

func TestServicesRepo_Create_WithNetworksAndPorts(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedStackForContainers(t, s, "stack-1")
	svc := newTestContainer("svc-1", "stack-1")
	svc.ContainerName = "api"
	svc.Image = "nginx:1.27"
	svc.Status = deploy.ServiceStatusRunning
	svc.Networks = []deploy.Network{{Name: "app-net", Address: "172.18.0.4"}}
	svc.Ports = []string{"8080:80"}
	svc.ObservedAt = time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	require.NoError(t, s.Services.Create(context.Background(), svc))

	got, err := s.Services.Get(context.Background(), "svc-1")
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.27", got.Image)
	assert.Equal(t, deploy.ServiceStatusRunning, got.Status)
	require.Len(t, got.Networks, 1)
	assert.Equal(t, "app-net", got.Networks[0].Name)
	assert.Equal(t, "172.18.0.4", got.Networks[0].Address)
	assert.Equal(t, []string{"8080:80"}, got.Ports)
	assert.True(t, got.ObservedAt.Equal(svc.ObservedAt))
}

func TestServicesRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Services.Get(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestServicesRepo_ListByStack(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedStackForContainers(t, s, "stack-1")
	seedStackForContainers(t, s, "stack-2")
	a := newTestContainer("svc-1", "stack-1")
	b := newTestContainer("svc-2", "stack-1")
	b.Name = "db"
	c := newTestContainer("svc-3", "stack-2")
	require.NoError(t, s.Services.Create(context.Background(), a))
	require.NoError(t, s.Services.Create(context.Background(), b))
	require.NoError(t, s.Services.Create(context.Background(), c))

	got, err := s.Services.ListByStack(context.Background(), "stack-1")
	require.NoError(t, err)
	require.Len(t, got, 2)

	other, err := s.Services.ListByStack(context.Background(), "stack-2")
	require.NoError(t, err)
	require.Len(t, other, 1)
	assert.Equal(t, "svc-3", other[0].ID)
}

func TestServicesRepo_Upsert_InsertsThenUpdatesByStackAndName(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedStackForContainers(t, s, "stack-1")
	svc := newTestContainer("svc-1", "stack-1")
	require.NoError(t, s.Services.Upsert(context.Background(), svc))

	// Same (stack_id, name), different id: the row is replaced in place, not duplicated.
	updated := newTestContainer("svc-2", "stack-1")
	updated.Status = deploy.ServiceStatusHealthy
	updated.ContainerName = "api"
	require.NoError(t, s.Services.Upsert(context.Background(), updated))

	got, err := s.Services.ListByStack(context.Background(), "stack-1")
	require.NoError(t, err)
	require.Len(t, got, 1, "upsert replaces the existing row for (stack_id, name) rather than inserting a second one")
	assert.Equal(t, deploy.ServiceStatusHealthy, got[0].Status)
	assert.Equal(t, "api", got[0].ContainerName)
}

func TestServicesRepo_DeleteByStack(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedStackForContainers(t, s, "stack-1")
	seedStackForContainers(t, s, "stack-2")
	a := newTestContainer("svc-1", "stack-1")
	b := newTestContainer("svc-2", "stack-2")
	require.NoError(t, s.Services.Create(context.Background(), a))
	require.NoError(t, s.Services.Create(context.Background(), b))

	require.NoError(t, s.Services.DeleteByStack(context.Background(), "stack-1"))

	got, err := s.Services.ListByStack(context.Background(), "stack-1")
	require.NoError(t, err)
	assert.Empty(t, got)

	other, err := s.Services.ListByStack(context.Background(), "stack-2")
	require.NoError(t, err)
	assert.Len(t, other, 1, "a different stack's services are untouched")
}

func TestServicesRepo_DeleteByStack_CascadesOnStackDelete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedStackForContainers(t, s, "stack-1")
	svc := newTestContainer("svc-1", "stack-1")
	require.NoError(t, s.Services.Create(context.Background(), svc))

	require.NoError(t, s.Stacks.Delete(context.Background(), "stack-1"))

	got, err := s.Services.ListByStack(context.Background(), "stack-1")
	require.NoError(t, err)
	assert.Empty(t, got, "ON DELETE CASCADE drops a stack's containers when the stack row goes")
}
