package gitprovider

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestHandlePROpened(t *testing.T) {
	tests := []struct {
		name       string
		payload    string
		linkErr    error
		wantLinked []string
		wantErr    bool
	}{
		{"links every referenced ticket", `{"owner":"acme","repo":"app","pr":{"number":7,"linked_ticket_ids":["42","43"]}}`, nil, []string{"42", "43"}, false},
		{"no references means no calls", `{"owner":"acme","repo":"app","pr":{"number":7,"linked_ticket_ids":[]}}`, nil, nil, false},
		{"malformed payload is fatal", "not-json", nil, nil, true},
		{"linker failure is retryable", `{"owner":"acme","repo":"app","pr":{"number":7,"linked_ticket_ids":["42"]}}`, errors.New("ticket store down"), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linker := &fakeLinker{linkErr: tt.linkErr}
			ev := eventbus.Event{Payload: json.RawMessage(tt.payload)}
			err := HandlePROpened(context.Background(), linker, ev)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantLinked, linker.linked)
		})
	}
}

func TestHandlePROpened_Errors(t *testing.T) {
	t.Run("fatal on bad payload", func(t *testing.T) {
		err := HandlePROpened(context.Background(), &fakeLinker{}, eventbus.Event{Payload: json.RawMessage("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("retryable on linker error", func(t *testing.T) {
		linker := &fakeLinker{linkErr: errors.New("down")}
		ev := eventbus.Event{Payload: json.RawMessage(`{"pr":{"linked_ticket_ids":["42"]}}`)}
		err := HandlePROpened(context.Background(), linker, ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}
