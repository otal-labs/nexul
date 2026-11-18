package imports

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// SourceAsana imports one Asana project, read with a personal access token.
const SourceAsana = "asana"

// AsanaAPI is Asana's REST API.
const AsanaAPI = "https://app.asana.com/api/1.0"

const asanaPage = 100

// asanaFileHosts are where an attachment's download_url points: Asana's private file buckets.
var asanaFileHosts = []string{"asana-user-private-us-east-1.s3.amazonaws.com", "asana-user-private-eu-west-1.s3.amazonaws.com"}

const asanaTaskFields = "name,notes,completed,due_on,permalink_url,num_subtasks,assignee.email,assignee.name,tags.name,memberships.project.gid,memberships.section.gid"

// AsanaSource is the Asana projects source.
type AsanaSource struct {
	api restAPI
}

var _ Source = (*AsanaSource)(nil)

// NewAsanaSource reads Asana at base, AsanaAPI outside tests; a nil client uses one with a one-minute bound.
func NewAsanaSource(base string, client *http.Client) *AsanaSource {
	return &AsanaSource{api: newRestAPI(base, client)}
}

// Info implements Source: a download_url is presigned, so files come down without the token.
func (s *AsanaSource) Info() Info { return Info{Title: "Asana", FileHosts: asanaFileHosts} }

type asanaRef struct {
	GID   string `json:"gid"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type asanaTask struct {
	GID          string     `json:"gid"`
	Name         string     `json:"name"`
	Notes        string     `json:"notes"`
	Completed    bool       `json:"completed"`
	DueOn        string     `json:"due_on"`
	PermalinkURL string     `json:"permalink_url"`
	NumSubtasks  int        `json:"num_subtasks"`
	Assignee     *asanaRef  `json:"assignee"`
	Tags         []asanaRef `json:"tags"`
	Memberships  []struct {
		Project asanaRef `json:"project"`
		Section asanaRef `json:"section"`
	} `json:"memberships"`
}

type asanaStory struct {
	GID       string    `json:"gid"`
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy *asanaRef `json:"created_by"`
}

type asanaAttachment struct {
	GID          string `json:"gid"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	DownloadURL  string `json:"download_url"`
	PermanentURL string `json:"permanent_url"`
	ViewURL      string `json:"view_url"`
}

type asanaPageOf[T any] struct {
	Data     []T `json:"data"`
	NextPage *struct {
		Offset string `json:"offset"`
	} `json:"next_page"`
}

// asanaRead is one project and where each of its sections lands.
type asanaRead struct {
	c       *restClient
	project asanaRef
	status  map[string]StatusRef
	order   []StatusRef
	done    StatusRef
}

// each hands every item of a paged list to fn, from offset onward; checkpoint, when set, records each next offset.
func asanaEach[T any](ctx context.Context, c *restClient, path string, q url.Values, offset string, fn func(T) error, checkpoint func(string) error) error {
	q.Set("limit", strconv.Itoa(asanaPage))
	for {
		q.Del("offset")
		if offset != "" {
			q.Set("offset", offset)
		}
		var page asanaPageOf[T]
		if err := c.get(ctx, path, q, &page); err != nil {
			return err
		}
		for _, item := range page.Data {
			if err := fn(item); err != nil {
				return err
			}
		}
		if page.NextPage == nil || page.NextPage.Offset == "" {
			return nil
		}
		offset = page.NextPage.Offset
		if checkpoint != nil {
			if err := checkpoint(offset); err != nil {
				return err
			}
		}
	}
}

func asanaAll[T any](ctx context.Context, c *restClient, path string, q url.Values) ([]T, error) {
	var out []T
	err := asanaEach(ctx, c, path, q, "", func(item T) error {
		out = append(out, item)
		return nil
	}, nil)
	return out, err
}

// Scan implements Source: it counts the project's tasks and their subtasks, and the assignees and sections it found.
// Comments go uncounted, since counting them reads every task's stories.
func (s *AsanaSource) Scan(ctx context.Context, job Job) (Scan, error) {
	r, err := s.open(ctx, job)
	if err != nil {
		return Scan{}, err
	}
	found := Scan{From: r.project.GID, Statuses: r.order}
	completed := false
	err = asanaEach(ctx, r.c, "/tasks", url.Values{"project": {r.project.GID}, "opt_fields": {asanaTaskFields}}, "", func(t asanaTask) error {
		found.Tickets += 1 + t.NumSubtasks
		completed = completed || t.Completed
		if p := asanaPerson(t.Assignee); !p.Zero() {
			found.People = append(found.People, p)
		}
		return nil
	}, nil)
	if completed && r.done.Name == "" {
		found.Statuses = append(found.Statuses, r.done)
	}
	return found, err
}

// Import implements Source: a page of tasks at a time from the checkpointed offset, each followed by its subtasks.
func (s *AsanaSource) Import(ctx context.Context, job Job, sink Sink) error {
	r, err := s.open(ctx, job)
	if err != nil {
		return err
	}
	err = r.tasks(ctx, sink, job.Cursor)
	if errors.Is(err, errRefused) && job.Cursor != "" {
		// A paging token Asana no longer honours starts over; the sink skips what already came in.
		return r.tasks(ctx, sink, "")
	}
	return err
}

// tasks hands every task from offset on, each followed by its subtasks.
func (r *asanaRead) tasks(ctx context.Context, sink Sink, offset string) error {
	q := url.Values{"project": {r.project.GID}, "opt_fields": {asanaTaskFields}}
	return asanaEach(ctx, r.c, "/tasks", q, offset, func(t asanaTask) error {
		status := r.statusOf(t)
		if err := r.hand(ctx, sink, t, status, ""); err != nil {
			return err
		}
		if t.NumSubtasks == 0 {
			return nil
		}
		subs, err := asanaAll[asanaTask](ctx, r.c, "/tasks/"+t.GID+"/subtasks", url.Values{"opt_fields": {asanaTaskFields}})
		if err != nil {
			return err
		}
		for _, sub := range subs {
			subStatus := status
			if sub.Completed {
				subStatus = r.done
			}
			if err := r.hand(ctx, sink, sub, subStatus, t.GID); err != nil {
				return err
			}
		}
		return nil
	}, func(offset string) error { return sink.Checkpoint(ctx, offset) })
}

// hand reads a task's comments and then its attachments, whose download links expire within minutes, and hands it
// to the sink at once, which copies the files before anything else is read.
func (r *asanaRead) hand(ctx context.Context, sink Sink, t asanaTask, status StatusRef, parent string) error {
	stories, err := asanaAll[asanaStory](ctx, r.c, "/tasks/"+t.GID+"/stories", url.Values{"opt_fields": {"type,text,created_at,created_by.name,created_by.email"}})
	if err != nil {
		return err
	}
	files, err := asanaAll[asanaAttachment](ctx, r.c, "/attachments", url.Values{"parent": {t.GID}, "opt_fields": {"name,host,download_url,permanent_url,view_url"}})
	if err != nil {
		return err
	}
	return sink.Ticket(ctx, AsanaTicket(t, status, parent, stories, files))
}

// open resolves the project the import names, or the only one the token sees, and plans its sections.
func (s *AsanaSource) open(ctx context.Context, job Job) (*asanaRead, error) {
	token := strings.TrimSpace(job.Credential)
	if token == "" {
		return nil, fmt.Errorf("%w: an Asana import takes a personal access token", apperrs.ErrInvalid)
	}
	c := &restClient{api: s.api, title: "Asana", header: http.Header{"Authorization": {"Bearer " + token}}}
	project, err := s.project(ctx, c, job.From)
	if err != nil {
		return nil, err
	}
	r := &asanaRead{c: c, project: project, status: map[string]StatusRef{}, done: StatusRef{Kind: StatusDone}}
	sections, err := asanaAll[asanaRef](ctx, c, "/projects/"+project.GID+"/sections", url.Values{"opt_fields": {"name"}})
	if err != nil {
		return nil, err
	}
	for i, sec := range sections {
		ref := listStatus(i, sec.Name)
		r.status[sec.GID] = ref
		r.order = append(r.order, ref)
		if ref.Kind == StatusDone && r.done.Name == "" {
			r.done = ref
		}
	}
	return r, nil
}

// asanaLink reads a project's id out of its URL, old (/0/<id>/...) or new (.../project/<id>/...).
var asanaLink = regexp.MustCompile(`app\.asana\.com/(?:0/|.*/project/)(\d+)`)

// project is the project from names: a project's URL or id, a workspace's id with one project, or with none named
// the token's only workspace's only project; otherwise the refusal lists what to pick from.
func (s *AsanaSource) project(ctx context.Context, c *restClient, from string) (asanaRef, error) {
	from = strings.TrimSpace(from)
	if m := asanaLink.FindStringSubmatch(from); m != nil {
		from = m[1]
	}
	if from != "" {
		var one struct{ Data asanaRef }
		err := c.get(ctx, "/projects/"+url.PathEscape(from), url.Values{"opt_fields": {"name"}}, &one)
		if err == nil {
			return one.Data, nil
		}
		if !errors.Is(err, apperrs.ErrNotFound) {
			return asanaRef{}, err
		}
		return s.only(ctx, c, from)
	}
	workspaces, err := asanaAll[asanaRef](ctx, c, "/workspaces", url.Values{"opt_fields": {"name"}})
	if err != nil {
		return asanaRef{}, err
	}
	if len(workspaces) != 1 {
		return asanaRef{}, choose("workspaces", "a workspace's or project's id", workspaces)
	}
	return s.only(ctx, c, workspaces[0].GID)
}

// only is a workspace's one unarchived project.
func (s *AsanaSource) only(ctx context.Context, c *restClient, workspace string) (asanaRef, error) {
	projects, err := asanaAll[asanaRef](ctx, c, "/projects", url.Values{"workspace": {workspace}, "archived": {"false"}, "opt_fields": {"name"}})
	if err != nil {
		return asanaRef{}, err
	}
	if len(projects) != 1 {
		return asanaRef{}, choose("projects", "a project's URL or id", projects)
	}
	return projects[0], nil
}

// choose is the refusal that lists what the token sees, so the person can name one.
func choose(what, by string, refs []asanaRef) error {
	if len(refs) == 0 {
		return fmt.Errorf("%w: the token sees no Asana %s", apperrs.ErrInvalid, what)
	}
	names := make([]string, 0, maxChoices)
	for _, r := range refs[:min(len(refs), maxChoices)] {
		names = append(names, r.Name+" ("+r.GID+")")
	}
	return fmt.Errorf("%w: the token sees %d %s; name one in from by %s: %s", apperrs.ErrInvalid, len(refs), what, by, strings.Join(names, ", "))
}

// statusOf is where a task lands: done when completed, else its section in this project.
func (r *asanaRead) statusOf(t asanaTask) StatusRef {
	if t.Completed {
		return r.done
	}
	for _, m := range t.Memberships {
		if ref, ok := r.status[m.Section.GID]; ok && m.Project.GID == r.project.GID {
			return ref
		}
	}
	return StatusRef{Kind: StatusBacklog}
}

func asanaPerson(ref *asanaRef) Person {
	if ref == nil || (ref.Email == "" && ref.Name == "") {
		return Person{}
	}
	return Person{Email: ref.Email, Name: ref.Name}
}

// AsanaTicket is the one place an Asana task becomes a ticket: its name, notes, attachments and a line back to it,
// its status, tags, assignee, due day and comments; a subtask names its parent's source id.
func AsanaTicket(t asanaTask, status StatusRef, parent string, stories []asanaStory, files []asanaAttachment) Ticket {
	out := Ticket{
		SourceID: t.GID, Title: t.Name, Status: status, Assignee: asanaPerson(t.Assignee), DueOn: t.DueOn, Parent: parent,
	}
	parts := []string{strings.TrimSpace(t.Notes)}
	var links []fileLink
	for _, f := range files {
		links = append(links, fileLink{Name: f.Name, URL: cmp.Or(f.PermanentURL, f.ViewURL)})
		if f.Host == "asana" && f.DownloadURL != "" {
			out.Files = append(out.Files, f.DownloadURL)
		}
	}
	if len(links) > 0 {
		parts = append(parts, fileLinks(links))
	}
	parts = append(parts, fmt.Sprintf("Imported from [Asana](%s).", t.PermalinkURL))
	out.Body = joinParts(parts)
	for _, tag := range t.Tags {
		out.Labels = append(out.Labels, tag.Name)
	}
	for _, st := range stories {
		if st.Type != "comment" {
			continue
		}
		out.Comments = append(out.Comments, Comment{SourceID: st.GID, Body: st.Text, Author: asanaPerson(st.CreatedBy), At: st.CreatedAt})
	}
	return out
}
