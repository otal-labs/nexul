package imports

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// progressEvery is how many records go by between progress frames; counts are saved on every record.
const progressEvery = 25

// sink writes one import's records as its source hands them over.
type sink struct {
	s       *Service
	imp     *Import
	info    Info
	members []Member
	cols    []Column
	// folders are the doc folders this run has found or made, by name.
	folders map[string]string
	// cats is the project's categories, read the first time a ticket names one.
	cats []Category
}

var _ Sink = (*sink)(nil)

// derivedID is the id a source record gets in a workspace, the same in every import, so a write whose record was lost
// (a crash, then a cancel) conflicts on the next attempt or the next import instead of writing it twice.
func derivedID(workspaceID, source, kind, sourceID string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("nexul-import:"+workspaceID+"/"+source+"/"+kind+"/"+sourceID)).String()
}

func (k *sink) id(kind, sourceID string) string {
	return derivedID(k.imp.WorkspaceID, k.imp.Source, kind, sourceID)
}

// Ticket implements Sink.
func (k *sink) Ticket(ctx context.Context, t Ticket) error {
	t.SourceID = strings.TrimSpace(t.SourceID)
	if t.SourceID == "" {
		return fmt.Errorf("%w: %s handed a ticket without a source id", apperrs.ErrInvalid, k.imp.Source)
	}
	item, ok, err := k.s.repo.Item(ctx, k.imp.ID, KindTicket, t.SourceID)
	if err != nil {
		return err
	}
	if ok && item.State != ItemCreated {
		return nil
	}
	if t.Body, err = k.parentLine(ctx, t); err != nil {
		return err
	}
	if !ok {
		_, before, err := k.s.repo.Imported(ctx, k.imp.WorkspaceID, k.imp.Source, KindTicket, t.SourceID)
		if err != nil {
			return err
		}
		if before {
			k.imp.Report.Skipped++
			return k.counted(ctx, true)
		}
		if item, err = k.create(ctx, t); err != nil {
			return k.failed(ctx, KindTicket, t.SourceID, t.Title, "", err)
		}
	}
	if err := k.fill(ctx, item.NexulID, t); err != nil {
		return k.failed(ctx, KindTicket, t.SourceID, t.Title, item.NexulID, err)
	}
	item.State = ItemFilled
	if err := k.s.repo.Record(ctx, k.imp.WorkspaceID, k.imp.Source, item); err != nil {
		return err
	}
	return k.counted(ctx, true)
}

// create files the ticket and records it, so a restart from here only fills it.
func (k *sink) create(ctx context.Context, t Ticket) (Item, error) {
	col, err := k.column(ctx, t.Status)
	if err != nil {
		return Item{}, err
	}
	categoryID, err := k.category(ctx, t.Category)
	if err != nil {
		return Item{}, err
	}
	nt := NewTicket{
		ID: k.id(KindTicket, t.SourceID), ProjectID: k.imp.ProjectID, Title: ticketTitle(t), Body: k.bodyOf(t),
		Status: col.ID, Developer: memberFor(k.members, t.Assignee), Source: k.imp.Source, Labels: t.Labels,
		CategoryID: categoryID, DueOn: t.DueOn,
	}
	err = k.s.desk.FileTicket(ctx, nt)
	if err != nil && !errors.Is(err, apperrs.ErrConflict) {
		return Item{}, err
	}
	item := Item{ImportID: k.imp.ID, Kind: KindTicket, SourceID: t.SourceID, NexulID: nt.ID, State: ItemCreated}
	if err := k.s.repo.Record(ctx, k.imp.WorkspaceID, k.imp.Source, item); err != nil {
		return Item{}, err
	}
	return item, nil
}

// fill copies the ticket's files and comments in; each piece is safe to repeat after a restart.
func (k *sink) fill(ctx context.Context, ticketID string, t Ticket) error {
	copied := k.copyFiles(ctx, ticketID, t.Files)
	if len(copied) > 0 {
		if err := k.s.desk.SetBody(ctx, ticketID, ticketTitle(t), rewriteLinks(k.bodyOf(t), copied)); err != nil {
			return fmt.Errorf("point the body at its files: %w", err)
		}
	}
	for _, c := range t.Comments {
		if strings.TrimSpace(c.Body) == "" {
			continue
		}
		nc := NewComment{
			ID: k.id(KindComment, c.SourceID), AuthorKey: k.imp.Source + ":" + personName(c.Author),
			Name: personName(c.Author), AvatarURL: c.AvatarURL, Body: rewriteLinks(c.Body, copied), Via: k.imp.Source, At: c.At,
		}
		err := k.s.desk.Comment(ctx, ticketID, nc)
		if err != nil && !errors.Is(err, apperrs.ErrConflict) {
			return fmt.Errorf("copy comment %s: %w", c.SourceID, err)
		}
	}
	return k.link(ctx, ticketID, t)
}

// link makes each blocked-by link whose other end is already in; the record that comes in second makes it.
func (k *sink) link(ctx context.Context, ticketID string, t Ticket) error {
	for _, sid := range t.BlockedBy {
		if err := k.linkTo(ctx, sid, func(other string) error { return k.s.desk.Block(ctx, ticketID, other) }); err != nil {
			return err
		}
	}
	for _, sid := range t.Blocks {
		if err := k.linkTo(ctx, sid, func(other string) error { return k.s.desk.Block(ctx, other, ticketID) }); err != nil {
			return err
		}
	}
	return nil
}

// linkTo makes one link to the ticket an import brought sourceID in as, if any import has.
func (k *sink) linkTo(ctx context.Context, sourceID string, makeLink func(other string) error) error {
	other, ok, err := k.s.repo.Imported(ctx, k.imp.WorkspaceID, k.imp.Source, KindTicket, sourceID)
	if err != nil || !ok {
		return err
	}
	err = makeLink(other)
	// A cycle, or an end deleted since, leaves that one link out rather than the ticket.
	if err != nil && !errors.Is(err, apperrs.ErrConflict) && !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("link a blocker: %w", err)
	}
	return nil
}

// category is the id of the project's category of that name, ignoring case, adding it the first time; "" for none.
func (k *sink) category(ctx context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if k.cats == nil {
		cats, err := k.s.board.Categories(ctx, k.imp.ProjectID)
		if err != nil {
			return "", fmt.Errorf("list categories: %w", err)
		}
		k.cats = append([]Category{}, cats...)
	}
	for _, c := range k.cats {
		if strings.EqualFold(strings.TrimSpace(c.Name), name) {
			return c.ID, nil
		}
	}
	c, err := k.s.board.AddCategory(ctx, k.imp.ProjectID, name)
	if err != nil {
		return "", fmt.Errorf("add category %s: %w", name, err)
	}
	k.cats = append(k.cats, c)
	return c.ID, nil
}

// parentLine is the body with a line naming the key of the ticket it is a subtask of, when that one came in.
func (k *sink) parentLine(ctx context.Context, t Ticket) (string, error) {
	if t.Parent == "" {
		return t.Body, nil
	}
	parentID, ok, err := k.s.repo.Imported(ctx, k.imp.WorkspaceID, k.imp.Source, KindTicket, t.Parent)
	if err != nil || !ok {
		return t.Body, err
	}
	key, err := k.s.desk.Key(ctx, parentID)
	if err != nil {
		return "", fmt.Errorf("read the key of %s's parent: %w", t.SourceID, err)
	}
	return withLine(t.Body, "Subtask of "+key), nil
}

// bodyOf is the ticket's body, naming an assignee no member of the project maps to, so who held it is not lost.
func (k *sink) bodyOf(t Ticket) string {
	if t.Assignee.Zero() || memberFor(k.members, t.Assignee) != "" {
		return t.Body
	}
	return withLine(t.Body, assignedLine(k.info.Title, t.Assignee))
}

// copyFiles downloads each file into an attachment on the ticket; one that fails or is too large stays a link.
func (k *sink) copyFiles(ctx context.Context, ticketID string, urls []string) map[string]string {
	return k.copyFilesTo(ctx, urls, func(name string, data []byte) (string, error) { return k.s.desk.Attach(ctx, ticketID, name, data) })
}

// copyFilesTo downloads each file and stores it with attach; one that fails or is too large stays a link.
func (k *sink) copyFilesTo(ctx context.Context, urls []string, attach func(name string, data []byte) (string, error)) map[string]string {
	copied := map[string]string{}
	for _, u := range urls {
		if _, done := copied[u]; done {
			continue
		}
		name, data, err := k.s.files.Fetch(ctx, u, k.info.FileHosts, k.fileHeader(u))
		if err != nil {
			k.s.log.InfoContext(ctx, "import kept a file as a link", "import_id", k.imp.ID, "error", err)
			k.imp.Report.FilesLinked++
			continue
		}
		id, err := attach(name, data)
		if err != nil {
			k.s.log.InfoContext(ctx, "import kept a file as a link", "import_id", k.imp.ID, "error", err)
			k.imp.Report.FilesLinked++
			continue
		}
		copied[u] = id
		k.imp.Report.FilesCopied++
	}
	return copied
}

// fileHeader is what the source signs a download of rawURL with, nil for a URL that is not its own API's.
func (k *sink) fileHeader(rawURL string) http.Header {
	if k.info.FileHeader == nil {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	return k.info.FileHeader(k.imp.Credential, u)
}

// column is where a status lands, adding the column the first time the project lacks it.
func (k *sink) column(ctx context.Context, ref StatusRef) (Column, error) {
	if c, ok := columnFor(k.cols, ref); ok {
		return c, nil
	}
	c, err := k.s.board.AddColumn(ctx, k.imp.ProjectID, newColumnName(ref), ref.Kind)
	if err != nil {
		return Column{}, fmt.Errorf("add column %s: %w", newColumnName(ref), err)
	}
	k.cols = append(k.cols, c)
	return c, nil
}

// failed records a record that did not come in and carries on with the next.
func (k *sink) failed(ctx context.Context, kind, sourceID, title, nexulID string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(cause, apperrs.ErrRetryable) {
		return cause
	}
	k.s.log.WarnContext(ctx, "import could not bring a record in", "import_id", k.imp.ID, "source_id", sourceID, "error", cause)
	reason := failureReason(cause)
	item := Item{ImportID: k.imp.ID, Kind: kind, SourceID: sourceID, NexulID: nexulID, State: ItemFailed, Error: reason}
	if err := k.s.repo.Record(ctx, k.imp.WorkspaceID, k.imp.Source, item); err != nil {
		return err
	}
	k.imp.Failed++
	if len(k.imp.Report.Failures) < MaxListedFailures {
		k.imp.Report.Failures = append(k.imp.Report.Failures, Failure{Kind: kind, SourceID: sourceID, Title: title, Error: reason})
	}
	return k.counted(ctx, false)
}

// counted saves the import's counts, sending a progress frame every progressEvery records; a cancel meanwhile stops
// the run.
func (k *sink) counted(ctx context.Context, done bool) error {
	if done {
		k.imp.Done++
	}
	k.imp.UpdatedAt = k.s.now().UTC()
	var ok bool
	var err error
	if (k.imp.Done+k.imp.Failed)%progressEvery == 0 {
		ok, err = k.s.repo.Update(ctx, k.imp, StateImporting, progressEvent(k.imp))
	}
	if (k.imp.Done+k.imp.Failed)%progressEvery != 0 {
		ok, err = k.s.repo.Update(ctx, k.imp, StateImporting)
	}
	if err != nil {
		return fmt.Errorf("save progress: %w", err)
	}
	if !ok {
		return errStopped
	}
	return nil
}

// DocLink implements Sink: the id is derived, so the link holds before the doc is written.
func (k *sink) DocLink(sourceID string) string {
	return "/docs/" + k.id(KindDoc, strings.TrimSpace(sourceID))
}

// Skip implements Sink.
func (k *sink) Skip(ctx context.Context, kind, sourceID string) (bool, error) {
	sourceID = strings.TrimSpace(sourceID)
	item, ok, err := k.s.repo.Item(ctx, k.imp.ID, kind, sourceID)
	if err != nil || ok {
		return ok && item.State != ItemCreated, err
	}
	_, before, err := k.s.repo.Imported(ctx, k.imp.WorkspaceID, k.imp.Source, kind, sourceID)
	if err != nil || !before {
		return false, err
	}
	k.imp.Report.Skipped++
	return true, k.counted(ctx, true)
}

// Fail implements Sink.
func (k *sink) Fail(ctx context.Context, kind, sourceID, title string, cause error) error {
	return k.failed(ctx, kind, strings.TrimSpace(sourceID), title, "", cause)
}

// Doc implements Sink.
func (k *sink) Doc(ctx context.Context, d Doc) error {
	d.SourceID = strings.TrimSpace(d.SourceID)
	if d.SourceID == "" {
		return fmt.Errorf("%w: %s handed a doc without a source id", apperrs.ErrInvalid, k.imp.Source)
	}
	item, ok, err := k.s.repo.Item(ctx, k.imp.ID, KindDoc, d.SourceID)
	if err != nil {
		return err
	}
	if ok && item.State != ItemCreated {
		return nil
	}
	if !ok {
		_, before, err := k.s.repo.Imported(ctx, k.imp.WorkspaceID, k.imp.Source, KindDoc, d.SourceID)
		if err != nil {
			return err
		}
		if before {
			k.imp.Report.Skipped++
			return k.counted(ctx, true)
		}
		if item, err = k.createDoc(ctx, d); err != nil {
			return k.failed(ctx, KindDoc, d.SourceID, d.Title, "", err)
		}
	}
	copied := k.copyFilesTo(ctx, d.Files, func(name string, data []byte) (string, error) {
		return k.s.shelf.AttachToDoc(ctx, item.NexulID, name, data)
	})
	if len(copied) > 0 {
		if err := k.s.shelf.SetDocBody(ctx, item.NexulID, docTitle(d), rewriteLinks(d.Body, copied)); err != nil {
			return k.failed(ctx, KindDoc, d.SourceID, d.Title, item.NexulID, fmt.Errorf("point the body at its files: %w", err))
		}
	}
	k.imp.Report.Unsupported += d.Unsupported
	item.State = ItemFilled
	if err := k.s.repo.Record(ctx, k.imp.WorkspaceID, k.imp.Source, item); err != nil {
		return err
	}
	return k.counted(ctx, true)
}

// createDoc writes the doc into its folder and records it, so a restart from here only fills it.
func (k *sink) createDoc(ctx context.Context, d Doc) (Item, error) {
	folderID, err := k.folder(ctx, d.Folder)
	if err != nil {
		return Item{}, err
	}
	nd := NewDoc{ID: k.id(KindDoc, d.SourceID), ProjectID: k.imp.ProjectID, FolderID: folderID, Title: docTitle(d), Body: d.Body}
	err = k.s.shelf.CreateDoc(ctx, nd)
	if err != nil && !errors.Is(err, apperrs.ErrConflict) {
		return Item{}, err
	}
	item := Item{ImportID: k.imp.ID, Kind: KindDoc, SourceID: d.SourceID, NexulID: nd.ID, State: ItemCreated}
	if err := k.s.repo.Record(ctx, k.imp.WorkspaceID, k.imp.Source, item); err != nil {
		return Item{}, err
	}
	return item, nil
}

// folder is the id of the doc folder named name, the project's default folder when name is empty.
func (k *sink) folder(ctx context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if id, ok := k.folders[strings.ToLower(name)]; ok {
		return id, nil
	}
	id, err := k.s.shelf.Folder(ctx, k.imp.ProjectID, name)
	if err != nil {
		return "", fmt.Errorf("folder %s: %w", name, err)
	}
	if k.folders == nil {
		k.folders = map[string]string{}
	}
	k.folders[strings.ToLower(name)] = id
	return id, nil
}

func docTitle(d Doc) string {
	if title := strings.TrimSpace(d.Title); title != "" {
		return title
	}
	return "Untitled"
}

// Checkpoint implements Sink.
func (k *sink) Checkpoint(ctx context.Context, cursor string) error {
	k.imp.Report.Cursor = cursor
	ok, err := k.s.repo.Update(ctx, k.imp, StateImporting)
	if err != nil {
		return fmt.Errorf("save cursor: %w", err)
	}
	if !ok {
		return errStopped
	}
	return nil
}

func ticketTitle(t Ticket) string {
	if title := strings.TrimSpace(t.Title); title != "" {
		return title
	}
	return "Untitled " + t.SourceID
}

func withLine(body, line string) string {
	body = strings.TrimRight(body, "\n ")
	if body == "" {
		return line
	}
	return body + "\n\n" + line
}
