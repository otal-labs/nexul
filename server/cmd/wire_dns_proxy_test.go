package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/runner"
)

func newDNSWireStore(t *testing.T) *storage.Store {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
}

// A fresh instance has no project, and its gateway stack needs none.
func TestDNSProvisioner_RetryUpdatesTheStack(t *testing.T) {
	s := newDNSWireStore(t)
	deploySvc := deploy.NewService(s.Deploys, s.Stacks, s.Services, deployProjectStore{projects: s.Projects})
	prov := dnsProvisioner{deploy: deploySvc}
	spec := dns.AgentSpec{
		Target: "box", Name: "nexul-proxy", Strategy: "run", DockerNetwork: "nexul_proxy",
		Image: "traefik:v3", Ports: []string{"80:80"}, Env: map[string]string{"A": "1"},
	}

	first, err := prov.Provision(t.Context(), spec)
	require.NoError(t, err)
	spec.Env = map[string]string{"A": "2"}
	spec.Mounts = []string{"nexul-traefik-acme:/letsencrypt"}
	second, err := prov.Provision(t.Context(), spec)
	require.NoError(t, err, "a deploy still pending is not an error")

	assert.Equal(t, first.ServiceID, second.ServiceID)
	stack, err := deploySvc.GetStack(t.Context(), first.ServiceID)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"A": "2"}, stack.Env)
	assert.Equal(t, []string{"nexul-traefik-acme:/letsencrypt"}, stack.Mounts)
	assert.Empty(t, stack.ProjectID)
}

func TestDNSInstancePlacement(t *testing.T) {
	t.Run("no instance runner is invalid", func(t *testing.T) {
		s := newDNSWireStore(t)
		_, err := dnsInstancePlacement{runners: s.Runners, machines: s.Machines}.InstanceMachine(t.Context())
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("an instance runner that never connected is invalid", func(t *testing.T) {
		s := newDNSWireStore(t)
		require.NoError(t, s.Runners.Create(t.Context(), &runner.Runner{ID: "r1", Name: "instance", CreatedAt: time.Now()}))
		_, err := dnsInstancePlacement{runners: s.Runners, machines: s.Machines}.InstanceMachine(t.Context())
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("the instance runner's machine", func(t *testing.T) {
		s := newDNSWireStore(t)
		require.NoError(t, s.Machines.Create(t.Context(), &runner.Machine{ID: "m1", Name: "box", FirstSeen: time.Now(), LastSeen: time.Now()}))
		require.NoError(t, s.Runners.Create(t.Context(), &runner.Runner{ID: "r1", Name: "instance", MachineID: "m1", CreatedAt: time.Now()}))
		p := dnsInstancePlacement{runners: s.Runners, machines: s.Machines}
		machine, err := p.InstanceMachine(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "box", machine)
	})
}
