package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPHandler_ListMachines(t *testing.T) {
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", FirstSeen: now, LastSeen: now}))
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(machines)
	srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/machines")
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got []Machine
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Len(t, got, 1)
	assert.Equal(t, "prod", got[0].Name)
}

func TestHTTPHandler_UpdateMachine(t *testing.T) {
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", StackRoot: defaultStackRoot, FirstSeen: now, LastSeen: now}))
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(machines)
	srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
	defer srv.Close()

	body, err := json.Marshal(updateMachineRequest{Name: "prod-1", StackRoot: "/srv/data"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/machines/m-1", bytes.NewReader(body))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got Machine
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "prod-1", got.Name)
	assert.Equal(t, "/srv/data", got.StackRoot)
}

func TestHTTPHandler_UpdateMachine_UnknownID_Errors(t *testing.T) {
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(newFakeMachineRepo())
	srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/machines/missing", bytes.NewReader([]byte(`{"name":"x"}`)))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

func TestHTTPHandler_DiscoverMachine(t *testing.T) {
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", FirstSeen: now, LastSeen: now}))
	dispatch := &fakeDispatch{discoverFn: func(_ context.Context, machine string, _ time.Duration) (DiscoverReport, error) {
		return DiscoverReport{Containers: []DiscoveredContainer{
			{Name: "web", Image: "nginx", Labels: map[string]string{composeProjectLabel: "myapp"}},
			{Name: "standalone", Image: "redis"},
			{Name: "cf", Image: "cloudflare/cloudflared:latest"},
		}}, nil
	}}
	svc := NewService(newFakeRunnerRepo(), dispatch).WithMachines(machines)
	srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/machines/m-1/discover", "application/json", nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got GroupedDiscovery
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Len(t, got.Stacks, 1)
	assert.Equal(t, "myapp", got.Stacks[0].Project)
	require.Len(t, got.Standalone, 1)
	require.Len(t, got.Gateways, 1)
}

func TestHTTPHandler_DiscoverMachine_UnknownMachine_Errors(t *testing.T) {
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(newFakeMachineRepo())
	srv := httptest.NewServer(NewHTTPHandler(svc).Routes())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/machines/missing/discover", "application/json", nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}
