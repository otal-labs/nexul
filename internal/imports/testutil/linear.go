// Package testutil holds a stub of Linear's GraphQL API for the import tests.
package testutil

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LinearIssue is one issue the stub serves; Blocks and Related name other issues by id.
type LinearIssue struct {
	ID, Identifier, Title, Description, DueDate string
	Priority                                    int
	StateName, StateType                        string
	AssigneeName, AssigneeEmail                 string
	Project, Parent                             string
	Labels                                      []string
	Blocks, Related                             []string
}

// LinearComment is one comment on an issue.
type LinearComment struct {
	ID, IssueID, Body, UserName, UserEmail string
	CreatedAt                              time.Time
}

// LinearRequest is one request the stub answered.
type LinearRequest struct {
	Authorization, SignFor string
	// Kind is teams, issues or comments.
	Kind  string
	After string
}

// LinearStub serves a workspace of Linear teams, issues and comments, a hundred to a page, to the one key it accepts.
type LinearStub struct {
	Key      string
	Teams    []struct{ ID, Key, Name string }
	Issues   []LinearIssue
	Comments []LinearComment
	// RateLimited answers that many requests with a 429 carrying RetryAfter, before answering normally.
	RateLimited int
	RetryAfter  string
	// LimitAfter is how many requests are answered before the RateLimited ones start.
	LimitAfter int
	// Header is set on every normal answer, such as X-RateLimit-Requests-Remaining.
	Header http.Header

	mu       sync.Mutex
	requests []LinearRequest
}

// AddTeam adds a team the key reaches.
func (s *LinearStub) AddTeam(id, key, name string) {
	s.Teams = append(s.Teams, struct{ ID, Key, Name string }{id, key, name})
}

// Requests is every request answered so far.
func (s *LinearStub) Requests() []LinearRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func (s *LinearStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req gqlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	after, _ := req.Variables["after"].(string)
	rec := LinearRequest{Authorization: r.Header.Get("Authorization"), SignFor: r.Header.Get("public-file-urls-expire-in"), Kind: kindOf(req.Query), After: after}
	s.mu.Lock()
	s.requests = append(s.requests, rec)
	limited := s.RateLimited > 0 && len(s.requests) > s.LimitAfter
	if limited {
		s.RateLimited--
	}
	s.mu.Unlock()
	if limited {
		w.Header().Set("Retry-After", s.RetryAfter)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"errors": []any{gqlError("Rate limit exceeded", "RATELIMITED")}})
		return
	}
	if rec.Authorization != s.Key {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": []any{gqlError("Authentication required, not authenticated", "AUTHENTICATION_ERROR")}})
		return
	}
	for k, v := range s.Header {
		w.Header()[k] = v
	}
	switch rec.Kind {
	case "teams":
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"teams": map[string]any{"nodes": s.teams()}}})
	case "issues":
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"team": s.issuesPage(req.Variables["team"], after)}})
	case "comments":
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"comments": s.commentsPage(req.Variables["issues"], after)}})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": []any{gqlError("unknown query", "GRAPHQL_VALIDATION_FAILED")}})
	}
}

func kindOf(query string) string {
	switch {
	case strings.Contains(query, "teams("):
		return "teams"
	case strings.Contains(query, "team(id"):
		return "issues"
	case strings.Contains(query, "comments("):
		return "comments"
	}
	return ""
}

func (s *LinearStub) teams() []any {
	out := []any{}
	for _, t := range s.Teams {
		out = append(out, map[string]any{"id": t.ID, "key": t.Key, "name": t.Name})
	}
	return out
}

const pageSize = 100

// page cuts n records at the cursor after, which names the index of the last one sent.
func page(n int, after string) (from, to int, info map[string]any) {
	if after != "" {
		i, _ := strconv.Atoi(strings.TrimPrefix(after, "cursor-"))
		from = i + 1
	}
	to = min(from+pageSize, n)
	from = min(from, to)
	info = map[string]any{"hasNextPage": to < n, "endCursor": nil}
	if to > from {
		info["endCursor"] = "cursor-" + strconv.Itoa(to-1)
	}
	return from, to, info
}

func (s *LinearStub) issuesPage(team any, after string) any {
	if !slices.ContainsFunc(s.Teams, func(t struct{ ID, Key, Name string }) bool { return t.ID == team }) {
		return nil
	}
	from, to, info := page(len(s.Issues), after)
	nodes := []any{}
	for _, i := range s.Issues[from:to] {
		nodes = append(nodes, s.issueNode(i))
	}
	return map[string]any{"issues": map[string]any{"nodes": nodes, "pageInfo": info}}
}

func (s *LinearStub) issueNode(i LinearIssue) map[string]any {
	labels := []any{}
	for _, l := range i.Labels {
		labels = append(labels, map[string]any{"name": l})
	}
	relations := []any{}
	for _, id := range i.Blocks {
		relations = append(relations, map[string]any{"type": "blocks", "relatedIssue": map[string]any{"id": id}})
	}
	for _, id := range i.Related {
		relations = append(relations, map[string]any{"type": "related", "relatedIssue": map[string]any{"id": id}})
	}
	inverse := []any{}
	for _, other := range s.Issues {
		for _, id := range other.Blocks {
			if id == i.ID {
				inverse = append(inverse, map[string]any{"type": "blocks", "issue": map[string]any{"id": other.ID}})
			}
		}
	}
	node := map[string]any{
		"id": i.ID, "identifier": i.Identifier, "title": i.Title, "description": i.Description, "priority": i.Priority,
		"dueDate": nil, "url": "https://linear.app/acme/issue/" + i.Identifier,
		"state": map[string]any{"name": i.StateName, "type": i.StateType}, "assignee": nil, "project": nil, "parent": nil,
		"labels": map[string]any{"nodes": labels}, "relations": map[string]any{"nodes": relations},
		"inverseRelations": map[string]any{"nodes": inverse},
	}
	if i.DueDate != "" {
		node["dueDate"] = i.DueDate
	}
	if i.AssigneeEmail != "" {
		node["assignee"] = map[string]any{"name": i.AssigneeName, "email": i.AssigneeEmail, "avatarUrl": ""}
	}
	if i.Project != "" {
		node["project"] = map[string]any{"name": i.Project}
	}
	if i.Parent != "" {
		node["parent"] = map[string]any{"identifier": i.Parent}
	}
	return node
}

func (s *LinearStub) commentsPage(issues any, after string) any {
	wanted := map[string]bool{}
	list, _ := issues.([]any)
	for _, id := range list {
		if s, ok := id.(string); ok {
			wanted[s] = true
		}
	}
	var matched []LinearComment
	for _, c := range s.Comments {
		if wanted[c.IssueID] {
			matched = append(matched, c)
		}
	}
	from, to, info := page(len(matched), after)
	nodes := []any{}
	for _, c := range matched[from:to] {
		node := map[string]any{"id": c.ID, "body": c.Body, "createdAt": c.CreatedAt.Format(time.RFC3339), "user": nil, "issue": map[string]any{"id": c.IssueID}}
		if c.UserEmail != "" {
			node["user"] = map[string]any{"name": c.UserName, "email": c.UserEmail, "avatarUrl": ""}
		}
		nodes = append(nodes, node)
	}
	return map[string]any{"nodes": nodes, "pageInfo": info}
}

func gqlError(msg, code string) map[string]any {
	return map[string]any{"message": msg, "extensions": map[string]any{"code": code}}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // a test stub: a broken pipe fails the caller's read
}
