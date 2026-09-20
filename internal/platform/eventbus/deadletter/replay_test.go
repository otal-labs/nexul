package deadletter_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
)

type recordingPublisher struct {
	mu            sync.Mutex
	topics        []string
	payloads      []json.RawMessage
	ids           []string
	failOnPublish bool
}

func (r *recordingPublisher) Publish(ctx context.Context, topic string, payload any) error {
	return r.publish("", topic, payload)
}

func (r *recordingPublisher) PublishWithID(ctx context.Context, id, topic string, payload any) error {
	return r.publish(id, topic, payload)
}

func (r *recordingPublisher) publish(id, topic string, payload any) error {
	if r.failOnPublish {
		return errors.New("publish refused")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.topics = append(r.topics, topic)
	r.payloads = append(r.payloads, payload.(json.RawMessage))
	r.ids = append(r.ids, id)
	return nil
}

func (r *recordingPublisher) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.topics)
}

func TestReplay_PublishesPayloadAndDeletes(t *testing.T) {
	a := testutil.NewStore(t).DeadLetters
	ctx := context.Background()
	require.NoError(t, a.Put(ctx, deadletter.DeadLetter{ID: "dl-1", Topic: "order.created", Payload: []byte(`{"n":1}`), Error: "e", Attempts: 2}))

	pub := &recordingPublisher{}
	require.NoError(t, deadletter.Replay(ctx, a, pub, "dl-1"))

	assert.Equal(t, 1, pub.calls())
	assert.Equal(t, []string{"order.created"}, pub.topics)
	assert.Equal(t, `{"n":1}`, string(pub.payloads[0]))
	assert.Equal(t, []string{"dl-1"}, pub.ids, "replay must preserve the dead letter's ID as the bus event ID")

	_, err := a.Get(ctx, "dl-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestReplay_MissingDeadLetter_ReturnsNotFound(t *testing.T) {
	a := testutil.NewStore(t).DeadLetters
	pub := &recordingPublisher{}
	err := deadletter.Replay(context.Background(), a, pub, "nope")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, 0, pub.calls())
}

func TestReplay_PublishFailure_KeepsDeadLetter(t *testing.T) {
	a := testutil.NewStore(t).DeadLetters
	ctx := context.Background()
	require.NoError(t, a.Put(ctx, deadletter.DeadLetter{ID: "dl-1", Topic: "t", Payload: []byte(`{}`), Error: "e", Attempts: 1}))

	pub := &recordingPublisher{failOnPublish: true}
	err := deadletter.Replay(ctx, a, pub, "dl-1")
	require.Error(t, err)

	dl, gerr := a.Get(ctx, "dl-1")
	require.NoError(t, gerr)
	assert.Equal(t, "dl-1", dl.ID)
}
