package pairing

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeTunnels scripts the Cloudflare side of a computer tunnel and records what was torn down.
type fakeTunnels struct {
	mu        sync.Mutex
	created   []string
	deleted   []ComputerTunnel
	status    string
	createErr error
	deleteErr error
	readErr   error
}

func (f *fakeTunnels) CreateTunnel(_ context.Context, name string, port int) (*ComputerTunnel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = append(f.created, name)
	return &ComputerTunnel{TunnelID: "tun-1", Hostname: "laptop-ab12cd34.example.com", ZoneID: "z1", RecordID: "rec-1", AccessAppID: "app-1"}, nil
}

func (f *fakeTunnels) DeleteTunnel(_ context.Context, t ComputerTunnel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, t)
	return nil
}

func (f *fakeTunnels) TunnelStatus(_ context.Context, _ string) (string, error) {
	return f.status, f.readErr
}

func (f *fakeTunnels) TunnelToken(_ context.Context, tunnelID string) (string, error) {
	if f.readErr != nil {
		return "", f.readErr
	}
	return "connector-token-" + tunnelID, nil
}

func newTunnelService(repo *fakeRepo, exch *fakeExchanger, tunnels Tunnels) (*Service, *[]string) {
	var changed []string
	svc := NewService(Config{
		Repo: repo, Harnesses: registry(exch), EncryptionKey: testEncKey, Tunnels: tunnels,
		Now:                func() time.Time { return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) },
		OnComputersChanged: func(userID string) { changed = append(changed, userID) },
	})
	return svc, &changed
}

func TestService_CreateComputerTunnel_StoresAnUnpairedComputerWithItsTunnel(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	tunnels := &fakeTunnels{}
	svc, changed := newTunnelService(repo, &fakeExchanger{}, tunnels)

	c, err := svc.CreateComputerTunnel(t.Context(), "u1", harness.KindT3Code, "  Laptop ", 3773)
	require.NoError(t, err)
	assert.Equal(t, []string{"Laptop"}, tunnels.created)
	assert.Equal(t, "https://laptop-ab12cd34.example.com", c.ServerURL)
	require.NotNil(t, c.Tunnel)
	assert.Equal(t, "app-1", c.Tunnel.AccessAppID)
	assert.Equal(t, []string{"u1"}, *changed)

	stored, err := repo.GetComputer(t.Context(), "u1", c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.Tunnel, stored.Tunnel)
	assert.Empty(t, stored.BearerToken, "unpaired until the harness pairs over the hostname")
	assert.True(t, stored.TokenExpiresAt.Before(time.Now()), "an unpaired computer never counts as a live session")

	require.Len(t, repo.outbox, 1)
	assert.Equal(t, TopicTunnelCreated, repo.outbox[0].Topic)
	assert.Equal(t, TunnelChangedEvent{ComputerID: c.ID, UserID: "u1", TunnelID: "tun-1", Hostname: "laptop-ab12cd34.example.com"}, repo.outbox[0].Payload)
}

func TestService_CreateComputerTunnel_Rejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		userID  string
		kind    harness.Kind
		compNm  string
		port    int
		tunnels Tunnels
		wantErr error
	}{
		{"missing user", "", harness.KindT3Code, "Laptop", 3773, &fakeTunnels{}, apperrs.ErrUnauthorized},
		{"unknown harness", "u1", "vim", "Laptop", 3773, &fakeTunnels{}, apperrs.ErrInvalid},
		{"blank name", "u1", harness.KindT3Code, " ", 3773, &fakeTunnels{}, apperrs.ErrInvalid},
		{"port zero", "u1", harness.KindT3Code, "Laptop", 0, &fakeTunnels{}, apperrs.ErrInvalid},
		{"port too high", "u1", harness.KindT3Code, "Laptop", 65536, &fakeTunnels{}, apperrs.ErrInvalid},
		{"tunnels not wired", "u1", harness.KindT3Code, "Laptop", 3773, nil, apperrs.ErrFatal},
		{"cloudflare refuses", "u1", harness.KindT3Code, "Laptop", 3773, &fakeTunnels{createErr: apperrs.ErrInvalid}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeRepo()
			svc, _ := newTunnelService(repo, &fakeExchanger{}, tt.tunnels)
			_, err := svc.CreateComputerTunnel(t.Context(), tt.userID, tt.kind, tt.compNm, tt.port)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, repo.computers)
		})
	}
}

func TestService_CreateComputerTunnel_SaveFailure_TearsTheTunnelDown(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	repo.saveErr = errBoom
	tunnels := &fakeTunnels{}
	svc, _ := newTunnelService(repo, &fakeExchanger{}, tunnels)

	_, err := svc.CreateComputerTunnel(t.Context(), "u1", harness.KindT3Code, "Laptop", 3773)
	require.ErrorIs(t, err, errBoom)
	require.Len(t, tunnels.deleted, 1)
	assert.Equal(t, "tun-1", tunnels.deleted[0].TunnelID)
}

// tunnelComputer stores one computer reached through its tunnel and returns its id.
func tunnelComputer(t *testing.T, svc *Service) string {
	t.Helper()
	c, err := svc.CreateComputerTunnel(t.Context(), "u1", harness.KindT3Code, "Laptop", 3773)
	require.NoError(t, err)
	return c.ID
}

func TestService_ComputerTunnelStatus_ProbesTheHarnessOnlyOnceCloudflareSeesTheConnector(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		status        string
		versionErr    error
		wantReachable bool
		wantProbe     bool
	}{
		{"never connected", "inactive", nil, false, false},
		{"connector down", "down", nil, false, false},
		{"healthy and answering", "healthy", nil, true, true},
		{"degraded and answering", "degraded", nil, true, true},
		{"healthy but harness silent", "healthy", apperrs.ErrRetryable, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exch := &fakeExchanger{version: "0.0.40", versionErr: tt.versionErr}
			svc, _ := newTunnelService(newFakeRepo(), exch, &fakeTunnels{status: tt.status})
			id := tunnelComputer(t, svc)

			got, err := svc.ComputerTunnelStatus(t.Context(), "u1", id)
			require.NoError(t, err)
			assert.Equal(t, tt.status, got.Tunnel)
			assert.Equal(t, tt.wantReachable, got.HarnessReachable)
			if tt.wantReachable {
				assert.Equal(t, "0.0.40", got.HarnessVersion)
			}
			if tt.wantProbe {
				assert.Equal(t, "https://laptop-ab12cd34.example.com", exch.probedURL)
				return
			}
			assert.Empty(t, exch.probedURL)
		})
	}
}

func TestService_ComputerTunnelReads_Errors(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	tunnels := &fakeTunnels{status: "healthy"}
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}}
	svc, _ := newTunnelService(repo, exch, tunnels)
	tunnelID := tunnelComputer(t, svc)
	urlPaired, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "VPS", "https://vps.example.com", "tok")
	require.NoError(t, err)

	token, err := svc.ComputerTunnelToken(t.Context(), "u1", tunnelID)
	require.NoError(t, err)
	assert.Equal(t, "connector-token-tun-1", token)

	tests := []struct {
		name       string
		userID     string
		computerID string
		readErr    error
		wantErr    error
	}{
		{"another user's computer", "u2", tunnelID, nil, apperrs.ErrNotFound},
		{"paired by URL", "u1", urlPaired.ID, nil, apperrs.ErrInvalid},
		{"missing id", "u1", " ", nil, apperrs.ErrInvalid},
		{"cloudflare unreachable", "u1", tunnelID, apperrs.ErrRetryable, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tunnels.readErr = tt.readErr
			_, err := svc.ComputerTunnelToken(t.Context(), tt.userID, tt.computerID)
			require.ErrorIs(t, err, tt.wantErr)
			_, err = svc.ComputerTunnelStatus(t.Context(), tt.userID, tt.computerID)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestService_DeleteComputer_TearsItsTunnelDownFirst(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	tunnels := &fakeTunnels{}
	svc, _ := newTunnelService(repo, &fakeExchanger{}, tunnels)
	id := tunnelComputer(t, svc)

	tunnels.deleteErr = errBoom
	require.ErrorIs(t, svc.DeleteComputer(t.Context(), "u1", id), errBoom)
	_, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err, "a failed teardown keeps the computer so removing it again retries")

	tunnels.deleteErr = nil
	require.NoError(t, svc.DeleteComputer(t.Context(), "u1", id))
	require.Len(t, tunnels.deleted, 1)
	assert.Equal(t, "app-1", tunnels.deleted[0].AccessAppID)
	_, err = repo.GetComputer(t.Context(), "u1", id)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.Len(t, repo.outbox, 2)
	assert.Equal(t, TopicTunnelRemoved, repo.outbox[1].Topic)
	assert.Equal(t, id, repo.outbox[1].Payload.(TunnelChangedEvent).ComputerID)
}

func TestService_DeleteComputer_WithTunnelButTunnelsUnwired_KeepsTheComputer(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	withTunnels, _ := newTunnelService(repo, &fakeExchanger{}, &fakeTunnels{})
	id := tunnelComputer(t, withTunnels)
	without, _ := newTunnelService(repo, &fakeExchanger{}, nil)

	require.ErrorIs(t, without.DeleteComputer(t.Context(), "u1", id), apperrs.ErrFatal)
	_, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err)
}

func TestService_Repair_KeepsTheComputersTunnel(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}}
	svc, _ := newTunnelService(repo, exch, &fakeTunnels{})
	id := tunnelComputer(t, svc)

	_, err := svc.Repair(t.Context(), "u1", id, "Laptop", "https://laptop-ab12cd34.example.com", "t3-pair-token")
	require.NoError(t, err)
	stored, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err)
	require.NotNil(t, stored.Tunnel)
	assert.Equal(t, "tun-1", stored.Tunnel.TunnelID)
}
