package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
)

// A CI run as a GitHub Actions workflow posts it to a Discord webhook: author, linked title, commit lines, four
// inline fields, footer, and timestamp.
const githubActionsPost = `{
  "username": "GitHub Actions",
  "avatar_url": "https://ci.example.com/actions.png",
  "embeds": [{
    "author": {"name": "alice", "url": "https://git.example.com/alice", "icon_url": "https://git.example.com/alice.png"},
    "title": "CI · acme/app #482 passed",
    "url": "https://git.example.com/acme/app/actions/runs/482",
    "description": "**3 new commits** to ` + "`main`" + `\n[` + "`a1b2c3d`" + `](https://git.example.com/acme/app/commit/a1b2c3d) Fix the login timeout - alice\n[` + "`e4f5a6b`" + `](https://git.example.com/acme/app/commit/e4f5a6b) Bump eslint - bob\n[` + "`c7d8e9f`" + `](https://git.example.com/acme/app/commit/c7d8e9f) Retry the deploy step - lena",
    "color": 2664261,
    "fields": [
      {"name": "Repository", "value": "[acme/app](https://git.example.com/acme/app)", "inline": true},
      {"name": "Ref", "value": "main", "inline": true},
      {"name": "Event", "value": "push", "inline": true},
      {"name": "Duration", "value": "2m 41s", "inline": true}
    ],
    "footer": {"text": "GitHub Actions", "icon_url": "https://ci.example.com/favicon.png"},
    "timestamp": "2026-10-06T21:11:00.000Z"
  }]
}`

// Grafana's Discord contact point, as its notifier encodes a firing alert: content always present, even empty.
const grafanaPost = `{
  "username": "Grafana",
  "content": "",
  "embeds": [{
    "title": "[FIRING:1] HighErrorRate",
    "type": "rich",
    "url": "https://grafana.example.com/alerting/list",
    "color": 14037554,
    "footer": {"text": "Grafana v11.0.0", "icon_url": "https://grafana.example.com/public/img/fav32.png"}
  }]
}`

// Uptime Kuma's DOWN notification, its timestamp in the heartbeat's own format rather than RFC 3339.
const uptimeKumaDownPost = `{
  "username": "Uptime Kuma",
  "embeds": [{
    "title": "❌ Your service API went down. ❌",
    "color": 16711680,
    "timestamp": "2026-10-06 21:11:00.123",
    "fields": [
      {"name": "Service Name", "value": "API"},
      {"name": "Service URL", "value": "https://api.example.com"},
      {"name": "Went Offline", "value": "<t:1791321060:F>"},
      {"name": "Time (Europe/London)", "value": "2026-10-06 22:11:00"},
      {"name": "Error", "value": "connect ECONNREFUSED 10.0.0.5:443"}
    ]
  }]
}`

const pngAvatar = "data:image/png;base64,iVBORw0KGgo="

// botRoutes is the public execute route over the wired services, logging into logs.
func botRoutes(f permFixture, logs *bytes.Buffer) http.Handler {
	return botwebhook.NewExecuteHandler(f.svc.botwebhookSvc, slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))).Routes()
}

func postAs(routes http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.RemoteAddr = "203.0.113.9:51000"
	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, req)
	return rec
}

// botPath is the path part of a bot's URL; tokenOfBot its last segment.
func botPath(b *botwebhook.Bot) string { return "/api/botwebhooks/" + b.ID + "/" + tokenOfBot(b) }

func tokenOfBot(b *botwebhook.Bot) string { return b.URL[strings.LastIndex(b.URL, "/")+1:] }

// assertNoToken proves no token reached a log line, an audit row, or an outbox payload.
func assertNoToken(t *testing.T, f permFixture, logs string, tokens ...string) {
	t.Helper()
	ctx := context.Background()
	audit, err := f.store.Audit.List(ctx, 1000)
	require.NoError(t, err)
	events, err := f.store.Outbox.Unpublished(ctx, 1000)
	require.NoError(t, err)
	for _, token := range tokens {
		assert.NotContains(t, logs, token, "a log line")
		for _, a := range audit {
			assert.NotContains(t, a.Action, token, "an audit row")
		}
		for _, e := range events {
			assert.NotContains(t, string(e.Payload), token, "a %s payload", e.Topic)
		}
	}
}

// TestIntegration_BotPostsBecomeMessages posts three real senders' payloads through the URL: each is one message by the
// bot with the name and avatar it showed and its embeds without their color, one audit row as the bot, one
// chat.message.created, and the bot's counters bumped; wait=true answers with the message.
func TestIntegration_BotPostsBecomeMessages(t *testing.T) {
	f := newPermFixture(t)
	ctx := context.Background()
	var logs bytes.Buffer
	routes := botRoutes(f, &logs)
	b, err := f.svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "CI", pngAvatar)
	require.NoError(t, err)

	rec := postAs(routes, botPath(b)+"?wait=true", githubActionsPost)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var answered botwebhook.Message
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &answered))
	assert.Equal(t, f.channel.ID, answered.ChannelID)
	assert.Equal(t, b.ID, answered.WebhookID)
	assert.Equal(t, botwebhook.MessageAuthor{ID: b.ID, Username: "GitHub Actions", Avatar: "https://ci.example.com/actions.png", Bot: true}, answered.Author)
	_, err = time.Parse(time.RFC3339, answered.Timestamp)
	require.NoError(t, err)
	require.Len(t, answered.Embeds, 1)
	assert.Len(t, answered.Embeds[0].Fields, 4)
	assert.NotContains(t, rec.Body.String(), "color")
	assert.Equal(t, "29", rec.Header().Get("X-RateLimit-Remaining"))

	for _, body := range []string{grafanaPost, uptimeKumaDownPost} {
		rec := postAs(routes, botPath(b), body)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.Empty(t, rec.Body.String())
	}
	require.Equal(t, http.StatusBadRequest, postAs(routes, botPath(b), `{"content":""}`).Code)

	msgs, err := f.store.Chat.ListMessages(ctx, f.channel.ID, 50)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	ownAvatar := fmt.Sprintf("/api/botwebhooks/%s/avatar?v=%d", b.ID, b.UpdatedAt.Unix())
	for i, want := range []struct{ name, avatar string }{{"GitHub Actions", "https://ci.example.com/actions.png"}, {"Grafana", ownAvatar}, {"Uptime Kuma", ownAvatar}} {
		m := msgs[i]
		assert.Equal(t, "CI", m.Via, "%s overrides the name, so the message says which bot sent it", want.name)
		assert.Equal(t, chat.AuthorBot, m.AuthorKind, want.name)
		assert.Equal(t, b.ID, m.AuthorID, want.name)
		assert.Equal(t, want.name, m.AuthorName)
		assert.Equal(t, want.avatar, m.AuthorAvatarURL, want.name)
		assert.NotContains(t, string(m.Embeds), "color", want.name)
	}
	assert.Equal(t, answered.ID, msgs[0].ID)
	assert.JSONEq(t, `[{"title":"[FIRING:1] HighErrorRate","url":"https://grafana.example.com/alerting/list",
		"footer":{"text":"Grafana v11.0.0","icon_url":"https://grafana.example.com/public/img/fav32.png"}}]`, string(msgs[1].Embeds))
	assert.Contains(t, string(msgs[2].Embeds), `"timestamp":"2026-10-06 21:11:00.123"`)

	got, err := f.store.Botwebhooks.Get(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, got.PostCount)
	require.NotNil(t, got.LastPostAt)

	audit, err := f.store.Audit.List(ctx, 100)
	require.NoError(t, err)
	require.Len(t, audit, 3, "a post is audited, a reject never")
	for _, a := range audit {
		assert.Equal(t, "bot", a.ActorType)
		assert.Equal(t, b.ID, a.ActorID)
		assert.Equal(t, "POST /api/botwebhooks/"+b.ID, a.Action)
	}
	events, err := f.store.Outbox.Unpublished(ctx, 100)
	require.NoError(t, err)
	created := 0
	for _, e := range events {
		if e.Topic == chat.TopicMessageCreated {
			created++
		}
	}
	assert.Equal(t, 3, created)
	assertNoToken(t, f, logs.String(), tokenOfBot(b))
}

// TestIntegration_BotDeletedMidPost_WritesNothing: a bot deleted between its URL check and its post leaves no message,
// no audit row, and no count, since the post's writes share one transaction.
func TestIntegration_BotDeletedMidPost_WritesNothing(t *testing.T) {
	f := newPermFixture(t)
	ctx := context.Background()
	b, err := f.svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "CI", "")
	require.NoError(t, err)
	checked, err := f.svc.botwebhookSvc.Authenticate(ctx, b.ID, tokenOfBot(b))
	require.NoError(t, err)
	_, err = f.svc.botwebhookSvc.Delete(as(uOwner), b.ID)
	require.NoError(t, err)

	_, err = f.svc.botwebhookSvc.Execute(ctx, checked, botwebhook.Payload{Content: "build passed"})
	require.ErrorIs(t, err, botwebhook.ErrUnknownWebhook)
	msgs, err := f.store.Chat.ListMessages(ctx, f.channel.ID, 50)
	require.NoError(t, err)
	assert.Empty(t, msgs)
	audit, err := f.store.Audit.List(ctx, 100)
	require.NoError(t, err)
	assert.Empty(t, audit)
	got, err := f.store.Botwebhooks.Get(ctx, b.ID)
	require.NoError(t, err)
	assert.Zero(t, got.PostCount)
}

// TestIntegration_BotRefusedURLsAnswerAlike: a wrong id, a wrong token, a regenerated token, a deleted bot, and the
// token from before a restore all answer the same 404 body, while the current token posts at once.
func TestIntegration_BotRefusedURLsAnswerAlike(t *testing.T) {
	f := newPermFixture(t)
	var logs bytes.Buffer
	routes := botRoutes(f, &logs)
	bots := f.svc.botwebhookSvc
	first, err := bots.Create(as(uOwner), f.channel.ID, "CI", "")
	require.NoError(t, err)
	const unknown = `{"message":"Unknown Webhook","code":10015}`
	refused := func(name, path string) {
		t.Helper()
		rec := postAs(routes, path, `{"content":"build passed"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code, name)
		assert.Equal(t, unknown, strings.TrimSpace(rec.Body.String()), name)
	}
	posts := func(name string, b *botwebhook.Bot) {
		t.Helper()
		assert.Equal(t, http.StatusNoContent, postAs(routes, botPath(b), `{"content":"build passed"}`).Code, name)
	}

	refused("a wrong id", "/api/botwebhooks/b-missing/"+tokenOfBot(first))
	refused("a wrong token", "/api/botwebhooks/"+first.ID+"/"+strings.Repeat("A", len(tokenOfBot(first))))
	posts("the bot's own URL", first)

	regenerated, err := bots.Update(as(uOwner), first.ID, botwebhook.Changes{Regenerate: true})
	require.NoError(t, err)
	refused("the token before a regenerate", botPath(first))
	posts("the regenerated URL", regenerated)

	_, err = bots.Delete(as(uOwner), first.ID)
	require.NoError(t, err)
	refused("a deleted bot", botPath(regenerated))

	restore := false
	restored, err := bots.Update(as(uOwner), first.ID, botwebhook.Changes{Deleted: &restore})
	require.NoError(t, err)
	refused("the token before a delete and restore", botPath(regenerated))
	posts("the restored URL", restored)

	assertNoToken(t, f, logs.String(), tokenOfBot(first), tokenOfBot(regenerated), tokenOfBot(restored))
	assert.Contains(t, logs.String(), "unknown webhook", "rejects are logged")
}

// TestIntegration_BotMentions: a bot's @handles mention the members the sender allowed, a handle it did not allow or
// nobody holds stays plain text, and @Agent never becomes the mention that starts a turn.
func TestIntegration_BotMentions(t *testing.T) {
	f := newPermFixture(t)
	ctx := context.Background()
	routes := botRoutes(f, &bytes.Buffer{})
	b, err := f.svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "Pager", "")
	require.NoError(t, err)
	const content = "@u-plain @u-reader and @u-nobody, ask @Agent"
	for _, body := range []string{
		`{"content":"` + content + `"}`,
		`{"content":"` + content + `","allowed_mentions":{"users":["u-plain"]}}`,
		`{"content":"` + content + `","allowed_mentions":{"parse":[]}}`,
	} {
		require.Equal(t, http.StatusNoContent, postAs(routes, botPath(b), body).Code)
	}

	msgs, err := f.store.Chat.ListMessages(ctx, f.channel.ID, 50)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	handles := func(m *chat.Message) []string {
		out := []string{}
		for _, mention := range m.Mentions {
			assert.Equal(t, chat.MentionUser, mention.Kind)
			out = append(out, mention.Handle)
		}
		return out
	}
	assert.Equal(t, []string{uPlain, uReader}, handles(msgs[0]), "absent allowed_mentions mentions every member named")
	assert.Equal(t, []string{uPlain}, handles(msgs[1]))
	assert.Empty(t, handles(msgs[2]))
	for _, m := range msgs {
		assert.Empty(t, m.Via, "no username override, no via")
		assert.Equal(t, chat.AuthorBot, m.AuthorKind, "the agent pipeline starts turns only on a person's message")
		assert.Equal(t, content, m.Body)
	}
}

// TestIntegration_BotAvatarFollowsConversationReaders: anyone who reads the bot's conversation sees its avatar, with
// no botwebhook permission; nobody else does, and a bot without one answers 404.
func TestIntegration_BotAvatarFollowsConversationReaders(t *testing.T) {
	f := newPermFixture(t)
	routes := botwebhook.NewHandler(f.svc.botwebhookSvc).Routes()
	withAvatar, err := f.svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "CI", pngAvatar)
	require.NoError(t, err)
	without, err := f.svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "Pager", "")
	require.NoError(t, err)
	inDM, err := f.svc.botwebhookSvc.Create(as(uOwner), f.dm.ID, "Reminders", pngAvatar)
	require.NoError(t, err)
	get := func(user string, b *botwebhook.Bot) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(as(user), http.MethodGet, "/api/botwebhooks/"+b.ID+"/avatar", nil)
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	rec := get(uPlain, withAvatar)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "\x89PNG\r\n\x1a\n", rec.Body.String())

	assert.Equal(t, http.StatusNotFound, get(uPlain, without).Code)
	assert.Equal(t, http.StatusNotFound, get(uOutsider, withAvatar).Code)
	assert.Equal(t, http.StatusNotFound, get(uPlain, inDM).Code, "a DM's bot stays with its participants")
	assert.Equal(t, http.StatusOK, get(uWriter, inDM).Code)
}

// TestIntegration_BotMediaFollowsConversationReaders reads the posted message through chat's own read rule: a DM's
// image reaches only its participants and only for a URL the message shows, and the wired guard never dials loopback.
func TestIntegration_BotMediaFollowsConversationReaders(t *testing.T) {
	f := newPermFixture(t)
	var hits atomic.Int32
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\n"))
	}))
	t.Cleanup(host.Close)
	inDM, err := f.svc.botwebhookSvc.Create(as(uOwner), f.dm.ID, "Reminders", "")
	require.NoError(t, err)
	image := host.URL + "/chart.png"
	rec := postAs(botRoutes(f, &bytes.Buffer{}), botPath(inDM)+"?wait=true", `{"embeds":[{"title":"Chart","image":{"url":"`+image+`"}}]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var posted botwebhook.Message
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &posted))
	routes := botwebhook.NewHandler(f.svc.botwebhookSvc).Routes()
	get := func(user, raw string) int {
		target := "/api/botwebhooks/media?message=" + url.QueryEscape(posted.ID) + "&url=" + url.QueryEscape(raw)
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequestWithContext(as(user), http.MethodGet, target, nil))
		return rec.Code
	}

	assert.Equal(t, http.StatusNotFound, get(uPlain, image), "a DM's images stay with its participants")
	assert.Equal(t, http.StatusNotFound, get(uWriter, host.URL+"/other.png"))
	assert.Equal(t, http.StatusBadGateway, get(uWriter, image), "a participant passes the gate, and the guard refuses loopback")
	assert.Zero(t, hits.Load())
}
