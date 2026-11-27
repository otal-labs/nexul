package plays

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnlyFull_AComputerLeftOutByTheRuleDoesNotMakeTheRunRetryEveryMinute(t *testing.T) {
	full, ruled, offline := PassedOver{Why: "full"}, PassedOver{Why: "rule_out"}, PassedOver{Why: "offline"}
	assert.True(t, onlyFull([]PassedOver{full}))
	assert.True(t, onlyFull([]PassedOver{ruled, full}), "it waits for a freed slot on the computers the rule allows")
	assert.True(t, onlyFull([]PassedOver{{Why: "no_auto_plays"}, full}), "a computer that takes no auto plays is no obstacle either")
	assert.False(t, onlyFull([]PassedOver{ruled}), "nothing allowed is full, so the run tries again on a timer")
	assert.False(t, onlyFull([]PassedOver{full, offline}))
	assert.False(t, onlyFull(nil))
}
