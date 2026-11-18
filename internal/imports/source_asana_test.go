package imports

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// sampleAsana is one workspace with one project of three sections and two pages of tasks; the first task has a
// completed subtask, a comment, and two attachments, one Asana holds and whose link works only until the next request.
func sampleAsana(t *testing.T) (*AsanaSource, *stubAPI, *sleeps, map[string]int) {
	t.Helper()
	issued := map[string]int{}
	var stub *stubAPI
	rateLimited := false
	page := func(data any, next string) map[string]any {
		out := map[string]any{"data": data, "next_page": nil}
		if next != "" {
			out["next_page"] = map[string]string{"offset": next}
		}
		return out
	}
	first := map[string]any{
		"gid": "301", "name": "Design the form", "notes": "Two fields.", "due_on": "2026-12-03", "num_subtasks": 1,
		"permalink_url": "https://app.asana.com/0/200/301", "assignee": map[string]string{"email": "alice@example.com", "name": "Alice Doe"},
		"tags":        []map[string]string{{"name": "ui"}},
		"memberships": []map[string]any{{"project": map[string]string{"gid": "200"}, "section": map[string]string{"gid": "402"}}},
	}
	fixed := map[string]any{
		"/workspaces":         page([]map[string]string{{"gid": "100", "name": "Acme"}}, ""),
		"/projects":           page([]map[string]string{{"gid": "200", "name": "Roadmap"}}, ""),
		"/projects/200":       map[string]any{"data": map[string]string{"gid": "200", "name": "Roadmap"}},
		"/tasks/301/subtasks": page([]map[string]any{{"gid": "302", "name": "Pick colours", "completed": true}}, ""),
		"/tasks/301/stories": page([]map[string]any{
			{"gid": "501", "type": "system", "text": "moved this task"},
			{"gid": "502", "type": "comment", "text": "Ready", "created_at": "2026-11-03T09:00:00Z", "created_by": map[string]string{"name": "Bob Roe", "email": "bob@example.com"}},
		}, ""),
		"/tasks/302/stories": page([]any{}, ""),
		"/tasks/303/stories": page([]any{}, ""),
	}
	stub = newStubAPI(t, func(r *http.Request) (int, http.Header, any) {
		q := r.URL.Query()
		if body, ok := fixed[r.URL.Path]; ok {
			return 200, nil, body
		}
		switch r.URL.Path {
		case "/projects/200/sections":
			if !rateLimited {
				rateLimited = true
				return http.StatusTooManyRequests, http.Header{"Retry-After": {"3"}}, nil
			}
			return 200, nil, page([]map[string]string{{"gid": "401", "name": "Backlog"}, {"gid": "402", "name": "In progress"}, {"gid": "403", "name": "Shipped"}}, "")
		case "/tasks":
			if q.Get("limit") != "100" {
				return 400, nil, map[string]string{"error": "limit"}
			}
			if q.Get("offset") == "expired" {
				return 400, nil, map[string]any{"errors": []map[string]string{{"message": "offset invalid"}}}
			}
			if q.Get("offset") == "o2" {
				return 200, nil, page([]map[string]any{{"gid": "303", "name": "Launch", "completed": true}}, "")
			}
			return 200, nil, page([]map[string]any{first}, "o2")
		case "/attachments":
			if q.Get("parent") != "301" {
				return 200, nil, page([]any{}, "")
			}
			link := fmt.Sprintf("https://asana-user-private-us-east-1.s3.amazonaws.com/assets/mock.png?sig=%d", stub.count())
			issued[link] = stub.count()
			return 200, nil, page([]map[string]string{
				{"gid": "601", "name": "mock.png", "host": "asana", "download_url": link, "permanent_url": "https://app.asana.com/app/asana/-/get_asset?asset_id=601"},
				{"gid": "602", "name": "Brief", "host": "gdrive", "view_url": "https://drive.example.com/brief"},
			}, "")
		}
		return 200, nil, nil
	})
	src := NewAsanaSource(stub.srv.URL, stub.srv.Client())
	sl := &sleeps{}
	src.api.sleep = sl.sleep
	return src, stub, sl, issued
}

// freshSink downloads each ticket's files the moment it gets them, as the import's sink does, and records whether
// each link was still the newest the stub had issued.
type freshSink struct {
	recordingSink
	stub   *stubAPI
	issued map[string]int
	stale  []string
}

func (s *freshSink) Ticket(ctx context.Context, t Ticket) error {
	for _, f := range t.Files {
		if s.issued[f] != s.stub.count() {
			s.stale = append(s.stale, f)
		}
	}
	return s.recordingSink.Ticket(ctx, t)
}

func TestAsanaSource_ScansTheOnlyProjectWithItsSubtasks(t *testing.T) {
	src, stub, sl, _ := sampleAsana(t)
	found, err := src.Scan(t.Context(), Job{Credential: "pat-1"})
	require.NoError(t, err)
	assert.Equal(t, Scan{
		From: "200", Tickets: 3, People: []Person{{Email: "alice@example.com", Name: "Alice Doe"}},
		Statuses: []StatusRef{{Name: "Backlog", Kind: StatusBacklog}, {Name: "In progress", Kind: StatusProgress}, {Name: "Shipped", Kind: StatusDone}},
	}, found)
	for _, r := range stub.requests {
		assert.Equal(t, "Bearer pat-1", r.Header.Get("Authorization"))
	}
	assert.Contains(t, sl.got, 3*time.Second, "a 429 waits out its Retry-After")
}

func TestAsanaSource_ImportsTasksThenSubtasksWithFreshLinksAndResumes(t *testing.T) {
	src, stub, _, issued := sampleAsana(t)
	sink := &freshSink{stub: stub, issued: issued}
	require.NoError(t, src.Import(t.Context(), Job{From: "https://app.asana.com/0/200/list", Credential: "pat-1"}, sink))
	assert.Empty(t, sink.stale, "every download link is handed over before anything else is read")
	assert.Equal(t, []string{"o2"}, sink.checkpoints)
	require.Len(t, sink.tickets, 3)
	shipped := StatusRef{Name: "Shipped", Kind: StatusDone}
	got := sink.tickets[0]
	require.Len(t, got.Files, 1, "only a file Asana holds is copied")
	assert.Equal(t, Ticket{
		SourceID: "301", Title: "Design the form",
		Body: "Two fields.\n\n**Attachments**\n\n- [mock.png](https://app.asana.com/app/asana/-/get_asset?asset_id=601)\n- [Brief](https://drive.example.com/brief)" +
			"\n\nImported from [Asana](https://app.asana.com/0/200/301).",
		Status: StatusRef{Name: "In progress", Kind: StatusProgress}, Labels: []string{"ui"},
		Assignee: Person{Email: "alice@example.com", Name: "Alice Doe"}, DueOn: "2026-12-03",
		Comments: []Comment{{SourceID: "502", Body: "Ready", Author: Person{Email: "bob@example.com", Name: "Bob Roe"}, At: time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)}},
		Files:    got.Files,
	}, got)
	assert.Equal(t, "302", sink.tickets[1].SourceID)
	assert.Equal(t, "301", sink.tickets[1].Parent, "a subtask names its parent")
	assert.Equal(t, shipped, sink.tickets[1].Status)
	assert.Equal(t, shipped, sink.tickets[2].Status, "a completed task goes to the done section")

	resumed := &freshSink{stub: stub, issued: issued}
	require.NoError(t, src.Import(t.Context(), Job{From: "200", Credential: "pat-1", Cursor: "o2"}, resumed))
	require.Len(t, resumed.tickets, 1)
	assert.Equal(t, "303", resumed.tickets[0].SourceID)
}

func TestAsanaSource_AnOffsetAsanaNoLongerHonoursStartsOver(t *testing.T) {
	src, _, _, _ := sampleAsana(t)
	sink := &recordingSink{}
	require.NoError(t, src.Import(t.Context(), Job{From: "200", Credential: "pat-1", Cursor: "expired"}, sink))
	assert.Len(t, sink.tickets, 3, "the import reads again from the first page")
}

func TestAsanaSource_Refusals(t *testing.T) {
	two := map[string]any{"data": []map[string]string{{"gid": "1", "name": "Acme"}, {"gid": "2", "name": "Globex"}}}
	tests := []struct {
		name   string
		cred   string
		answer func(r *http.Request) (int, http.Header, any)
		want   error
		says   string
	}{
		{"no token", "", nil, apperrs.ErrInvalid, "personal access token"},
		{"a refused token", "pat", func(*http.Request) (int, http.Header, any) { return 401, nil, map[string]string{} }, apperrs.ErrUnauthorized, "refused the credential"},
		{"several workspaces and none named", "pat", func(*http.Request) (int, http.Header, any) { return 200, nil, two }, apperrs.ErrInvalid, "Acme (1), Globex (2)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer := tt.answer
			if answer == nil {
				answer = func(*http.Request) (int, http.Header, any) { return 200, nil, nil }
			}
			stub := newStubAPI(t, answer)
			src := NewAsanaSource(stub.srv.URL, stub.srv.Client())
			src.api.sleep = (&sleeps{}).sleep
			_, err := src.Scan(t.Context(), Job{Credential: tt.cred})
			assert.ErrorIs(t, err, tt.want)
			assert.ErrorContains(t, err, tt.says)
		})
	}
}

func TestAsanaSource_AWorkspaceIDListsItsProjects(t *testing.T) {
	stub := newStubAPI(t, func(r *http.Request) (int, http.Header, any) {
		if r.URL.Path == "/projects" && r.URL.Query().Get("workspace") == "100" {
			return 200, nil, map[string]any{"data": []map[string]string{{"gid": "7", "name": "Ops"}, {"gid": "8", "name": "Web"}}}
		}
		return 404, nil, map[string]string{}
	})
	src := NewAsanaSource(stub.srv.URL, stub.srv.Client())
	_, err := src.Scan(t.Context(), Job{From: "100", Credential: "pat"})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "Ops (7), Web (8)")
}
