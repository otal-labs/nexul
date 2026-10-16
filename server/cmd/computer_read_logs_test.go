package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/runner"
)

func TestComputerRules_ReadLogsIsOffByDefaultImpliesSeeAndGoesWithIt(t *testing.T) {
	c := newSharingCast(t)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	held, err := c.held(t, "u-dave")
	require.NoError(t, err)
	assert.False(t, held.Has(permissions.ComputersLogs), "a share is See alone until the owner adds more")

	code, body = c.share(t, c.alice, `"user_ids":["u-bob"],"actions":["computers:read_logs"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)
	held, err = c.held(t, "u-bob")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersLogs), held, "Read logs alone brings See along")

	code, body = c.share(t, c.alice, `"user_ids":["u-bob"],"actions":["computers:see"],"state":"fallback"`)
	require.Equal(t, http.StatusNoContent, code, body)
	_, err = c.held(t, "u-bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "taking See off takes Read logs with it")
}

type seenFrames struct {
	mu     sync.Mutex
	frames []runner.Frame
}

func (s *seenFrames) add(f runner.Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames = append(s.frames, f)
}

func (s *seenFrames) last() runner.Frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frames[len(s.frames)-1]
}

func dialRunnerLogs(ctx context.Context, srv *httptest.Server, computerID, token string) (*websocket.Conn, *http.Response, error) {
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/pairing/computers/"+computerID+"/runner/logs?tail=20&token="+url.QueryEscape(token), nil)
}

// TestComputerReadLogs_OwnerOrAShareThatAllowsIt: a runner's own log and its containers are the owner's, and a person
// the owner's rule gives Read logs reads both over HTTP, MCP and the sockets, masked; See alone, no rule, and a role
// holding every bit reach nothing, and nothing of theirs reaches the computer.
func TestComputerReadLogs_OwnerOrAShareThatAllowsIt(t *testing.T) {
	c := newSharingCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	seen := &seenFrames{}
	requests := connectRecordingComputerRunner(t, c.privacyCast, srv, c.laptop, seen.add)
	reportDocker(t, c.privacyCast, c.laptop, true)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	containers := "/api/pairing/computers/" + c.laptop + "/containers"
	containerLogs := containers + "/shop-web-1/logs?tail=5"
	runnerLogs := "/api/pairing/computers/" + c.laptop + "/runner/logs?tail=5"
	runnerTool := `{"id":"` + c.laptop + `","runner_logs":{"lines":5}}`
	containerTool := `{"id":"` + c.laptop + `","container_logs":{"container":"shop-web-1","tail":5}}`

	t.Run("the owner reads the runner's log unmasked", func(t *testing.T) {
		rec := callAPI(t, c.h, http.MethodGet, runnerLogs, c.alice, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "runner connected")
		assert.Equal(t, runner.Frame{Type: runner.FrameLogsRequest, ID: seen.last().ID, Target: runner.LogTargetRunner, Tail: 5}, seen.last())
		assert.Contains(t, callTool(t, c.h, c.alice, "computer_list", runnerTool), "runner connected")

		rec = callAPI(t, c.h, http.MethodGet, containerLogs, c.alice, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.False(t, seen.last().Mask, "the owner's own output is not masked")
	})

	t.Run("a share of See alone, no share, and every bit reach nothing", func(t *testing.T) {
		before := requests.Load()
		for who, token := range map[string]string{"dave with See alone": c.dave, "erin sharing no workspace": c.erin, "workspace Owner": c.owner, "every bit": c.everyBits} {
			for _, path := range []string{runnerLogs, containers, containerLogs} {
				rec := callAPI(t, c.h, http.MethodGet, path, token, "")
				assert.Equal(t, http.StatusNotFound, rec.Code, "GET %s as %s", path, who)
				assert.NotContains(t, rec.Body.String(), "shop-web-1", "GET %s as %s", path, who)
			}
			for _, args := range []string{runnerTool, containerTool} {
				body := callToolResult(t, c.h, token, "computer_list", args)
				assert.Contains(t, body, "not found", "computer_list as %s", who)
				assert.NotContains(t, body, "listening", "computer_list as %s", who)
				assert.NotContains(t, body, "runner connected", "computer_list as %s", who)
			}
			_, resp, err := dialRunnerLogs(t.Context(), srv, c.laptop, token)
			require.Error(t, err, "the runner log socket as %s", who)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "the runner log socket as %s", who)
		}
		assert.Equal(t, before, requests.Load(), "no call of theirs reached alice's computer")
	})

	code, body = c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:read_logs"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	t.Run("a share that allows Read logs reads the runner's log, the containers and their logs, masked", func(t *testing.T) {
		rec := callAPI(t, c.h, http.MethodGet, runnerLogs, c.dave, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "runner connected")

		rec = callAPI(t, c.h, http.MethodGet, containers, c.dave, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "shop-web-1")

		rec = callAPI(t, c.h, http.MethodGet, containerLogs, c.dave, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.True(t, seen.last().Mask, "anyone but the owner reads a container's logs masked")

		body := callTool(t, c.h, c.dave, "computer_list", `{"id":"`+c.laptop+`","runner_logs":{},"container_logs":{"container":"shop-web-1"}}`)
		assert.Contains(t, body, "runner connected")
		assert.Contains(t, body, "listening on :8080")
		assert.Contains(t, body, "shop-web-1", "the container list too")

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		viewer, _, err := dialRunnerLogs(ctx, srv, c.laptop, c.dave)
		require.NoError(t, err)
		var msg struct {
			Lines []struct {
				Line string `json:"line"`
			} `json:"lines"`
		}
		require.NoError(t, wsjson.Read(ctx, viewer, &msg))
		require.Len(t, msg.Lines, 1)
		assert.Contains(t, msg.Lines[0].Line, "runner connected")
		assert.True(t, seen.last().Follow)
		_ = viewer.Close(websocket.StatusNormalClosure, "")
	})

	t.Run("Read logs stops with the share's See", func(t *testing.T) {
		code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:see"],"state":"fallback"`)
		require.Equal(t, http.StatusNoContent, code, body)
		for _, path := range []string{runnerLogs, containers, containerLogs} {
			rec := callAPI(t, c.h, http.MethodGet, path, c.dave, "")
			assert.Equal(t, http.StatusNotFound, rec.Code, path)
		}
	})
}

func TestComputerReadLogs_ARunnerLogNeedsNoDockerButTheContainersDo(t *testing.T) {
	c := newSharingCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	connectComputerRunner(t, c.privacyCast, srv, c.laptop)
	reportDocker(t, c.privacyCast, c.laptop, false)
	code, body := c.share(t, c.alice, `"user_ids":["u-dave"],"actions":["computers:read_logs"],"state":"allow"`)
	require.Equal(t, http.StatusNoContent, code, body)

	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+c.laptop+"/runner/logs", c.dave, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+c.laptop+"/containers", c.dave, "")
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
}
