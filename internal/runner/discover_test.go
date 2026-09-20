package runner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeDiscoverCmd answers docker commands keyed by "<name> <args...>"; unmatched keys return empty output.
type fakeDiscoverCmd struct {
	mu      sync.Mutex
	calls   []string
	outputs map[string]string
	errs    map[string]error
}

func newFakeDiscoverCmd() *fakeDiscoverCmd {
	return &fakeDiscoverCmd{outputs: map[string]string{}, errs: map[string]error{}}
}

func (f *fakeDiscoverCmd) key(name string, args []string) string {
	return name + " " + strings.Join(args, " ")
}

func (f *fakeDiscoverCmd) on(name string, args []string, output string) {
	f.outputs[f.key(name, args)] = output
}

func (f *fakeDiscoverCmd) failOn(name string, args []string, err error) {
	f.errs[f.key(name, args)] = err
}

func (f *fakeDiscoverCmd) run(ctx context.Context, _ string, logf func(string), name string, args ...string) error {
	key := f.key(name, args)
	f.mu.Lock()
	f.calls = append(f.calls, key)
	err := f.errs[key]
	out, ok := f.outputs[key]
	f.mu.Unlock()
	if err != nil {
		return err
	}
	if ok {
		logf(out)
	}
	return nil
}

func TestDiscoverHost_ExcludesSelfAndItsComposeProject(t *testing.T) {
	cmd := newFakeDiscoverCmd()
	cmd.on("docker", []string{"ps", "-a", "--format", "json"},
		`{"ID":"selfid1234"}`+"\n"+`{"ID":"sibling5678"}`+"\n"+`{"ID":"web9999"}`+"\n")
	cmd.on("docker", []string{"inspect", "--format", "{{json .}}", "selfid1234"},
		`{"Id":"selfid1234deadbeef","Name":"/runner","Config":{"Image":"nexul-runner","Labels":{"com.docker.compose.project":"nexul","com.docker.compose.service":"runner"}},"State":{"Status":"running"},"NetworkSettings":{"Networks":{},"Ports":{}}}`)
	cmd.on("docker", []string{"inspect", "--format", "{{json .}}", "sibling5678"},
		`{"Id":"sibling5678","Name":"/server","Config":{"Image":"nexul-server","Labels":{"com.docker.compose.project":"nexul","com.docker.compose.service":"server"}},"State":{"Status":"running"},"NetworkSettings":{"Networks":{},"Ports":{}}}`)
	cmd.on("docker", []string{"inspect", "--format", "{{json .}}", "web9999"},
		`{"Id":"web9999","Name":"/myapp-web-1","Config":{"Image":"nginx:latest","Labels":{"com.docker.compose.project":"myapp","com.docker.compose.service":"web"}},"State":{"Status":"running"},"NetworkSettings":{"Networks":{"myapp_default":{"IPAddress":"172.20.0.2"}},"Ports":{"80/tcp":[{"HostIp":"0.0.0.0","HostPort":"8080"}]}}}`)
	cmd.on("docker", []string{"network", "ls", "--format", "json"},
		`{"Name":"bridge"}`+"\n"+`{"Name":"myapp_default"}`+"\n"+`{"Name":"host"}`+"\n")

	report, err := discoverHost(context.Background(), cmd.run, "selfid1234")
	require.NoError(t, err)

	require.Len(t, report.Containers, 1, "the runner's own container and its compose siblings are excluded")
	c := report.Containers[0]
	assert.Equal(t, "myapp-web-1", c.Name)
	assert.Equal(t, "nginx:latest", c.Image)
	assert.Equal(t, "running", c.Status)
	assert.Equal(t, "myapp", c.Labels[composeProjectLabel])
	require.Len(t, c.Networks, 1)
	assert.Equal(t, "myapp_default", c.Networks[0].Name)
	assert.Equal(t, "172.20.0.2", c.Networks[0].Address)
	assert.Equal(t, []string{"8080:80/tcp"}, c.Ports)

	require.Len(t, report.Networks, 1, "builtin networks are excluded")
	assert.Equal(t, "myapp_default", report.Networks[0].Name)
}

func TestDiscoverHost_NoSelfMatch_ReturnsEverything(t *testing.T) {
	cmd := newFakeDiscoverCmd()
	cmd.on("docker", []string{"ps", "-a", "--format", "json"}, `{"ID":"c1"}`+"\n")
	cmd.on("docker", []string{"inspect", "--format", "{{json .}}", "c1"},
		`{"Id":"c1","Name":"/standalone","Config":{"Image":"redis","Labels":{}},"State":{"Status":"running"},"NetworkSettings":{"Networks":{},"Ports":{}}}`)
	cmd.on("docker", []string{"network", "ls", "--format", "json"}, "")

	report, err := discoverHost(context.Background(), cmd.run, "")
	require.NoError(t, err)
	require.Len(t, report.Containers, 1)
	assert.Equal(t, "standalone", report.Containers[0].Name)
}

func TestDiscoverHost_SkipsContainerThatVanishesBeforeInspect(t *testing.T) {
	cmd := newFakeDiscoverCmd()
	cmd.on("docker", []string{"ps", "-a", "--format", "json"}, `{"ID":"gone"}`+"\n"+`{"ID":"c1"}`+"\n")
	cmd.failOn("docker", []string{"inspect", "--format", "{{json .}}", "gone"}, fmt.Errorf("no such container"))
	cmd.on("docker", []string{"inspect", "--format", "{{json .}}", "c1"},
		`{"Id":"c1","Name":"/ok","Config":{"Image":"redis","Labels":{}},"State":{"Status":"running"},"NetworkSettings":{"Networks":{},"Ports":{}}}`)
	cmd.on("docker", []string{"network", "ls", "--format", "json"}, "")

	report, err := discoverHost(context.Background(), cmd.run, "")
	require.NoError(t, err)
	require.Len(t, report.Containers, 1)
	assert.Equal(t, "ok", report.Containers[0].Name)
}

func TestDiscoverHost_PsFailure_Errors(t *testing.T) {
	cmd := newFakeDiscoverCmd()
	cmd.failOn("docker", []string{"ps", "-a", "--format", "json"}, fmt.Errorf("docker not found"))
	_, err := discoverHost(context.Background(), cmd.run, "")
	require.Error(t, err)
}

func TestDiscoverHost_NetworkLsFailure_Errors(t *testing.T) {
	cmd := newFakeDiscoverCmd()
	cmd.on("docker", []string{"ps", "-a", "--format", "json"}, "")
	cmd.failOn("docker", []string{"network", "ls", "--format", "json"}, fmt.Errorf("docker daemon unreachable"))
	_, err := discoverHost(context.Background(), cmd.run, "")
	require.Error(t, err)
}

func TestGroupDiscovery_GroupsByProjectThenStandaloneThenGateways(t *testing.T) {
	report := DiscoverReport{
		Containers: []DiscoveredContainer{
			{Name: "web", Image: "nginx", Labels: map[string]string{composeProjectLabel: "myapp"}},
			{Name: "db", Image: "postgres", Labels: map[string]string{composeProjectLabel: "myapp"}},
			{Name: "redis-standalone", Image: "redis"},
			{Name: "cf", Image: "cloudflare/cloudflared:latest"},
			{Name: "tr", Image: "traefik:v3"},
			{Name: "api", Image: "myorg/api", Labels: map[string]string{composeProjectLabel: "another"}},
		},
		Networks: []NetworkInfo{{Name: "myapp_default"}},
	}

	grouped := GroupDiscovery(report)
	require.Len(t, grouped.Stacks, 2)
	assert.Equal(t, "another", grouped.Stacks[0].Project, "stacks are sorted by project name")
	assert.Equal(t, "myapp", grouped.Stacks[1].Project)
	require.Len(t, grouped.Stacks[1].Containers, 2)

	require.Len(t, grouped.Standalone, 1)
	assert.Equal(t, "redis-standalone", grouped.Standalone[0].Name)

	require.Len(t, grouped.Gateways, 2)
	assert.Equal(t, []NetworkInfo{{Name: "myapp_default"}}, grouped.Networks)
}

func TestHandler_Discover_NoIdleRunner(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	_, err := h.Discover(context.Background(), "prod", time.Second)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrConflict))
}

// startDiscoverTestClient connects a client on machine, executing exec's Discover on a discover frame.
func startDiscoverTestClient(t *testing.T, srvURL, machine string, exec Executor) {
	t.Helper()
	client := NewClient(ClientConfig{
		URL: srvURL, Token: "s3cr3t", RunnerID: "r-disc-" + machine, Name: "disc", Machine: machine,
		Logger: testLogger(), Executor: exec,
		HeartbeatInterval: 20 * time.Millisecond, ConnectTimeout: time.Second,
		BackoffBase: 5 * time.Millisecond, BackoffMax: 50 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = client.Run(ctx) }()
}

func TestHandler_Discover_EndToEnd_Success(t *testing.T) {
	bus := newFakeBus()
	h, srv := newIntegrationHandler(t, bus, newFakeRunnerRepo())
	exec := &fakeExecutor{discoverFn: func(context.Context) (DiscoverReport, error) {
		return DiscoverReport{
			Containers: []DiscoveredContainer{{Name: "web", Image: "nginx"}},
			Networks:   []NetworkInfo{{Name: "app-net"}},
		}, nil
	}}
	startDiscoverTestClient(t, wsURL(srv), "prod", exec)
	eventually(t, time.Second, func() bool { return len(h.Runners()) == 1 })

	report, err := h.Discover(context.Background(), "prod", time.Second)
	require.NoError(t, err)
	require.Len(t, report.Containers, 1)
	assert.Equal(t, "web", report.Containers[0].Name)
	require.Len(t, report.Networks, 1)
	assert.Equal(t, "app-net", report.Networks[0].Name)
}

func TestHandler_Discover_Timeout(t *testing.T) {
	bus := newFakeBus()
	h, srv := newIntegrationHandler(t, bus, newFakeRunnerRepo())
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	exec := &fakeExecutor{discoverFn: func(ctx context.Context) (DiscoverReport, error) {
		select {
		case <-block:
		case <-ctx.Done():
		}
		return DiscoverReport{}, nil
	}}
	startDiscoverTestClient(t, wsURL(srv), "prod", exec)
	eventually(t, time.Second, func() bool { return len(h.Runners()) == 1 })

	_, err := h.Discover(context.Background(), "prod", 50*time.Millisecond)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestHandler_Discover_WrongMachine_NoIdleRunner(t *testing.T) {
	bus := newFakeBus()
	h, srv := newIntegrationHandler(t, bus, newFakeRunnerRepo())
	exec := &fakeExecutor{}
	startDiscoverTestClient(t, wsURL(srv), "staging", exec)
	eventually(t, time.Second, func() bool { return len(h.Runners()) == 1 })

	_, err := h.Discover(context.Background(), "prod", 50*time.Millisecond)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrConflict))
}

// A cloudflared TUNNEL_TOKEN is base64 JSON {a, t, s}; only t may leave the machine.
func cloudflaredToken(t *testing.T, tunnelID string) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString([]byte(`{"a":"acct1","t":"` + tunnelID + `","s":"sup3rs3cret"}`))
}

func TestDiscoverHost_CloudflaredCarriesTunnelIDNeverTheSecret(t *testing.T) {
	cmd := newFakeDiscoverCmd()
	cmd.on("docker", []string{"ps", "-a", "--format", "json"}, `{"ID":"cf1"}`+"\n")
	cmd.on("docker", []string{"inspect", "--format", "{{json .}}", "cf1"},
		`{"Id":"cf1","Name":"/cloudflared-local","Config":{"Image":"cloudflare/cloudflared:latest","Labels":{},"Env":["PATH=/bin","TUNNEL_TOKEN=`+cloudflaredToken(t, "tun-123")+`"]},"State":{"Status":"running"},"NetworkSettings":{"Networks":{},"Ports":{}}}`)
	cmd.on("docker", []string{"network", "ls", "--format", "json"}, "")

	report, err := discoverHost(context.Background(), cmd.run, "")
	require.NoError(t, err)
	require.Len(t, report.Containers, 1)
	assert.Equal(t, "tun-123", report.Containers[0].TunnelID)
	wire, err := json.Marshal(report)
	require.NoError(t, err)
	assert.NotContains(t, string(wire), "sup3rs3cret")
	assert.NotContains(t, string(wire), "acct1")
}

func TestTunnelIDFromEnv(t *testing.T) {
	assert.Equal(t, "", tunnelIDFromEnv([]string{"PATH=/bin"}), "no token")
	assert.Equal(t, "", tunnelIDFromEnv([]string{"TUNNEL_TOKEN=not-base64!"}), "garbage token")
	assert.Equal(t, "tun-1", tunnelIDFromEnv([]string{"TUNNEL_TOKEN=" + cloudflaredToken(t, "tun-1")}))
	unpadded := strings.TrimRight(cloudflaredToken(t, "tun-2"), "=")
	assert.Equal(t, "tun-2", tunnelIDFromEnv([]string{"TUNNEL_TOKEN=" + unpadded}), "padding stripped")
}

// fakeTunnelDescriber answers DescribeTunnel per tunnel id; unknown ids error like a provider would.
type fakeTunnelDescriber struct {
	infos map[string]*TunnelInfo
}

func (f fakeTunnelDescriber) DescribeTunnel(_ context.Context, id string) (*TunnelInfo, error) {
	info, ok := f.infos[id]
	if !ok {
		return nil, fmt.Errorf("%w: tunnel %s", apperrs.ErrNotFound, id)
	}
	return info, nil
}

func TestDiscoverForImport_DescribesGatewaysAndCarriesFailuresPerRow(t *testing.T) {
	report := DiscoverReport{Containers: []DiscoveredContainer{
		{Name: "web", Image: "nginx"},
		{Name: "cloudflared-a", Image: "cloudflare/cloudflared:latest", TunnelID: "tun-a"},
		{Name: "cloudflared-b", Image: "cloudflare/cloudflared:latest", TunnelID: "tun-b"},
		{Name: "cloudflared-c", Image: "cloudflare/cloudflared:latest"},
	}}
	machines := newFakeMachineRepo()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m1", Name: "prod"}))
	s := &Service{
		live: &fakeDispatch{discoverFn: func(context.Context, string, time.Duration) (DiscoverReport, error) {
			return report, nil
		}},
		machines: machines,
		tunnels: fakeTunnelDescriber{infos: map[string]*TunnelInfo{
			"tun-a": {ID: "tun-a", Name: "home", Status: "healthy", Routes: []TunnelRoute{{Hostname: "app.example.com", Service: "http://web:80"}}},
		}},
	}

	grouped, err := s.DiscoverForImport(context.Background(), "m1")
	require.NoError(t, err)
	require.Len(t, grouped.Gateways, 3)
	require.NotNil(t, grouped.Gateways[0].Tunnel)
	assert.Equal(t, "home", grouped.Gateways[0].Tunnel.Name)
	assert.Equal(t, "app.example.com", grouped.Gateways[0].Tunnel.Routes[0].Hostname)
	assert.Nil(t, grouped.Gateways[1].Tunnel)
	assert.Contains(t, grouped.Gateways[1].TunnelError, "tun-b", "a failed describe lands on its own row, not the scan")
	assert.Empty(t, grouped.Gateways[2].TunnelError, "a gateway without a token has nothing to describe")
	assert.Len(t, grouped.Standalone, 1)
}
