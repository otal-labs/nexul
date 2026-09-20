package dns

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// newTunnelService wires a Service over fakes with tunnel provider + provisioner.
func newTunnelService(repo *fakeRepo, tunnel *fakeTunnelProvider, prov *fakeProvisioner) *Service {
	cfg := Config{
		Repo:           repo,
		Provider:       newFakeProvider(),
		TunnelProvider: tunnel,
		EncryptionKey:  []byte("0123456789abcdef0123456789abcdef"),
		Settings:       &fakeSettings{instanceURL: "https://deploy.example.com"},
		Tokens:         &fakeTokenProvider{token: "at"},
		Now:            func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) },
	}
	if prov != nil {
		cfg.Provisioner = prov
	}
	return NewService(cfg)
}

func TestService_CreateTunnel(t *testing.T) {
	t.Run("creates tunnel and stores token encrypted", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		got, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "tunnel-1"})
		require.NoError(t, err)
		assert.Equal(t, "tunnel-1", got.ID)
		assert.Equal(t, "tunnel-1", got.Name)
		assert.NotContains(t, got.Token, "tunnel-token", "token must be encrypted at rest")
		stored, err := repo.GetTunnel(context.Background(), "tunnel-1")
		require.NoError(t, err)
		assert.NotContains(t, stored.Token, "tunnel-token-1")
		assert.Contains(t, repo.topics(), TopicTunnelChanged)
	})

	t.Run("same name again returns the stored tunnel without a second provider call", func(t *testing.T) {
		repo := newFakeRepo()
		tunnel := newFakeTunnelProvider()
		s := newTunnelService(repo, tunnel, nil)
		first, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "tunnel-1"})
		require.NoError(t, err)
		tunnel.createErr = errBoom
		again, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "tunnel-1"})
		require.NoError(t, err)
		assert.Equal(t, first.ID, again.ID)
	})

	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		_, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "  "})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("provider error is classified", func(t *testing.T) {
		tunnel := newFakeTunnelProvider()
		tunnel.createErr = apperrs.Retryable(errBoom)
		s := newTunnelService(newFakeRepo(), tunnel, nil)
		_, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "t"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})

	t.Run("no tunnel provider is invalid", func(t *testing.T) {
		cfg := Config{
			Repo:          newFakeRepo(),
			Provider:      newFakeProvider(),
			Tokens:        &fakeTokenProvider{token: "cf-token"},
			EncryptionKey: testKey(),
			Now:           time.Now,
		}
		s := NewService(cfg)
		_, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "t"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestService_ListTunnels(t *testing.T) {
	repo := newFakeRepo()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "a", CreatedAt: time.Now()}))
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t2", Name: "b", CreatedAt: time.Now()}))
	s := newTunnelService(repo, newFakeTunnelProvider(), nil)
	tunnels, err := s.ListTunnels(context.Background())
	require.NoError(t, err)
	assert.Len(t, tunnels, 2)
}

func TestService_GetTunnel(t *testing.T) {
	t.Run("returns stored tunnel", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "a"}))
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		got, err := s.GetTunnel(context.Background(), "t1")
		require.NoError(t, err)
		assert.Equal(t, "t1", got.ID)
	})

	t.Run("missing tunnel is not found", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		_, err := s.GetTunnel(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})

	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		_, err := s.GetTunnel(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestService_RouteTunnelHostname(t *testing.T) {
	t.Run("routes hostname via ingress + CNAME record", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "tunnel-1"}))
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		got, err := s.RouteTunnelHostname(context.Background(), RouteTunnelInput{
			TunnelID: "t1", Hostname: "app.example.com", ZoneID: "z1", Zone: "example.com", Service: "http://localhost:80",
		})
		require.NoError(t, err)
		assert.Equal(t, "app.example.com", got.Hostname)
		assert.Equal(t, "r1", got.RecordID)
		assert.Equal(t, "http://localhost:80", got.Service)
		stored, err := repo.GetTunnel(context.Background(), "t1")
		require.NoError(t, err)
		assert.Equal(t, "app.example.com", stored.Hostname)
		assert.Contains(t, repo.topics(), TopicTunnelChanged)
		assert.Contains(t, repo.topics(), TopicRecordChanged)
	})

	t.Run("the CNAME is proxied and re-routing updates it in place", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "tunnel-1"}))
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		first, err := s.RouteTunnelHostname(context.Background(), RouteTunnelInput{
			TunnelID: "t1", Hostname: "app.example.com", ZoneID: "z1", Zone: "example.com", Service: "http://web:80",
		})
		require.NoError(t, err)
		again, err := s.RouteTunnelHostname(context.Background(), RouteTunnelInput{
			TunnelID: "t1", Hostname: "app2.example.com", ZoneID: "z1", Zone: "example.com", Service: "http://web:80",
		})
		require.NoError(t, err)
		assert.Equal(t, first.RecordID, again.RecordID, "same tunnel keeps one record")
		records, err := s.provider.ListRecords(context.Background(), "z1")
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.True(t, records[0].Proxied, "cfargotunnel CNAMEs must be proxied")
		assert.Equal(t, "app2", records[0].Name)
	})

	t.Run("invalid input rejected", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		_, err := s.RouteTunnelHostname(context.Background(), RouteTunnelInput{TunnelID: "", Hostname: "x"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("ingress failure classified", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1"}))
		tunnel := newFakeTunnelProvider()
		tunnel.routeErr = apperrs.Retryable(errBoom)
		s := newTunnelService(repo, tunnel, nil)
		_, err := s.RouteTunnelHostname(context.Background(), RouteTunnelInput{
			TunnelID: "t1", Hostname: "app.example.com", ZoneID: "z1", Zone: "example.com", Service: "http://localhost:80",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func TestService_RotateTunnelCredentials(t *testing.T) {
	t.Run("rotates token and stores encrypted", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "tunnel-1", Token: "old-enc"}))
		tunnel := newFakeTunnelProvider()
		tunnel.rotated = "brand-new-token"
		s := newTunnelService(repo, tunnel, nil)
		got, err := s.RotateTunnelCredentials(context.Background(), "t1")
		require.NoError(t, err)
		assert.Equal(t, "t1", got.ID)
		stored, err := repo.GetTunnel(context.Background(), "t1")
		require.NoError(t, err)
		assert.NotContains(t, stored.Token, "brand-new-token")
		assert.Contains(t, repo.topics(), TopicTunnelChanged)
	})

	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		_, err := s.RotateTunnelCredentials(context.Background(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("missing tunnel is not found", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		_, err := s.RotateTunnelCredentials(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestService_DeleteTunnel(t *testing.T) {
	t.Run("deletes at provider and locally", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "tunnel-1"}))
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		require.NoError(t, s.DeleteTunnel(context.Background(), "t1"))
		_, err := repo.GetTunnel(context.Background(), "t1")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
		assert.Contains(t, repo.topics(), TopicTunnelChanged)
	})

	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		err := s.DeleteTunnel(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("missing tunnel is not found", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		err := s.DeleteTunnel(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})

	t.Run("provider delete error surfaces", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1"}))
		tunnel := newFakeTunnelProvider()
		tunnel.deleteErr = apperrs.Retryable(errBoom)
		s := newTunnelService(repo, tunnel, nil)
		err := s.DeleteTunnel(context.Background(), "t1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func TestService_CreateTunnel_TokenEncryptionErrors(t *testing.T) {
	t.Run("missing encryption key is invalid", func(t *testing.T) {
		cfg := Config{
			Repo:           newFakeRepo(),
			Provider:       newFakeProvider(),
			TunnelProvider: newFakeTunnelProvider(),
			Settings:       &fakeSettings{instanceURL: "https://deploy.example.com"},
			Now:            time.Now,
		}
		s := NewService(cfg)
		_, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "t"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("constructor resolves from the token provider", func(t *testing.T) {
		cfg := Config{
			Repo:          newFakeRepo(),
			Provider:      newFakeProvider(),
			EncryptionKey: testKey(),
			Settings:      &fakeSettings{instanceURL: "https://deploy.example.com"},
			Tokens:        &fakeTokenProvider{token: "at"},
			NewTunnelProvider: func(_ context.Context, _ string) (TunnelProvider, error) {
				return newFakeTunnelProvider(), nil
			},
			Now: time.Now,
		}
		s := NewService(cfg)
		got, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "t"})
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
	})

	t.Run("constructor error surfaces", func(t *testing.T) {
		cfg := Config{
			Repo:          newFakeRepo(),
			Provider:      newFakeProvider(),
			Tokens:        &fakeTokenProvider{token: "cf-token"},
			EncryptionKey: testKey(),
			Settings:      &fakeSettings{instanceURL: "https://deploy.example.com"},
			NewTunnelProvider: func(_ context.Context, _ string) (TunnelProvider, error) {
				return nil, errBoom
			},
			Now: time.Now,
		}
		s := NewService(cfg)
		_, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "t"})
		require.Error(t, err)
		assert.ErrorIs(t, err, errBoom)
	})
}

// The web's provision call carries no health URL, and the service model requires one; the agent supplies its own.
func TestService_ProvisionTunnelAgent_DefaultsHealthURL(t *testing.T) {
	repo := newFakeRepo()
	key := []byte("0123456789abcdef0123456789abcdef")
	enc, err := encryptTunnelTokenForTest(key, "secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "instance", Token: enc}))
	prov := &fakeProvisioner{}
	s := newTunnelService(repo, newFakeTunnelProvider(), prov)
	_, err = s.ProvisionTunnelAgent(context.Background(), "t1", AgentSpec{ProjectID: "p1", Target: "instance", DockerNetwork: "nexul_default"})
	require.NoError(t, err)
	require.Len(t, prov.calls, 1)
	assert.Equal(t, cloudflaredHealthURL, prov.calls[0].HealthURL)
	assert.Equal(t, "run", prov.calls[0].Strategy)
}

// TunnelStatus reflects the connector state Cloudflare reports, never the stale stored row, and never the token.
func TestService_TunnelStatus_ReadsProviderLive(t *testing.T) {
	repo := newFakeRepo()
	tunnel := newFakeTunnelProvider()
	s := newTunnelService(repo, tunnel, nil)
	created, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "instance"})
	require.NoError(t, err)
	tunnel.mu.Lock()
	tunnel.tunnels[created.ID].Status = "healthy"
	tunnel.mu.Unlock()

	got, err := s.TunnelStatus(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "healthy", got.Status)
	assert.Empty(t, got.Token)
}

func TestService_ProvisionTunnelAgent_DecryptError(t *testing.T) {
	repo := newFakeRepo()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Token: "not-ciphertext"}))
	s := newTunnelService(repo, newFakeTunnelProvider(), &fakeProvisioner{})
	_, err := s.ProvisionTunnelAgent(context.Background(), "t1", AgentSpec{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrFatal), "undecryptable stored token is fatal")
}

func TestRouteTunnelInput_Validate(t *testing.T) {
	valid := RouteTunnelInput{TunnelID: "t", Hostname: "h.example.com", ZoneID: "z", Zone: "example.com", Service: "http://localhost:80"}
	require.NoError(t, valid.Validate())
	for name, mutate := range map[string]func(*RouteTunnelInput){
		"empty tunnel id": func(in *RouteTunnelInput) { in.TunnelID = " " },
		"empty hostname":  func(in *RouteTunnelInput) { in.Hostname = "" },
		"empty zone id":   func(in *RouteTunnelInput) { in.ZoneID = "" },
		"empty zone":      func(in *RouteTunnelInput) { in.Zone = " " },
		"empty service":   func(in *RouteTunnelInput) { in.Service = "" },
	} {
		t.Run(name, func(t *testing.T) {
			in := valid
			mutate(&in)
			require.Error(t, in.Validate())
		})
	}
}

func TestService_ListTunnels_RepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.tunnelErr = errBoom
	s := newTunnelService(repo, newFakeTunnelProvider(), nil)
	_, err := s.ListTunnels(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom)
}

func TestService_ProvisionTunnelAgent(t *testing.T) {
	t.Run("provisions cloudflared with the tunnel token", func(t *testing.T) {
		repo := newFakeRepo()
		key := []byte("0123456789abcdef0123456789abcdef")
		enc, err := encryptTunnelTokenForTest(key, "the-tunnel-secret")
		require.NoError(t, err)
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "tunnel-1", Token: enc}))
		prov := &fakeProvisioner{}
		s := newTunnelService(repo, newFakeTunnelProvider(), prov)
		got, err := s.ProvisionTunnelAgent(context.Background(), "t1", AgentSpec{
			ProjectID: "p1", Target: "10.0.0.1", Name: "cloudflared-tunnel-1",
			Strategy: "run", DockerNetwork: "nexul", HealthURL: "http://localhost:20131/ready",
		})
		require.NoError(t, err)
		assert.Equal(t, "svc-1", got.ServiceID)
		require.Len(t, prov.calls, 1)
		assert.Equal(t, "the-tunnel-secret", prov.calls[0].Env["TUNNEL_TOKEN"], "token is fed to the agent env")
		assert.Equal(t, cloudflaredImage, prov.calls[0].Image, "without an image nothing deploys")
		assert.Equal(t, []string{"tunnel", "--no-autoupdate", "--metrics", "0.0.0.0:20241", "run"}, prov.calls[0].Command, "without tunnel run cloudflared exits")
		assert.Equal(t, "http://localhost:20131/ready", prov.calls[0].HealthURL, "a caller's health URL is kept")
		stored, err := repo.GetTunnel(context.Background(), "t1")
		require.NoError(t, err)
		assert.Equal(t, "svc-1", stored.AgentServiceID)
	})

	t.Run("empty tunnel id is invalid", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), &fakeProvisioner{})
		_, err := s.ProvisionTunnelAgent(context.Background(), "", AgentSpec{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("no provisioner wired is a configuration error", func(t *testing.T) {
		repo := newFakeRepo()
		enc, err := encryptTunnelTokenForTest(testKey(), "the-tunnel-secret")
		require.NoError(t, err)
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Token: enc}))
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		_, err = s.ProvisionTunnelAgent(context.Background(), "t1", AgentSpec{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})

	t.Run("provisioner error surfaces", func(t *testing.T) {
		repo := newFakeRepo()
		enc, err := encryptTunnelTokenForTest(testKey(), "the-tunnel-secret")
		require.NoError(t, err)
		require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Token: enc}))
		prov := &fakeProvisioner{err: errBoom}
		s := newTunnelService(repo, newFakeTunnelProvider(), prov)
		_, err = s.ProvisionTunnelAgent(context.Background(), "t1", AgentSpec{})
		require.Error(t, err)
		assert.ErrorIs(t, err, errBoom)
	})
}

func TestService_ProvisionReverseProxy(t *testing.T) {
	t.Run("provisions reverse proxy service", func(t *testing.T) {
		prov := &fakeProvisioner{}
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), prov)
		got, err := s.ProvisionReverseProxy(context.Background(), AgentSpec{
			ProjectID: "p1", Target: "10.0.0.1", Name: "proxy", Strategy: "run",
			Ports: []string{"80:80", "443:443"}, Image: "traefik:v3", HealthURL: "http://localhost:8080/ping",
		})
		require.NoError(t, err)
		assert.Equal(t, "svc-1", got.ServiceID)
		require.Len(t, prov.calls, 1)
		assert.Equal(t, "proxy", prov.calls[0].Name)
		assert.Equal(t, []string{"80:80", "443:443"}, prov.calls[0].Ports)
	})

	t.Run("default name applied", func(t *testing.T) {
		prov := &fakeProvisioner{}
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), prov)
		_, err := s.ProvisionReverseProxy(context.Background(), AgentSpec{ProjectID: "p1", Target: "10.0.0.1"})
		require.NoError(t, err)
		assert.Equal(t, "reverse-proxy", prov.calls[0].Name)
	})

	t.Run("no provisioner wired is a configuration error", func(t *testing.T) {
		s := newTunnelService(newFakeRepo(), newFakeTunnelProvider(), nil)
		_, err := s.ProvisionReverseProxy(context.Background(), AgentSpec{ProjectID: "p1", Target: "10.0.0.1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

func encryptTunnelTokenForTest(key []byte, token string) (string, error) {
	svc := &Service{key: key}
	return svc.encryptTunnelToken(token)
}

func TestService_DescribeTunnel(t *testing.T) {
	repo := newFakeRepo()
	tunnel := newFakeTunnelProvider()
	prov := newFakeProvider()
	prov.zones = []Zone{{ID: "z1", Name: "example.com"}, {ID: "z2", Name: "other.com"}}
	prov.records["z1"] = []Record{
		{ID: "r1", ZoneID: "z1", Type: RecordCNAME, Name: "app.example.com", Content: "tunnel-1.cfargotunnel.com"},
		{ID: "r2", ZoneID: "z1", Type: RecordCNAME, Name: "elsewhere.example.com", Content: "tunnel-9.cfargotunnel.com"},
		{ID: "r3", ZoneID: "z1", Type: RecordA, Name: "box.example.com", Content: "10.0.0.1"},
	}
	prov.records["z2"] = []Record{
		{ID: "r4", ZoneID: "z2", Type: RecordCNAME, Name: "api.other.com", Content: "tunnel-1.cfargotunnel.com"},
	}
	s := NewService(Config{
		Repo: repo, Provider: prov, TunnelProvider: tunnel,
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
		Settings:      &fakeSettings{instanceURL: "https://deploy.example.com"},
		Tokens:        &fakeTokenProvider{token: "at"},
		Now:           func() time.Time { return time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC) },
	})
	created, err := s.CreateTunnel(context.Background(), CreateTunnelInput{Name: "tunnel-1"})
	require.NoError(t, err)
	require.NoError(t, tunnel.RouteTunnelHostname(context.Background(), created.ID, "app.example.com", "http://web:80"))

	t.Run("a tracked tunnel with routes and records", func(t *testing.T) {
		info, err := s.DescribeTunnel(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "tunnel-1", info.Name)
		assert.True(t, info.Tracked)
		require.Len(t, info.Routes, 1)
		assert.Equal(t, "app.example.com", info.Routes[0].Hostname)
		require.Len(t, info.Records, 2, "only CNAMEs targeting this tunnel, from every zone")
		assert.Equal(t, "app.example.com", info.Records[0].Name)
		assert.Equal(t, "api.other.com", info.Records[1].Name)
	})

	t.Run("a tunnel this instance never created is described but not tracked", func(t *testing.T) {
		foreign, err := tunnel.CreateTunnel(context.Background(), "someone-elses")
		require.NoError(t, err)
		info, err := s.DescribeTunnel(context.Background(), foreign.ID)
		require.NoError(t, err)
		assert.False(t, info.Tracked)
		assert.Empty(t, info.Routes)
		assert.Empty(t, info.Records)
		assert.NotNil(t, info.Routes, "empty lists serialize as [] for the wizard")
	})

	t.Run("an unknown tunnel id is the provider's not-found", func(t *testing.T) {
		_, err := s.DescribeTunnel(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestService_AdoptTunnelGateway(t *testing.T) {
	newAdoptService := func(t *testing.T) (*Service, *fakeRepo, string) {
		t.Helper()
		repo := newFakeRepo()
		tunnel := newFakeTunnelProvider()
		prov := newFakeProvider()
		prov.zones = []Zone{{ID: "z1", Name: "example.com"}}
		s := NewService(Config{
			Repo: repo, Provider: prov, TunnelProvider: tunnel,
			EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
			Settings:      &fakeSettings{instanceURL: "https://deploy.example.com"},
			Tokens:        &fakeTokenProvider{token: "at"},
			Now:           func() time.Time { return time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC) },
		})
		// A tunnel somebody made outside Nexul, already routing two hostnames.
		foreign, err := tunnel.CreateTunnel(context.Background(), "local")
		require.NoError(t, err)
		require.NoError(t, tunnel.RouteTunnelHostname(context.Background(), foreign.ID, "api.example.com", "http://hello-api:3000"))
		require.NoError(t, tunnel.RouteTunnelHostname(context.Background(), foreign.ID, "web.example.com", "http://web:5173"))
		prov.records["z1"] = []Record{
			{ID: "r1", ZoneID: "z1", Type: RecordCNAME, Name: "api.example.com", Content: foreign.ID + ".cfargotunnel.com"},
		}
		return s, repo, foreign.ID
	}
	input := func(tunnelID string) AdoptTunnelGatewayInput {
		return AdoptTunnelGatewayInput{
			TunnelID: tunnelID, Machine: "prod", ServiceID: "svc-cf", ServiceName: "cloudflared-local",
			Networks: []string{"app_default", "other_default"},
			Targets:  map[string]string{"hello-api": "svc-api"},
		}
	}

	t.Run("tracks the tunnel, creates the gateway, exposes matched routes, reports the rest", func(t *testing.T) {
		s, repo, tunnelID := newAdoptService(t)
		got, err := s.AdoptTunnelGateway(context.Background(), input(tunnelID))
		require.NoError(t, err)

		stored, err := repo.GetTunnel(context.Background(), tunnelID)
		require.NoError(t, err, "the foreign tunnel is now tracked")
		assert.NotEmpty(t, stored.Token)
		assert.NotEqual(t, "tunnel-token", stored.Token, "token stored encrypted")

		assert.Equal(t, GatewayTunnel, got.Gateway.Kind)
		assert.Equal(t, "app_default", got.Gateway.DockerNetwork)
		assert.Equal(t, []string{"app_default", "other_default"}, got.Gateway.Networks)
		assert.Equal(t, "svc-cf", got.Gateway.ServiceID)
		assert.Equal(t, tunnelID, got.Gateway.TunnelID)
		assert.Equal(t, []string{"api.example.com"}, got.Exposed)
		assert.Equal(t, []string{"web.example.com"}, got.Unmatched, "a route to a container Nexul does not track is reported, not exposed")

		exposures, err := repo.ListExposuresByGateway(context.Background(), got.Gateway.ID)
		require.NoError(t, err)
		require.Len(t, exposures, 1)
		assert.Equal(t, "svc-api", exposures[0].ServiceID)
		assert.Equal(t, 3000, exposures[0].Port)
		assert.Equal(t, "example.com", exposures[0].Zone)
		assert.Equal(t, "r1", exposures[0].RecordID, "the existing CNAME is linked, never recreated")
	})

	t.Run("adopting again reuses the gateway and adds no duplicate exposures", func(t *testing.T) {
		s, repo, tunnelID := newAdoptService(t)
		first, err := s.AdoptTunnelGateway(context.Background(), input(tunnelID))
		require.NoError(t, err)
		second, err := s.AdoptTunnelGateway(context.Background(), input(tunnelID))
		require.NoError(t, err)
		assert.Equal(t, first.Gateway.ID, second.Gateway.ID)
		assert.Equal(t, []string{"api.example.com"}, second.Exposed)
		exposures, err := repo.ListExposuresByGateway(context.Background(), first.Gateway.ID)
		require.NoError(t, err)
		assert.Len(t, exposures, 1)
		gateways, err := repo.ListGateways(context.Background())
		require.NoError(t, err)
		assert.Len(t, gateways, 1)
	})

	t.Run("a network already homing another tunnel's gateway conflicts", func(t *testing.T) {
		s, repo, tunnelID := newAdoptService(t)
		require.NoError(t, repo.SaveGateway(context.Background(), Gateway{ID: "gw-x", Kind: GatewayTunnel, DockerNetwork: "app_default", Networks: []string{"app_default"}, Machine: "prod", TunnelID: "some-other-tunnel"}))
		_, err := s.AdoptTunnelGateway(context.Background(), input(tunnelID))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
}

func TestRouteTarget(t *testing.T) {
	host, port := routeTarget("http://hello-api:3000")
	assert.Equal(t, "hello-api", host)
	assert.Equal(t, 3000, port)
	host, port = routeTarget("https://web")
	assert.Equal(t, "web", host)
	assert.Equal(t, 443, port)
	host, port = routeTarget("http_status:404")
	assert.Equal(t, "", host)
	assert.Equal(t, 0, port)
}
