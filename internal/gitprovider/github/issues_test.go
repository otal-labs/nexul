package github

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/gitprovider"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func issueJSON(number int, pr bool) string {
	extra := ""
	if pr {
		extra = `,"pull_request":{"url":"https://api.github.com/repos/acme/app/pulls/` + strconv.Itoa(number) + `"}`
	}
	return fmt.Sprintf(`{"id":%d,"number":%d,"title":"Issue %d","body":"b","state":"open","state_reason":null,`+
		`"labels":[{"name":"bug"}],"assignees":[{"login":"bob"}],"user":{"login":"alice"},`+
		`"html_url":"https://github.com/acme/app/issues/%d","updated_at":"2026-11-03T09:30:00Z"%s}`,
		5000+number, number, number, number, extra)
}

func commentJSON(id int64, body string) string {
	return fmt.Sprintf(`{"id":%d,"body":%q,"user":{"login":"bob","avatar_url":"https://avatars.example.com/bob"},`+
		`"created_at":"2026-11-01T10:00:00Z","updated_at":"2026-11-02T10:00:00Z"}`, id, body)
}

func TestListIssues(t *testing.T) {
	t.Run("one page with its filters, pull requests left out, and the next page", func(t *testing.T) {
		var query string
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/acme/app/issues", r.URL.Path)
			query = r.URL.RawQuery
			w.Header().Set("Link", `<http://x/repos/acme/app/issues?page=3>; rel="next"`)
			_, _ = fmt.Fprint(w, "["+issueJSON(1, false)+","+issueJSON(2, true)+","+issueJSON(3, false)+"]") // test server: write errors are irrelevant
		}))
		since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
		got, next, err := c.ListIssues(t.Context(), "acme", "app", gitprovider.IssueOpts{State: "all", Labels: []string{"bug", "ui"}, Since: since, Page: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, next)
		require.Len(t, got, 2)
		assert.Equal(t, []int{1, 3}, []int{got[0].Number, got[1].Number})
		assert.Equal(t, gitprovider.Issue{
			ID: 5001, Number: 1, Title: "Issue 1", Body: "b", State: "open", Labels: []string{"bug"}, Assignees: []string{"bob"},
			Author: "alice", HTMLURL: "https://github.com/acme/app/issues/1", UpdatedAt: time.Date(2026, 11, 3, 9, 30, 0, 0, time.UTC),
		}, *got[0])
		assert.Equal(t, "labels=bug%2Cui&page=2&per_page=100&since=2026-11-01T00%3A00%3A00Z&state=all", query)
	})
	t.Run("the last page has no next", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, "[]") // test server: write errors are irrelevant
		}))
		got, next, err := c.ListIssues(t.Context(), "acme", "app", gitprovider.IssueOpts{})
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Zero(t, next)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, _, err := c.ListIssues(t.Context(), "acme", "app", gitprovider.IssueOpts{})
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestGetIssue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/acme/app/issues/12", r.URL.Path)
			_, _ = fmt.Fprint(w, issueJSON(12, false)) // test server: write errors are irrelevant
		}))
		got, err := c.GetIssue(t.Context(), "acme", "app", 12)
		require.NoError(t, err)
		assert.Equal(t, 12, got.Number)
	})
	t.Run("not found maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, err := c.GetIssue(t.Context(), "acme", "app", 12)
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestCreateIssue(t *testing.T) {
	t.Run("sends the issue", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/repos/acme/app/issues", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, map[string]any{"title": "Login fails", "body": "On Safari", "labels": []any{"bug"}}, body, "no assignees given sends none")
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, issueJSON(13, false)) // test server: write errors are irrelevant
		}))
		got, err := c.CreateIssue(t.Context(), "acme", "app", gitprovider.IssueInput{Title: "Login fails", Body: "On Safari", Labels: []string{"bug"}})
		require.NoError(t, err)
		assert.Equal(t, 13, got.Number)
	})
	t.Run("validation failure maps to ErrConflict", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusUnprocessableEntity))
		_, err := c.CreateIssue(t.Context(), "acme", "app", gitprovider.IssueInput{Title: "x"})
		assertErrorIs(t, err, apperrs.ErrConflict)
	})
}

func TestUpdateIssue(t *testing.T) {
	t.Run("sends only what the patch sets, and an empty list clears", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/repos/acme/app/issues/12", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, map[string]any{"state": "closed", "state_reason": "not_planned", "assignees": []any{}}, body)
			_, _ = fmt.Fprint(w, issueJSON(12, false)) // test server: write errors are irrelevant
		}))
		closed, reason, none := "closed", "not_planned", []string{}
		_, err := c.UpdateIssue(t.Context(), "acme", "app", 12, gitprovider.IssuePatch{State: &closed, StateReason: &reason, Assignees: &none})
		require.NoError(t, err)
	})
	t.Run("forbidden maps to ErrUnauthorized", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusForbidden))
		_, err := c.UpdateIssue(t.Context(), "acme", "app", 12, gitprovider.IssuePatch{})
		assertErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("a deleted issue's gone maps to ErrNotFound", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusGone))
		_, err := c.UpdateIssue(t.Context(), "acme", "app", 12, gitprovider.IssuePatch{})
		assertErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestListIssueComments(t *testing.T) {
	t.Run("every page since the given time", func(t *testing.T) {
		var queries []string
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/acme/app/issues/12/comments", r.URL.Path)
			queries = append(queries, r.URL.RawQuery)
			if r.URL.Query().Get("page") == "2" {
				_, _ = fmt.Fprint(w, "["+commentJSON(2, "second")+"]") // test server: write errors are irrelevant
				return
			}
			w.Header().Set("Link", `<http://x/repos/acme/app/issues/12/comments?page=2>; rel="next"`)
			_, _ = fmt.Fprint(w, "["+commentJSON(1, "first")+"]") // test server: write errors are irrelevant
		}))
		got, err := c.ListIssueComments(t.Context(), "acme", "app", 12, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, gitprovider.IssueComment{
			ID: 1, Body: "first", Author: "bob", AuthorAvatarURL: "https://avatars.example.com/bob",
			CreatedAt: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
		}, *got[0])
		assert.Equal(t, "second", got[1].Body)
		assert.Equal(t, []string{
			"per_page=100&since=2026-11-01T00%3A00%3A00Z",
			"page=2&per_page=100&since=2026-11-01T00%3A00%3A00Z",
		}, queries)
	})
	t.Run("a zero since sends none", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.URL.Query().Get("since"))
			_, _ = fmt.Fprint(w, "[]") // test server: write errors are irrelevant
		}))
		_, err := c.ListIssueComments(t.Context(), "acme", "app", 12, time.Time{})
		require.NoError(t, err)
	})
	t.Run("server error is retryable", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusBadGateway))
		_, err := c.ListIssueComments(t.Context(), "acme", "app", 12, time.Time{})
		assertErrorIs(t, err, apperrs.ErrRetryable)
	})
}

func TestIssueCommentWrites(t *testing.T) {
	t.Run("create posts the body on the issue", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/repos/acme/app/issues/12/comments", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, map[string]any{"body": "Fixed in 1.2"}, body)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, commentJSON(7, "Fixed in 1.2")) // test server: write errors are irrelevant
		}))
		got, err := c.CreateIssueComment(t.Context(), "acme", "app", 12, "Fixed in 1.2")
		require.NoError(t, err)
		assert.Equal(t, int64(7), got.ID)
	})
	t.Run("update patches the comment by id", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/repos/acme/app/issues/comments/7", r.URL.Path)
			_, _ = fmt.Fprint(w, commentJSON(7, "Fixed in 1.3")) // test server: write errors are irrelevant
		}))
		got, err := c.UpdateIssueComment(t.Context(), "acme", "app", 7, "Fixed in 1.3")
		require.NoError(t, err)
		assert.Equal(t, "Fixed in 1.3", got.Body)
	})
	t.Run("delete removes the comment by id", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/repos/acme/app/issues/comments/7", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}))
		require.NoError(t, c.DeleteIssueComment(t.Context(), "acme", "app", 7))
	})
	t.Run("failures map to the sentinels", func(t *testing.T) {
		c := newTestClient(t, errorHandler(http.StatusNotFound))
		_, err := c.CreateIssueComment(t.Context(), "acme", "app", 12, "x")
		assertErrorIs(t, err, apperrs.ErrNotFound)
		_, err = c.UpdateIssueComment(t.Context(), "acme", "app", 7, "x")
		assertErrorIs(t, err, apperrs.ErrNotFound)
		assertErrorIs(t, c.DeleteIssueComment(t.Context(), "acme", "app", 7), apperrs.ErrNotFound)
	})
}

func TestMapErr_RateLimitsAreRetryableWithTheirWait(t *testing.T) {
	reset := strconv.FormatInt(time.Now().Add(90*time.Second).Unix(), 10)
	tests := []struct {
		name     string
		status   int
		headers  map[string]string
		body     string
		minWait  time.Duration
		maxWait  time.Duration
		limited  bool
		sentinel error
	}{
		{"primary limit on a 403 waits until the reset", http.StatusForbidden,
			map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": reset}, "", 80 * time.Second, 90 * time.Second, true, nil},
		{"primary limit on a 429 waits until the reset", http.StatusTooManyRequests,
			map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": reset}, "", 80 * time.Second, 90 * time.Second, true, nil},
		{"Retry-After on a 403 is the wait", http.StatusForbidden,
			map[string]string{"Retry-After": "30"}, "", 30 * time.Second, 30 * time.Second, true, nil},
		{"Retry-After on a 429 is the wait", http.StatusTooManyRequests,
			map[string]string{"Retry-After": "45"}, "", 45 * time.Second, 45 * time.Second, true, nil},
		{"secondary limit with Retry-After", http.StatusForbidden,
			map[string]string{"Retry-After": "60"}, `{"message":"slow down","documentation_url":"https://docs.github.com/rest#secondary-rate-limits"}`,
			60 * time.Second, 60 * time.Second, true, nil},
		{"a plain 403 is still unauthorized", http.StatusForbidden,
			map[string]string{"X-RateLimit-Remaining": "4999"}, "", 0, 0, false, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, cmp.Or(tt.body, `{"message":"api error"}`)) // test server: write errors are irrelevant
			}))
			_, _, err := c.ListIssues(t.Context(), "acme", "app", gitprovider.IssueOpts{})
			require.Error(t, err)
			if !tt.limited {
				assert.ErrorIs(t, err, tt.sentinel)
				assert.NotErrorIs(t, err, apperrs.ErrRetryable)
				return
			}
			assert.ErrorIs(t, err, apperrs.ErrRetryable)
			assert.NotErrorIs(t, err, apperrs.ErrUnauthorized)
			var wait *apperrs.RetryAfterError
			require.True(t, errors.As(err, &wait))
			assert.GreaterOrEqual(t, wait.Wait, tt.minWait)
			assert.LessOrEqual(t, wait.Wait, tt.maxWait)
		})
	}
}

func TestRateLimit_ReadsTheCoreLimit(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rate_limit", r.URL.Path)
		_, _ = fmt.Fprint(w, `{"resources":{"core":{"limit":5000,"remaining":150,"reset":1795000000}}}`) // test server: write errors are irrelevant
	}))
	got, err := c.RateLimit(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 150, got.Remaining)
	assert.True(t, got.Reset.Equal(time.Unix(1795000000, 0)), "reset is %v", got.Reset)

	failing := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	_, err = failing.RateLimit(t.Context())
	assert.ErrorIs(t, err, apperrs.ErrRetryable)
}
