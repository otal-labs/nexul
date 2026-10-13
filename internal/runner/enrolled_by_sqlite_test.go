package runner_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/runner"
)

// TestEnrolledBy_RealSQLite: whoever made the install command is kept on the runner it enrolls, named on its
// connect, and counts as having a connected deploy server while it is.
func TestEnrolledBy_RealSQLite(t *testing.T) {
	in := newInstance(t)
	connects := make(chan runner.RunnerConnectedEvent, 1)
	require.NoError(t, in.bus.Subscribe(context.Background(), runner.TopicRunnerConnected, func(_ context.Context, ev eventbus.Event) error {
		var e runner.RunnerConnectedEvent
		require.NoError(t, json.Unmarshal(ev.Payload, &e))
		connects <- e
		return nil
	}))
	enrolled := in.enroll(t, "builder", "prod")

	stored, err := in.store.Runners.GetByName(t.Context(), "builder")
	require.NoError(t, err)
	assert.Equal(t, "admin", stored.EnrolledBy)
	ready, err := in.store.Runners.PeopleWithConnectedDeployRunner(t.Context(), []string{"admin"})
	require.NoError(t, err)
	assert.False(t, ready["admin"], "enrolled but not connected yet")

	in.start(t, "builder", enrolled["credential"].(string), &recordingExecutor{})
	select {
	case e := <-connects:
		assert.Equal(t, "admin", e.EnrolledBy)
	case <-time.After(3 * time.Second):
		t.Fatal("the runner never connected")
	}
	ready, err = in.store.Runners.PeopleWithConnectedDeployRunner(t.Context(), []string{"admin", "lena"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"admin": true}, ready)
}
