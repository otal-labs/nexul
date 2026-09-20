package runner

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestDeployRequestedEvent_ToFrame(t *testing.T) {
	t.Run("build maps to assign_build", func(t *testing.T) {
		req := DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Repo: "org/app", Ref: "main", Steps: []string{"go build"},
			Network: "nexul_default", Ports: []string{"80:80"}, Mounts: []string{"/a:/b"}, Command: []string{"serve"}}
		got, err := req.toFrame(nil)
		require.NoError(t, err)
		assert.Equal(t, FrameAssignBuild, got.Type)
		assert.Equal(t, "org/app", got.Repo)
		assert.Equal(t, []string{"go build"}, got.Steps)
		// A built image starts like any run-strategy service: on its network, with its ports, mounts and command.
		assert.Equal(t, "nexul_default", got.Network)
		assert.Equal(t, []string{"80:80"}, got.Ports)
		assert.Equal(t, []string{"/a:/b"}, got.Mounts)
		assert.Equal(t, []string{"serve"}, got.Command)
	})

	t.Run("deploy maps to assign_deploy", func(t *testing.T) {
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", Image: "img", Strategy: "run"}
		got, err := req.toFrame(map[string]string{"K": "v"})
		require.NoError(t, err)
		assert.Equal(t, FrameAssignDeploy, got.Type)
		assert.Equal(t, "run", got.Strategy)
	})

	t.Run("env comes from the resolved param, not the request's own field", func(t *testing.T) {
		// The request's Env field is what the bus carried — redacted keys only
		// (ticket 14). The frame must use the resolved value passed in, proving
		// the two are decoupled.
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", Env: map[string]string{"K": ""}}
		got, err := req.toFrame(map[string]string{"K": "real-value"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"K": "real-value"}, got.Env)
	})

	t.Run("unknown kind is invalid", func(t *testing.T) {
		_, err := DeployRequestedEvent{ID: "x", Kind: "frobnicate"}.toFrame(nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}
