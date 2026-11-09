package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/otal-labs/nexul/internal/auth"
)

func TestLivePushTopics_TokenLifecycle_ReachesTheBrowser(t *testing.T) {
	t.Parallel()
	assert.Contains(t, bridgedLiveTopics, auth.TopicTokenMinted)
	assert.Contains(t, bridgedLiveTopics, auth.TopicTokenRevoked)
}

func TestLivePushTopics_SessionLifecycle_ReachesTheBrowser(t *testing.T) {
	t.Parallel()
	assert.Contains(t, bridgedLiveTopics, auth.TopicSessionCreated)
	assert.Contains(t, bridgedLiveTopics, auth.TopicSessionRevoked)
}
