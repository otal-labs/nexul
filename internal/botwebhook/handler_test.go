package botwebhook

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandler_Routes drives each gateway route as the HTTP client sends it: methods, paths, the deleted query, and
// the status each answer carries.
func TestHandler_Routes(t *testing.T) {
	repo := newFakeRepo()
	routes := NewHandler(newTestService(repo)).Routes()
	serve := func(user, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(as(user), method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	rec := serve(editor, http.MethodPost, "/api/conversations/c-eng/botwebhooks", `{"name":"CI"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created Bot
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Contains(t, created.URL, "/api/botwebhooks/"+created.ID+"/")
	assert.NotContains(t, rec.Body.String(), `"token"`)

	assert.Equal(t, http.StatusBadRequest, serve(editor, http.MethodPost, "/api/conversations/c-eng/botwebhooks", `{`).Code)

	rec = serve(editor, http.MethodPatch, "/api/botwebhooks/"+created.ID, `{"name":"Builds"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"name":"Builds"`)
	assert.Equal(t, http.StatusBadRequest, serve(editor, http.MethodPatch, "/api/botwebhooks/"+created.ID, `[`).Code)
	assert.Equal(t, http.StatusNotFound, serve(editor, http.MethodPatch, "/api/botwebhooks/b-gone", `{}`).Code)

	assert.Equal(t, http.StatusForbidden, serve(editor, http.MethodDelete, "/api/botwebhooks/"+created.ID, "").Code)
	assert.Equal(t, http.StatusNoContent, serve(deleter, http.MethodDelete, "/api/botwebhooks/"+created.ID, "").Code)

	rec = serve(editor, http.MethodGet, "/api/conversations/c-eng/botwebhooks?deleted=true", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var deleted []Bot
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &deleted))
	require.Len(t, deleted, 1)
	assert.Equal(t, created.ID, deleted[0].ID)
	assert.Empty(t, deleted[0].URL)
}

// executeFixture is the public route over the fakes, its limiters on a clock the test moves.
type executeFixture struct {
	routes http.Handler
	poster *fakePoster
	bot    *Bot
	token  string
	now    time.Time
}

func newExecuteFixture(t *testing.T) *executeFixture {
	t.Helper()
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")
	h := NewExecuteHandler(s, slog.New(slog.NewTextHandler(io.Discard, nil)))
	f := &executeFixture{routes: h.Routes(), poster: s.poster.(*fakePoster), bot: b, token: tokenOf(t, repo, b.ID), now: fixedNow}
	h.posts.now = func() time.Time { return f.now }
	h.rejects.now = func() time.Time { return f.now }
	return f
}

func (f *executeFixture) post(addr, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.RemoteAddr = addr + ":40000"
	rec := httptest.NewRecorder()
	f.routes.ServeHTTP(rec, req)
	return rec
}

func (f *executeFixture) url() string { return "/api/botwebhooks/" + f.bot.ID + "/" + f.token }

// TestExecuteHandler_PerBotLimit takes 30 posts a minute per bot with Discord's headers, answers the 31st with
// Discord's 429, never spends a slot on a rejected post, and frees a slot once the oldest post leaves the minute.
func TestExecuteHandler_PerBotLimit(t *testing.T) {
	f := newExecuteFixture(t)
	assert.Equal(t, http.StatusBadRequest, f.post("192.0.2.1", f.url(), `{"content":" "}`).Code)
	for i := range postsPerMinute {
		rec := f.post("192.0.2.1", f.url(), `{"content":"build passed"}`)
		require.Equal(t, http.StatusNoContent, rec.Code, "post %d", i)
		assert.Equal(t, "30", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, fmt.Sprint(postsPerMinute-1-i), rec.Header().Get("X-RateLimit-Remaining"))
		f.now = f.now.Add(time.Second)
	}
	rec := f.post("192.0.2.1", f.url(), `{"content":"build passed"}`)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.JSONEq(t, `{"message":"You are being rate limited.","retry_after":30,"global":false}`, rec.Body.String())
	assert.Equal(t, "30", rec.Header().Get("Retry-After"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "30.000", rec.Header().Get("X-RateLimit-Reset-After"))
	assert.Len(t, f.poster.posts, postsPerMinute)

	f.now = f.now.Add(30 * time.Second)
	assert.Equal(t, http.StatusNoContent, f.post("192.0.2.1", f.url(), `{"content":"build passed"}`).Code)
}

// TestExecuteHandler_PerAddressRejects answers an address past 60 rejected requests a minute with 429 before any work,
// even on a good URL, while another address still posts.
func TestExecuteHandler_PerAddressRejects(t *testing.T) {
	f := newExecuteFixture(t)
	for i := range rejectsPerMinute {
		path := f.url() + "x"
		if i%2 == 0 {
			path = "/api/botwebhooks/b-missing/" + f.token
		}
		require.Equal(t, http.StatusNotFound, f.post("198.51.100.7", path, `{"content":"hi"}`).Code, "reject %d", i)
	}
	rec := f.post("198.51.100.7", f.url(), `{"content":"hi"}`)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("X-RateLimit-Limit"))
	assert.Empty(t, f.poster.posts, "a refused address does no work")

	assert.Equal(t, http.StatusNoContent, f.post("198.51.100.8", f.url(), `{"content":"hi"}`).Code)
	f.now = f.now.Add(time.Minute)
	assert.Equal(t, http.StatusNoContent, f.post("198.51.100.7", f.url(), `{"content":"hi"}`).Code)
}

// TestExecuteHandler_BadBodies answers a body Nexul cannot post with 400 and its normal error body, naming what broke.
func TestExecuteHandler_BadBodies(t *testing.T) {
	f := newExecuteFixture(t)
	for name, tc := range map[string]struct{ body, says string }{
		"oversize":  {`{"content":"` + strings.Repeat("a", MaxBodyBytes) + `"}`, "64 KiB"},
		"not JSON":  {`content=hi`, "not a JSON post"},
		"empty":     {`{"content":"","embeds":[]}`, "content or an embed"},
		"too long":  {`{"content":"` + strings.Repeat("a", 2001) + `"}`, "content is 2001 characters"},
		"bad embed": {`{"embeds":[{"title":"t","color":"red","image":{"url":"ftp://example.com/a.png"}}]}`, "embeds[0].image.url"},
	} {
		rec := f.post("192.0.2.1", f.url(), tc.body)
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
		var env struct{ Message, Code string }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), name)
		assert.Equal(t, "INVALID", env.Code, name)
		assert.Contains(t, env.Message, tc.says, name)
	}
	assert.Empty(t, f.poster.posts)
}
