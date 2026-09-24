package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/otal-labs/nexul/internal/auth"
)

func TestLivePushTopics_TokenLifecycle_ReachesTheBrowser(t *testing.T) {
	assert.Contains(t, livePushTopics, auth.TopicTokenMinted)
	assert.Contains(t, livePushTopics, auth.TopicTokenRevoked)
}
