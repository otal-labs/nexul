package imports

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// SourceTrello imports one Trello board, read with an API key and token given as key:token.
const SourceTrello = "trello"

// TrelloAPI is Trello's REST API.
const TrelloAPI = "https://api.trello.com/1"

const (
	trelloPage = 1000
	// trelloEvery spaces requests at 8 a second, under the 100 per 10 seconds a token may make.
	trelloEvery = 125 * time.Millisecond
	maxChoices  = 20
)

// trelloFileHosts are where Trello serves uploaded attachments, and where a download redirects.
var trelloFileHosts = []string{"trello.com", "api.trello.com", "trello-attachments.s3.amazonaws.com"}

// TrelloSource is the Trello boards source.
type TrelloSource struct {
	api  restAPI
	page int
}

var _ Source = (*TrelloSource)(nil)

// NewTrelloSource reads Trello at base, TrelloAPI outside tests; a nil client uses one with a one-minute bound.
func NewTrelloSource(base string, client *http.Client) *TrelloSource {
	return &TrelloSource{api: newRestAPI(base, client), page: trelloPage}
}

// Info implements Source: an upload is downloaded from Trello's API signed with the import's key and token.
func (s *TrelloSource) Info() Info {
	return Info{Title: "Trello", FileHosts: trelloFileHosts, FileHeader: func(credential string, u *url.URL) http.Header {
		host := strings.ToLower(u.Hostname())
		if host != "trello.com" && host != "api.trello.com" {
			return nil
		}
		h, err := trelloHeader(credential)
		if err != nil {
			return nil
		}
		return h
	}}
}

// trelloHeader is the Authorization header for key:token; Trello takes it there, never in the URL.
func trelloHeader(credential string) (http.Header, error) {
	key, token, ok := strings.Cut(strings.TrimSpace(credential), ":")
	key, token = strings.TrimSpace(key), strings.TrimSpace(token)
	if !ok || key == "" || token == "" {
		return nil, fmt.Errorf("%w: a Trello import takes its API key and token as key:token", apperrs.ErrInvalid)
	}
	return http.Header{"Authorization": {fmt.Sprintf(`OAuth oauth_consumer_key="%s", oauth_token="%s"`, key, token)}}, nil
}

func (s *TrelloSource) client(job Job) (*restClient, error) {
	h, err := trelloHeader(job.Credential)
	if err != nil {
		return nil, err
	}
	return &restClient{api: s.api, title: "Trello", header: h, every: trelloEvery, pause: trelloPause}, nil
}

// trelloPause waits out the rest of the interval once a key or token has no requests left in it.
func trelloPause(h http.Header) time.Duration {
	var wait time.Duration
	for _, kind := range []string{"key", "token"} {
		prefix := "X-Rate-Limit-Api-" + kind + "-"
		if h.Get(prefix+"Remaining") != "0" {
			continue
		}
		ms, err := strconv.Atoi(h.Get(prefix + "Interval-Ms"))
		if err != nil {
			ms = 10_000
		}
		wait = max(wait, time.Duration(ms)*time.Millisecond)
	}
	return wait
}

type trelloBoard struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortLink string `json:"shortLink"`
	Closed    bool   `json:"closed"`
}

type trelloList struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

type trelloMember struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	AvatarURL string `json:"avatarUrl"`
}

type trelloCard struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Desc      string   `json:"desc"`
	Closed    bool     `json:"closed"`
	IDList    string   `json:"idList"`
	IDMembers []string `json:"idMembers"`
	Due       string   `json:"due"`
	ShortURL  string   `json:"shortUrl"`
	Labels    []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	Badges struct {
		Comments int `json:"comments"`
	} `json:"badges"`
	Checklists  []trelloChecklist `json:"checklists"`
	Attachments []struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		IsUpload bool   `json:"isUpload"`
	} `json:"attachments"`
}

type trelloChecklist struct {
	Name       string            `json:"name"`
	Pos        float64           `json:"pos"`
	CheckItems []trelloCheckItem `json:"checkItems"`
}

type trelloCheckItem struct {
	Name  string  `json:"name"`
	State string  `json:"state"`
	Pos   float64 `json:"pos"`
}

type trelloAction struct {
	ID   string    `json:"id"`
	Date time.Time `json:"date"`
	Data struct {
		Text string `json:"text"`
	} `json:"data"`
	MemberCreator trelloMember `json:"memberCreator"`
}

// trelloRead is one board's lists and members, which every card refers to by id.
type trelloRead struct {
	c      *restClient
	board  trelloBoard
	lists  map[string]trelloList
	status map[string]StatusRef
	// order is the open lists' statuses in board order.
	order   []StatusRef
	members map[string]trelloMember
}

// Scan implements Source: it counts the board's open cards on open lists, and the archived ones apart.
func (s *TrelloSource) Scan(ctx context.Context, job Job) (Scan, error) {
	r, err := s.open(ctx, job)
	if err != nil {
		return Scan{}, err
	}
	found := Scan{From: r.board.ShortLink, Statuses: r.order}
	for _, l := range r.lists {
		if l.Closed {
			found.Archived++
		}
	}
	err = s.cards(ctx, r, "", func(c trelloCard) error {
		if !r.open(c) {
			found.Archived++
			return nil
		}
		found.Tickets++
		found.Comments += c.Badges.Comments
		if p := r.assignee(c); !p.Zero() {
			found.People = append(found.People, p)
		}
		return nil
	}, nil)
	return found, err
}

// Import implements Source: a page of cards at a time from the checkpointed card, each open one with its comments.
func (s *TrelloSource) Import(ctx context.Context, job Job, sink Sink) error {
	r, err := s.open(ctx, job)
	if err != nil {
		return err
	}
	return s.cards(ctx, r, job.Cursor, func(c trelloCard) error {
		if !r.open(c) {
			return nil
		}
		var comments []trelloAction
		if c.Badges.Comments > 0 {
			// shortcut: one page of a card's comments, so past 1,000 the oldest are left out; page with before once a board needs it.
			q := url.Values{"filter": {"commentCard"}, "limit": {"1000"}}
			if err := r.c.get(ctx, "/cards/"+c.ID+"/actions", q, &comments); err != nil {
				return err
			}
		}
		return sink.Ticket(ctx, r.ticket(c, comments))
	}, func(cursor string) error { return sink.Checkpoint(ctx, cursor) })
}

// cards hands every card of the board to each, a page at a time from the newest, starting before cursor;
// checkpoint, when set, records where the next page starts.
func (s *TrelloSource) cards(ctx context.Context, r *trelloRead, cursor string, each func(trelloCard) error, checkpoint func(string) error) error {
	q := url.Values{
		"fields":            {"name,desc,closed,idList,idMembers,due,shortUrl,labels,badges"},
		"checklists":        {"all"},
		"attachments":       {"true"},
		"attachment_fields": {"name,url,isUpload"},
		"limit":             {strconv.Itoa(s.page)},
	}
	for before := cursor; ; {
		if before != "" {
			q.Set("before", before)
		}
		var page []trelloCard
		if err := r.c.get(ctx, "/boards/"+r.board.ID+"/cards/all", q, &page); err != nil {
			return err
		}
		for _, c := range page {
			if err := each(c); err != nil {
				return err
			}
		}
		if len(page) < s.page {
			return nil
		}
		// Card ids grow with time, so the oldest on the page is where the next one starts.
		before = slices.MinFunc(page, func(a, b trelloCard) int { return strings.Compare(a.ID, b.ID) }).ID
		if checkpoint != nil {
			if err := checkpoint(before); err != nil {
				return err
			}
		}
	}
}

// open reads the board the import names, or the person's only open board, with its lists and members.
func (s *TrelloSource) open(ctx context.Context, job Job) (*trelloRead, error) {
	c, err := s.client(job)
	if err != nil {
		return nil, err
	}
	id, err := s.boardID(ctx, c, job.From)
	if err != nil {
		return nil, err
	}
	r := &trelloRead{c: c, lists: map[string]trelloList{}, status: map[string]StatusRef{}, members: map[string]trelloMember{}}
	if err := c.get(ctx, "/boards/"+url.PathEscape(id), url.Values{"fields": {"name,shortLink,closed"}}, &r.board); err != nil {
		return nil, err
	}
	var lists []trelloList
	if err := c.get(ctx, "/boards/"+r.board.ID+"/lists", url.Values{"filter": {"all"}, "fields": {"name,closed"}}, &lists); err != nil {
		return nil, err
	}
	open := 0
	for _, l := range lists {
		r.lists[l.ID] = l
		if l.Closed {
			continue
		}
		r.status[l.ID] = listStatus(open, l.Name)
		r.order = append(r.order, r.status[l.ID])
		open++
	}
	var members []trelloMember
	if err := c.get(ctx, "/boards/"+r.board.ID+"/members", url.Values{"fields": {"username,fullName,avatarUrl"}}, &members); err != nil {
		return nil, err
	}
	for _, m := range members {
		r.members[m.ID] = m
	}
	return r, nil
}

// trelloBoardLink reads a board's short link out of its URL, such as https://trello.com/b/abc123/roadmap.
var trelloBoardLink = regexp.MustCompile(`trello\.com/b/([A-Za-z0-9]+)`)

// boardID is the board from names, as a URL, short link or id; with none named, the person's only open board.
func (s *TrelloSource) boardID(ctx context.Context, c *restClient, from string) (string, error) {
	from = strings.TrimSpace(from)
	if m := trelloBoardLink.FindStringSubmatch(from); m != nil {
		return m[1], nil
	}
	if from != "" {
		return from, nil
	}
	var boards []trelloBoard
	if err := c.get(ctx, "/members/me/boards", url.Values{"filter": {"open"}, "fields": {"name,shortLink"}}, &boards); err != nil {
		return "", err
	}
	if len(boards) == 1 {
		return boards[0].ID, nil
	}
	if len(boards) == 0 {
		return "", fmt.Errorf("%w: the credential sees no open Trello board", apperrs.ErrInvalid)
	}
	names := make([]string, 0, maxChoices)
	for _, b := range boards[:min(len(boards), maxChoices)] {
		names = append(names, b.Name+" ("+b.ShortLink+")")
	}
	return "", fmt.Errorf("%w: the credential sees %d boards; name one in from by its URL or short link: %s", apperrs.ErrInvalid, len(boards), strings.Join(names, ", "))
}

// open reports whether a card comes in: it and its list are both open.
func (r *trelloRead) open(c trelloCard) bool {
	_, listOpen := r.status[c.IDList]
	return !c.Closed && listOpen
}

func (r *trelloRead) person(memberID string) Person {
	m, ok := r.members[memberID]
	if !ok {
		return Person{}
	}
	return Person{Username: m.Username, Name: m.FullName}
}

func (r *trelloRead) assignee(c trelloCard) Person {
	if len(c.IDMembers) == 0 {
		return Person{}
	}
	return r.person(c.IDMembers[0])
}

// ticket is the one place a Trello card becomes a ticket: its name, its description with its checklists and
// attachments and a line back to the card, its list, labels, first member, due day and comments.
func (r *trelloRead) ticket(c trelloCard, comments []trelloAction) Ticket {
	parts := []string{strings.TrimSpace(c.Desc)}
	lists := slices.Clone(c.Checklists)
	slices.SortStableFunc(lists, func(a, b trelloChecklist) int { return cmp.Compare(a.Pos, b.Pos) })
	for _, cl := range lists {
		items := slices.Clone(cl.CheckItems)
		slices.SortStableFunc(items, func(a, b trelloCheckItem) int { return cmp.Compare(a.Pos, b.Pos) })
		var checks []checkItem
		for _, it := range items {
			checks = append(checks, checkItem{Name: it.Name, Done: it.State == "complete"})
		}
		parts = append(parts, taskList(cl.Name, checks))
	}
	t := Ticket{SourceID: c.ID, Title: c.Name, Status: r.status[c.IDList], Assignee: r.assignee(c), DueOn: dayOf(c.Due)}
	var files []fileLink
	for _, a := range c.Attachments {
		files = append(files, fileLink{Name: a.Name, URL: a.URL})
		if a.IsUpload {
			t.Files = append(t.Files, a.URL)
		}
	}
	if len(files) > 0 {
		parts = append(parts, fileLinks(files))
	}
	parts = append(parts, fmt.Sprintf("Imported from [Trello](%s), board %s.", c.ShortURL, r.board.Name))
	t.Body = joinParts(parts)
	for _, l := range c.Labels {
		t.Labels = append(t.Labels, labelName(l.Name, l.Color))
	}
	for _, a := range comments {
		m := a.MemberCreator
		avatar := ""
		if m.AvatarURL != "" {
			avatar = m.AvatarURL + "/50.png"
		}
		t.Comments = append(t.Comments, Comment{
			SourceID: a.ID, Body: a.Data.Text, Author: Person{Username: m.Username, Name: m.FullName}, AvatarURL: avatar, At: a.Date,
		})
	}
	slices.SortStableFunc(t.Comments, func(a, b Comment) int { return a.At.Compare(b.At) })
	return t
}

// labelName is a label's name, or for one Trello shows only as a colour, the colour.
func labelName(name, color string) string {
	return cmp.Or(strings.TrimSpace(name), strings.TrimSpace(color))
}

// dayOf is the UTC day of an RFC 3339 time, "" for none or one it cannot read.
func dayOf(v string) string {
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
	if err != nil {
		return ""
	}
	return at.UTC().Format("2006-01-02")
}

// joinParts is a body of the non-empty parts, a blank line apart.
func joinParts(parts []string) string {
	kept := slices.DeleteFunc(slices.Clone(parts), func(p string) bool { return strings.TrimSpace(p) == "" })
	return strings.Join(kept, "\n\n")
}
