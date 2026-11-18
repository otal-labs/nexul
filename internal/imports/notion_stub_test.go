package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const stubToken = "ntn_test_token"

// stubNotion is an in-memory Notion workspace behind Notion's API shape: search, databases, blocks and children,
// and the signed file links its uploads are served from, which expire an hour after they are handed out.
type stubNotion struct {
	mu        sync.Mutex
	srv       *httptest.Server
	pages     []map[string]any
	databases map[string]map[string]any
	blocks    map[string]map[string]any
	children  map[string][]map[string]any
	// now is the stub's clock, which file links expire by.
	now time.Time
	// throttle answers the next search with a 429 asking to wait that many seconds.
	throttle int
	requests int
	seq      int
}

func newStubNotion(t *testing.T) *stubNotion {
	s := &stubNotion{
		databases: map[string]map[string]any{}, blocks: map[string]map[string]any{}, children: map[string][]map[string]any{},
		now: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.srv.Close)
	return s
}

// stubID is the n-th made-up Notion id.
func stubID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", n)
}

func (s *stubNotion) id() string {
	s.seq++
	return stubID(s.seq)
}

func stubText(text string) []map[string]any {
	return []map[string]any{{"type": "text", "plain_text": text, "text": map[string]any{"content": text}, "annotations": map[string]any{}}}
}

func titleProps(title string) map[string]any {
	return map[string]any{"Name": map[string]any{"type": "title", "title": stubText(title)}}
}

// page adds a page under parent, a page id or "" for the workspace, with a child_page block in its parent.
func (s *stubNotion) page(parent, title string) string {
	id := s.id()
	p := map[string]any{"type": "workspace", "workspace": true}
	if parent != "" {
		p = map[string]any{"type": "page_id", "page_id": parent}
		s.children[parent] = append(s.children[parent], map[string]any{
			"id": id, "type": "child_page", "has_children": true, "child_page": map[string]any{"title": title},
		})
	}
	s.pages = append(s.pages, map[string]any{
		"object": "page", "id": id, "created_time": s.now.Add(time.Duration(s.seq) * time.Minute), "parent": p, "properties": titleProps(title),
	})
	return id
}

// database adds a database inside page, with a child_database block there.
func (s *stubNotion) database(page, title string) string {
	id := s.id()
	s.databases[id] = map[string]any{"object": "database", "id": id, "title": stubText(title), "parent": map[string]any{"type": "page_id", "page_id": page}}
	s.children[page] = append(s.children[page], map[string]any{
		"id": id, "type": "child_database", "has_children": false, "child_database": map[string]any{"title": title},
	})
	return id
}

// row adds a row to database with extra properties beside its title.
func (s *stubNotion) row(database, title string, props map[string]any) string {
	id := s.id()
	all := titleProps(title)
	for k, v := range props {
		all[k] = v
	}
	s.pages = append(s.pages, map[string]any{
		"object": "page", "id": id, "created_time": s.now.Add(time.Duration(s.seq) * time.Minute),
		"parent": map[string]any{"type": "data_source_id", "data_source_id": "ds-" + database, "database_id": database}, "properties": all,
	})
	return id
}

// add appends a block of type to parent's content.
func (s *stubNotion) add(parent, typ string, content map[string]any) string {
	id := s.id()
	b := map[string]any{"id": id, "type": typ, "has_children": false, typ: content, "parent": map[string]any{"type": "page_id", "page_id": parent}}
	s.blocks[id] = b
	s.children[parent] = append(s.children[parent], b)
	return id
}

// image adds a Notion-hosted image; its signed link is minted on every read of the page, valid for an hour.
func (s *stubNotion) image(parent, name string) {
	s.add(parent, "image", map[string]any{"type": "file", "file": map[string]any{"url": "minted:" + name}, "caption": []any{}})
}

func (s *stubNotion) fileURL(name string) string {
	return s.srv.URL + "/files/" + name + "?expires=" + strconv.FormatInt(s.now.Add(time.Hour).Unix(), 10)
}

func (s *stubNotion) advance(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = s.now.Add(d)
}

func (s *stubNotion) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests++
	if strings.HasPrefix(r.URL.Path, "/files/") {
		s.serveFile(w, r)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+stubToken {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"unauthorized","message":"API token is invalid."}`))
		return
	}
	if r.Header.Get("Notion-Version") != NotionVersion {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v1/search":
		s.search(w, r)
	case strings.HasPrefix(r.URL.Path, "/v1/databases/"):
		s.one(w, s.databases[strings.TrimPrefix(r.URL.Path, "/v1/databases/")])
	case strings.HasSuffix(r.URL.Path, "/children"):
		s.list(w, r, s.childrenOf(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/blocks/"), "/children")))
	case strings.HasPrefix(r.URL.Path, "/v1/blocks/"):
		s.one(w, s.blocks[strings.TrimPrefix(r.URL.Path, "/v1/blocks/")])
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// childrenOf is a block's children with each Notion-hosted file's link minted now.
func (s *stubNotion) childrenOf(id string) []map[string]any {
	out := make([]map[string]any, 0, len(s.children[id]))
	for _, b := range s.children[id] {
		img, ok := b["image"].(map[string]any)
		if !ok || img["type"] != "file" {
			out = append(out, b)
			continue
		}
		name := strings.TrimPrefix(img["file"].(map[string]any)["url"].(string), "minted:")
		minted := map[string]any{}
		for k, v := range b {
			minted[k] = v
		}
		minted["image"] = map[string]any{"type": "file", "file": map[string]any{"url": s.fileURL(name)}, "caption": img["caption"]}
		out = append(out, minted)
	}
	return out
}

func (s *stubNotion) serveFile(w http.ResponseWriter, r *http.Request) {
	exp, _ := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
	if s.now.Unix() > exp {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Request has expired"))
		return
	}
	_, _ = w.Write([]byte("PNG bytes of " + strings.TrimPrefix(r.URL.Path, "/files/")))
}

func (s *stubNotion) search(w http.ResponseWriter, r *http.Request) {
	if s.throttle > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(s.throttle))
		s.throttle = 0
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	var body struct {
		StartCursor string `json:"start_cursor"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	start, _ := strconv.Atoi(body.StartCursor)
	s.page100(w, s.pages, start)
}

func (s *stubNotion) list(w http.ResponseWriter, r *http.Request, items []map[string]any) {
	start, _ := strconv.Atoi(r.URL.Query().Get("start_cursor"))
	s.page100(w, items, start)
}

func (s *stubNotion) page100(w http.ResponseWriter, items []map[string]any, start int) {
	end := min(start+100, len(items))
	out := map[string]any{"object": "list", "results": items[start:end], "has_more": end < len(items), "next_cursor": nil}
	if end < len(items) {
		out["next_cursor"] = strconv.Itoa(end)
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *stubNotion) one(w http.ResponseWriter, v map[string]any) {
	if v == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"object_not_found","message":"Could not find it."}`))
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// stubFiles downloads from the stub over plain HTTP, as Fetcher does over HTTPS from the source's hosts.
type stubFiles struct{}

func (stubFiles) Fetch(ctx context.Context, rawURL string, _ []string) (string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("download answered %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	return strings.TrimPrefix(req.URL.Path, "/files/"), data, err
}

// fakeShelf keeps the docs and folders an import wrote.
type fakeShelf struct {
	mu       sync.Mutex
	folders  map[string]string
	docs     map[string]NewDoc
	archived map[string]bool
	files    map[string][]byte
	// failBody answers SetDocBody with this error once.
	failBody error
	created  int
}

func newFakeShelf() *fakeShelf {
	return &fakeShelf{folders: map[string]string{}, docs: map[string]NewDoc{}, archived: map[string]bool{}, files: map[string][]byte{}}
}

func (f *fakeShelf) Folder(_ context.Context, _, name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.ToLower(name)
	if id, ok := f.folders[key]; ok {
		return id, nil
	}
	f.folders[key] = "folder-" + key
	return f.folders[key], nil
}

func (f *fakeShelf) CreateDoc(_ context.Context, d NewDoc) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.docs[d.ID]; ok {
		delete(f.archived, d.ID)
		return apperrs.ErrConflict
	}
	f.docs[d.ID] = d
	f.created++
	return nil
}

func (f *fakeShelf) SetDocBody(_ context.Context, docID, title, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failBody != nil {
		err := f.failBody
		f.failBody = nil
		return err
	}
	d := f.docs[docID]
	d.Title, d.Body = title, body
	f.docs[docID] = d
	return nil
}

func (f *fakeShelf) AttachToDoc(_ context.Context, _, name string, data []byte) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := "att-" + name
	f.files[id] = data
	return id, nil
}

func (f *fakeShelf) ArchiveDoc(_ context.Context, docID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.docs[docID]; !ok {
		return apperrs.ErrNotFound
	}
	f.archived[docID] = true
	return nil
}

// byTitle is the doc of that title.
func (f *fakeShelf) byTitle(title string) (NewDoc, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.docs {
		if d.Title == title {
			return d, true
		}
	}
	return NewDoc{}, false
}

// newNotionService runs imports from the stub into the fakes.
func newNotionService(stub *stubNotion, shelf *fakeShelf) (*Service, *fakeRepo) {
	repo := newFakeRepo()
	client := NewNotionClient(stub.srv.URL, stub.srv.Client(), 0)
	return NewService(Deps{
		Repo: repo, Owners: fakeOwners{owner: testOwner}, Projects: fakeProjects{testProject: testWorkspace},
		Board: &fakeBoard{}, People: fakePeople{}, Desk: newFakeDesk(), Shelf: shelf, Files: stubFiles{},
		Sources: map[string]Source{SourceNotion: NewNotionSource(client)},
	}), repo
}
