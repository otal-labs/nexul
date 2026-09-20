package automations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestFrame_Validate(t *testing.T) {
	tests := []struct {
		name    string
		frame   Frame
		wantErr bool
	}{
		{"missing type", Frame{}, true},
		{"unknown type", Frame{Type: "bogus"}, true},
		{"announce valid", Frame{Type: FrameAnnounce, Name: "PR notifier"}, false},
		{"announce missing name", Frame{Type: FrameAnnounce}, true},
		{"hello always valid", Frame{Type: FrameHello}, false},
		{"event valid", Frame{Type: FrameEvent, RunID: "r1", EventID: "e1", Topic: "ticket.created"}, false},
		{"event missing run_id", Frame{Type: FrameEvent, EventID: "e1", Topic: "ticket.created"}, true},
		{"event missing event_id", Frame{Type: FrameEvent, RunID: "r1", Topic: "ticket.created"}, true},
		{"event missing topic", Frame{Type: FrameEvent, RunID: "r1", EventID: "e1"}, true},
		{"run_started valid", Frame{Type: FrameRunStarted, RunID: "r1"}, false},
		{"run_started missing run_id", Frame{Type: FrameRunStarted}, true},
		{"run_log valid", Frame{Type: FrameRunLog, RunID: "r1", Log: "hi"}, false},
		{"run_log missing run_id", Frame{Type: FrameRunLog}, true},
		{"run_finished success", Frame{Type: FrameRunFinished, RunID: "r1", Outcome: OutcomeSuccess}, false},
		{"run_finished failure", Frame{Type: FrameRunFinished, RunID: "r1", Outcome: OutcomeFailure}, false},
		{"run_finished missing run_id", Frame{Type: FrameRunFinished, Outcome: OutcomeSuccess}, true},
		{"run_finished bad outcome", Frame{Type: FrameRunFinished, RunID: "r1", Outcome: "maybe"}, true},
		{"run_crashed valid", Frame{Type: FrameRunCrashed, RunID: "r1", Error: "boom"}, false},
		{"run_crashed missing run_id", Frame{Type: FrameRunCrashed}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.frame.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, apperrs.ErrInvalid)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestParseFrame_InvalidJSON_ReturnsError(t *testing.T) {
	_, err := ParseFrame([]byte("not json"))
	require.Error(t, err)
}

func TestParseFrame_InvalidFrame_ReturnsError(t *testing.T) {
	_, err := ParseFrame([]byte(`{"type":"bogus"}`))
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestParseFrame_Valid_Roundtrips(t *testing.T) {
	f, err := ParseFrame([]byte(`{"type":"announce","name":"PR notifier"}`))
	require.NoError(t, err)
	assert.Equal(t, FrameAnnounce, f.Type)
	assert.Equal(t, "PR notifier", f.Name)
}

func TestFrame_Encode_InvalidFrame_ReturnsError(t *testing.T) {
	f := Frame{Type: FrameAnnounce}
	_, err := f.Encode()
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestFrame_Encode_ValidFrame_Roundtrips(t *testing.T) {
	f := Frame{Type: FrameRunFinished, RunID: "r1", Outcome: OutcomeSuccess}
	data, err := f.Encode()
	require.NoError(t, err)

	got, err := ParseFrame(data)
	require.NoError(t, err)
	assert.Equal(t, f, *got)
}
