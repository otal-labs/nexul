package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/runner"
)

// machineShellCast is the privacy cast with machine prod and its runner edge connected: bob, the instance Owner, added
// it; carol's role holds every bit and a rule lets that role see prod, so she holds Run commands there.
type machineShellCast struct {
	privacyCast
	machineID, runnerID string
}

func newMachineShellCast(t *testing.T, allowShell bool) machineShellCast {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the commands are POSIX shell")
	}
	c, _ := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)

	owner := identity.WithActor(t.Context(), identity.Actor{ID: "u-bob"})
	_, err := c.svc.authSvc.UpdateInstanceURL(owner, "u-bob", "https://nexul.example.com")
	require.NoError(t, err)
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/enrollments", c.owner, `{"name":"edge","machine":"prod"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var code struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &code))
	rec = callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"code":"`+code.Code+`","name":"edge","machine":"prod","os":"linux","arch":"amd64"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var enrolled struct {
		ID         string `json:"id"`
		Credential string `json:"credential"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &enrolled))
	m, err := c.store.Machines.GetByName(t.Context(), "prod")
	require.NoError(t, err)
	require.NoError(t, c.svc.accessSvc.SetRules(owner, "u-bob", "machine", []string{m.ID}, access.Targets{RoleIDs: []string{"role-everything"}},
		[]permissions.Action{permissions.MachinesSee}, access.StateAllow))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	edge := runner.NewClient(runner.ClientConfig{
		URL: "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/runner", Credential: enrolled.Credential, Name: "edge",
		Version: "v0.3.101-beta", Logger: logger, Executor: runner.NewShellExecutor(nil, runner.ExecutorConfig{}, logger),
		ShellEnabled: allowShell,
	})
	ctx, stop := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- edge.Run(ctx) }()
	t.Cleanup(func() {
		stop()
		<-stopped
	})
	require.Eventually(t, func() bool {
		rn, err := c.store.Runners.GetByID(t.Context(), enrolled.ID)
		return err == nil && rn.Connected
	}, 10*time.Second, 20*time.Millisecond)
	return machineShellCast{privacyCast: c, machineID: m.ID, runnerID: enrolled.ID}
}

func (c machineShellCast) turnOn(t *testing.T) {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPut, "/api/machines/"+c.machineID+"/shell", c.owner, `{"enabled":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"shell_enabled":true`)
}

func TestMachineShell_ARoleWithoutRunCommandsCannotStartListOrReadThem(t *testing.T) {
	t.Parallel()
	c := newMachineShellCast(t, true)
	c.turnOn(t)
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/"+c.runnerID+"/commands", c.everyBits, `{"command":"true"}`)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	var job struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &job))

	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "u-bob"})
	require.NoError(t, c.svc.accessSvc.SetRules(ctx, "u-bob", "machine", []string{c.machineID}, access.Targets{RoleIDs: []string{"role-everything"}},
		[]permissions.Action{permissions.MachinesShell}, access.StateDeny))
	for name, token := range map[string]string{"denied Run commands on the machine": c.everyBits, "a member who can't see it": c.alice} {
		t.Run(name, func(t *testing.T) {
			for _, r := range []struct{ method, path, body string }{
				{http.MethodPost, "/api/runners/" + c.runnerID + "/commands", `{"command":"id"}`},
				{http.MethodGet, "/api/machines/" + c.machineID + "/commands", ""},
				{http.MethodGet, "/api/runners/commands/" + job.ID, ""},
				{http.MethodPost, "/api/runners/commands/" + job.ID + "/cancel", ""},
				{http.MethodPut, "/api/machines/" + c.machineID + "/shell", `{"enabled":false}`},
			} {
				rec := callAPI(t, c.h, r.method, r.path, token, r.body)
				assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, rec.Code, r.method+" "+r.path+": "+rec.Body.String())
			}
			got := callToolResult(t, c.h, token, "command_run", `{"runner_id":"`+c.runnerID+`","command":"id"}`)
			assert.Contains(t, got, `"isError":true`)
			assert.NotContains(t, callToolResult(t, c.h, token, "machine_list", `{"id":"`+c.machineID+`","commands":{}}`), `"command":"true"`)
		})
	}
}

func TestMachineShell_AScopedTokenNeedsRunCommandsAmongItsOwnScopes(t *testing.T) {
	t.Parallel()
	c := newMachineShellCast(t, true)
	c.turnOn(t)
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "u-bob"})
	for scope, want := range map[integrations.Scope]int{"machines:deploy": http.StatusForbidden, "machines:shell": http.StatusAccepted} {
		token, _, _, err := c.svc.integrationsSvc.Install(ctx, "u-bob", "ops-"+strings.ReplaceAll(string(scope), ":", "-"), integrations.TrustCommunity, "https://example.com/hook", []integrations.Scope{scope})
		require.NoError(t, err)
		rec := callAPI(t, c.h, http.MethodPost, "/api/runners/"+c.runnerID+"/commands", token, `{"command":"true"}`)
		assert.Equal(t, want, rec.Code, string(scope)+": "+rec.Body.String())
	}
}

func TestMachineShell_RunsOnTheRunnerAndIsRecorded(t *testing.T) {
	t.Parallel()
	c := newMachineShellCast(t, true)

	got := callToolResult(t, c.h, c.everyBits, "command_run", `{"runner_id":"`+c.runnerID+`","command":"echo hi"}`)
	assert.Contains(t, got, "off on machine prod", "Nexul's switch starts off")

	c.turnOn(t)
	got = callToolResult(t, c.h, c.everyBits, "command_run", `{"runner_id":"`+c.runnerID+`","command":"echo hello; exit 3"}`)
	assert.Contains(t, got, `\"status\":\"failed\"`)
	assert.Contains(t, got, `\"exit_code\":3`)
	assert.Contains(t, got, "hello")

	rec := callAPI(t, c.h, http.MethodGet, "/api/machines/"+c.machineID+"/commands", c.everyBits, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"total":1`)
	assert.Contains(t, rec.Body.String(), `"runner_name":"edge"`)
	assert.Contains(t, callToolResult(t, c.h, c.everyBits, "machine_list", `{"id":"`+c.machineID+`","commands":{}}`), "echo hello")
}

func TestMachineShell_AMachineInstalledWithoutAllowShellRefusesEveryJob(t *testing.T) {
	t.Parallel()
	c := newMachineShellCast(t, false)
	c.turnOn(t)

	got := callToolResult(t, c.h, c.owner, "command_run", `{"runner_id":"`+c.runnerID+`","command":"echo should-not-run"}`)

	assert.Contains(t, got, `\"status\":\"refused\"`)
	assert.Contains(t, got, "--allow-shell")
	assert.NotContains(t, got, "should-not-run\\n")
	rec := callAPI(t, c.h, http.MethodGet, "/api/machines/"+c.machineID+"/commands", c.owner, "")
	assert.Contains(t, rec.Body.String(), `"status":"refused"`, "the refusal is on the record")
}
