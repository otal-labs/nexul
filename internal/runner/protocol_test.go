package runner

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestFrame_Encode(t *testing.T) {
	tests := []struct {
		name string
		got  func() Frame
		want string
	}{
		{
			name: "assign_build wire shape",
			got: func() Frame {
				return Frame{Type: FrameAssignBuild, ID: "build-123", Repo: "org/app", Ref: "main", Steps: []string{"go build"}}
			},
			want: `{"type":"assign_build","id":"build-123","repo":"org/app","ref":"main","steps":["go build"]}`,
		},
		{
			name: "assign_deploy wire shape",
			got: func() Frame {
				return Frame{Type: FrameAssignDeploy, ID: "deploy-456", Service: "api", Image: "ghcr.io/org/api:1.0", Env: map[string]string{"PORT": "8080"}}
			},
			want: `{"type":"assign_deploy","id":"deploy-456","service":"api","image":"ghcr.io/org/api:1.0","env":{"PORT":"8080"}}`,
		},
		{
			name: "cancel wire shape",
			got: func() Frame {
				return Frame{Type: FrameCancel, ID: "build-123"}
			},
			want: `{"type":"cancel","id":"build-123"}`,
		},
		{
			name: "heartbeat wire shape",
			got: func() Frame {
				return Frame{Type: FrameHeartbeat, RunnerID: "r-1", TS: 1700000000}
			},
			want: `{"type":"heartbeat","runner_id":"r-1","ts":1700000000}`,
		},
		{
			name: "build_progress wire shape",
			got: func() Frame {
				return Frame{Type: FrameBuildProgress, ID: "build-123", Step: 2, Total: 5, Log: "ok"}
			},
			want: `{"type":"build_progress","id":"build-123","step":2,"total":5,"log":"ok"}`,
		},
		{
			name: "build_result wire shape",
			got: func() Frame {
				return Frame{Type: FrameBuildResult, ID: "build-123", Status: BuildStatusSuccess, Artifacts: []string{"bin/api"}}
			},
			want: `{"type":"build_result","id":"build-123","status":"success","artifacts":["bin/api"]}`,
		},
		{
			name: "deploy_progress wire shape",
			got: func() Frame {
				return Frame{Type: FrameDeployProgress, ID: "deploy-456", Phase: DeployPhasePulling, Log: "pulling..."}
			},
			want: `{"type":"deploy_progress","id":"deploy-456","phase":"pulling","log":"pulling..."}`,
		},
		{
			name: "deploy_result wire shape",
			got: func() Frame {
				return Frame{Type: FrameDeployResult, ID: "deploy-456", Status: DeployStatusFailed, Error: "no space"}
			},
			want: `{"type":"deploy_result","id":"deploy-456","status":"failed","error":"no space"}`,
		},
		{
			name: "assign_deploy carries the gateway join step",
			got: func() Frame {
				return Frame{Type: FrameAssignDeploy, ID: "deploy-456", Service: "api", Image: "ghcr.io/org/api:1.0", GatewayContainer: "gw", JoinNetworks: []string{"api_default"}}
			},
			want: `{"type":"assign_deploy","id":"deploy-456","service":"api","image":"ghcr.io/org/api:1.0","gateway_container":"gw","join_networks":["api_default"]}`,
		},
		{
			name: "join_networks wire shape",
			got: func() Frame {
				return Frame{Type: FrameJoinNetworks, GatewayContainer: "gw", JoinNetworks: []string{"net1", "net2"}}
			},
			want: `{"type":"join_networks","gateway_container":"gw","join_networks":["net1","net2"]}`,
		},
		{
			name: "join_networks_result wire shape",
			got: func() Frame {
				return Frame{Type: FrameJoinNetworksResult, GatewayContainer: "gw", Status: BuildStatusFailed, Error: "not found"}
			},
			want: `{"type":"join_networks_result","gateway_container":"gw","status":"failed","error":"not found"}`,
		},
		{
			name: "update wire shape",
			got: func() Frame {
				return Frame{Type: FrameUpdate, Version: "v0.2.0", URL: "https://instance/api/runners/download/linux-amd64?version=v0.2.0", Sha256: "abc123"}
			},
			want: `{"type":"update","version":"v0.2.0","url":"https://instance/api/runners/download/linux-amd64?version=v0.2.0","sha256":"abc123"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.got()
			data, err := g.Encode()
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestFrame_Encode_Invalid(t *testing.T) {
	tests := []struct {
		name string
		f    Frame
	}{
		{name: "missing type", f: Frame{}},
		{name: "unknown type", f: Frame{Type: "nuke"}},
		{name: "heartbeat missing runner", f: Frame{Type: FrameHeartbeat}},
		{name: "assign_build missing ref", f: Frame{Type: FrameAssignBuild, ID: "b1", Repo: "r"}},
		{name: "assign_deploy missing image", f: Frame{Type: FrameAssignDeploy, ID: "d1", Service: "s"}},
		{name: "cancel missing id", f: Frame{Type: FrameCancel}},
		{name: "build_progress step zero", f: Frame{Type: FrameBuildProgress, ID: "b1", Step: 0, Total: 1}},
		{name: "build_progress step exceeds total", f: Frame{Type: FrameBuildProgress, ID: "b1", Step: 3, Total: 2}},
		{name: "build_result bad status", f: Frame{Type: FrameBuildResult, ID: "b1", Status: "meh"}},
		{name: "deploy_progress bad phase", f: Frame{Type: FrameDeployProgress, ID: "d1", Phase: "draining"}},
		{name: "deploy_result bad status", f: Frame{Type: FrameDeployResult, ID: "d1", Status: "running"}},
		{name: "join_networks missing gateway_container", f: Frame{Type: FrameJoinNetworks, JoinNetworks: []string{"net1"}}},
		{name: "join_networks missing networks", f: Frame{Type: FrameJoinNetworks, GatewayContainer: "gw"}},
		{name: "join_networks_result missing gateway_container", f: Frame{Type: FrameJoinNetworksResult, Status: BuildStatusSuccess}},
		{name: "update missing url", f: Frame{Type: FrameUpdate, Version: "v0.2.0"}},
		{name: "update missing version", f: Frame{Type: FrameUpdate, URL: "https://instance/download"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.f.Encode()
			require.Error(t, err)
			assert.True(t, errors.Is(err, apperrs.ErrInvalid))
		})
	}
}

func TestParseFrame(t *testing.T) {
	t.Run("valid frames round-trip", func(t *testing.T) {
		frames := []Frame{
			{Type: FrameHeartbeat, RunnerID: "r-1", TS: 1700000000},
			{Type: FrameAssignBuild, ID: "b1", Repo: "org/app", Ref: "main", Steps: []string{"a", "b"}},
			{Type: FrameCancel, ID: "b1"},
			{Type: FrameBuildProgress, ID: "b1", Step: 1, Total: 2, Log: "x"},
			{Type: FrameBuildResult, ID: "b1", Status: BuildStatusFailed, Error: "boom"},
			{Type: FrameJoinNetworks, GatewayContainer: "gw", JoinNetworks: []string{"net1"}},
			{Type: FrameJoinNetworksResult, GatewayContainer: "gw", Status: BuildStatusSuccess},
			{Type: FrameUpdate, Version: "v0.2.0", URL: "https://instance/download", Sha256: "abc123"},
		}
		for _, want := range frames {
			data, err := (&want).Encode()
			require.NoError(t, err)
			got, err := ParseFrame(data)
			require.NoError(t, err)
			assert.Equal(t, want, *got)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		_, err := ParseFrame([]byte(`{"type":`))
		require.Error(t, err)
	})

	t.Run("rejects invalid status on read", func(t *testing.T) {
		_, err := ParseFrame([]byte(`{"type":"build_result","id":"b1","status":"nope"}`))
		require.Error(t, err)
	})
}

func TestParseFrame_ExtraFieldsIgnored(t *testing.T) {
	got, err := ParseFrame([]byte(`{"type":"heartbeat","runner_id":"r-1","ts":1,"wat":"ignored"}`))
	require.NoError(t, err)
	assert.Equal(t, "r-1", got.RunnerID)
}

func TestFrame_JSONRoundTripPreservesEnvOrder(t *testing.T) {
	f := Frame{Type: FrameAssignDeploy, ID: "d1", Service: "s", Image: "i", Env: map[string]string{"B": "2", "A": "1"}}
	data, err := json.Marshal(f)
	require.NoError(t, err)
	var got Frame
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, f.Env, got.Env)
}
