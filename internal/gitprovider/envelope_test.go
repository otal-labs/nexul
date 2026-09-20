package gitprovider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestNormalizeProviderEvent(t *testing.T) {
	tests := []struct {
		name        string
		body        []byte
		deliveryID  string
		wantAction  string
		wantRepo    *RepositoryRef
		wantPayload jsonRaw
	}{
		{
			name:       "full repository extracted",
			body:       []byte(`{"action":"opened","repository":{"id":1,"name":"app","full_name":"acme/app","owner":{"login":"acme"},"html_url":"https://github.com/acme/app","default_branch":"main"}}`),
			deliveryID: "d1",
			wantAction: "opened",
			wantRepo: &RepositoryRef{
				ID:            1,
				Name:          "app",
				FullName:      "acme/app",
				Owner:         "acme",
				HTMLURL:       "https://github.com/acme/app",
				DefaultBranch: "main",
			},
			wantPayload: jsonRaw(`{"action":"opened","repository":{"id":1,"name":"app","full_name":"acme/app","owner":{"login":"acme"},"html_url":"https://github.com/acme/app","default_branch":"main"}}`),
		},
		{
			name:        "no repository leaves field nil",
			body:        []byte(`{"action":"deleted"}`),
			deliveryID:  "d2",
			wantAction:  "deleted",
			wantRepo:    nil,
			wantPayload: jsonRaw(`{"action":"deleted"}`),
		},
		{
			name:        "no action leaves field empty",
			body:        []byte(`{}`),
			wantAction:  "",
			wantPayload: jsonRaw(`{}`),
		},
		{
			name:       "invalid json is best-effort",
			body:       []byte(`not json`),
			wantAction: "",
			wantRepo:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewWebhookHandler("", newFakeBus())
			before := time.Now().UTC()
			env := h.normalizeProviderEvent("push", tt.deliveryID, tt.body)
			after := time.Now().UTC()

			assert.Equal(t, providerGitHub, env.Provider)
			assert.Equal(t, "push", env.EventType)
			assert.Equal(t, tt.deliveryID, env.DeliveryID)
			assert.Equal(t, tt.wantAction, env.Action)
			assert.Equal(t, tt.wantRepo, env.Repository)
			assert.True(t, !env.ReceivedAt.Before(before) && !env.ReceivedAt.After(after), "received_at should be ~now")
			if tt.wantPayload != nil {
				assert.Equal(t, json.RawMessage(tt.wantPayload), env.Payload)
			}
		})
	}
}

func TestWebhookHandler_Opened_PublishesEnvelopeThenTyped(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("opened", false)))

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, bus.published, 2)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
	assert.Equal(t, TopicPROpened, bus.published[1].Topic)

	env := envelopeFrom(t, bus)
	assert.Equal(t, "github", env.Provider)
	assert.Equal(t, "pull_request", env.EventType)
	assert.Equal(t, "delivery-123", env.DeliveryID)
	assert.Equal(t, "opened", env.Action)
	require.NotNil(t, env.Repository)
	assert.Equal(t, "acme", env.Repository.Owner)
	assert.Equal(t, "app", env.Repository.Name)
	assert.JSONEq(t, string(prEventPayload("opened", false)), string(env.Payload))
}

func TestWebhookHandler_EnvelopePublishFailure_Returns500(t *testing.T) {
	bus := newFakeBus()
	bus.publishErr = errors.New("bus closed")
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "ping", []byte(`{"zen":"ok"}`)))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Empty(t, bus.published)
}

func TestWebhookHandler_TypedPublishFailure_Returns500(t *testing.T) {
	bus := newFakeBus()
	require.NoError(t, bus.Subscribe(context.Background(), TopicPROpened, func(context.Context, eventbus.Event) error {
		return errors.New("consumer unavailable")
	}))
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("opened", false)))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, TopicProviderEvent, bus.published[0].Topic)
}

func TestWebhookHandler_InvalidJSONBody_Rejected(t *testing.T) {
	bus := newFakeBus()
	h := NewWebhookHandler("shhh", bus)
	rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "push", []byte(`not json`)))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, bus.published)
}

// jsonRaw is a named []byte so the table literal syntax reads like JSON.
type jsonRaw = []byte
