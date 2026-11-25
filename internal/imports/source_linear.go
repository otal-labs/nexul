package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// SourceLinear imports one Linear team's issues, read with a personal API key.
const SourceLinear = "linear"

// LinearAPI is Linear's GraphQL endpoint.
const LinearAPI = "https://api.linear.app/graphql"

const (
	// linearSignFor is how long, in seconds, Linear signs the file links it returns, so files download without the key.
	linearSignFor  = "3600"
	linearAttempts = 3
	// linearMaxWait is the longest a rate limit is waited out in place; past it the import pauses and resumes from
	// its cursor, so one import never holds the loop for long.
	linearMaxWait = 30 * time.Second
	// linearDefaultWait is the wait a 429 without a Retry-After or reset header gets.
	linearDefaultWait = 10 * time.Second
	linearMaxBody     = 32 << 20
)

// LinearSource is the Linear source: a team's issues, comments, labels, projects and blocking relations.
type LinearSource struct {
	endpoint string
	client   *http.Client
	wait     func(ctx context.Context, d time.Duration) error
	now      func() time.Time
}

var _ Source = (*LinearSource)(nil)

// NewLinearSource reads from endpoint, LinearAPI outside tests; a nil client gets a one-minute timeout per request.
func NewLinearSource(endpoint string, client *http.Client) *LinearSource {
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	return &LinearSource{endpoint: endpoint, client: client, wait: waitFor, now: time.Now}
}

func waitFor(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Info implements Source.
func (l *LinearSource) Info() Info {
	return Info{Title: "Linear", FileHosts: []string{"uploads.linear.app"}}
}

type linearTeam struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type linearUser struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarUrl"`
}

type linearIssue struct {
	ID          string  `json:"id"`
	Identifier  string  `json:"identifier"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Priority    float64 `json:"priority"`
	DueDate     string  `json:"dueDate"`
	URL         string  `json:"url"`
	State       struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"state"`
	Assignee *linearUser `json:"assignee"`
	Project  *struct {
		Name string `json:"name"`
	} `json:"project"`
	Parent *struct {
		Identifier string `json:"identifier"`
	} `json:"parent"`
	Labels           linearConn[linearLabel]    `json:"labels"`
	Relations        linearConn[linearRelation] `json:"relations"`
	InverseRelations linearConn[linearInverse]  `json:"inverseRelations"`
}

// linearConn is one of an issue's nested lists, a page of it.
type linearConn[T any] struct {
	Nodes    []T            `json:"nodes"`
	PageInfo linearPageInfo `json:"pageInfo"`
}

type linearLabel struct {
	Name string `json:"name"`
}

type linearRelation struct {
	Type         string `json:"type"`
	RelatedIssue struct {
		ID string `json:"id"`
	} `json:"relatedIssue"`
}

type linearInverse struct {
	Type  string `json:"type"`
	Issue struct {
		ID string `json:"id"`
	} `json:"issue"`
}

type linearComment struct {
	ID        string      `json:"id"`
	Body      string      `json:"body"`
	CreatedAt time.Time   `json:"createdAt"`
	User      *linearUser `json:"user"`
	Issue     struct {
		ID string `json:"id"`
	} `json:"issue"`
}

type linearPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

const linearTeamsQuery = `query { teams(first: 100) { nodes { id key name } } }`

const (
	linearLabelNodes    = `name`
	linearRelationNodes = `type relatedIssue { id }`
	linearInverseNodes  = `type issue { id }`
	// One past 20 labels and 10 relations shows an issue has more; pageInfo here could pass the complexity cap.
	linearFirstLabels    = 21
	linearFirstRelations = 11
)

// Nested connections stay small so 100 issues keep under Linear's 10,000-point query complexity.
var linearIssuesQuery = fmt.Sprintf(`query($team: String!, $after: String) {
  team(id: $team) {
    issues(first: 100, after: $after, orderBy: createdAt, includeArchived: true) {
      nodes {
        id identifier title description priority dueDate url
        state { name type }
        assignee { name email avatarUrl }
        project { name }
        parent { identifier }
        labels(first: %[1]d) { nodes { %[3]s } }
        relations(first: %[2]d) { nodes { %[4]s } }
        inverseRelations(first: %[2]d) { nodes { %[5]s } }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`, linearFirstLabels, linearFirstRelations, linearLabelNodes, linearRelationNodes, linearInverseNodes)

var (
	linearAllLabels    = linearAllQuery("labels", linearLabelNodes)
	linearAllRelations = linearAllQuery("relations", linearRelationNodes)
	linearAllInverse   = linearAllQuery("inverseRelations", linearInverseNodes)
)

// linearAllQuery pages one issue's nested list, aliased to all so one decoder reads any of them.
func linearAllQuery(field, nodes string) string {
	return `query($id: String!, $after: String) {
  issue(id: $id) { all: ` + field + `(first: 100, after: $after) { nodes { ` + nodes + ` } pageInfo { hasNextPage endCursor } } }
}`
}

const linearCommentsQuery = `query($issues: [ID!], $after: String) {
  comments(first: 100, after: $after, includeArchived: true, filter: { issue: { id: { in: $issues } } }) {
    nodes { id body createdAt user { name email avatarUrl } issue { id } }
    pageInfo { hasNextPage endCursor }
  }
}`

// Scan implements Source: it reads every issue and comment once to count them and the people, states and relations.
func (l *LinearSource) Scan(ctx context.Context, job Job) (Scan, error) {
	team, err := l.team(ctx, job)
	if err != nil {
		return Scan{}, err
	}
	found := job.Scanned
	found.From, found.Cursor = team.Key, ""
	states := map[StatusRef]bool{}
	for _, ref := range found.Statuses {
		states[ref] = true
	}
	at := job.Cursor
	err = l.pages(ctx, job.Credential, team.ID, job.Cursor, func(issues []linearIssue, comments map[string][]linearComment, next string) error {
		for _, i := range issues {
			found.Tickets++
			found.Comments += len(comments[i.ID])
			if i.Assignee != nil {
				found.People = append(found.People, linearPerson(i.Assignee))
			}
			ref := linearStatus(i.State.Name, i.State.Type)
			if !states[ref] {
				states[ref] = true
				found.Statuses = append(found.Statuses, ref)
			}
			for _, r := range i.Relations.Nodes {
				if r.Type != "blocks" {
					found.RelationsSkipped++
				}
			}
		}
		at = next
		return nil
	})
	if err != nil {
		found.Cursor = at
		return found, err
	}
	return found, nil
}

// Import implements Source: a page of issues at a time from the checkpointed cursor, each with its comments.
func (l *LinearSource) Import(ctx context.Context, job Job, sink Sink) error {
	team, err := l.team(ctx, job)
	if err != nil {
		return err
	}
	return l.pages(ctx, job.Credential, team.ID, job.Cursor, func(issues []linearIssue, comments map[string][]linearComment, next string) error {
		for _, i := range issues {
			if err := sink.Ticket(ctx, LinearTicket(i, comments[i.ID])); err != nil {
				return err
			}
		}
		if next == "" {
			return nil
		}
		return sink.Checkpoint(ctx, next)
	})
}

// pages hands fn each page of the team's issues after cursor, with their comments, and the cursor after it, empty
// on the last page.
func (l *LinearSource) pages(ctx context.Context, key, teamID, cursor string, fn func([]linearIssue, map[string][]linearComment, string) error) error {
	for {
		var out struct {
			Team *struct {
				Issues struct {
					Nodes    []linearIssue  `json:"nodes"`
					PageInfo linearPageInfo `json:"pageInfo"`
				} `json:"issues"`
			} `json:"team"`
		}
		vars := map[string]any{"team": teamID, "after": nil}
		if cursor != "" {
			vars["after"] = cursor
		}
		if err := l.query(ctx, key, linearIssuesQuery, vars, &out); err != nil {
			return err
		}
		if out.Team == nil {
			return fmt.Errorf("%w: the Linear team is gone, or the key no longer reaches it", apperrs.ErrNotFound)
		}
		page := out.Team.Issues
		for n := range page.Nodes {
			if err := l.rest(ctx, key, &page.Nodes[n]); err != nil {
				return err
			}
		}
		comments, err := l.comments(ctx, key, page.Nodes)
		if err != nil {
			return err
		}
		next := ""
		if page.PageInfo.HasNextPage {
			next = page.PageInfo.EndCursor
		}
		if err := fn(page.Nodes, comments, next); err != nil {
			return err
		}
		if next == "" {
			return nil
		}
		cursor = next
	}
}

// rest reads again, whole, any of an issue's labels and relations that filled the page; others cost no request.
func (l *LinearSource) rest(ctx context.Context, key string, i *linearIssue) error {
	if err := linearAll(ctx, l, key, i.ID, linearAllLabels, linearFirstLabels, &i.Labels); err != nil {
		return err
	}
	if err := linearAll(ctx, l, key, i.ID, linearAllRelations, linearFirstRelations, &i.Relations); err != nil {
		return err
	}
	return linearAll(ctx, l, key, i.ID, linearAllInverse, linearFirstRelations, &i.InverseRelations)
}

// linearAll replaces conn with every page of the list when its first page came back full.
func linearAll[T any](ctx context.Context, l *LinearSource, key, issueID, query string, first int, conn *linearConn[T]) error {
	if len(conn.Nodes) < first {
		return nil
	}
	var all []T
	var after any
	for {
		var out struct {
			Issue *struct {
				All linearConn[T] `json:"all"`
			} `json:"issue"`
		}
		if err := l.query(ctx, key, query, map[string]any{"id": issueID, "after": after}, &out); err != nil {
			return err
		}
		// Deleted since its page was read: keep that page rather than fail the rest of the import.
		if out.Issue == nil {
			return nil
		}
		all = append(all, out.Issue.All.Nodes...)
		if !out.Issue.All.PageInfo.HasNextPage {
			conn.Nodes = all
			return nil
		}
		after = out.Issue.All.PageInfo.EndCursor
	}
}

// comments reads every comment on the issues, oldest first per issue.
func (l *LinearSource) comments(ctx context.Context, key string, issues []linearIssue) (map[string][]linearComment, error) {
	out := map[string][]linearComment{}
	if len(issues) == 0 {
		return out, nil
	}
	ids := make([]string, 0, len(issues))
	for _, i := range issues {
		ids = append(ids, i.ID)
	}
	var after any
	for {
		var page struct {
			Comments struct {
				Nodes    []linearComment `json:"nodes"`
				PageInfo linearPageInfo  `json:"pageInfo"`
			} `json:"comments"`
		}
		if err := l.query(ctx, key, linearCommentsQuery, map[string]any{"issues": ids, "after": after}, &page); err != nil {
			return nil, err
		}
		for _, c := range page.Comments.Nodes {
			out[c.Issue.ID] = append(out[c.Issue.ID], c)
		}
		if !page.Comments.PageInfo.HasNextPage {
			break
		}
		after = page.Comments.PageInfo.EndCursor
	}
	for id := range out {
		slices.SortStableFunc(out[id], func(a, b linearComment) int { return a.CreatedAt.Compare(b.CreatedAt) })
	}
	return out, nil
}

// team is the team the import names by key, name or id, or the key's only team when it names none.
func (l *LinearSource) team(ctx context.Context, job Job) (linearTeam, error) {
	if strings.TrimSpace(job.Credential) == "" {
		return linearTeam{}, fmt.Errorf("%w: a Linear import needs a personal API key in credential", apperrs.ErrInvalid)
	}
	var out struct {
		Teams struct {
			Nodes []linearTeam `json:"nodes"`
		} `json:"teams"`
	}
	if err := l.query(ctx, job.Credential, linearTeamsQuery, nil, &out); err != nil {
		return linearTeam{}, err
	}
	teams := out.Teams.Nodes
	from := strings.TrimSpace(job.From)
	keys := make([]string, 0, len(teams))
	for _, t := range teams {
		if from != "" && (strings.EqualFold(t.Key, from) || strings.EqualFold(t.Name, from) || t.ID == from) {
			return t, nil
		}
		keys = append(keys, t.Key)
	}
	if len(teams) == 0 {
		return linearTeam{}, fmt.Errorf("%w: the API key reaches no Linear team", apperrs.ErrInvalid)
	}
	if from == "" && len(teams) == 1 {
		return teams[0], nil
	}
	if from == "" {
		return linearTeam{}, fmt.Errorf("%w: the API key reaches %d teams; name one in from (%s)", apperrs.ErrInvalid, len(teams), strings.Join(keys, ", "))
	}
	return linearTeam{}, fmt.Errorf("%w: the API key reaches no team %s; it reaches %s", apperrs.ErrInvalid, from, strings.Join(keys, ", "))
}

type linearError struct {
	Message    string `json:"message"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

var errLinearKey = fmt.Errorf("%w: Linear refused the API key; create a personal API key in Linear's security settings and start again", apperrs.ErrUnauthorized)

// query posts one GraphQL query, waiting out a short rate limit and pausing the import on a long one.
func (l *LinearSource) query(ctx context.Context, key, query string, vars map[string]any, out any) error {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	for attempt := 1; ; attempt++ {
		wait, err := l.post(ctx, key, payload, out)
		if wait == 0 {
			return err
		}
		if wait > linearMaxWait || attempt == linearAttempts {
			return &apperrs.RetryAfterError{Wait: wait, Err: fmt.Errorf("linear rate limit, resets in %s", wait.Round(time.Second))}
		}
		if err := l.wait(ctx, wait); err != nil {
			return err
		}
	}
}

// post sends one request; a positive duration is a rate limit to wait out before sending it again.
func (l *LinearSource) post(ctx context.Context, key string, payload []byte, out any) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	// A personal API key goes as is; Linear reads a Bearer prefix as an OAuth token.
	req.Header.Set("Authorization", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("public-file-urls-expire-in", linearSignFor)
	resp, err := l.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, apperrs.Retryable(fmt.Errorf("reach linear: %w", err))
	}
	defer func() { _ = resp.Body.Close() }() // a read-only body: nothing to lose on close
	body, err := io.ReadAll(io.LimitReader(resp.Body, linearMaxBody))
	if err != nil {
		return 0, apperrs.Retryable(fmt.Errorf("read linear's answer: %w", err))
	}
	data, wait, err := l.answer(resp, body)
	if wait > 0 || err != nil {
		return wait, err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return 0, fmt.Errorf("read linear's data: %w", err)
	}
	// Spent the window: wait out its reset before the next request rather than be refused; a long one is left to
	// the refusal, which pauses the import.
	if wait := l.exhausted(resp.Header); wait > 0 && wait <= linearMaxWait {
		return 0, l.wait(ctx, wait)
	}
	return 0, nil
}

// answer reads a GraphQL answer into its data, a rate limit to wait out, or why it failed.
func (l *LinearSource) answer(resp *http.Response, body []byte) (json.RawMessage, time.Duration, error) {
	var answer struct {
		Data   json.RawMessage `json:"data"`
		Errors []linearError   `json:"errors"`
	}
	decodeErr := json.Unmarshal(body, &answer)
	codes := make([]string, 0, len(answer.Errors))
	for _, e := range answer.Errors {
		codes = append(codes, e.Extensions.Code)
	}
	if resp.StatusCode == http.StatusTooManyRequests || slices.Contains(codes, "RATELIMITED") {
		return nil, l.rateLimitWait(resp.Header), nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || slices.Contains(codes, "AUTHENTICATION_ERROR") {
		return nil, 0, errLinearKey
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, 0, apperrs.Retryable(fmt.Errorf("linear answered %s", resp.Status))
	}
	if decodeErr != nil {
		return nil, 0, fmt.Errorf("read linear's answer (%s): %w", resp.Status, decodeErr)
	}
	if len(answer.Errors) > 0 {
		return nil, 0, fmt.Errorf("linear answered %s: %s", resp.Status, answer.Errors[0].Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("linear answered %s", resp.Status)
	}
	return answer.Data, 0, nil
}

// rateLimitWait is how long a refused request waits: Retry-After, else the reset of the spent window.
func (l *LinearSource) rateLimitWait(h http.Header) time.Duration {
	if s, err := strconv.Atoi(strings.TrimSpace(h.Get("Retry-After"))); err == nil && s > 0 {
		return time.Duration(s) * time.Second
	}
	if wait := l.exhausted(h); wait > 0 {
		return wait
	}
	return linearDefaultWait
}

// exhausted is the time until the request or complexity window resets, when either has nothing left; 0 otherwise.
func (l *LinearSource) exhausted(h http.Header) time.Duration {
	var wait time.Duration
	for _, window := range []string{"Requests", "Complexity"} {
		left, err := strconv.Atoi(strings.TrimSpace(h.Get("X-RateLimit-" + window + "-Remaining")))
		if err != nil || left > 0 {
			continue
		}
		resetMs, err := strconv.ParseInt(strings.TrimSpace(h.Get("X-RateLimit-"+window+"-Reset")), 10, 64)
		if err != nil {
			wait = max(wait, linearDefaultWait)
			continue
		}
		wait = max(wait, time.UnixMilli(resetMs).Sub(l.now()), time.Second)
	}
	return wait
}

// LinearTicket is the one place a Linear issue becomes a ticket: its title, its body with a line back to the issue
// and its parent, its state by type, its labels with its priority and a canceled mark, its project as the category,
// its due date and assignee, its blocking relations, its comments and the files they hold.
func LinearTicket(i linearIssue, comments []linearComment) Ticket {
	body := i.Description
	if i.Parent != nil && i.Parent.Identifier != "" {
		body = withLine(body, "Sub-issue of "+i.Parent.Identifier+".")
	}
	body = withLine(body, fmt.Sprintf("Imported from [%s](%s).", i.Identifier, i.URL))
	t := Ticket{
		SourceID: i.ID, Title: i.Title, Body: body, Status: linearStatus(i.State.Name, i.State.Type),
		DueOn: i.DueDate, Files: linearFiles(i.Description),
	}
	for _, label := range i.Labels.Nodes {
		t.Labels = append(t.Labels, label.Name)
	}
	if p := linearPriority(i.Priority); p != "" {
		t.Labels = append(t.Labels, p)
	}
	if i.State.Type == "canceled" {
		t.Labels = append(t.Labels, "canceled")
	}
	if i.Project != nil {
		t.Category = i.Project.Name
	}
	if i.Assignee != nil {
		t.Assignee = linearPerson(i.Assignee)
	}
	for _, r := range i.Relations.Nodes {
		if r.Type == "blocks" {
			t.Blocks = append(t.Blocks, r.RelatedIssue.ID)
		}
	}
	for _, r := range i.InverseRelations.Nodes {
		if r.Type == "blocks" {
			t.BlockedBy = append(t.BlockedBy, r.Issue.ID)
		}
	}
	for _, c := range comments {
		author := Person{Name: "Linear"}
		avatar := ""
		if c.User != nil {
			author, avatar = linearPerson(c.User), c.User.AvatarURL
		}
		t.Comments = append(t.Comments, Comment{SourceID: c.ID, Body: c.Body, Author: author, AvatarURL: avatar, At: c.CreatedAt})
		t.Files = append(t.Files, linearFiles(c.Body)...)
	}
	return t
}

func linearPerson(u *linearUser) Person { return Person{Email: u.Email, Name: u.Name} }

// linearStatus maps a workflow state by its type; the column of the state's own name wins when the project has one.
func linearStatus(name, kind string) StatusRef {
	ref := StatusRef{Name: name, Kind: StatusBacklog}
	switch kind {
	case "started":
		ref.Kind = StatusProgress
	case "completed", "canceled":
		ref.Kind = StatusDone
	}
	return ref
}

func linearPriority(p float64) string {
	switch int(p) {
	case 1:
		return "priority:urgent"
	case 2:
		return "priority:high"
	case 3:
		return "priority:medium"
	case 4:
		return "priority:low"
	}
	return ""
}

// linearFile matches the links Linear writes when someone drops a file into an issue or comment, signed or not.
var linearFile = regexp.MustCompile(`https://uploads\.linear\.app/[^\s)"'<>\]]+`)

func linearFiles(text string) []string {
	return linearFile.FindAllString(text, -1)
}
