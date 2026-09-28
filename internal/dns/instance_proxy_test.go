package dns

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const testDomain = "nexul.example.com"

// newInstanceProxyService wires the instance proxy use-case over fakes: the bundled machine is host1 and the
// domain resolves to one IPv6 and one IPv4 address.
func newInstanceProxyService(repo *fakeRepo, prov *fakeProvisioner) *Service {
	return NewService(Config{
		Repo: repo, Provisioner: prov, Settings: &fakeSettings{},
		InstanceOrigin: testOrigin,
		Placement:      fakePlacement{machine: "host1"},
		Resolver:       fakeResolver{addrs: map[string][]string{testDomain: {"2001:db8::1", "203.0.113.10"}}},
	})
}

func TestService_ProvisionInstanceProxy_Errors(t *testing.T) {
	tests := []struct {
		name  string
		in    InstanceProxyInput
		setup func(*fakeRepo, *fakeProvisioner) *Service
		want  error
	}{
		{name: "an empty domain is invalid", in: InstanceProxyInput{}, want: apperrs.ErrInvalid},
		{name: "a single label is invalid", in: InstanceProxyInput{Domain: "localhost"}, want: apperrs.ErrInvalid},
		{name: "a URL is invalid", in: InstanceProxyInput{Domain: "https://nexul.example.com"}, want: apperrs.ErrInvalid},
		{name: "an empty label is invalid", in: InstanceProxyInput{Domain: "nexul..example.com"}, want: apperrs.ErrInvalid},
		{name: "a leading dash is invalid", in: InstanceProxyInput{Domain: "-nexul.example.com"}, want: apperrs.ErrInvalid},
		{name: "an underscore is invalid", in: InstanceProxyInput{Domain: "nex_ul.example.com"}, want: apperrs.ErrInvalid},
		{name: "a malformed email is invalid", in: InstanceProxyInput{Domain: testDomain, Email: "owner"}, want: apperrs.ErrInvalid},
		{name: "no machine and no placement is invalid", in: InstanceProxyInput{Domain: testDomain},
			setup: func(r *fakeRepo, p *fakeProvisioner) *Service {
				return NewService(Config{Repo: r, Provisioner: p, InstanceOrigin: testOrigin, Resolver: fakeResolver{}})
			}, want: apperrs.ErrInvalid},
		{name: "an instance runner that never connected surfaces", in: InstanceProxyInput{Domain: testDomain},
			setup: func(r *fakeRepo, p *fakeProvisioner) *Service {
				return NewService(Config{Repo: r, Provisioner: p, InstanceOrigin: testOrigin, Resolver: fakeResolver{},
					Placement: fakePlacement{err: apperrs.ErrInvalid}})
			}, want: apperrs.ErrInvalid},
		{name: "no container address for the server is a configuration error", in: InstanceProxyInput{Domain: testDomain},
			setup: func(r *fakeRepo, p *fakeProvisioner) *Service {
				return NewService(Config{Repo: r, Provisioner: p, Resolver: fakeResolver{}, Placement: fakePlacement{machine: "host1"}})
			}, want: apperrs.ErrFatal},
		{name: "a network another gateway homes on conflicts", in: InstanceProxyInput{Domain: testDomain},
			setup: func(r *fakeRepo, p *fakeProvisioner) *Service {
				r.gateways["g0"] = &Gateway{ID: "g0", Kind: GatewayTunnel, DockerNetwork: instanceProxyNetwork, Machine: "host2"}
				return newInstanceProxyService(r, p)
			}, want: apperrs.ErrConflict},
		{name: "a failed gateway read surfaces", in: InstanceProxyInput{Domain: testDomain},
			setup: func(r *fakeRepo, p *fakeProvisioner) *Service {
				r.gatewayErr = errBoom
				return newInstanceProxyService(r, p)
			}, want: errBoom},
		{name: "a failed deploy surfaces", in: InstanceProxyInput{Domain: testDomain},
			setup: func(r *fakeRepo, p *fakeProvisioner) *Service {
				p.err = errBoom
				return newInstanceProxyService(r, p)
			}, want: errBoom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, prov := newFakeRepo(), &fakeProvisioner{}
			s := newInstanceProxyService(repo, prov)
			if tt.setup != nil {
				s = tt.setup(repo, prov)
			}
			_, err := s.ProvisionInstanceProxy(t.Context(), tt.in)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestService_ProvisionInstanceProxy_DeploysTraefikWithTheInstanceRoute(t *testing.T) {
	repo, prov := newFakeRepo(), &fakeProvisioner{}
	s := newInstanceProxyService(repo, prov)

	g, err := s.ProvisionInstanceProxy(t.Context(), InstanceProxyInput{Domain: " Nexul.Example.com. ", Email: "owner@example.com"})
	require.NoError(t, err)

	require.Len(t, prov.calls, 1)
	assert.Equal(t, AgentSpec{
		Target: "host1", Name: "nexul-proxy", Strategy: "run", DockerNetwork: "nexul_proxy",
		Image:  "traefik:v3",
		Ports:  []string{"80:80", "443:443"},
		Mounts: []string{"/var/run/docker.sock:/var/run/docker.sock:ro", "nexul-traefik-acme:/letsencrypt"},
		Command: []string{"sh", "-c",
			`mkdir -p /etc/traefik && printf '%s' "$NEXUL_TRAEFIK_DYNAMIC" > /etc/traefik/nexul.yml && exec traefik`},
		Env: map[string]string{
			"TRAEFIK_PROVIDERS_DOCKER":                                                "true",
			"TRAEFIK_PROVIDERS_DOCKER_EXPOSEDBYDEFAULT":                               "false",
			"TRAEFIK_PROVIDERS_FILE_FILENAME":                                         "/etc/traefik/nexul.yml",
			"TRAEFIK_ENTRYPOINTS_WEB_ADDRESS":                                         ":80",
			"TRAEFIK_ENTRYPOINTS_WEB_HTTP_REDIRECTIONS_ENTRYPOINT_TO":                 "websecure",
			"TRAEFIK_ENTRYPOINTS_WEB_HTTP_REDIRECTIONS_ENTRYPOINT_SCHEME":             "https",
			"TRAEFIK_ENTRYPOINTS_WEBSECURE_ADDRESS":                                   ":443",
			"TRAEFIK_ENTRYPOINTS_WEBSECURE_HTTP_TLS_CERTRESOLVER":                     "letsencrypt",
			"TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_STORAGE":                  "/letsencrypt/acme.json",
			"TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_HTTPCHALLENGE_ENTRYPOINT": "web",
			"TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_EMAIL":                    "owner@example.com",
			"NEXUL_TRAEFIK_DYNAMIC": `{"http":{"routers":{"nexul":{"entryPoints":["websecure"],"rule":"Host(` +
				"`nexul.example.com`" + `)","service":"nexul","tls":{"certResolver":"letsencrypt"}}},` +
				`"services":{"nexul":{"loadBalancer":{"servers":[{"url":"http://host.docker.internal:5123"}]}}}}}`,
		},
	}, prov.calls[0])

	assert.Equal(t, GatewayProxy, g.Kind)
	assert.Equal(t, "host1", g.Machine)
	assert.Equal(t, "svc-1", g.ServiceID)
	assert.Equal(t, "203.0.113.10", g.ServerAddress, "exposure records point at the domain's IPv4 address first")
	stored, err := repo.GetGateway(t.Context(), g.ID)
	require.NoError(t, err)
	assert.Equal(t, "svc-1", stored.ServiceID)
	assert.Equal(t, []string{TopicGatewayChanged}, repo.topics())
}

func TestService_ProvisionInstanceProxy_RetryReusesTheGateway(t *testing.T) {
	repo, prov := newFakeRepo(), &fakeProvisioner{result: &AgentProvisioned{ServiceID: "stack-1"}}
	s := newInstanceProxyService(repo, prov)

	first, err := s.ProvisionInstanceProxy(t.Context(), InstanceProxyInput{Domain: testDomain})
	require.NoError(t, err)
	second, err := s.ProvisionInstanceProxy(t.Context(), InstanceProxyInput{Domain: testDomain})
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "stack-1", second.ServiceID)
	require.Len(t, prov.calls, 2, "a retry redeploys")
	assert.Equal(t, prov.calls[0].Name, prov.calls[1].Name)
	assert.NotContains(t, prov.calls[0].Env, "TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_EMAIL", "email is optional")
	assert.Len(t, repo.gateways, 1)
	assert.Equal(t, []string{TopicGatewayChanged}, repo.topics(), "only the first call creates the gateway")
}

func TestService_ProvisionInstanceProxy_ExtendsTheMachinesProxyGateway(t *testing.T) {
	repo, prov := newFakeRepo(), &fakeProvisioner{}
	repo.gateways["g1"] = &Gateway{
		ID: "g1", Kind: GatewayProxy, DockerNetwork: "shop_default", Networks: []string{"shop_default"}, Machine: "host1",
		ServiceID: "stack-9", ServiceName: "traefik-abc", ServerAddress: "198.51.100.7",
	}
	s := newInstanceProxyService(repo, prov)

	g, err := s.ProvisionInstanceProxy(t.Context(), InstanceProxyInput{Domain: testDomain})
	require.NoError(t, err)

	assert.Equal(t, "g1", g.ID)
	require.Len(t, prov.calls, 1)
	assert.Equal(t, "traefik-abc", prov.calls[0].Name, "one Traefik per machine, never a second on ports 80 and 443")
	assert.Equal(t, "shop_default", prov.calls[0].DockerNetwork)
	assert.Contains(t, prov.calls[0].Env[dynamicConfigEnv], "Host(`nexul.example.com`)")
	assert.Equal(t, "198.51.100.7", g.ServerAddress, "an existing server address is kept")
	assert.Empty(t, repo.topics())
}

func TestService_ResolveHost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		resolver fakeResolver
		want     []string
		wantErr  error
	}{
		{name: "an invalid host is invalid", host: "not a host", wantErr: apperrs.ErrInvalid},
		{name: "a lookup failure surfaces", host: testDomain, resolver: fakeResolver{err: errBoom}, wantErr: errBoom},
		{name: "a name that does not resolve yet is empty", host: testDomain, want: []string{}},
		{name: "addresses come back sorted without duplicates", host: "Nexul.example.com",
			resolver: fakeResolver{addrs: map[string][]string{testDomain: {"203.0.113.10", "2001:db8::1", "203.0.113.10"}}},
			want:     []string{"2001:db8::1", "203.0.113.10"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(Config{Repo: newFakeRepo(), Resolver: tt.resolver})
			got, err := s.ResolveHost(t.Context(), tt.host)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHandler_InstanceProxyAndResolve(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     any
		wantCode int
		wantBody string
	}{
		{name: "a non-object body is a bad request", method: http.MethodPost, path: "/api/dns/instance-proxy", body: "x", wantCode: http.StatusBadRequest},
		{name: "an invalid domain is a bad request", method: http.MethodPost, path: "/api/dns/instance-proxy",
			body: map[string]string{"domain": "localhost"}, wantCode: http.StatusBadRequest},
		{name: "a deploy answers with the stack to watch", method: http.MethodPost, path: "/api/dns/instance-proxy",
			body: map[string]string{"domain": testDomain}, wantCode: http.StatusCreated, wantBody: `{"service_id":"svc-1"}`},
		{name: "resolve without a host is a bad request", method: http.MethodGet, path: "/api/dns/resolve", wantCode: http.StatusBadRequest},
		{name: "resolve lists the addresses", method: http.MethodGet, path: "/api/dns/resolve?host=" + testDomain,
			wantCode: http.StatusOK, wantBody: `{"addresses":["2001:db8::1","203.0.113.10"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(newInstanceProxyService(newFakeRepo(), &fakeProvisioner{}))
			rec := doJSON(t, h.Routes(), tt.method, tt.path, tt.body)
			require.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, rec.Body.String())
			}
		})
	}
}

func TestGatewayCreateTool_InstanceDomain(t *testing.T) {
	t.Run("kind tunnel with an instance domain is invalid", func(t *testing.T) {
		_, err := newToolFakes(t).call(t, "gateway_create", `{"kind":"tunnel","instance_domain":"nexul.example.com"}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("an invalid instance domain is invalid", func(t *testing.T) {
		_, err := newToolFakes(t).call(t, "gateway_create", `{"kind":"proxy","instance_domain":"localhost"}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("routes the domain through the instance machine's proxy gateway", func(t *testing.T) {
		f := newToolFakes(t)
		got, err := f.call(t, "gateway_create", `{"kind":"proxy","instance_domain":"nexul.example.com","email":"owner@example.com"}`)
		require.NoError(t, err)
		var res gatewayResult
		require.NoError(t, json.Unmarshal([]byte(asJSON(t, got)), &res))
		assert.Equal(t, GatewayProxy, res.Kind)
		assert.Equal(t, "host1", res.Machine)
		assert.Equal(t, "nexul_proxy", res.DockerNetwork)
		assert.Equal(t, "svc-1", res.StackID)
		assert.Equal(t, "203.0.113.10", res.ServerAddress)
		require.Len(t, f.prov.calls, 1)
		assert.Equal(t, "owner@example.com", f.prov.calls[0].Env["TRAEFIK_CERTIFICATESRESOLVERS_LETSENCRYPT_ACME_EMAIL"])
	})
	t.Run("a proxy gateway without an instance domain still needs its zone", func(t *testing.T) {
		_, err := newToolFakes(t).call(t, "gateway_create", `{"kind":"proxy","machine":"host1","docker_network":"n","server_address":"1.2.3.4"}`)
		require.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}
