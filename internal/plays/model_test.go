package plays

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestPlay_Validate(t *testing.T) {
	progress := StageProgress
	bogus := Stage("bogus")
	tests := []struct {
		name    string
		play    Play
		wantErr bool
	}{
		{"valid ticket play", Play{Label: "Fix", Type: TypeTicket, ShowWhenStage: &progress}, false},
		{"valid doc play", Play{Label: "Doc", Type: TypeDoc}, false},
		{"empty label", Play{Label: "  ", Type: TypeDoc}, true},
		{"invalid type", Play{Label: "X", Type: "bogus"}, true},
		{"ticket play missing stage", Play{Label: "Fix", Type: TypeTicket}, true},
		{"ticket play invalid stage", Play{Label: "Fix", Type: TypeTicket, ShowWhenStage: &bogus}, true},
		{"doc play with stage", Play{Label: "Doc", Type: TypeDoc, ShowWhenStage: &progress}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.play.Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, apperrs.ErrInvalid)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStage_Valid(t *testing.T) {
	assert.True(t, StageBacklog.valid())
	assert.True(t, StageDone.valid())
	assert.False(t, Stage("nope").valid())
}

func TestTopics_ListsEveryPublishedEvent(t *testing.T) {
	assert.ElementsMatch(t, []string{"play.created", "play.updated", "play.deleted", "play.run_started", "play.run_waiting", "play.run_finished"}, Topics())
}
