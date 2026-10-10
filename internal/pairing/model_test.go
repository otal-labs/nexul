package pairing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotConfiguredError_ReadsAsAPlainSentenceAndKeepsItsReason(t *testing.T) {
	tests := []struct {
		reason NotConfiguredReason
		want   string
	}{
		{ReasonUnpaired, "no computer is paired to run it on; add one in Settings → Computers"},
		{ReasonExpiredToken, "the paired computer's session expired; pair it again in Settings → Computers"},
		{ReasonNoDefault, "no T3 project is picked for this project on the paired computer; link one in Settings → Computers → Projects, or set a fallback under Defaults"},
		{ReasonNoDefaultComputer, "several computers are paired and none is the default; pick one under Defaults in Settings → Computers"},
		{NotConfiguredReason("future_reason"), "pairing not configured: future_reason"},
	}
	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			err := &NotConfiguredError{Reason: tt.reason}
			assert.Equal(t, tt.want, err.Error())
			assert.Equal(t, RefusalDetails{Reason: tt.reason}, err.ErrorDetails())
		})
	}
}
