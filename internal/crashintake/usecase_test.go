package crashintake

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var (
	fp    = strings.Repeat("ab", 32)
	build = strings.Repeat("c", 40)
)

// reportFields is a valid crash-report/1 embed's fields; a test overrides or deletes the ones it is about.
func reportFields() map[string]string {
	return map[string]string{
		"Contract": Contract, "Fingerprint": fp, "Kind": "ui-error", "Process type": "renderer", "App version": "1.4.0",
		"Build commit": build, "OS": "linux", "Arch": "x64", "First seen": "2026-10-20T10:00:00Z", "Signature": "TypeError",
		"Stack": "renderPanel\nloadPanel", "Truncated": "false",
	}
}

func postOf(fields map[string]string) Post {
	type field struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Inline bool   `json:"inline"`
	}
	var fs []field
	for name, value := range fields {
		fs = append(fs, field{Name: name, Value: value})
	}
	raw, err := json.Marshal([]map[string]any{{"title": "GetSource crash report", "fields": fs}})
	if err != nil {
		panic(err)
	}
	return Post{Content: FingerprintLine + fp, Embeds: raw}
}

func validPost() Post { return postOf(reportFields()) }

var cfg = Config{ProjectID: "p1"}

func newTestService(desk *fakeDesk, notes *fakeNotes, fixes fakeFixes) *Service {
	s := NewService(desk, notes, fixes, nil)
	s.now = func() time.Time { return time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC) }
	return s
}

func TestIntake_NoTicketHasTheFingerprint_FilesABugOfUnknownOrigin(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	res, err := newTestService(desk, notes, fakeFixes{}).Intake(t.Context(), cfg, validPost())
	require.NoError(t, err)

	assert.Equal(t, Result{Outcome: OutcomeFiled, TicketID: "t1"}, res)
	require.Len(t, desk.filed, 1)
	bug := desk.filed[0]
	assert.Equal(t, Bug{ProjectID: "p1", TypeName: DefaultBugType, Title: "Crash: TypeError in renderPanel (ui-error)", Body: bug.Body}, bug)
	assert.True(t, strings.HasPrefix(bug.Body, FingerprintLine+fp+"\n"), "the fingerprint line opens the body")
	for _, want := range []string{"- Kind: ui-error", "- Process type: renderer", "- First version: 1.4.0", "- OS: linux x64", "renderPanel\nloadPanel"} {
		assert.Contains(t, bug.Body, want)
	}
	assert.Zero(t, notes.posts)
}

func TestIntake_OpenTicket_KeepsOneRunningNote(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	desk.add(Match{ID: "open"}, fp)
	s := newTestService(desk, notes, fakeFixes{})

	res, err := s.Intake(t.Context(), cfg, validPost())
	require.NoError(t, err)
	assert.Equal(t, Result{Outcome: OutcomeSeenAgain, TicketID: "open"}, res)

	again := reportFields()
	again["App version"], again["Suppressed duplicates"] = "1.5.0", "3"
	_, err = s.Intake(t.Context(), cfg, postOf(again))
	require.NoError(t, err)

	assert.Empty(t, desk.filed, "a repeat never files a duplicate")
	assert.Equal(t, 1, notes.posts, "the first repeat posts the note")
	assert.Equal(t, 1, notes.replaces, "the next one updates it in place")
	note := notes.markdown("open", seenAgain.file)
	assert.Contains(t, note, "Reports: 5", "one report, then one more holding three held back")
	assert.Contains(t, note, "Versions: 1.4.0, 1.5.0")
	assert.Contains(t, note, "Last seen: 2026-10-24T12:00:00Z")
}

func TestIntake_DoneTicketWithoutAMergedFix_OnlyNotes(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	desk.add(Match{ID: "closed", Done: true}, fp)

	res, err := newTestService(desk, notes, fakeFixes{}).Intake(t.Context(), cfg, validPost())
	require.NoError(t, err)

	assert.Equal(t, Result{Outcome: OutcomeSeenAgain, TicketID: "closed"}, res)
	assert.Empty(t, desk.filed)
	assert.Contains(t, notes.markdown("closed", seenAgain.file), "Reports: 1")
}

func TestIntake_DoneTicketAndTheBuildHoldsEveryFix_FilesAHighPriorityRegression(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	desk.add(Match{ID: "fixed", Done: true, Merged: []PR{{"acme", "app", 1}, {"acme", "app", 2}}}, fp)
	fixes := fakeFixes{in: map[int]bool{1: true, 2: true}}

	res, err := newTestService(desk, notes, fixes).Intake(t.Context(), Config{ProjectID: "p1", BugType: "Defect", Label: "urgent"}, validPost())
	require.NoError(t, err)

	assert.Equal(t, OutcomeRegression, res.Outcome)
	require.Len(t, desk.filed, 1)
	bug := desk.filed[0]
	assert.Equal(t, "fixed", bug.FoundIn)
	assert.Equal(t, "urgent", bug.Label, "a returned crash carries the high-priority label")
	assert.Equal(t, "Defect", bug.TypeName)
	assert.Contains(t, bug.Body, "came back in a build that contains every merged fix")
	assert.Zero(t, notes.posts)
}

func TestIntake_RegressionThenRepeat_NotesOnTheNewBug(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	desk.add(Match{ID: "fixed", Done: true, Merged: []PR{{"acme", "app", 1}}, CreatedAt: time.Unix(1, 0)}, fp)
	s := newTestService(desk, notes, fakeFixes{in: map[int]bool{1: true}})

	_, err := s.Intake(t.Context(), cfg, validPost())
	require.NoError(t, err)
	res, err := s.Intake(t.Context(), cfg, validPost())
	require.NoError(t, err)

	assert.Len(t, desk.filed, 1, "the newest match, the open regression, decides")
	assert.Equal(t, Result{Outcome: OutcomeSeenAgain, TicketID: "t2"}, res)
}

func TestIntake_DoneTicketAndTheBuildLacksAFix_NotesAnOldBuild(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	desk.add(Match{ID: "fixed", Done: true, Merged: []PR{{"acme", "app", 1}, {"acme", "app", 2}}}, fp)
	fixes := fakeFixes{in: map[int]bool{1: false}, errs: map[int]error{2: apperrs.ErrNotFound}}

	res, err := newTestService(desk, notes, fixes).Intake(t.Context(), cfg, validPost())
	require.NoError(t, err)

	assert.Equal(t, Result{Outcome: OutcomeOldBuild, TicketID: "fixed"}, res, "a fix the build lacks decides over one GitHub can't place")
	assert.Empty(t, desk.filed)
	note := notes.markdown("fixed", oldBuild.file)
	assert.Contains(t, note, "## Old build")
	assert.Contains(t, note, "Builds: "+build[:12])
}

func TestIntake_GitHubCantAnswer_NotesWhyAndFilesNothing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"unknown build commit", apperrs.ErrNotFound, "Nexul couldn't find build commit " + build[:12] + " or acme/app#1 on GitHub"},
		{"repository not readable", apperrs.ErrUnauthorized, "Nexul can't read acme/app on GitHub"},
		{"fix not merged on GitHub", apperrs.ErrConflict, "acme/app#1 has not merged"},
		{"API down", apperrs.Retryable(errors.New("502")), "GitHub didn't answer for acme/app#1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desk, notes := &fakeDesk{}, newFakeNotes()
			desk.add(Match{ID: "fixed", Done: true, Merged: []PR{{"acme", "app", 1}}}, fp)
			fixes := fakeFixes{errs: map[int]error{1: tt.err}}

			res, err := newTestService(desk, notes, fixes).Intake(t.Context(), cfg, validPost())
			require.NoError(t, err)

			assert.Equal(t, Result{Outcome: OutcomeCantTell, TicketID: "fixed", Reason: tt.want}, res)
			assert.Empty(t, desk.filed, "can't tell is never a regression")
			assert.Contains(t, notes.markdown("fixed", cantTell.file), "Reason: "+tt.want)
		})
	}
}

func TestIntake_TwoPostsAtOnce_FileOneTicket(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	s := newTestService(desk, notes, fakeFixes{})

	var wg sync.WaitGroup
	results := make([]Result, 2)
	for i := range results {
		wg.Go(func() {
			res, err := s.Intake(t.Context(), cfg, validPost())
			assert.NoError(t, err)
			results[i] = res
		})
	}
	wg.Wait()

	assert.Len(t, desk.filed, 1)
	assert.ElementsMatch(t, []Outcome{OutcomeFiled, OutcomeSeenAgain}, []Outcome{results[0].Outcome, results[1].Outcome})
}

func TestIntake_NotACrashReport_ChangesNothing(t *testing.T) {
	desk, notes := &fakeDesk{}, newFakeNotes()
	desk.add(Match{ID: "open"}, fp)
	other := reportFields()
	other["Contract"] = "crash-report/2"

	res, err := newTestService(desk, notes, fakeFixes{}).Intake(t.Context(), cfg, postOf(other))
	require.NoError(t, err)

	assert.Equal(t, OutcomeIgnored, res.Outcome)
	assert.Contains(t, res.Reason, "crash-report/2")
	assert.Empty(t, desk.filed)
	assert.Zero(t, notes.posts)
}

func TestIntake_Failures_AreReturned(t *testing.T) {
	boom := errors.New("boom")
	t.Run("no project", func(t *testing.T) {
		_, err := newTestService(&fakeDesk{}, newFakeNotes(), fakeFixes{}).Intake(t.Context(), Config{}, validPost())
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("search fails", func(t *testing.T) {
		_, err := newTestService(&fakeDesk{findErr: boom}, newFakeNotes(), fakeFixes{}).Intake(t.Context(), cfg, validPost())
		assert.ErrorIs(t, err, boom)
	})
	t.Run("filing fails", func(t *testing.T) {
		_, err := newTestService(&fakeDesk{fileErr: boom}, newFakeNotes(), fakeFixes{}).Intake(t.Context(), cfg, validPost())
		assert.ErrorIs(t, err, boom)
	})
	t.Run("reading the note fails", func(t *testing.T) {
		desk, notes := &fakeDesk{}, newFakeNotes()
		desk.add(Match{ID: "open"}, fp)
		notes.err = boom
		_, err := newTestService(desk, notes, fakeFixes{}).Intake(t.Context(), cfg, validPost())
		assert.ErrorIs(t, err, boom)
	})
}
