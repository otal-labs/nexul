package gitprovider

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOpenPRs struct {
	refs []PRRef
	err  error
}

func (f fakeOpenPRs) ListOpenPRs(context.Context) ([]PRRef, error) { return f.refs, f.err }

// fakePRReader answers GetPR by number; a number with no entry fails like an unreachable host.
type fakePRReader map[int]*PR

func (f fakePRReader) GetPR(_ context.Context, _, _ string, number int) (*PR, error) {
	pr, ok := f[number]
	if !ok {
		return nil, errors.New("host unreachable")
	}
	return pr, nil
}

func TestReconcilePRsOnce_PublishesWhatTheWebhookMissed(t *testing.T) {
	tests := []struct {
		name       string
		open       fakeOpenPRs
		host       fakePRReader
		wantTopics []string
	}{
		{"merged pr publishes pr_merged", fakeOpenPRs{refs: []PRRef{{Owner: "o", Repo: "r", Number: 1}}},
			fakePRReader{1: {Number: 1, State: PRStateClosed, Merged: true}}, []string{TopicPRMerged}},
		{"closed unmerged pr publishes pr_closed", fakeOpenPRs{refs: []PRRef{{Owner: "o", Repo: "r", Number: 2}}},
			fakePRReader{2: {Number: 2, State: PRStateClosed}}, []string{TopicPRClosed}},
		{"still open pr publishes nothing", fakeOpenPRs{refs: []PRRef{{Owner: "o", Repo: "r", Number: 3}}},
			fakePRReader{3: {Number: 3, State: PRStateOpen}}, nil},
		{"one unreachable pr does not stop the rest", fakeOpenPRs{refs: []PRRef{{Owner: "o", Repo: "r", Number: 9}, {Owner: "o", Repo: "r", Number: 1}}},
			fakePRReader{1: {Number: 1, State: PRStateClosed, Merged: true}}, []string{TopicPRMerged}},
		{"list failure publishes nothing", fakeOpenPRs{err: errors.New("db locked")}, fakePRReader{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newFakeBus()
			reconcilePRsOnce(t.Context(), tt.open, tt.host, bus, slog.New(slog.DiscardHandler))
			var topics []string
			for _, ev := range bus.published {
				topics = append(topics, ev.Topic)
			}
			assert.Equal(t, tt.wantTopics, topics)
		})
	}
}

func TestReconcilePRsOnce_MergedPayload_CarriesTheLinkIdentity(t *testing.T) {
	bus := newFakeBus()
	host := fakePRReader{7: {Number: 7, Title: "Ship it", State: PRStateClosed, Merged: true, HeadSHA: "abc"}}
	reconcilePRsOnce(t.Context(), fakeOpenPRs{refs: []PRRef{{Owner: "Onik97", Repo: "getsource", Number: 7}}}, host, bus, slog.New(slog.DiscardHandler))

	var got PREvent
	require.NoError(t, json.Unmarshal(bus.lastPublished().Payload, &got))
	assert.Equal(t, "Onik97", got.Owner)
	assert.Equal(t, "getsource", got.Repo)
	assert.Equal(t, 7, got.PR.Number)
	assert.Equal(t, "abc", got.PR.HeadSHA)
}
