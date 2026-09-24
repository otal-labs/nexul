package memories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopics(t *testing.T) {
	assert.ElementsMatch(t, []string{TopicCreated, TopicUpdated, TopicDeleted, TopicInterviewTemplateUpdated}, Topics())
}
