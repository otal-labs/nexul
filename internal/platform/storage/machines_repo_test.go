package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/runner"
)

func newTestMachine(id, name string) *runner.Machine {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	return &runner.Machine{ID: id, Name: name, StackRoot: "/data/nexul", ReportedHostname: name, FirstSeen: now, LastSeen: now}
}

func TestMachinesRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestMachine("m-1", "prod")
	require.NoError(t, s.Machines.Create(context.Background(), want))

	got, err := s.Machines.Get(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, "prod", got.Name)
	assert.Equal(t, "/data/nexul", got.StackRoot)
	assert.Equal(t, "prod", got.ReportedHostname)
}

func TestMachinesRepo_Create_DuplicateName_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Machines.Create(context.Background(), newTestMachine("m-1", "prod")))
	err := s.Machines.Create(context.Background(), newTestMachine("m-2", "prod"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestMachinesRepo_GetByName(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Machines.Create(context.Background(), newTestMachine("m-1", "prod")))

	got, err := s.Machines.GetByName(context.Background(), "prod")
	require.NoError(t, err)
	assert.Equal(t, "m-1", got.ID)

	_, err = s.Machines.GetByName(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestMachinesRepo_List_OrdersByName(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Machines.Create(context.Background(), newTestMachine("m-2", "staging")))
	require.NoError(t, s.Machines.Create(context.Background(), newTestMachine("m-1", "prod")))

	got, err := s.Machines.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "prod", got[0].Name)
	assert.Equal(t, "staging", got[1].Name)
}

func TestMachinesRepo_Rename(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Machines.Create(context.Background(), newTestMachine("m-1", "prod")))

	require.NoError(t, s.Machines.Rename(context.Background(), "m-1", "prod-primary"))

	got, err := s.Machines.Get(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, "prod-primary", got.Name)
}

func TestMachinesRepo_SetStackRoot(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Machines.Create(context.Background(), newTestMachine("m-1", "prod")))

	require.NoError(t, s.Machines.SetStackRoot(context.Background(), "m-1", "/srv/data"))

	got, err := s.Machines.Get(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, "/srv/data", got.StackRoot)
}

func TestMachinesRepo_Touch_UpdatesLastSeenAndReportedHostname(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Machines.Create(context.Background(), newTestMachine("m-1", "prod")))

	at := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	require.NoError(t, s.Machines.Touch(context.Background(), "m-1", "prod.local", at))

	got, err := s.Machines.Get(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, at, got.LastSeen)
	assert.Equal(t, "prod.local", got.ReportedHostname)
}

func TestMachinesRepo_DeletingAMachineDeletesItsRules(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	s := New(db, testEncKey)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("bob", "bob", "bob"))
	require.NoError(t, err)
	require.NoError(t, s.Machines.Create(ctx, newTestMachine("m-1", "prod")))
	require.NoError(t, s.Machines.Create(ctx, newTestMachine("m-2", "stage")))
	require.NoError(t, s.Access.Set(ctx, "machine", "m-1", "bob", permissions.SetOf(permissions.MachinesSee), nil))
	require.NoError(t, s.Access.Set(ctx, "machine", "m-2", "bob", permissions.SetOf(permissions.MachinesSee), nil))

	_, err = db.Exec(`DELETE FROM machines WHERE id = 'm-1'`)
	require.NoError(t, err)

	_, err = s.Access.Get(ctx, "machine", "m-1", "bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a machine's rules go with it")
	_, err = s.Access.Get(ctx, "machine", "m-2", "bob")
	require.NoError(t, err, "another machine's stay")
}
