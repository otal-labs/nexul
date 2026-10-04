package docs

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// ponytail: search ranks at most 1,000 hits and pages them in memory; add FTS offsets if a query ever matches more.
const docSearchScan = 1000

// docResult is a doc as the model reads it: the body as markdown, never the stored rich-text tree.
type docResult struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	FolderID  string    `json:"folder_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Version   int       `json:"version"`
	Archived  bool      `json:"archived"`
	Locked    bool      `json:"locked"`
	UpdatedAt time.Time `json:"updated_at"`
	// Watchers are the people who get the doc's change notifications; Watching says whether the caller is one.
	Watchers []watcherResult `json:"watchers"`
	Watching bool            `json:"watching"`
	// Clarification is the doc's rounds of questions and answers, empty until a Clarify via AI run opens one.
	Clarification *Clarification `json:"clarification"`
}

type watcherResult struct {
	UserID string        `json:"user_id"`
	Source WatcherSource `json:"source"`
}

type docListIn struct {
	Query           string `json:"query,omitempty" jsonschema:"Full-text search over titles and bodies, for example deploy rollback. Omit to browse. Archived docs never match a search."`
	ProjectID       string `json:"project_id,omitempty" jsonschema:"Only docs in this project; project_list lists projects. Omit for every project."`
	FolderID        string `json:"folder_id,omitempty" jsonschema:"Only docs in this folder. Every doc lives in exactly one folder of its project, the default folder Main unless placed elsewhere; project_get lists a project's doc_folders. Omit for every folder."`
	IncludeArchived bool   `json:"include_archived,omitzero" jsonschema:"Also list archived docs when browsing without a query. Defaults to false."`
	mcptool.PageArgs
}

type docGetIn struct {
	ID string `json:"id" jsonschema:"The doc's id, from doc_list."`
}

type docCreateIn struct {
	ProjectID   string `json:"project_id" jsonschema:"The project the doc belongs to; project_list lists projects."`
	Title       string `json:"title,omitempty" jsonschema:"The doc's title, for example Storage spine. Required unless clone_from_id is set."`
	Body        string `json:"body,omitempty" jsonschema:"The doc's body as markdown. Omit for an empty doc."`
	FolderID    string `json:"folder_id,omitempty" jsonschema:"The folder of project_id the new doc goes in, from project_get's doc_folders; a folder groups a project's docs one level deep. Omit for the project's default folder, Main. Not allowed with clone_from_id."`
	CloneFromID string `json:"clone_from_id,omitempty" jsonschema:"The id of a doc to copy, from doc_list, with its attachments, into project_id instead of writing a new one. Its own project_id duplicates it there."`
}

type docUpdateIn struct {
	ID       string  `json:"id" jsonschema:"The doc's id, from doc_list."`
	Title    *string `json:"title,omitempty" jsonschema:"New title. Omit to keep the current one."`
	Body     *string `json:"body,omitempty" jsonschema:"New body as markdown, replacing the whole body. Omit to keep the current body."`
	Archived *bool   `json:"archived,omitempty" jsonschema:"true archives the doc (hidden from search), false restores it. Omit to leave it as is."`
	Locked   *bool   `json:"locked,omitempty" jsonschema:"true locks the doc read-only so its title and body refuse edits, false unlocks it; either needs docs:lock. Omit to leave it as is."`
	FolderID *string `json:"folder_id,omitempty" jsonschema:"Moves the doc to this folder of its own project, from project_get's doc_folders; every doc lives in exactly one folder, and a locked doc moves too. Omit to leave it where it is."`
	Watch    *bool   `json:"watch,omitempty" jsonschema:"true makes you a watcher of the doc, so its edits reach your notifications; false stops that, and your own later edits do not start it again. Needs only read access. Omit to leave it as is."`

	Answers             []clarifyAnswerIn   `json:"answers,omitempty" jsonschema:"Answers to the doc's clarification questions, by question id from doc_get's clarification; each replaces that question's answer, clears it, or skips it. Needs docs:write, and works on a locked doc."`
	Questions           []clarifyQuestionIn `json:"questions,omitempty" jsonschema:"The running Clarify round's questions, posted once by the run that round belongs to; anyone else is refused. Post them, then end the turn."`
	AnythingElseReply   *string             `json:"anything_else_reply,omitempty" jsonschema:"The running Clarify round's one-line reply to the previous round's Anything else? text, when it had one; from that round's run only."`
	NoGaps              bool                `json:"no_gaps,omitzero" jsonschema:"true from the running Clarify round's run when it found no gaps left, sent with body: the whole rewritten doc, which goes through the lock the run holds. Instead of questions; title is not allowed with it."`
	ClarificationClosed *bool               `json:"clarification_closed,omitempty" jsonschema:"true closes the doc's clarification; it needs docs:write and plays:run on the Clarify via AI play, and a new round started with play_run reopens it. Omit to leave it as is."`
}

type clarifyAnswerIn struct {
	QuestionID string   `json:"question_id" jsonschema:"The question's id, from doc_get's clarification rounds."`
	Selected   []string `json:"selected,omitempty" jsonschema:"The picked options' labels."`
	Text       string   `json:"text,omitempty" jsonschema:"A free-text answer, alone or beside the picked options."`
	Skip       bool     `json:"skip,omitzero" jsonschema:"true skips the question instead of answering it; omit selected and text."`
	Clear      bool     `json:"clear,omitzero" jsonschema:"true makes the question unanswered again; omit the other fields."`
}

type clarifyQuestionIn struct {
	Question    string          `json:"question" jsonschema:"The question as the people answering read it, in plain words."`
	Why         string          `json:"why,omitempty" jsonschema:"One line on why the doc needs this answer."`
	Options     []clarifyOption `json:"options,omitempty" jsonschema:"The choices to pick from; omit for a free-text question. A free-text box is always offered beside them."`
	MultiSelect bool            `json:"multi_select,omitzero" jsonschema:"true lets several options be picked. Defaults to false."`
}

type clarifyOption struct {
	Label       string `json:"label" jsonschema:"The option's short label."`
	Description string `json:"description,omitempty" jsonschema:"One line explaining the option."`
}

type docDeleteIn struct {
	ID string `json:"id" jsonschema:"The doc's id, from doc_list."`
}

// MCPTools returns the docs tools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{docListTool(s), docGetTool(s), docCreateTool(s), docUpdateTool(s), docDeleteTool(s)}
}

func docListTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_list", "List docs",
		"Lists docs by title, or full-text searches them when query is set, optionally within one project or one folder. "+
			"A folder groups a project's docs one level deep: every doc lives in exactly one, the project's default folder Main unless placed elsewhere, and project_get lists them. "+
			"Use it to find a doc's id, then doc_get to read its body. "+
			"Returns items without bodies, each with its folder_id, ranked by relevance with a query and oldest first without one; "+
			"a doc you cannot open is listed with can_open false when browsing and left out of a search. "+
			"Archived docs are hidden unless include_archived is set.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in docListIn) (any, error) {
			items, err := listDocs(ctx, s, in.ProjectID)
			if err != nil {
				return nil, err
			}
			if in.Query != "" {
				if items, err = rankBySearch(ctx, s, in.Query, items); err != nil {
					return nil, err
				}
			}
			if !in.IncludeArchived {
				items = slices.DeleteFunc(items, func(d *DocListItem) bool { return d.Archived })
			}
			if in.FolderID != "" {
				items = slices.DeleteFunc(items, func(d *DocListItem) bool { return d.FolderID != in.FolderID })
			}
			return mcptool.Paginate(items, in.PageArgs), nil
		})
}

func listDocs(ctx context.Context, s *Service, projectID string) ([]*DocListItem, error) {
	if projectID != "" {
		return s.ListByProject(ctx, projectID)
	}
	return s.List(ctx)
}

// rankBySearch keeps the listed docs the search matched, in the search's rank order.
func rankBySearch(ctx context.Context, s *Service, query string, items []*DocListItem) ([]*DocListItem, error) {
	hits, err := s.Search(ctx, query, docSearchScan)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*DocListItem, len(items))
	for _, d := range items {
		byID[d.ID] = d
	}
	ranked := make([]*DocListItem, 0, len(hits))
	for _, h := range hits {
		if d, ok := byID[h.ID]; ok {
			ranked = append(ranked, d)
		}
	}
	return ranked, nil
}

func docGetTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_get", "Get doc",
		"Returns one doc with its full body as markdown, its project, the folder it lives in (folder_id), version, archived and locked state, "+
			"and its watchers: the people its edits notify, who are its creator, everyone who edited it, and anyone who chose to watch, with watching true when you are one. "+
			"It also returns the doc's clarification: every round of questions with their answers, each round's Anything else? text and its reply, whether a round is running, and whether it is closed. "+
			"Use it after doc_list has given you the id, and before doc_update so you edit the current text. "+
			"Fails with forbidden when you cannot read the doc.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in docGetIn) (any, error) {
			d, err := s.Get(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			return toDocResult(ctx, s, d)
		})
}

func docCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_create", "Create doc",
		"Creates a doc in a project from a markdown title and body, or copies one with clone_from_id, and makes you its owner. "+
			"Use it for what people read to decide what to build: requirements, product scope, acceptance criteria, business processes, and plans. "+
			"Reusable technical guidance meant to steer implementation, such as research, architecture, coding standards, and library choices, belongs in memory_create instead, however long or cited it is, unless the person asked for a doc; a mixed deliverable splits into a doc and a memory that link each other. "+
			"Every doc lives in exactly one folder of its project: a new doc goes in folder_id, or the project's default folder Main when omitted, and a copy goes in the destination's default folder, or beside the original within its own project. "+
			"project_get lists a project's folders and project_update creates them. "+
			"Copying needs docs:clone on the source and docs:write in the destination project. "+
			"Returns the new doc with its body as markdown.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in docCreateIn) (any, error) {
			d, err := createOrClone(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return toDocResult(ctx, s, d)
		})
}

func createOrClone(ctx context.Context, s *Service, in docCreateIn) (*Doc, error) {
	if in.CloneFromID == "" {
		return s.CreateInFolder(ctx, in.ProjectID, in.FolderID, in.Title, in.Body)
	}
	if in.Title != "" || in.Body != "" || in.FolderID != "" {
		return nil, fmt.Errorf("%w: clone_from_id copies the source's title and body into a default place; omit title, body, and folder_id, then change the copy with doc_update", apperrs.ErrInvalid)
	}
	return s.Clone(ctx, in.CloneFromID, in.ProjectID)
}

func docUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_update", "Update doc",
		"Changes a doc's title, body, folder, archived, or locked state, or whether you watch it; only the fields you send change. "+
			"folder_id moves the doc to another folder of its project, one of the folders project_get lists. "+
			"A new title or body saves a new version, and archived true hides the doc from search until archived false restores it; archiving keeps the doc, and doc_delete removes it for good. "+
			"A locked doc refuses title and body changes until locked false, which you may send with the edit to unlock first; "+
			"locking and unlocking need docs:lock, and a doc play locks its doc when its run starts. "+
			"A title or body edit makes you a watcher, notified of the doc's later edits, unless you stopped watching it; watch true or false starts or stops that for you alone. "+
			"The answers field answers, skips, or clears clarification questions by id, and clarification_closed true closes the clarification; "+
			"questions, anything_else_reply, and no_gaps with the new body come only from the running Clarify via AI round's own run. "+
			"Read the doc with doc_get first, because body replaces the whole body. "+
			"Returns the doc as it now stands.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in docUpdateIn) (any, error) {
			d, err := updateDoc(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return toDocResult(ctx, s, d)
		})
}

func docDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_delete", "Delete doc",
		"Deletes a doc for good, with its versions, thread, and attachments, the same as Delete in the doc's menu; it needs docs:delete and works on a locked or archived doc too. "+
			"Archiving with doc_update archived true is the reversible alternative: it only hides the doc, which doc_get and doc_list include_archived still return. "+
			"It fails with a conflict while tickets are filed from the doc; ticket_list with doc_id finds them, and ticket_update with an empty doc_id clears each one's source. "+
			"Returns the deleted id with deleted true, and doc_get then answers not found.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in docDeleteIn) (any, error) {
			if err := s.Delete(ctx, in.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(in.ID), nil
		})
}

// updateDoc unlocks before editing and locks after it, so one call can do either around a title or body change.
func updateDoc(ctx context.Context, s *Service, in docUpdateIn) (*Doc, error) {
	if in.NoGaps {
		return writeNoGaps(ctx, s, in)
	}
	d, err := s.Get(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	var applied []string
	if is(in.Locked, false) && d.Locked {
		if d, err = s.Unlock(ctx, in.ID); err != nil {
			return nil, err
		}
		applied = append(applied, "locked")
	}
	if edited := editedFields(in); len(edited) > 0 {
		if d, err = s.Update(ctx, in.ID, deref(in.Title, d.Title), deref(in.Body, d.Body)); err != nil {
			return nil, stepErr(applied, "title and body", err)
		}
		applied = append(applied, edited...)
	}
	if d, applied, err = placeDoc(ctx, s, in, d, applied); err != nil {
		return nil, err
	}
	if is(in.Locked, true) && !d.Locked {
		if d, err = s.Lock(ctx, in.ID); err != nil {
			return nil, stepErr(applied, "locked", err)
		}
		applied = append(applied, "locked")
	}
	if in.Watch != nil {
		if _, err := s.SetWatching(ctx, in.ID, *in.Watch); err != nil {
			return nil, stepErr(applied, "watch", err)
		}
		applied = append(applied, "watch")
	}
	if err := clarifyDoc(ctx, s, in, applied); err != nil {
		return nil, err
	}
	return d, nil
}

// writeNoGaps takes a Clarify round's no-gaps rewrite, with the reply to the last Anything else? when it has one.
func writeNoGaps(ctx context.Context, s *Service, in docUpdateIn) (*Doc, error) {
	if in.Body == nil || in.Title != nil || len(in.Questions) > 0 {
		return nil, fmt.Errorf("%w: no_gaps comes with body, the whole rewritten doc, and without title or questions", apperrs.ErrInvalid)
	}
	if in.AnythingElseReply != nil {
		if err := s.PostRound(ctx, in.ID, nil, *in.AnythingElseReply); err != nil {
			return nil, err
		}
	}
	d, err := s.WriteNoGaps(ctx, in.ID, *in.Body)
	if err != nil && in.AnythingElseReply != nil {
		return nil, stepErr([]string{"anything_else_reply"}, "no_gaps", err)
	}
	return d, err
}

// clarifyDoc applies the clarification fields after the doc's own, so a refusal names what was already saved.
func clarifyDoc(ctx context.Context, s *Service, in docUpdateIn, applied []string) error {
	if len(in.Questions) > 0 || in.AnythingElseReply != nil {
		if err := s.PostRound(ctx, in.ID, toQuestions(in.Questions), deref(in.AnythingElseReply, "")); err != nil {
			return stepErr(applied, "questions", err)
		}
		applied = append(applied, "questions")
	}
	for i, a := range in.Answers {
		if err := answer(ctx, s, in.ID, a); err != nil {
			return stepErr(applied, fmt.Sprintf("answer %d of %d (%s)", i+1, len(in.Answers), a.QuestionID), err)
		}
		applied = append(applied, fmt.Sprintf("answer %d", i+1))
	}
	if in.ClarificationClosed == nil {
		return nil
	}
	if !*in.ClarificationClosed {
		return stepErr(applied, "clarification_closed", fmt.Errorf("%w: a closed clarification reopens with a new round; start the doc's Clarify via AI play with play_run", apperrs.ErrInvalid))
	}
	if _, err := s.CloseClarification(ctx, in.ID); err != nil {
		return stepErr(applied, "clarification_closed", err)
	}
	return nil
}

func answer(ctx context.Context, s *Service, docID string, a clarifyAnswerIn) error {
	if a.Clear {
		_, err := s.ClearAnswer(ctx, docID, a.QuestionID)
		return err
	}
	_, err := s.AnswerQuestion(ctx, docID, a.QuestionID, Answer{Selected: a.Selected, Text: a.Text, Skipped: a.Skip})
	return err
}

func toQuestions(in []clarifyQuestionIn) []ClarificationQuestion {
	out := make([]ClarificationQuestion, 0, len(in))
	for _, q := range in {
		options := make([]QuestionOption, 0, len(q.Options))
		for _, o := range q.Options {
			options = append(options, QuestionOption(o))
		}
		out = append(out, ClarificationQuestion{Question: q.Question, Why: q.Why, Options: options, MultiSelect: q.MultiSelect})
	}
	return out
}

// placeDoc applies the folder and archived fields, the ones that change where a doc shows rather than its text.
func placeDoc(ctx context.Context, s *Service, in docUpdateIn, d *Doc, applied []string) (*Doc, []string, error) {
	var err error
	if in.FolderID != nil && *in.FolderID != d.FolderID {
		if d, err = s.MoveToFolder(ctx, in.ID, *in.FolderID); err != nil {
			return nil, applied, stepErr(applied, "folder_id", err)
		}
		applied = append(applied, "folder_id")
	}
	if in.Archived != nil && *in.Archived != d.Archived {
		if d, err = setArchived(ctx, s, in.ID, *in.Archived); err != nil {
			return nil, applied, stepErr(applied, "archived", err)
		}
		applied = append(applied, "archived")
	}
	return d, applied, nil
}

func editedFields(in docUpdateIn) []string {
	var edited []string
	if in.Title != nil {
		edited = append(edited, "title")
	}
	if in.Body != nil {
		edited = append(edited, "body")
	}
	return edited
}

func is(p *bool, want bool) bool {
	return p != nil && *p == want
}

func setArchived(ctx context.Context, s *Service, id string, archived bool) (*Doc, error) {
	if archived {
		return s.Archive(ctx, id)
	}
	return s.Restore(ctx, id)
}

// stepErr says which fields were already saved when a later step failed, even when err is hidden.
func stepErr(applied []string, step string, err error) error {
	if len(applied) == 0 {
		return err
	}
	return &mcptool.PartialError{Applied: applied, Err: fmt.Errorf("%s: %w", step, err)}
}

func deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

func toDocResult(ctx context.Context, s *Service, d *Doc) (docResult, error) {
	md, err := richtext.ToMarkdown(d.Body)
	if err != nil {
		return docResult{}, fmt.Errorf("render doc %s: %w", d.ID, err)
	}
	ws, err := s.watchersOf(ctx, d.ID)
	if err != nil {
		return docResult{}, err
	}
	watchers := make([]watcherResult, 0, len(ws.Watchers))
	for _, w := range ws.Watchers {
		watchers = append(watchers, watcherResult{UserID: w.UserID, Source: w.Source})
	}
	c, err := s.clarificationOf(ctx, d)
	if err != nil {
		return docResult{}, err
	}
	return docResult{
		ID: d.ID, ProjectID: d.ProjectID, FolderID: d.FolderID, Title: d.Title, Body: md,
		Version: d.Version, Archived: d.Archived, Locked: d.Locked, UpdatedAt: d.UpdatedAt,
		Watchers: watchers, Watching: ws.Watching, Clarification: c,
	}, nil
}
