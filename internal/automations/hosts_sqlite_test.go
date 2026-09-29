package automations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

type instanceURL string

func (u instanceURL) GetInstanceURL(context.Context) (string, error) { return string(u), nil }

// recordingConns records which automations lost their live connection, then drops them for real.
type recordingConns struct {
	mu   sync.Mutex
	ids  []string
	next automations.ConnectionRegistry
}

func (c *recordingConns) Disconnect(automationID, reason string) {
	c.mu.Lock()
	c.ids = append(c.ids, automationID)
	c.mu.Unlock()
	c.next.Disconnect(automationID, reason)
}

func (c *recordingConns) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.ids...)
}

type hostsFixture struct {
	store     *storage.Store
	svc       *automations.Service
	hosts     *automations.HostsService
	conns     *recordingConns
	srv       *httptest.Server
	enrollDir string
}

func adminCtx() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "admin"})
}

// allowAnywhere passes every instance-level check, standing in for an Owner.
type allowAnywhere struct{}

func (allowAnywhere) RequireAnywhere(context.Context, permissions.Action) error { return nil }

// newHostsFixture serves the automations host endpoints, the placement endpoint, a token-gated self-read and the
// dial-in socket over a real, migrated SQLite database, the way the composition root mounts them.
func newHostsFixture(t *testing.T) *hostsFixture {
	t.Helper()
	_, store := newTestStore(t)
	svc := automations.NewService(store.Automations, allowAllPerm{})
	svc.SetGateway(func(string, string, []string) bool { return false }, nil)
	enrollDir := filepath.Join(t.TempDir(), "enroll")
	hosts := automations.NewHostsService(store.AutomationHosts, store.Automations, []byte("host-token-key")).
		WithGate(allowAnywhere{}).WithInstanceURL(instanceURL("https://nexul.example.com/")).WithEnrollDir(enrollDir)
	svc.SetHosts(hosts)
	dial := automations.NewDialinHandler(svc, automations.DialinConfig{
		Repo: store.Automations, Cursors: store.AutomationCursors, EventLog: store.AutomationEventLog,
		Runs: store.AutomationRuns, Logger: testutil.DiscardLogger(), PollInterval: 10 * time.Millisecond,
	})
	conns := &recordingConns{next: dial}
	svc.SetConnectionRegistry(conns)
	hosts.SetConnectionRegistry(conns)

	asAdmin := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: "admin"})))
		})
	}
	hostsHTTP := automations.NewHostsHandler(hosts)
	automationsHTTP := automations.NewHandler(svc).Routes()
	mux := http.NewServeMux()
	public := hostsHTTP.PublicRoutes()
	mux.Handle("POST /api/automation-hosts/enroll", public)
	mux.Handle("POST /api/automation-hosts/self/remove", public)
	mux.Handle("GET /api/automation-hosts/self/assignments", public)
	mux.Handle("/api/automation-hosts", asAdmin(hostsHTTP.Routes()))
	mux.Handle("/api/automation-hosts/", asAdmin(hostsHTTP.Routes()))
	mux.Handle("PATCH /api/automations/{id}/host", asAdmin(automationsHTTP))
	mux.Handle("GET /api/automations/{id}", svc.RequireAutomation(fellThroughUserAuth, automationsHTTP))
	mux.Handle("/ws/automations", dial)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &hostsFixture{store: store, svc: svc, hosts: hosts, conns: conns, srv: srv, enrollDir: enrollDir}
}

func (f *hostsFixture) do(t *testing.T, method, path, bearer string, body any) (int, []byte) {
	t.Helper()
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, f.srv.URL+path, payload)
	require.NoError(t, err)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // read-only response
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, data
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var e struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(body, &e), string(body))
	return e.Code
}

// enroll mints a code for name through the admin endpoint and trades it, returning the host id and credential.
func (f *hostsFixture) enroll(t *testing.T, name string) (id, credential string) {
	t.Helper()
	status, body := f.do(t, http.MethodPost, "/api/automation-hosts/enrollments", "", map[string]string{"name": name})
	require.Equal(t, http.StatusCreated, status, string(body))
	var e automations.HostEnrollment
	require.NoError(t, json.Unmarshal(body, &e))
	status, body = f.do(t, http.MethodPost, "/api/automation-hosts/enroll", "", automations.HostEnrollRequest{
		Code: e.Code, Name: name, OS: "linux", Arch: "amd64", Version: "v0.3.0",
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	var got automations.HostEnrolled
	require.NoError(t, json.Unmarshal(body, &got))
	return got.ID, got.Credential
}

func (f *hostsFixture) assignments(t *testing.T, credential string) []automations.Assignment {
	t.Helper()
	status, body := f.do(t, http.MethodGet, "/api/automation-hosts/self/assignments", credential, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var a automations.Assignments
	require.NoError(t, json.Unmarshal(body, &a))
	return a.Automations
}

func (f *hostsFixture) tokenFor(t *testing.T, credential, automationID string) string {
	t.Helper()
	for _, a := range f.assignments(t, credential) {
		if a.ID == automationID {
			return a.Token
		}
	}
	t.Fatalf("automation %s is not assigned to this host", automationID)
	return ""
}

func (f *hostsFixture) createAutomation(t *testing.T, name string, enabled bool) string {
	t.Helper()
	a, _, err := f.svc.Create(adminCtx(), "admin", name, []string{"tickets:read"})
	require.NoError(t, err)
	if enabled {
		_, err = f.svc.SetEnabled(adminCtx(), "admin", a.ID, true)
		require.NoError(t, err)
	}
	return a.ID
}

// selfRead is the host's state fetch: the automation reading itself with its worker's token.
func (f *hostsFixture) selfRead(t *testing.T, automationID, token string) int {
	t.Helper()
	status, _ := f.do(t, http.MethodGet, "/api/automations/"+automationID, token, nil)
	return status
}

// dialStatus is the HTTP status /ws/automations answers the token with: 101 once upgraded.
func (f *hostsFixture) dialStatus(t *testing.T, token string) int {
	t.Helper()
	url := "ws" + strings.TrimPrefix(f.srv.URL, "http") + "/ws/automations?token=" + token
	conn, resp, err := websocket.Dial(t.Context(), url, nil)
	if err == nil {
		_ = conn.CloseNow() // the upgrade is all this checks
	}
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func TestHostEnroll_RefusedCodes(t *testing.T) {
	f := newHostsFixture(t)
	status, body := f.do(t, http.MethodPost, "/api/automation-hosts/enrollments", "", map[string]string{"name": "jobs"})
	require.Equal(t, http.StatusCreated, status)
	var e automations.HostEnrollment
	require.NoError(t, json.Unmarshal(body, &e))

	expired, expiredHash, err := hostcred.MintCode()
	require.NoError(t, err)
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, f.store.AutomationHosts.CreateHostEnrollment(t.Context(), &automations.HostEnrollmentCode{
		CodeHash: expiredHash, Name: "old", CreatedAt: past, ExpiresAt: past.Add(time.Hour),
	}))

	tests := []struct {
		name       string
		code       string
		host       string
		wantStatus int
		wantCode   string
	}{
		{"an unknown code", "nxe_nope", "jobs", http.StatusUnauthorized, "invalid_code"},
		{"an expired code", expired, "old", http.StatusUnauthorized, "invalid_code"},
		{"a code made for another name", e.Code, "other", http.StatusConflict, "name_mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := f.do(t, http.MethodPost, "/api/automation-hosts/enroll", "", automations.HostEnrollRequest{Code: tt.code, Name: tt.host})
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantCode, errorCode(t, body))
		})
	}

	t.Run("a used code", func(t *testing.T) {
		status, _ := f.do(t, http.MethodPost, "/api/automation-hosts/enroll", "", automations.HostEnrollRequest{Code: e.Code, Name: "jobs"})
		require.Equal(t, http.StatusCreated, status)
		status, body := f.do(t, http.MethodPost, "/api/automation-hosts/enroll", "", automations.HostEnrollRequest{Code: e.Code, Name: "jobs"})
		assert.Equal(t, http.StatusUnauthorized, status)
		assert.Equal(t, "invalid_code", errorCode(t, body))
	})
}

func TestHostCreateEnrollment_Refusals(t *testing.T) {
	f := newHostsFixture(t)
	f.enroll(t, "taken")
	noURL := automations.NewHostsService(f.store.AutomationHosts, f.store.Automations, []byte("k")).WithGate(allowAnywhere{})

	tests := []struct {
		name string
		svc  *automations.HostsService
		ctx  context.Context
		host string
		want error
	}{
		{"no instance URL to point the command at", noURL, adminCtx(), "jobs", apperrs.ErrConflict},
		{"a name the installer cannot use", f.hosts, adminCtx(), "Jobs Box", apperrs.ErrInvalid},
		{"a name already enrolled", f.hosts, adminCtx(), "taken", apperrs.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.CreateEnrollment(tt.ctx, tt.host, "")
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestHostEnroll_WithoutAMachineOnTheCode_UsesTheReportedHostname(t *testing.T) {
	f := newHostsFixture(t)
	e, err := f.hosts.CreateEnrollment(adminCtx(), "jobs", "")
	require.NoError(t, err)

	got, err := f.hosts.Enroll(t.Context(), automations.HostEnrollRequest{Code: e.Code, Name: "jobs", Machine: "box-1"})
	require.NoError(t, err)

	assert.Equal(t, "box-1", got.Machine)
}

func TestHostEnroll_RendersCommandsAndFilesUnderMachine(t *testing.T) {
	f := newHostsFixture(t)
	e, err := f.hosts.CreateEnrollment(adminCtx(), "jobs", "prod")
	require.NoError(t, err)
	assert.Equal(t, "curl -fsSL https://nexul.io/automations.sh | sh -s -- --server https://nexul.example.com --name jobs --code "+e.Code, e.Commands.Unix)
	assert.Equal(t, "& ([scriptblock]::Create((irm https://nexul.io/automations.ps1))) --server https://nexul.example.com --name jobs --code "+e.Code, e.Commands.Windows)

	got, err := f.hosts.Enroll(t.Context(), automations.HostEnrollRequest{Code: e.Code, Name: "jobs", OS: "linux", Arch: "arm64", Version: "v1"})
	require.NoError(t, err)
	assert.Equal(t, "prod", got.Machine)
	assert.True(t, strings.HasPrefix(got.Credential, "nxa_"))

	list, err := f.hosts.List(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, automations.HostView{
		ID: got.ID, Name: "jobs", Machine: "prod", OS: "linux", Arch: "arm64", Version: "v1", Connected: true, LastSeen: list[0].LastSeen,
	}, list[0])
}

func TestHostAssignments_RefusesUnknownAndRemovedHosts(t *testing.T) {
	f := newHostsFixture(t)
	id, credential := f.enroll(t, "jobs")

	status, _ := f.do(t, http.MethodGet, "/api/automation-hosts/self/assignments", "", nil)
	assert.Equal(t, http.StatusUnauthorized, status, "no credential")
	status, _ = f.do(t, http.MethodGet, "/api/automation-hosts/self/assignments", "nxa_unknown", nil)
	assert.Equal(t, http.StatusUnauthorized, status, "unknown credential")

	status, _ = f.do(t, http.MethodDelete, "/api/automation-hosts/"+id, "", nil)
	require.Equal(t, http.StatusNoContent, status)
	status, body := f.do(t, http.MethodGet, "/api/automation-hosts/self/assignments", credential, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.JSONEq(t, automations.HostRemovedRefusal, string(body))
}

func TestHostAssignments_FollowPlacement(t *testing.T) {
	f := newHostsFixture(t)
	_, instanceCred := f.enroll(t, automations.InstanceHostName)
	jobsID, jobsCred := f.enroll(t, "jobs")
	onInstance := f.createAutomation(t, "on instance", true)
	disabled := f.createAutomation(t, "disabled", false)
	moved := f.createAutomation(t, "moved", true)

	status, body := f.do(t, http.MethodPatch, "/api/automations/"+moved+"/host", "", map[string]string{"host_id": jobsID})
	require.Equal(t, http.StatusOK, status, string(body))

	ids := func(as []automations.Assignment) []string {
		out := []string{}
		for _, a := range as {
			out = append(out, a.ID)
		}
		return out
	}
	assert.ElementsMatch(t, []string{onInstance}, ids(f.assignments(t, instanceCred)), "unplaced enabled automations run on instance; %s is disabled", disabled)
	assert.ElementsMatch(t, []string{moved}, ids(f.assignments(t, jobsCred)))
	assert.Contains(t, f.conns.snapshot(), moved, "a move drops the live connection")

	status, _ = f.do(t, http.MethodPatch, "/api/automations/"+moved+"/host", "", map[string]any{"host_id": nil})
	require.Equal(t, http.StatusOK, status)
	assert.ElementsMatch(t, []string{onInstance, moved}, ids(f.assignments(t, instanceCred)))
	assert.Empty(t, f.assignments(t, jobsCred))
}

func TestSetHost_Refusals(t *testing.T) {
	f := newHostsFixture(t)
	instanceID, _ := f.enroll(t, automations.InstanceHostName)
	id := f.createAutomation(t, "a", true)

	status, _ := f.do(t, http.MethodPatch, "/api/automations/"+id+"/host", "", map[string]string{"host_id": "ghost"})
	assert.Equal(t, http.StatusNotFound, status, "an unknown host")
	status, _ = f.do(t, http.MethodPatch, "/api/automations/ghost/host", "", map[string]string{"host_id": ""})
	assert.Equal(t, http.StatusNotFound, status, "an unknown automation")

	a, err := f.svc.SetHost(adminCtx(), "admin", id, instanceID)
	require.NoError(t, err)
	assert.Nil(t, a.HostID, "the instance host by id is stored as no host")

	unwired := automations.NewService(f.store.Automations, allowAllPerm{})
	_, err = unwired.SetHost(adminCtx(), "admin", id, instanceID)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestHostToken_AcceptedOnlyWhilePlacementAndCredentialHold(t *testing.T) {
	f := newHostsFixture(t)
	instanceID, instanceCred := f.enroll(t, automations.InstanceHostName)
	jobsID, jobsCred := f.enroll(t, "jobs")
	id := f.createAutomation(t, "a", true)

	onInstance := f.tokenFor(t, instanceCred, id)
	assert.Equal(t, http.StatusOK, f.selfRead(t, id, onInstance), "the automation auth path accepts it")
	assert.Equal(t, http.StatusSwitchingProtocols, f.dialStatus(t, onInstance), "the dial-in accepts it")

	t.Run("a forged or malformed token is refused", func(t *testing.T) {
		for _, token := range []string{"dat_h." + id + ".AAAA", "dat_h." + id, "dat_h..AAAA", "dat_h.ghost." + strings.Split(onInstance, ".")[2]} {
			assert.Equal(t, http.StatusUnauthorized, f.selfRead(t, id, token), token)
			assert.Equal(t, http.StatusUnauthorized, f.dialStatus(t, token), token)
		}
	})

	t.Run("a move invalidates the old host's token", func(t *testing.T) {
		_, err := f.svc.SetHost(adminCtx(), "admin", id, jobsID)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, f.selfRead(t, id, onInstance))
		assert.Equal(t, http.StatusUnauthorized, f.dialStatus(t, onInstance))
		onJobs := f.tokenFor(t, jobsCred, id)
		assert.Equal(t, http.StatusOK, f.selfRead(t, id, onJobs))

		status, _ := f.do(t, http.MethodDelete, "/api/automation-hosts/"+jobsID, "", nil)
		require.Equal(t, http.StatusNoContent, status)
		assert.Equal(t, http.StatusUnauthorized, f.selfRead(t, id, onJobs), "a removed host's token")
		assert.Equal(t, http.StatusUnauthorized, f.dialStatus(t, onJobs))
		assert.Equal(t, http.StatusOK, f.selfRead(t, id, f.tokenFor(t, instanceCred, id)), "it falls back to the instance host")
	})

	t.Run("rotating or revoking the automation's token invalidates it", func(t *testing.T) {
		token := f.tokenFor(t, instanceCred, id)
		_, _, err := f.svc.MintToken(adminCtx(), "admin", id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, f.selfRead(t, id, token))

		token = f.tokenFor(t, instanceCred, id)
		_, err = f.svc.RevokeToken(adminCtx(), "admin", id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, f.selfRead(t, id, token))
	})

	t.Run("no instance host enrolled", func(t *testing.T) {
		other := f.createAutomation(t, "b", true)
		token := f.tokenFor(t, instanceCred, other)
		status, _ := f.do(t, http.MethodDelete, "/api/automation-hosts/"+instanceID, "", nil)
		require.Equal(t, http.StatusNoContent, status)
		assert.Equal(t, http.StatusUnauthorized, f.selfRead(t, other, token))
	})
}

func TestHostRemove(t *testing.T) {
	f := newHostsFixture(t)
	jobsID, _ := f.enroll(t, "jobs")
	id := f.createAutomation(t, "a", true)
	_, err := f.svc.SetHost(adminCtx(), "admin", id, jobsID)
	require.NoError(t, err)

	status, _ := f.do(t, http.MethodDelete, "/api/automation-hosts/ghost", "", nil)
	assert.Equal(t, http.StatusNotFound, status, "an unknown host")

	status, _ = f.do(t, http.MethodDelete, "/api/automation-hosts/"+jobsID, "", nil)
	require.Equal(t, http.StatusNoContent, status)
	a, err := f.svc.Get(adminCtx(), "admin", id)
	require.NoError(t, err)
	assert.Nil(t, a.HostID, "its automations move back to the instance host")
	assert.Contains(t, f.conns.snapshot(), id)
	list, err := f.hosts.List(t.Context())
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestHostRemoveSelf(t *testing.T) {
	f := newHostsFixture(t)
	_, credential := f.enroll(t, "jobs")

	status, _ := f.do(t, http.MethodPost, "/api/automation-hosts/self/remove", "nxa_unknown", nil)
	assert.Equal(t, http.StatusUnauthorized, status)

	status, _ = f.do(t, http.MethodPost, "/api/automation-hosts/self/remove", credential, nil)
	require.Equal(t, http.StatusNoContent, status)
	status, _ = f.do(t, http.MethodPost, "/api/automation-hosts/self/remove", credential, nil)
	assert.Equal(t, http.StatusNoContent, status, "a host already removed has nothing left to remove")
	list, err := f.hosts.List(t.Context())
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestWriteInstanceEnrollment_UntilTheInstanceHostEnrolls(t *testing.T) {
	f := newHostsFixture(t)
	file := filepath.Join(f.enrollDir, "automations-instance")

	require.NoError(t, f.hosts.WriteInstanceEnrollment(t.Context()))
	info, err := os.Stat(file)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	code, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(code), hostcred.CodePrefix))

	_, err = f.hosts.Enroll(t.Context(), automations.HostEnrollRequest{Code: string(code), Name: automations.InstanceHostName})
	require.NoError(t, err)
	assert.NoFileExists(t, file, "enrolling the instance host deletes its code")

	require.NoError(t, os.WriteFile(file, []byte("stale"), 0o600))
	require.NoError(t, f.hosts.WriteInstanceEnrollment(t.Context()))
	assert.NoFileExists(t, file, "a boot with the instance host enrolled writes no code")
}

func TestAutomationUpdateTool_MovesBetweenHosts(t *testing.T) {
	f := newHostsFixture(t)
	jobsID, _ := f.enroll(t, "jobs")
	id := f.createAutomation(t, "a", true)
	tools := automations.MCPTools(f.svc)
	var update func(ctx context.Context, args json.RawMessage) (any, error)
	for _, tool := range tools {
		if tool.Name == "automation_update" {
			update = tool.Call
		}
	}
	require.NotNil(t, update)

	_, err := update(adminCtx(), json.RawMessage(`{"id":"`+id+`","enabled":false,"host_id":"ghost"}`))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, err.Error(), "enabled", "the error says what already took effect")

	got, err := update(adminCtx(), json.RawMessage(`{"id":"`+id+`","host_id":"`+jobsID+`"}`))
	require.NoError(t, err)
	out, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"host_id":"`+jobsID+`"`)

	got, err = update(adminCtx(), json.RawMessage(`{"id":"`+id+`","host_id":""}`))
	require.NoError(t, err)
	out, err = json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "host_id", "back on the instance host")
}
