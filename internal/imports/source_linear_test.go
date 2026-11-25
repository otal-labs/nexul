package imports

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/imports/testutil"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const linearKey = "lin_api_test"

func newLinearStub(issues int) *testutil.LinearStub {
	stub := &testutil.LinearStub{Key: linearKey}
	stub.AddTeam("team-1", "ENG", "Engineering")
	for n := 1; n <= issues; n++ {
		stub.Issues = append(stub.Issues, testutil.LinearIssue{
			ID: fmt.Sprintf("issue-%d", n), Identifier: fmt.Sprintf("ENG-%d", n), Title: fmt.Sprintf("Issue %d", n),
			StateName: "Todo", StateType: "unstarted",
		})
	}
	return stub
}

// serveLinear runs the stub and a source pointed at it whose waits are recorded instead of slept.
func serveLinear(t *testing.T, stub *testutil.LinearStub) (*LinearSource, *[]time.Duration) {
	t.Helper()
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)
	src := NewLinearSource(srv.URL, srv.Client())
	var waits []time.Duration
	src.wait = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	return src, &waits
}

func linearJob() Job { return Job{Credential: linearKey} }

func TestLinear_PagesAHundredAtATime_CheckpointsTheCursor_AndResumesFromIt(t *testing.T) {
	stub := newLinearStub(120)
	stub.Comments = []testutil.LinearComment{{ID: "c-1", IssueID: "issue-110", Body: "Seen", UserName: "Bob", UserEmail: "bob@example.com"}}
	src, _ := serveLinear(t, stub)

	found, err := src.Scan(context.Background(), linearJob())
	require.NoError(t, err)
	assert.Equal(t, "ENG", found.From)
	assert.Equal(t, 120, found.Tickets)
	assert.Equal(t, 1, found.Comments)
	assert.Equal(t, []StatusRef{{Name: "Todo", Kind: StatusBacklog}}, found.Statuses)

	sink := &recordingSink{}
	require.NoError(t, src.Import(context.Background(), linearJob(), sink))
	require.Len(t, sink.tickets, 120)
	assert.Equal(t, []string{"cursor-99"}, sink.checkpoints, "the cursor after the first page; the last page needs none")
	assert.Len(t, sink.tickets[109].Comments, 1)

	before := len(stub.Requests())
	resumed := &recordingSink{}
	job := linearJob()
	job.Cursor = "cursor-99"
	require.NoError(t, src.Import(context.Background(), job, resumed))
	require.Len(t, resumed.tickets, 20)
	assert.Equal(t, "Issue 101", resumed.tickets[0].Title)
	var issuesAfter []string
	for _, r := range stub.Requests()[before:] {
		if r.Kind == "issues" {
			issuesAfter = append(issuesAfter, r.After)
		}
	}
	assert.Equal(t, []string{"cursor-99"}, issuesAfter, "a resumed import reads from its cursor, not the first page")
	for _, r := range stub.Requests() {
		assert.Equal(t, linearKey, r.Authorization, "the key goes as is, with no Bearer")
		assert.Equal(t, "3600", r.SignFor, "file links come back signed, so they download without the key")
	}
}

func TestLinearTicket_MapsEachField(t *testing.T) {
	stub := newLinearStub(0)
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	stub.Issues = []testutil.LinearIssue{
		{
			ID: "a", Identifier: "ENG-1", Title: "Crash on save", Description: "See ![shot](https://uploads.linear.app/x/y/shot.png)",
			StateName: "In Progress", StateType: "started", Priority: 1, DueDate: "2026-12-01", AssigneeName: "Alice",
			AssigneeEmail: "alice@example.com", Project: "Mobile", Labels: []string{"bug"}, Blocks: []string{"b"}, Related: []string{"c"},
		},
		{ID: "b", Identifier: "ENG-2", Title: "Ship", StateName: "Won't do", StateType: "canceled", Priority: 4, Parent: "ENG-1"},
		{ID: "c", Identifier: "ENG-3", Title: "Triage me", StateName: "Triage", StateType: "triage"},
		{ID: "d", Identifier: "ENG-4", Title: "Done", StateName: "Done", StateType: "completed", Priority: 2},
		{ID: "e", Identifier: "ENG-5", Title: "Later", StateName: "Backlog", StateType: "backlog", Priority: 3},
	}
	stub.Comments = []testutil.LinearComment{
		{ID: "c2", IssueID: "a", Body: "later", UserName: "Bob", UserEmail: "bob@example.com", CreatedAt: at.Add(time.Hour)},
		{ID: "c1", IssueID: "a", Body: "log https://uploads.linear.app/x/z/log.txt", UserName: "Bob", UserEmail: "bob@example.com", CreatedAt: at},
		{ID: "c3", IssueID: "b", Body: "from an integration", CreatedAt: at},
	}
	src, _ := serveLinear(t, stub)
	sink := &recordingSink{}
	require.NoError(t, src.Import(context.Background(), linearJob(), sink))
	require.Len(t, sink.tickets, 5)

	a := sink.tickets[0]
	assert.Equal(t, Ticket{
		SourceID: "a", Title: "Crash on save",
		Body:   "See ![shot](https://uploads.linear.app/x/y/shot.png)\n\nImported from [ENG-1](https://linear.app/acme/issue/ENG-1).",
		Status: StatusRef{Name: "In Progress", Kind: StatusProgress}, Labels: []string{"bug", "priority:urgent"},
		Assignee: Person{Email: "alice@example.com", Name: "Alice"}, Category: "Mobile", DueOn: "2026-12-01", Blocks: []string{"b"},
		Comments: []Comment{
			{SourceID: "c1", Body: "log https://uploads.linear.app/x/z/log.txt", Author: Person{Email: "bob@example.com", Name: "Bob"}, At: at},
			{SourceID: "c2", Body: "later", Author: Person{Email: "bob@example.com", Name: "Bob"}, At: at.Add(time.Hour)},
		},
		Files: []string{"https://uploads.linear.app/x/y/shot.png", "https://uploads.linear.app/x/z/log.txt"},
	}, a)

	b := sink.tickets[1]
	assert.Equal(t, StatusRef{Name: "Won't do", Kind: StatusDone}, b.Status)
	assert.Equal(t, []string{"priority:low", "canceled"}, b.Labels)
	assert.Equal(t, []string{"a"}, b.BlockedBy)
	assert.Equal(t, "Sub-issue of ENG-1.\n\nImported from [ENG-2](https://linear.app/acme/issue/ENG-2).", b.Body)
	assert.Equal(t, Person{Name: "Linear"}, b.Comments[0].Author)

	assert.Equal(t, StatusRef{Name: "Triage", Kind: StatusBacklog}, sink.tickets[2].Status)
	assert.Empty(t, sink.tickets[2].BlockedBy, "a related relation is no link")
	assert.Equal(t, StatusRef{Name: "Done", Kind: StatusDone}, sink.tickets[3].Status)
	assert.Equal(t, []string{"priority:high"}, sink.tickets[3].Labels)
	assert.Equal(t, StatusRef{Name: "Backlog", Kind: StatusBacklog}, sink.tickets[4].Status)
	assert.Equal(t, []string{"priority:medium"}, sink.tickets[4].Labels)

	found, err := src.Scan(context.Background(), linearJob())
	require.NoError(t, err)
	assert.Equal(t, 1, found.RelationsSkipped, "the related relation is counted and skipped")
	assert.Equal(t, 3, found.Comments)
	assert.Equal(t, []Person{{Email: "alice@example.com", Name: "Alice"}}, found.People)
}

func TestLinear_RateLimits(t *testing.T) {
	t.Run("a 429 waits out its Retry-After and tries again", func(t *testing.T) {
		stub := newLinearStub(1)
		stub.RateLimited, stub.RetryAfter = 1, "2"
		src, waits := serveLinear(t, stub)
		found, err := src.Scan(context.Background(), linearJob())
		require.NoError(t, err)
		assert.Equal(t, 1, found.Tickets)
		assert.Equal(t, []time.Duration{2 * time.Second}, *waits)
	})
	t.Run("a long wait pauses the import, which resumes later from its cursor", func(t *testing.T) {
		stub := newLinearStub(1)
		stub.RateLimited, stub.RetryAfter = 1, "3600"
		src, waits := serveLinear(t, stub)
		_, err := src.Scan(context.Background(), linearJob())
		limited, ok := errors.AsType[*apperrs.RetryAfterError](err)
		require.True(t, ok, "the pause carries Linear's reset: %v", err)
		assert.Equal(t, time.Hour, limited.Wait)
		assert.Empty(t, *waits)
	})
	t.Run("refused on every attempt, it pauses", func(t *testing.T) {
		stub := newLinearStub(1)
		stub.RateLimited, stub.RetryAfter = 10, "1"
		src, waits := serveLinear(t, stub)
		_, err := src.Scan(context.Background(), linearJob())
		require.ErrorIs(t, err, apperrs.ErrRetryable)
		assert.Len(t, *waits, linearAttempts-1)
	})
	t.Run("a spent window is waited out before the next request", func(t *testing.T) {
		stub := newLinearStub(1)
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		stub.Header = http.Header{
			"X-Ratelimit-Requests-Remaining": {"0"},
			"X-Ratelimit-Requests-Reset":     {strconv.FormatInt(now.Add(5*time.Second).UnixMilli(), 10)},
		}
		src, waits := serveLinear(t, stub)
		src.now = func() time.Time { return now }
		_, err := src.Scan(context.Background(), linearJob())
		require.NoError(t, err)
		require.NotEmpty(t, *waits)
		assert.Equal(t, 5*time.Second, (*waits)[0])
	})
}

func TestLinear_AnHourLongRateLimitHoldsTheImportUntilItResets(t *testing.T) {
	stub := newLinearStub(1)
	stub.RateLimited, stub.RetryAfter = 1000, "3600"
	src, _ := serveLinear(t, stub)
	svc, repo := newTestService(src)
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject, Credential: linearKey})
	require.NoError(t, err)
	start := time.Now()
	svc.now = func() time.Time { return start }

	assert.Equal(t, time.Hour, svc.Pass(t.Context()))
	for _, at := range []time.Duration{0, time.Minute, 30 * time.Minute, 59 * time.Minute} {
		svc.now = func() time.Time { return start.Add(at) }
		assert.Equal(t, time.Hour-at, svc.Pass(t.Context()), "a commit wakes the loop at %s; the paused import still waits", at)
	}
	assert.Len(t, stub.Requests(), 1, "one refused request in the hour, not one a minute")
	assert.Equal(t, StateScanning, repo.stored(imp.ID).State)

	svc.now = func() time.Time { return start.Add(time.Hour) }
	svc.Pass(t.Context())
	assert.Len(t, stub.Requests(), 2, "read again once the window resets")
}

func TestLinear_AScanPausedOnAPageResumesOnThatPage(t *testing.T) {
	stub := newLinearStub(250)
	stub.Issues[0].StateName, stub.Issues[0].StateType = "In Progress", "started"
	stub.Issues[0].AssigneeName, stub.Issues[0].AssigneeEmail = "Alice", "alice@example.com"
	stub.Issues[0].Related = []string{"issue-2"}
	// teams, then page 1's issues and comments, then page 2's issues are refused for an hour.
	stub.LimitAfter, stub.RateLimited, stub.RetryAfter = 3, 1, "3600"
	src, _ := serveLinear(t, stub)
	svc, repo := newTestService(src)
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject, Credential: linearKey})
	require.NoError(t, err)
	start := time.Now()
	svc.now = func() time.Time { return start }

	svc.Pass(t.Context())
	paused := repo.stored(imp.ID)
	require.Equal(t, StateScanning, paused.State)
	assert.Equal(t, "cursor-99", paused.Report.Cursor, "the page the scan stopped on")
	assert.Equal(t, 100, paused.Report.Tickets, "the first page is counted once")

	before := len(stub.Requests())
	svc.now = func() time.Time { return start.Add(time.Hour) }
	svc.Pass(t.Context())
	var after []string
	for _, r := range stub.Requests()[before:] {
		if r.Kind == "issues" {
			after = append(after, r.After)
		}
	}
	assert.Equal(t, []string{"cursor-99", "cursor-199"}, after, "resumed on page 2, not page 1")

	got := repo.stored(imp.ID)
	require.Equal(t, StateReady, got.State)
	assert.Equal(t, 250, got.Report.Tickets)
	assert.Equal(t, 250, got.Total)
	assert.Equal(t, 1, got.Report.RelationsSkipped, "a count from before the pause survives it")
	assert.Empty(t, got.Report.Cursor, "the import itself starts at the first page")
	require.Len(t, got.Report.People, 1)
	assert.Equal(t, "alice@example.com", got.Report.People[0].Email)
	var statuses []string
	for _, st := range got.Report.Statuses {
		statuses = append(statuses, st.Name)
	}
	assert.ElementsMatch(t, []string{"In Progress", "Todo"}, statuses)
}

func TestLinear_ARefusedKeyFailsWithWhatToFix(t *testing.T) {
	stub := newLinearStub(1)
	src, _ := serveLinear(t, stub)
	_, err := src.Scan(context.Background(), Job{Credential: "Bearer " + linearKey})
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	assert.Equal(t, "Linear refused the API key; create a personal API key in Linear's security settings and start again", failureReason(err))

	_, err = src.Scan(context.Background(), Job{})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "no key at all")
}

func TestLinear_TeamIsTheOneNamed_OrTheOnlyOne(t *testing.T) {
	stub := newLinearStub(0)
	stub.AddTeam("team-2", "OPS", "Operations")
	src, _ := serveLinear(t, stub)
	tests := []struct {
		from, want, err string
	}{
		{from: "ops", want: "OPS"},
		{from: "Engineering", want: "ENG"},
		{from: "", err: "the API key reaches 2 teams; name one in from (ENG, OPS)"},
		{from: "WEB", err: "the API key reaches no team WEB; it reaches ENG, OPS"},
	}
	for _, tt := range tests {
		found, err := src.Scan(context.Background(), Job{Credential: linearKey, From: tt.from})
		if tt.err != "" {
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Equal(t, tt.err, failureReason(err))
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tt.want, found.From)
	}
}

func TestLinear_AnIssuePastTheFirstPageOfLabelsAndRelations_ImportsEveryOne(t *testing.T) {
	stub := newLinearStub(26)
	hub := &stub.Issues[0]
	var labels, blocks, blockedBy []string
	for n := 1; n <= 25; n++ {
		labels = append(labels, fmt.Sprintf("label-%d", n))
	}
	for n := 2; n <= 13; n++ {
		blocks = append(blocks, fmt.Sprintf("issue-%d", n))
	}
	for n := 14; n <= 25; n++ {
		blockedBy = append(blockedBy, fmt.Sprintf("issue-%d", n))
		stub.Issues[n-1].Blocks = []string{hub.ID}
	}
	hub.Labels, hub.Blocks, hub.Related = labels, blocks, []string{"issue-26"}
	var many []string
	for n := 1; n <= 130; n++ {
		many = append(many, fmt.Sprintf("tag-%d", n))
	}
	stub.Issues[25].Labels = many
	src, _ := serveLinear(t, stub)

	found, err := src.Scan(context.Background(), linearJob())
	require.NoError(t, err)
	assert.Equal(t, 1, found.RelationsSkipped, "the related link past the first ten relations is counted")

	before := len(stub.Requests())
	sink := &recordingSink{}
	require.NoError(t, src.Import(context.Background(), linearJob(), sink))
	require.Len(t, sink.tickets, 26)
	got := sink.tickets[0]
	assert.Equal(t, labels, got.Labels)
	assert.Equal(t, blocks, got.Blocks)
	assert.Equal(t, blockedBy, got.BlockedBy)
	assert.Equal(t, many, sink.tickets[25].Labels, "a list longer than a page of 100 is read to its end")

	var later []testutil.LinearRequest
	for _, r := range stub.Requests()[before:] {
		if r.Kind != "teams" && r.Kind != "issues" && r.Kind != "comments" {
			later = append(later, testutil.LinearRequest{Kind: r.Kind, Issue: r.Issue, After: r.After})
		}
	}
	assert.Equal(t, []testutil.LinearRequest{
		{Kind: "labels", Issue: hub.ID},
		{Kind: "relations", Issue: hub.ID},
		{Kind: "inverseRelations", Issue: hub.ID},
		{Kind: "labels", Issue: "issue-26"},
		{Kind: "labels", Issue: "issue-26", After: "cursor-99"},
	}, later, "only the lists past a first page are read again; the other 24 issues cost no request")
}
