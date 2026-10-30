package docs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRuns holds one running play run, found only for its starter on its target, as the trail lookup does.
type fakeRuns struct {
	starter, targetType, targetID, label string
	err                                  error
	asked                                []string
}

func (f *fakeRuns) RunningOn(_ context.Context, userID, targetType, targetID string) (string, string, bool, error) {
	f.asked = append(f.asked, userID+" "+targetType+" "+targetID)
	if f.err != nil {
		return "", "", false, f.err
	}
	if userID != f.starter || targetType != f.targetType || targetID != f.targetID {
		return "", "", false, nil
	}
	return f.label, "trail-1", true, nil
}

func TestDocUpdate_InsideAPlayRun_RecordsThePlayAsTheVia(t *testing.T) {
	s := newTestService(newFakeRepo())
	d := mustDoc(t, s, "project-1", "Spec", "one")
	runs := &fakeRuns{starter: "user-1", targetType: "doc", targetID: d.ID, label: "Update doc via AI"}
	s.SetRunningPlays(runs)

	_, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","body":"two"}`)
	require.NoError(t, err)
	_, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","revert_to_version":1}`)
	require.NoError(t, err)

	h, err := s.History(testCtx(), d.ID)
	require.NoError(t, err)
	assert.Equal(t, "Update doc via AI", h.Versions[0].Via, "a restore made in the run names the play too")
	assert.Equal(t, "Update doc via AI", h.Versions[1].Via)
	assert.Equal(t, []string{"user-1 doc " + d.ID, "user-1 doc " + d.ID}, runs.asked)
}

func TestDocUpdate_WithoutTheCallersOwnLiveRun_StaysMCP(t *testing.T) {
	tests := []struct {
		name string
		runs func(docID string) *fakeRuns
	}{
		{"another person's run on the doc", func(docID string) *fakeRuns {
			return &fakeRuns{starter: "user-2", targetType: "doc", targetID: docID, label: "Update doc via AI"}
		}},
		{"a run on another doc", func(string) *fakeRuns {
			return &fakeRuns{starter: "user-1", targetType: "doc", targetID: "other", label: "Update doc via AI"}
		}},
		{"a failed lookup", func(string) *fakeRuns { return &fakeRuns{err: errors.New("storage is down")} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(newFakeRepo())
			d := mustDoc(t, s, "project-1", "Spec", "one")
			s.SetRunningPlays(tt.runs(d.ID))

			_, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","body":"two"}`)
			require.NoError(t, err, "the lookup never fails the save")

			h, err := s.History(testCtx(), d.ID)
			require.NoError(t, err)
			assert.Equal(t, viaMCP, h.Versions[0].Via)
		})
	}
}
