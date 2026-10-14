package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMachines_ListNamesWhatTheViewerHolds: GET /api/machines and machine_list carry the actions the caller holds on
// each machine, so a client shows Manage controls to the instance Owner and Manage holders alone, and a Deny on Manage
// takes the entry away from a person who still sees the machine.
func TestMachines_ListNamesWhatTheViewerHolds(t *testing.T) {
	c, db := newPrivacyCast(t)
	seedEdgeAndCore(t, db)
	_, err := db.Exec(`UPDATE permission_overwrites SET deny = '["machines:write"]' WHERE resource_id = 'm-edge' AND user_id = 'u-carol'`)
	require.NoError(t, err)
	every := []string{"machines:delete", "machines:deploy", "machines:see", "machines:write"}
	people := []struct {
		name, token string
		held        map[string][]string
	}{
		{"the instance Owner", c.owner, map[string][]string{"edge-box": every, "core-box": every}},
		{"a person on edge denied Manage", c.everyBits, map[string][]string{"edge-box": {"machines:delete", "machines:deploy", "machines:see"}}},
	}
	for _, p := range people {
		t.Run(p.name, func(t *testing.T) {
			var listed []struct {
				Name    string   `json:"name"`
				Actions []string `json:"actions"`
			}
			rec := callAPI(t, c.h, http.MethodGet, "/api/machines", p.token, "")
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed), rec.Body.String())
			got := map[string][]string{}
			for _, m := range listed {
				got[m.Name] = m.Actions
			}
			assert.Equal(t, p.held, got, "GET /api/machines")

			var rpc struct {
				Result struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"result"`
			}
			require.NoError(t, json.Unmarshal([]byte(callTool(t, c.h, p.token, "machine_list", `{}`)), &rpc))
			require.NotEmpty(t, rpc.Result.Content)
			var tool struct {
				Items []struct {
					Name    string   `json:"name"`
					Actions []string `json:"actions"`
				} `json:"items"`
			}
			require.NoError(t, json.Unmarshal([]byte(rpc.Result.Content[0].Text), &tool))
			viaTool := map[string][]string{}
			for _, m := range tool.Items {
				viaTool[m.Name] = m.Actions
			}
			assert.Equal(t, p.held, viaTool, "machine_list")
		})
	}
}
