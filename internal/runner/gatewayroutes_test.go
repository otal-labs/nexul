package runner

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const appRoutes = `{"routers":{"nexul-e1":{"rule":"Host(` + "`app.example.com`" + `)"}},"services":{"nexul-e1":{"url":"http://app:80"}}}`

func TestMergeRoutes_AddsRoutesBesideTheBaseAndReplacesSameNames(t *testing.T) {
	base := `{"http":{"routers":{"nexul":{"rule":"base"},"nexul-e1":{"rule":"stale"}},"services":{"nexul":{"s":1}}}}`

	out, err := mergeRoutes(base+"\n", appRoutes)
	require.NoError(t, err)

	var got struct {
		HTTP struct {
			Routers  map[string]map[string]any
			Services map[string]map[string]any
		}
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "base", got.HTTP.Routers["nexul"]["rule"], "the base route stays")
	assert.Equal(t, "Host(`app.example.com`)", got.HTTP.Routers["nexul-e1"]["rule"], "a route sent again replaces the old one")
	assert.Contains(t, got.HTTP.Services, "nexul")
	assert.Contains(t, got.HTTP.Services, "nexul-e1")
}

// Traefik rejects a file with an empty routers or services map and keeps the old routes, so no route left is no section.
func TestMergeRoutes_NoRoutesLeftIsAnEmptyConfig(t *testing.T) {
	out, err := mergeRoutes("{}", `{"routers":{},"services":{}}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, out)

	out, err = mergeRoutes(`{"http":{"routers":{"nexul":{}},"services":{"nexul":{}}}}`, `{"routers":{},"services":{}}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{"http":{"routers":{"nexul":{}},"services":{"nexul":{}}}}`, out, "the base route stays")
}

func TestMergeRoutes_RefusesWhatItCannotParse(t *testing.T) {
	_, err := mergeRoutes("{", appRoutes)
	require.Error(t, err)
	_, err = mergeRoutes("{}", "nope")
	require.Error(t, err)
}

// execRecorder answers the base config read with env and records every docker command.
type execRecorder struct {
	env   string
	calls [][]string
	fail  map[string]error
}

func (r *execRecorder) run(_ context.Context, _ string, logf func(string), name string, args ...string) error {
	r.calls = append(r.calls, append([]string{name}, args...))
	if err := r.fail[args[2]]; err != nil {
		return err
	}
	if args[2] == "printenv" {
		logf(r.env)
	}
	return nil
}

func TestSetGatewayRoutes_ReadsTheBaseThenSwapsTheFileIn(t *testing.T) {
	rec := &execRecorder{env: `{"http":{"routers":{"nexul":{}}}}` + "\n"}

	require.NoError(t, setGatewayRoutes(t.Context(), rec.run, "traefik-1", appRoutes))

	require.Len(t, rec.calls, 2)
	assert.Equal(t, []string{"docker", "exec", "traefik-1", "printenv", "NEXUL_TRAEFIK_DYNAMIC"}, rec.calls[0])
	write := rec.calls[1]
	assert.Equal(t, []string{"docker", "exec", "traefik-1", "sh", "-c"}, write[:5])
	assert.Contains(t, write[5], "mv /etc/traefik/nexul.yml.tmp /etc/traefik/nexul.yml", "renamed in, never written in place")
	assert.JSONEq(t, `{"http":{"routers":{"nexul":{},"nexul-e1":{"rule":"Host(`+"`app.example.com`"+`)"}},"services":{"nexul-e1":{"url":"http://app:80"}}}}`, write[7])
}

func TestShellExecutor_SetRoutes_ReportsTheOutcome(t *testing.T) {
	boom := errors.New("no such container")
	for name, tc := range map[string]struct {
		fail   map[string]error
		status string
	}{
		"written":                 {status: BuildStatusSuccess},
		"the base cannot be read": {fail: map[string]error{"printenv": boom}, status: BuildStatusFailed},
		"the write fails":         {fail: map[string]error{"sh": boom}, status: BuildStatusFailed},
	} {
		t.Run(name, func(t *testing.T) {
			rec := &execRecorder{env: "{}", fail: tc.fail}
			rf := &frameRecorder{}

			newTestExecutor(rec.run).SetRoutes(t.Context(), "traefik-1", appRoutes, rf.send)

			last := rf.last()
			assert.Equal(t, FrameGatewayRoutesResult, last.Type)
			assert.Equal(t, "traefik-1", last.GatewayContainer)
			assert.Equal(t, tc.status, last.Status)
		})
	}
}
