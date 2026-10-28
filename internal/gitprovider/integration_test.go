package gitprovider

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Integration: signed webhook delivery -> bus event -> ticket linking handler.
func TestIntegration_WebhookToTicketLinking(t *testing.T) {
	t.Run("opened PR links tickets", func(t *testing.T) {
		bus := newFakeBus()
		linker := &fakeLinker{}
		require.NoError(t, bus.Subscribe(context.Background(), TopicPROpened, func(ctx context.Context, ev eventbus.Event) error {
			return HandlePROpened(ctx, linker, ev)
		}))

		h := NewWebhookHandler("shhh", bus)
		rec := serveWebhook(t, h, signedWebhookRequest(t, "shhh", "pull_request", prEventPayload("opened", false)))

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, TopicPROpened, bus.lastPublished().Topic)
		assert.Equal(t, []string{"42"}, linker.linked)
	})

}
