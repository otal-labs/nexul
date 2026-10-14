package runner_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/runner"
)

func (in *instance) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, in.db.QueryRow(query, args...).Scan(&n))
	return n
}

// TestMachineDelete_RealSQLite: a machine is refused while a stack targets it or a runner on it is connected, and once
// free it goes with its offline runners, their credentials, its rules and the codes waiting to enroll one, writing
// machine.deleted in the same commit; a retired runner that comes back uninstalls itself.
func TestMachineDelete_RealSQLite(t *testing.T) {
	ctx := context.Background()
	in := newInstance(t)
	live := in.enroll(t, "alpha", "prod")
	idle := in.enroll(t, "gamma", "prod")
	status, _ := in.post(t, "/api/runners/enrollments", map[string]string{"name": "delta", "machine": "prod"})
	require.Equal(t, http.StatusCreated, status)
	machine, err := in.store.Machines.GetByName(ctx, "prod")
	require.NoError(t, err)
	_, err = in.db.Exec(`INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at) VALUES ('s-1', 'api', 'api', 'prod', 'run', 1, 1)`)
	require.NoError(t, err)
	require.Equal(t, 1, in.count(t, `SELECT COUNT(*) FROM permission_overwrites WHERE resource_type = 'machine' AND resource_id = ?`, machine.ID), "enrolling gave its adder a rule")
	in.start(t, "alpha", live["credential"].(string), &recordingExecutor{})
	require.Eventually(t, func() bool { return in.connected(t)["alpha"] }, 3*time.Second, 10*time.Millisecond)

	var inUse *apperrs.InUseError
	require.True(t, errors.As(in.svc.DeleteMachine(ctx, machine.ID), &inUse))
	assert.Equal(t, []string{"stack api", "connected runner alpha"}, inUse.Users)

	_, err = in.db.Exec(`DELETE FROM stacks WHERE id = 's-1'`)
	require.NoError(t, err)
	require.True(t, errors.As(in.svc.DeleteMachine(ctx, machine.ID), &inUse))
	assert.Equal(t, []string{"connected runner alpha"}, inUse.Users, "a free stack does not clear the live runner")

	require.NoError(t, in.svc.RemoveRunner(ctx, live["id"].(string)))
	require.NoError(t, in.svc.DeleteMachine(ctx, machine.ID))

	_, err = in.store.Machines.Get(ctx, machine.ID)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Zero(t, in.count(t, `SELECT COUNT(*) FROM runners WHERE machine_id = ?`, machine.ID), "the offline runner went with it")
	assert.Zero(t, in.count(t, `SELECT COUNT(*) FROM permission_overwrites WHERE resource_type = 'machine' AND resource_id = ?`, machine.ID))
	assert.Zero(t, in.count(t, `SELECT COUNT(*) FROM runner_enrollment_codes WHERE machine = 'prod'`), "no code can put a runner back on it")
	assert.Equal(t, 1, in.count(t, `SELECT COUNT(*) FROM outbox WHERE topic = ?`, runner.TopicMachineDeleted))
	assert.Zero(t, in.count(t, `SELECT COUNT(*) FROM runner_credentials WHERE runner_id = ? AND revoked_at IS NULL`, idle["id"]))

	returning := &recordingExecutor{}
	select {
	case err := <-in.start(t, "gamma", idle["credential"].(string), returning):
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("a runner of a deleted machine was not refused")
	}
	_, uninstalls := returning.snapshot()
	assert.Equal(t, []string{"gamma"}, uninstalls)
}
