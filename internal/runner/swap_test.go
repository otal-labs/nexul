package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptCmd answers each command by the longest scripted prefix of its command line; a scripted list is consumed one
// answer per call, its last answer repeating. An unscripted command succeeds silently.
type scriptCmd struct {
	mu      sync.Mutex
	answers map[string][]scripted
	calls   []string
}

type scripted struct {
	out string
	err error
}

func (s *scriptCmd) run(_ context.Context, _ string, logf func(string), name string, args ...string) error {
	line := name + " " + strings.Join(args, " ")
	s.mu.Lock()
	s.calls = append(s.calls, line)
	key := ""
	for k := range s.answers {
		if strings.HasPrefix(line, k) && len(k) > len(key) {
			key = k
		}
	}
	var a scripted
	if seq := s.answers[key]; len(seq) > 0 {
		a = seq[0]
		if len(seq) > 1 {
			s.answers[key] = seq[1:]
		}
	}
	s.mu.Unlock()
	if a.err != nil {
		return a.err
	}
	logf(a.out)
	return nil
}

// ran reports whether a command line starting with prefix ran.
func (s *scriptCmd) ran(prefix string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// order returns the calls that start with any of prefixes, in the order they ran.
func (s *scriptCmd) order(prefixes ...string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, c := range s.calls {
		for _, p := range prefixes {
			if strings.HasPrefix(c, p) {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

const (
	stateStarting  = `{"Status":"running","Health":{"Status":"starting","Log":[]}}`
	stateHealthy   = `{"Status":"running","Health":{"Status":"healthy","Log":[]}}`
	stateUnhealthy = `{"Status":"running","Health":{"Status":"unhealthy","Log":[` +
		`{"ExitCode":1,"Output":"connection refused"},{"ExitCode":1,"Output":"curl: (7) failed to connect"}]}}`
	stateExited     = `{"Status":"exited","Health":{"Status":"starting","Log":[{"ExitCode":1,"Output":"boot failed"}]}}`
	withHealthcheck = `{"Test":["CMD","curl","-f","http://localhost/health"]}`
)

// swapRunScript is a routed run stack whose old container runs and whose image has a healthcheck; states is what the new
// container reports, one per look.
func swapRunScript(states ...string) *scriptCmd {
	looks := make([]scripted, 0, len(states))
	for _, s := range states {
		looks = append(looks, scripted{out: s})
	}
	return &scriptCmd{answers: map[string][]scripted{
		"docker inspect --format {{.State.Status}} api":    {{out: "running"}},
		"docker image inspect --format":                    {{out: withHealthcheck}},
		"docker inspect --format {{json .State}} api-next": looks,
	}}
}

func swapRunRequest() DeployRequestedEvent {
	return DeployRequestedEvent{
		ID: "d1", Kind: RequestDeploy, Service: "api", StackSlug: "api", Image: "img:2", Strategy: "run",
		Network: "app-net", SwapServices: []string{"api"},
	}
}

func TestShellExecutor_Deploy_SwapFailures_LeaveTheOldContainerServing(t *testing.T) {
	tests := []struct {
		name    string
		script  func() *scriptCmd
		wantErr []string
	}{
		{
			name:    "the new container turns unhealthy",
			script:  func() *scriptCmd { return swapRunScript(stateStarting, stateUnhealthy) },
			wantErr: []string{"api-next turned unhealthy", "health log:", "exit 1: connection refused", "exit 1: curl: (7) failed to connect"},
		},
		{
			name:    "the new container exits before it turns healthy",
			script:  func() *scriptCmd { return swapRunScript(stateExited) },
			wantErr: []string{"api-next stopped (exited)", "exit 1: boot failed"},
		},
		{
			name:    "the new container never turns healthy",
			script:  func() *scriptCmd { return swapRunScript(stateStarting) },
			wantErr: []string{"api-next was not healthy after 5m0s"},
		},
		{
			name: "the new container cannot be inspected",
			script: func() *scriptCmd {
				s := swapRunScript()
				s.answers["docker inspect --format {{json .State}} api-next"] = []scripted{{err: errors.New("no such container")}}
				return s
			},
			wantErr: []string{"wait for api-next to turn healthy: no such container"},
		},
		{
			name: "the new container fails to start",
			script: func() *scriptCmd {
				s := swapRunScript()
				s.answers["docker run"] = []scripted{{err: errors.New("port is already allocated")}}
				return s
			},
			wantErr: []string{"port is already allocated"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cmd := tt.script()
				rec := &frameRecorder{}
				newTestExecutor(cmd.run).Deploy(t.Context(), swapRunRequest(), rec.send)

				result := rec.last()
				require.Equal(t, DeployStatusFailed, result.Status)
				for _, want := range tt.wantErr {
					assert.Contains(t, result.Error, want)
				}
				assert.True(t, cmd.ran("docker rm -f api-next"), "the new container is removed")
				assert.False(t, cmd.ran("docker rename"), "the old container keeps its name")
				assert.False(t, cmd.ran("docker stop"), "the old container keeps serving")
				assert.NotContains(t, cmd.order("docker rm"), "docker rm -f api", "the old container is never removed")
				assert.Contains(t, rec.logText(LogPhaseDeploy), "start new: docker run --name api-next img:2")
			})
		})
	}
}

func TestShellExecutor_Deploy_SwapFailures_HealthLogReachesTheDeployLog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapRunScript(stateUnhealthy)
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Deploy(t.Context(), swapRunRequest(), rec.send)

		log := rec.logText(LogPhaseDeploy)
		assert.Contains(t, log, "wait healthy: api-next\n")
		assert.Contains(t, log, "exit 1: connection refused")
	})
}

func TestShellExecutor_Deploy_SwapRenameFails_GivesTheOldContainerItsNameBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapRunScript(stateHealthy)
		cmd.answers["docker rename api-next api"] = []scripted{{err: errors.New("daemon gone")}}
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Deploy(t.Context(), swapRunRequest(), rec.send)

		require.Equal(t, DeployStatusFailed, rec.last().Status)
		assert.Contains(t, rec.last().Error, "swap api: daemon gone")
		assert.Equal(t, []string{
			"docker rm -f api-next api-old",
			"docker rename api api-old",
			"docker rename api-next api",
			"docker rename api-old api",
			"docker rm -f api-next",
		}, cmd.order("docker rename", "docker rm -f api-next"))
		assert.False(t, cmd.ran("docker stop"))
	})
}

func TestShellExecutor_Deploy_SwapCancelled_RemovesTheNewContainer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapRunScript(stateStarting)
		ctx, cancel := context.WithCancel(t.Context())
		e := newTestExecutor(cmd.run)
		done := make(chan struct{})
		go func() {
			defer close(done)
			e.Deploy(ctx, swapRunRequest(), (&frameRecorder{}).send)
		}()
		synctest.Wait()
		cancel()
		<-done

		assert.True(t, cmd.ran("docker rm -f api-next"))
		assert.False(t, cmd.ran("docker rename"))
	})
}

func TestShellExecutor_Deploy_Swap_StartsWaitsSwapsThenStopsTheOld(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapRunScript(stateStarting, stateStarting, stateHealthy)
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Deploy(t.Context(), swapRunRequest(), rec.send)

		require.Equal(t, DeployStatusHealthy, rec.last().Status, rec.last().Error)
		assert.Equal(t, []string{
			"docker rm -f api-next api-old",
			"docker run -d --restart unless-stopped --name api-next --add-host host.docker.internal:host-gateway --network app-net img:2",
			"docker rename api api-old",
			"docker rename api-next api",
			"docker stop api-old",
			"docker rm -f api-old",
		}, cmd.order("docker rm", "docker run", "docker rename", "docker stop"))
		assert.Equal(t, "docker pull img:2\n"+
			"start new: docker run --name api-next img:2\n"+
			"wait healthy: api-next\n"+
			"swap: api-next takes the name api\n"+
			"stop old: api-old\n", rec.logText(LogPhaseDeploy))
	})
}

func TestShellExecutor_Deploy_Swap_AnOldContainerThatWillNotStopIsReportedNotFailed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapRunScript(stateHealthy)
		cmd.answers["docker stop api-old"] = []scripted{{err: errors.New("timed out")}}
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Deploy(t.Context(), swapRunRequest(), rec.send)

		assert.Equal(t, DeployStatusHealthy, rec.last().Status)
		assert.Contains(t, rec.logText(LogPhaseDeploy), "stop old failed: timed out")
		assert.True(t, cmd.ran("docker rm -f api-old"))
	})
}

func TestShellExecutor_Deploy_NoSwap_RestartsInPlaceAndSaysWhy(t *testing.T) {
	tests := []struct {
		name    string
		req     func(*DeployRequestedEvent)
		script  func(*scriptCmd)
		wantLog string
	}{
		{name: "no gateway routes to it", req: func(r *DeployRequestedEvent) { r.SwapServices = nil }},
		{name: "it publishes a host port", req: func(r *DeployRequestedEvent) { r.Ports = []string{"8080:80"} }, wantLog: noSwapHostPort},
		{
			name:    "its image has no healthcheck",
			script:  func(s *scriptCmd) { s.answers["docker image inspect --format"] = []scripted{{out: "null"}} },
			wantLog: noSwapNoHealthcheck,
		},
		{
			name: "its image turns the healthcheck off",
			script: func(s *scriptCmd) {
				s.answers["docker image inspect --format"] = []scripted{{out: `{"Test":["NONE"]}`}}
			},
			wantLog: noSwapNoHealthcheck,
		},
		{
			name: "nothing runs yet",
			script: func(s *scriptCmd) {
				s.answers["docker inspect --format {{.State.Status}} api"] = []scripted{{err: errors.New("no such container")}}
			},
			wantLog: noSwapNotRunning,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := swapRunScript(stateHealthy)
			if tt.script != nil {
				tt.script(cmd)
			}
			req := swapRunRequest()
			if tt.req != nil {
				tt.req(&req)
			}
			rec := &frameRecorder{}
			newTestExecutor(cmd.run).Deploy(t.Context(), req, rec.send)

			require.Equal(t, DeployStatusHealthy, rec.last().Status)
			assert.True(t, cmd.ran("docker rm -f api"), "the old container is replaced as before")
			assert.True(t, cmd.ran("docker run -d --restart unless-stopped --name api "))
			assert.False(t, cmd.ran("docker rename"))
			log := rec.logText(LogPhaseDeploy)
			if tt.wantLog == "" {
				assert.NotContains(t, log, "restarts in place", "a service no gateway routes to deploys as it always has")
				return
			}
			assert.Contains(t, log, "api restarts in place: "+tt.wantLog+"\n")
		})
	}
}

const composeConfig = `{"services":{
	"web":{"image":"web:2","healthcheck":{"test":["CMD","true"]}},
	"worker":{"image":"worker:2"}}}`

// swapComposeScript is a compose stack whose routed web service runs one container; states is what the new one reports.
func swapComposeScript(config string, states ...string) *scriptCmd {
	looks := make([]scripted, 0, len(states))
	for _, s := range states {
		looks = append(looks, scripted{out: s})
	}
	return &scriptCmd{answers: map[string][]scripted{
		"docker compose -p api config --format json": {{out: config}},
		"docker ps --filter label=com.docker.compose.project=api --filter label=com.docker.compose.service=web": {
			{out: "api-web-1\n"}, {out: "api-web-2\napi-web-1\n"},
		},
		"docker inspect --format {{json .State}} api-web-2": looks,
	}}
}

func composeSwapRequest(t *testing.T) DeployRequestedEvent {
	req := buildRequest(t, "b1")
	req.Strategy = "compose"
	req.SwapServices = []string{"web"}
	return req
}

func TestShellExecutor_Build_ComposeSwap_SwapsTheRoutedServiceAndUpdatesTheRest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapComposeScript(composeConfig, stateStarting, stateHealthy)
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Build(t.Context(), composeSwapRequest(t), rec.send)

		require.Equal(t, DeployStatusHealthy, rec.last().Status, rec.last().Error)
		assert.Equal(t, []string{
			"docker compose -p api up -d --no-deps worker",
			"docker compose -p api up -d --no-deps --no-recreate --scale web=2 web",
			"docker rename api-web-1 api-web-1-old",
			"docker rename api-web-2 api-web-1",
			"docker stop api-web-1-old",
			"docker rm -f api-web-1-old",
			"docker compose -p api up -d --remove-orphans",
		}, cmd.order("docker compose -p api up", "docker rename", "docker stop", "docker rm"))
		log := rec.logText(LogPhaseDeploy)
		assert.Contains(t, log, "start new: docker compose -p api up -d --no-deps --no-recreate --scale web=2 web\n")
		assert.Contains(t, log, "wait healthy: api-web-2\n")
		assert.Contains(t, log, "swap: api-web-2 takes the name api-web-1\n")
		assert.Contains(t, log, "stop old: api-web-1-old\n")
	})
}

func TestShellExecutor_Build_ComposeSwap_AnUnhealthyNewContainerFailsWithTheOldServing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapComposeScript(composeConfig, stateUnhealthy)
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Build(t.Context(), composeSwapRequest(t), rec.send)

		require.Equal(t, DeployStatusFailed, rec.last().Status)
		assert.Contains(t, rec.last().Error, "api-web-2 turned unhealthy")
		assert.Contains(t, rec.last().Error, "connection refused")
		assert.True(t, cmd.ran("docker rm -f api-web-2"))
		assert.False(t, cmd.ran("docker rename"))
		assert.False(t, cmd.ran("docker compose -p api up -d --remove-orphans"), "nothing recreates the old container")
	})
}

func TestShellExecutor_Build_ComposeSwap_AFailedScaleUpRemovesWhatItStarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapComposeScript(composeConfig)
		cmd.answers["docker compose -p api up -d --no-deps --no-recreate"] = []scripted{{err: errors.New("pull access denied")}}
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Build(t.Context(), composeSwapRequest(t), rec.send)

		require.Equal(t, DeployStatusFailed, rec.last().Status)
		assert.True(t, cmd.ran("docker rm -f api-web-2"))
		assert.NotContains(t, cmd.order("docker rm"), "docker rm -f api-web-1")
	})
}

func TestShellExecutor_Build_ComposeNoSwap_UpsAsBeforeAndSaysWhy(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		script  func(*scriptCmd)
		wantLog string
	}{
		{name: "it sets container_name", config: `{"services":{"web":{"container_name":"web","healthcheck":{"test":["CMD","true"]}}}}`, wantLog: noSwapFixedName},
		{name: "it publishes a host port", config: `{"services":{"web":{"ports":[{"target":80,"published":"8080"}],"healthcheck":{"test":["CMD","true"]}}}}`, wantLog: noSwapHostPort},
		{name: "its healthcheck is disabled", config: `{"services":{"web":{"healthcheck":{"disable":true}}}}`, wantLog: noSwapNoHealthcheck},
		{
			name:   "neither it nor its image has a healthcheck",
			config: `{"services":{"web":{}}}`,
			script: func(s *scriptCmd) {
				s.answers["docker image inspect --format {{json .Config.Healthcheck}} api-web"] = []scripted{{out: "null"}}
			},
			wantLog: noSwapNoHealthcheck,
		},
		{
			name:    "it runs two containers",
			config:  composeConfig,
			script:  func(s *scriptCmd) { s.answers[composeWebPS] = []scripted{{out: "api-web-1\napi-web-2\n"}} },
			wantLog: noSwapReplicas,
		},
		{
			name:    "nothing runs yet",
			config:  composeConfig,
			script:  func(s *scriptCmd) { s.answers[composeWebPS] = []scripted{{out: ""}} },
			wantLog: noSwapNotRunning,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := swapComposeScript(tt.config, stateHealthy)
			if tt.script != nil {
				tt.script(cmd)
			}
			rec := &frameRecorder{}
			newTestExecutor(cmd.run).Build(t.Context(), composeSwapRequest(t), rec.send)

			require.Equal(t, DeployStatusHealthy, rec.last().Status, rec.last().Error)
			assert.Equal(t, []string{"docker compose -p api up -d --remove-orphans"}, cmd.order("docker compose -p api up"))
			assert.False(t, cmd.ran("docker rename"))
			assert.Contains(t, rec.logText(LogPhaseDeploy), "web restarts in place: "+tt.wantLog+"\n")
		})
	}
}

const composeWebPS = "docker ps --filter label=com.docker.compose.project=api --filter label=com.docker.compose.service=web"

func TestShellExecutor_Build_ComposeSwap_AnImageHealthcheckIsEnough(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cmd := swapComposeScript(`{"services":{"web":{}}}`, stateHealthy)
		cmd.answers["docker image inspect --format {{json .Config.Healthcheck}} api-web"] = []scripted{{out: withHealthcheck}}
		rec := &frameRecorder{}
		newTestExecutor(cmd.run).Build(t.Context(), composeSwapRequest(t), rec.send)

		require.Equal(t, DeployStatusHealthy, rec.last().Status, rec.last().Error)
		assert.True(t, cmd.ran("docker rename api-web-2 api-web-1"), "a built image is named after its project and service")
	})
}
