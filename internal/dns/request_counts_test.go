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

func TestTraefikSpec_ServesRouterMetricsOnAnUnpublishedPort(t *testing.T) {
	spec := traefikSpec("", "", "")

	assert.Equal(t, ":8082", spec.Env["TRAEFIK_ENTRYPOINTS_METRICS_ADDRESS"])
	assert.Equal(t, "true", spec.Env["TRAEFIK_METRICS_PROMETHEUS"])
	assert.Equal(t, "true", spec.Env["TRAEFIK_METRICS_PROMETHEUS_ADDROUTERSLABELS"])
	assert.Equal(t, "metrics", spec.Env["TRAEFIK_METRICS_PROMETHEUS_ENTRYPOINT"])
	assert.Equal(t, []string{"80:80", "443:443"}, spec.Ports, "the metrics port is never published")
}

func TestService_RequestRoutes_ListsEveryExposureWithItsGatewaysKindAndMachine(t *testing.T) {
	s, repo, _, _, tunnelGW := newExposureFixture(t)
	proxyGW := &Gateway{ID: "gw-proxy", Kind: GatewayProxy, DockerNetwork: "net2", Machine: "host2"}
	require.NoError(t, repo.SaveGateway(t.Context(), *proxyGW))
	for _, e := range []Exposure{
		{ID: "e-tunnel", GatewayID: tunnelGW.ID, Hostname: "app.example.com", ServiceID: "c-app", ZoneID: "z1"},
		{ID: "e-proxy", GatewayID: "gw-proxy", Hostname: "other.example.com", ServiceID: "c-other", ZoneID: "z2"},
		{ID: "e-legacy", GatewayID: tunnelGW.ID, Hostname: "old.example.com", Service: "old"},
		{ID: "e-orphan", GatewayID: "gone", Hostname: "x.example.com", ServiceID: "c-x"},
	} {
		require.NoError(t, repo.SaveExposure(t.Context(), e))
	}

	routes, err := s.RequestRoutes(t.Context())

	require.NoError(t, err)
	assert.ElementsMatch(t, []RequestRoute{
		{Router: "nexul-e-tunnel", Hostname: "app.example.com", ContainerID: "c-app", ZoneID: "z1", Machine: "host1", Tunnel: true},
		{Router: "nexul-e-proxy", Hostname: "other.example.com", ContainerID: "c-other", ZoneID: "z2", Machine: "host2"},
	}, routes, "an exposure with no container or no gateway reaches nothing to count")
}

// fakeAnalytics adds request counts to the fake DNS provider.
type fakeAnalytics struct {
	*fakeProvider
	counts []HostMinute
	err    error
	zone   string
	hosts  []string
	from   time.Time
	to     time.Time
}

func (f *fakeAnalytics) HostnameRequests(_ context.Context, zoneID string, hostnames []string, from, to time.Time) ([]HostMinute, error) {
	f.zone, f.hosts, f.from, f.to = zoneID, hostnames, from, to
	return f.counts, f.err
}

func TestService_HostnameRequests_AsksTheProviderThatReadsAnalytics(t *testing.T) {
	counts := []HostMinute{{Hostname: "app.example.com", Minute: time.Date(2026, 10, 16, 10, 39, 0, 0, time.UTC), Requests: 5, Errors: 1}}
	provider := &fakeAnalytics{fakeProvider: newFakeProvider(), counts: counts}
	s := NewService(Config{Repo: newFakeRepo(), Provider: provider, Tokens: &fakeTokenProvider{token: "at"}})
	from := time.Date(2026, 10, 16, 10, 39, 0, 0, time.UTC)

	got, err := s.HostnameRequests(t.Context(), "z1", []string{"app.example.com"}, from, from.Add(time.Minute))

	require.NoError(t, err)
	assert.Equal(t, counts, got)
	assert.Equal(t, "z1", provider.zone)
	assert.Equal(t, []string{"app.example.com"}, provider.hosts)
	assert.Equal(t, from, provider.from)
}

func TestService_HostnameRequests_Failures(t *testing.T) {
	from := time.Now()
	t.Run("a provider that reads no analytics", func(t *testing.T) {
		s := NewService(Config{Repo: newFakeRepo(), Provider: newFakeProvider(), Tokens: &fakeTokenProvider{token: "at"}})
		_, err := s.HostnameRequests(t.Context(), "z1", nil, from, from)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("a token that cannot read them", func(t *testing.T) {
		denied := fmtInvalid("the Cloudflare token can't read analytics")
		s := NewService(Config{Repo: newFakeRepo(), Provider: &fakeAnalytics{fakeProvider: newFakeProvider(), err: denied}, Tokens: &fakeTokenProvider{token: "at"}})
		_, err := s.HostnameRequests(t.Context(), "z1", nil, from, from)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("Cloudflare not connected", func(t *testing.T) {
		s := NewService(Config{Repo: newFakeRepo(), Tokens: &fakeTokenProvider{err: apperrs.ErrNotFound}})
		_, err := s.HostnameRequests(t.Context(), "z1", nil, from, from)
		assert.Error(t, err)
	})
}

func fmtInvalid(msg string) error { return errors.Join(apperrs.ErrInvalid, errors.New(msg)) }
