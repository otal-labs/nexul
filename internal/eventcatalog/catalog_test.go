package eventcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllTopics_NoDuplicates_SortedAndPopulated(t *testing.T) {
	topics := AllTopics()
	require.NotEmpty(t, topics)

	seen := map[string]bool{}
	for i, topic := range topics {
		assert.False(t, seen[topic], "duplicate topic %q", topic)
		seen[topic] = true
		if i > 0 {
			assert.Less(t, topics[i-1], topic, "topics must be sorted")
		}
	}
}

func TestAllTopics_IncludesTicket07Additions(t *testing.T) {
	topics := AllTopics()
	assert.Contains(t, topics, "ticket.deleted")
	assert.Contains(t, topics, "doc.deleted")
	assert.Contains(t, topics, "git.pr_comment")
}

func TestAllTopics_IncludesInstanceUpgrade(t *testing.T) {
	topics := AllTopics()
	assert.Contains(t, topics, "instance.upgrade_changed")
	assert.Contains(t, topics, "instance.upgrade_requested")
}

func TestAllTopics_IncludesMemories(t *testing.T) {
	topics := AllTopics()
	assert.Contains(t, topics, "memory.created")
	assert.Contains(t, topics, "memory.updated")
	assert.Contains(t, topics, "memory.deleted")
}

func TestAllTopics_IncludesComputerSetup(t *testing.T) {
	topics := AllTopics()
	assert.Contains(t, topics, "computer.setup_confirmed")
	assert.Contains(t, topics, "computer.setup_unconfirmed")
}

func TestAllTopics_IncludesPlayRuns(t *testing.T) {
	topics := AllTopics()
	assert.Contains(t, topics, "play.run_started")
	assert.Contains(t, topics, "play.run_waiting")
	assert.Contains(t, topics, "play.run_finished")
	assert.NotContains(t, topics, "play.run", "the live topic is ephemeral, never catalogued")
}

func TestAllTopics_IncludesInvitationLifecycle(t *testing.T) {
	topics := AllTopics()
	assert.Contains(t, topics, "invitation.created")
	assert.Contains(t, topics, "invitation.deleted")
	assert.Contains(t, topics, "account.admitted")
	assert.Contains(t, topics, "workspace.member.added")
}

func TestAllTopics_IncludesDeployLog(t *testing.T) {
	assert.Contains(t, AllTopics(), "deploy.log")
	assert.Contains(t, AllTopics(), "deploy.updated")
}
