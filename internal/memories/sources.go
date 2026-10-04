package memories

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	maxSourceRefChars   = 1_000
	maxSourceLabelChars = 200
)

// ListSources returns a project's interview sources oldest first, refs resolved with the caller's access; memories:read.
func (s *Service) ListSources(ctx context.Context, projectID string) ([]*InterviewSource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireProject(ctx, projectID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	srcs, err := s.repo.ListSources(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list interview sources for project %s: %w", projectID, err)
	}
	for _, src := range srcs {
		if err := s.resolveSource(ctx, src); err != nil {
			return nil, err
		}
	}
	return srcs, nil
}

// AddSource adds a source to src.ProjectID's interview; memories:write, plus read on a doc, memory, or project it names.
func (s *Service) AddSource(ctx context.Context, src InterviewSource) (*InterviewSource, error) {
	if err := normalizeSource(&src); err != nil {
		return nil, err
	}
	actorID, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	workspaceID, err := s.projectForWrite(ctx, src.ProjectID, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	if err := s.checkRef(ctx, workspaceID, &src); err != nil {
		return nil, err
	}
	// ponytail: count then insert, so two adds at once can pass the cap by one; count inside the insert if it matters.
	n, err := s.repo.CountSources(ctx, src.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("count interview sources for project %s: %w", src.ProjectID, err)
	}
	if n >= MaxSources {
		return nil, fmt.Errorf("%w: a project's interview holds at most %d sources; remove one first", apperrs.ErrInvalid, MaxSources)
	}
	now := s.now().UTC()
	src.ID, src.WorkspaceID, src.AddedBy, src.AddedAt, src.UpdatedAt = ids.New(), workspaceID, actorID, now, now
	err = s.repo.InsertSource(ctx, &src, sourceEvent(TopicSourceAdded, &src, actorID, now))
	if errors.Is(err, apperrs.ErrConflict) {
		return nil, fmt.Errorf("%w: the interview already has this %s as a source", apperrs.ErrConflict, src.Kind)
	}
	if err != nil {
		return nil, fmt.Errorf("add interview source to project %s: %w", src.ProjectID, err)
	}
	if err := s.resolveSource(ctx, &src); err != nil {
		return nil, err
	}
	return &src, nil
}

// UpdateSource changes a source's stance or pasted text's label, a nil field kept; memories:write on its project.
func (s *Service) UpdateSource(ctx context.Context, id string, stance, label *string) (*InterviewSource, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: source id is required", apperrs.ErrInvalid)
	}
	if stance == nil && label == nil {
		return nil, fmt.Errorf("%w: send a stance or a label to change", apperrs.ErrInvalid)
	}
	actorID, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	src, err := s.writableSource(ctx, id)
	if err != nil {
		return nil, err
	}
	if stance != nil {
		if err := checkStance(*stance); err != nil {
			return nil, err
		}
		src.Stance = *stance
	}
	if label != nil {
		if src.Kind != SourceText {
			return nil, fmt.Errorf("%w: only pasted text has a label to change; a %s shows its own name", apperrs.ErrInvalid, src.Kind)
		}
		if src.Label, err = checkLabel(*label); err != nil {
			return nil, err
		}
	}
	src.UpdatedAt = s.now().UTC()
	if err := s.repo.UpdateSource(ctx, src, sourceEvent(TopicSourceChanged, src, actorID, src.UpdatedAt)); err != nil {
		return nil, fmt.Errorf("update interview source %s: %w", id, err)
	}
	if err := s.resolveSource(ctx, src); err != nil {
		return nil, err
	}
	return src, nil
}

// RemoveSource takes a source off its project's interview; drafts made from it stay. Requires memories:write.
func (s *Service) RemoveSource(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: source id is required", apperrs.ErrInvalid)
	}
	actorID, err := requireActor(ctx)
	if err != nil {
		return err
	}
	src, err := s.writableSource(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteSource(ctx, id, sourceEvent(TopicSourceRemoved, src, actorID, s.now().UTC())); err != nil {
		return fmt.Errorf("remove interview source %s: %w", id, err)
	}
	return nil
}

// ListDrafts returns a project's interview drafts; requires memories:read on the project.
func (s *Service) ListDrafts(ctx context.Context, projectID string) ([]*InterviewDraft, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireProject(ctx, projectID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	ds, err := s.repo.ListDrafts(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list interview drafts for project %s: %w", projectID, err)
	}
	return ds, nil
}

// SaveDrafts stores drafts, each replacing its question's earlier one, dropping any equal to the answer; memories:write.
func (s *Service) SaveDrafts(ctx context.Context, projectID string, drafts []InterviewDraft) ([]*InterviewDraft, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if len(drafts) == 0 {
		return nil, fmt.Errorf("%w: send at least one draft", apperrs.ErrInvalid)
	}
	actorID, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	workspaceID, err := s.projectForWrite(ctx, projectID, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	follow, answered, err := s.draftContext(ctx, projectID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	seen := map[string]bool{}
	rows := []*InterviewDraft{}
	evts := []eventbus.OutboxEvent{}
	for i, d := range drafts {
		if err := normalizeDraft(&d, projectID, follow); err != nil {
			return nil, fmt.Errorf("draft %d of %d (%q), none saved: %w", i+1, len(drafts), d.Question, err)
		}
		if seen[d.Question] {
			return nil, fmt.Errorf("%w: the question %q is drafted twice", apperrs.ErrInvalid, d.Question)
		}
		seen[d.Question] = true
		if sameAsAnswer(answered[d.Question], &d) {
			continue
		}
		d.ID, d.WorkspaceID, d.ProjectID, d.DraftedBy, d.DraftedAt = ids.New(), workspaceID, projectID, actorID, now
		rows = append(rows, &d)
		evts = append(evts, draftEvent(TopicDraftSaved, &d, actorID, now))
	}
	if len(rows) == 0 {
		return rows, nil
	}
	if err := s.repo.SaveDrafts(ctx, rows, evts...); err != nil {
		return nil, fmt.Errorf("save interview drafts for project %s: %w", projectID, err)
	}
	return rows, nil
}

// DismissDraft deletes a draft; a later drafting run may write it again. Requires memories:write on its project.
func (s *Service) DismissDraft(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: draft id is required", apperrs.ErrInvalid)
	}
	actorID, err := requireActor(ctx)
	if err != nil {
		return err
	}
	d, err := s.repo.GetDraft(ctx, id)
	if err != nil {
		return fmt.Errorf("get interview draft %s: %w", id, err)
	}
	if err := s.requireProject(ctx, d.ProjectID, permissions.MemoriesWrite); err != nil {
		return err
	}
	if err := s.repo.DeleteDraft(ctx, id, draftEvent(TopicDraftDismissed, d, actorID, s.now().UTC())); err != nil {
		return fmt.Errorf("dismiss interview draft %s: %w", id, err)
	}
	return nil
}

// ClearSuggestions deletes a project's drafts on questions that already have an answer, as a drafting run starts, so
// only the suggested changes it drafts again come back (ADR 0122); drafts on unanswered questions stay. memories:write.
func (s *Service) ClearSuggestions(ctx context.Context, projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	actorID, err := requireActor(ctx)
	if err != nil {
		return err
	}
	if err := s.requireProject(ctx, projectID, permissions.MemoriesWrite); err != nil {
		return err
	}
	_, answered, err := s.draftContext(ctx, projectID)
	if err != nil {
		return err
	}
	ds, err := s.repo.ListDrafts(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list interview drafts for project %s: %w", projectID, err)
	}
	now := s.now().UTC()
	for _, d := range ds {
		if answered[d.Question] == nil {
			continue
		}
		err := s.repo.DeleteDraft(ctx, d.ID, draftEvent(TopicDraftDismissed, d, actorID, now))
		if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			return fmt.Errorf("clear suggested change %s: %w", d.ID, err)
		}
	}
	return nil
}

func (s *Service) writableSource(ctx context.Context, id string) (*InterviewSource, error) {
	src, err := s.repo.GetSource(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get interview source %s: %w", id, err)
	}
	if err := s.requireProject(ctx, src.ProjectID, permissions.MemoriesWrite); err != nil {
		return nil, err
	}
	return src, nil
}

// draftContext is the project's follow source ids and its answered (not skipped) template questions.
func (s *Service) draftContext(ctx context.Context, projectID string) (map[string]bool, map[string]*InterviewAnswer, error) {
	srcs, err := s.repo.ListSources(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("list interview sources for project %s: %w", projectID, err)
	}
	follow := map[string]bool{}
	for _, src := range srcs {
		follow[src.ID] = src.Stance == StanceFollow
	}
	as, err := s.repo.ListAnswers(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("list interview answers for project %s: %w", projectID, err)
	}
	answered := map[string]*InterviewAnswer{}
	for _, a := range as {
		if a.Round == 0 && !a.Skipped {
			answered[a.Question] = a
		}
	}
	return follow, answered, nil
}

// checkRef refuses a doc, memory, or project the caller cannot read, or one outside the project's workspace.
func (s *Service) checkRef(ctx context.Context, workspaceID string, src *InterviewSource) error {
	switch src.Kind {
	case SourceDoc:
		return s.checkDocRef(ctx, workspaceID, src.Ref)
	case SourceMemory:
		m, err := s.repo.GetByID(ctx, src.Ref)
		if err != nil {
			return fmt.Errorf("memory %s: %w", src.Ref, err)
		}
		if err := s.requireProject(ctx, m.ProjectID, permissions.MemoriesRead); err != nil {
			return err
		}
		return sameWorkspace(m.WorkspaceID, workspaceID, "memory", src.Ref)
	case SourceProject:
		if src.Ref == src.ProjectID {
			return fmt.Errorf("%w: a project cannot be its own interview source", apperrs.ErrInvalid)
		}
		if err := s.requireProject(ctx, src.Ref, permissions.MemoriesRead); err != nil {
			return err
		}
		ws, err := s.projects.WorkspaceForProject(ctx, src.Ref)
		if err != nil {
			return fmt.Errorf("project %s: %w", src.Ref, err)
		}
		return sameWorkspace(ws, workspaceID, "project", src.Ref)
	}
	return nil
}

func (s *Service) checkDocRef(ctx context.Context, workspaceID, docID string) error {
	if s.docs == nil {
		return fmt.Errorf("%w: no doc lookup wired", apperrs.ErrForbidden)
	}
	d, err := s.docs.DocForSource(ctx, docID)
	if err != nil {
		return fmt.Errorf("doc %s: %w", docID, err)
	}
	if !s.canReadDoc(ctx, docID) {
		return fmt.Errorf("%w: docs:read on doc %s is required to add it as a source", apperrs.ErrForbidden, docID)
	}
	return sameWorkspace(d.WorkspaceID, workspaceID, "doc", docID)
}

func sameWorkspace(got, want, kind, id string) error {
	if got == want {
		return nil
	}
	return fmt.Errorf("%w: %s %s is in another workspace; a source must be in this project's workspace", apperrs.ErrInvalid, kind, id)
}

// resolveSource names a source for the caller: an unreadable ref shows its kind alone, a deleted one is flagged gone.
func (s *Service) resolveSource(ctx context.Context, src *InterviewSource) error {
	if err := s.resolveRef(ctx, src); err != nil {
		return err
	}
	if src.NotVisible {
		src.Ref = ""
	}
	return nil
}

func (s *Service) resolveRef(ctx context.Context, src *InterviewSource) error {
	switch src.Kind {
	case SourcePath:
		src.Label = src.Ref
	case SourceDoc:
		return s.resolveDoc(ctx, src)
	case SourceMemory:
		return s.resolveMemory(ctx, src)
	case SourceProject:
		return s.resolveProject(ctx, src)
	}
	return nil
}

func (s *Service) resolveDoc(ctx context.Context, src *InterviewSource) error {
	if s.docs == nil {
		src.NotVisible = true
		return nil
	}
	d, err := s.docs.DocForSource(ctx, src.Ref)
	if errors.Is(err, apperrs.ErrNotFound) {
		src.Gone = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve doc of interview source %s: %w", src.ID, err)
	}
	if !s.canReadDoc(ctx, src.Ref) {
		src.NotVisible = true
		return nil
	}
	src.Label, src.RefUpdatedAt = d.Title, &d.UpdatedAt
	return nil
}

func (s *Service) resolveMemory(ctx context.Context, src *InterviewSource) error {
	m, err := s.repo.GetByID(ctx, src.Ref)
	if errors.Is(err, apperrs.ErrNotFound) {
		src.Gone = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve memory of interview source %s: %w", src.ID, err)
	}
	if s.requireProject(ctx, m.ProjectID, permissions.MemoriesRead) != nil {
		src.NotVisible = true
		return nil
	}
	src.Label, src.RefUpdatedAt = m.Title, &m.UpdatedAt
	return nil
}

func (s *Service) resolveProject(ctx context.Context, src *InterviewSource) error {
	name, err := s.projects.ProjectName(ctx, src.Ref)
	if errors.Is(err, apperrs.ErrNotFound) {
		src.Gone = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve project of interview source %s: %w", src.ID, err)
	}
	if s.requireProject(ctx, src.Ref, permissions.MemoriesRead) != nil {
		src.NotVisible = true
		return nil
	}
	src.Label = name
	return nil
}

// canReadDoc asks access about the doc itself, so its own sharing applies on top of the project's.
func (s *Service) canReadDoc(ctx context.Context, docID string) bool {
	if identity.Internal(ctx) {
		return true
	}
	if s.access == nil {
		return false
	}
	actor, _ := identity.ActorFromCtx(ctx)
	ok, err := s.access.Can(ctx, actor.ID, docID, permissions.DocsRead)
	return err == nil && ok
}

// normalizeSource trims and checks a source as given, per kind.
func normalizeSource(src *InterviewSource) error {
	src.ProjectID = strings.TrimSpace(src.ProjectID)
	if src.ProjectID == "" {
		return fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := checkStance(src.Stance); err != nil {
		return err
	}
	src.Ref = strings.TrimSpace(src.Ref)
	if utf8.RuneCountInString(src.Ref) > maxSourceRefChars {
		return fmt.Errorf("%w: the ref is over %d characters", apperrs.ErrInvalid, maxSourceRefChars)
	}
	if src.Kind == SourceText {
		return normalizeText(src)
	}
	if strings.TrimSpace(src.Label) != "" || strings.TrimSpace(src.Body) != "" {
		return fmt.Errorf("%w: only pasted text carries a label and body; a %s source is its ref alone", apperrs.ErrInvalid, src.Kind)
	}
	src.Label, src.Body = "", ""
	switch src.Kind {
	case SourcePath:
		return normalizePath(src)
	case SourceDoc, SourceMemory, SourceProject:
		if src.Ref == "" {
			return fmt.Errorf("%w: a %s source needs the id of the %s as its ref", apperrs.ErrInvalid, src.Kind, src.Kind)
		}
		return nil
	}
	return fmt.Errorf("%w: kind is path, doc, memory, project, or text, not %q", apperrs.ErrInvalid, src.Kind)
}

// normalizePath keeps a path inside the project's checkout: relative, with no .. segment.
func normalizePath(src *InterviewSource) error {
	ref := strings.ReplaceAll(src.Ref, `\`, "/")
	if ref == "" {
		return fmt.Errorf("%w: a path source needs a file or folder relative to the project's checkout", apperrs.ErrInvalid)
	}
	if path.IsAbs(ref) || hasDriveLetter(ref) {
		return fmt.Errorf("%w: %q is absolute; give the path relative to the project's checkout", apperrs.ErrInvalid, src.Ref)
	}
	if slices.Contains(strings.Split(ref, "/"), "..") {
		return fmt.Errorf("%w: %q leaves the project's checkout; .. is not allowed", apperrs.ErrInvalid, src.Ref)
	}
	src.Ref = path.Clean(ref)
	return nil
}

func hasDriveLetter(p string) bool {
	return len(p) >= 2 && p[1] == ':' && unicode.IsLetter(rune(p[0]))
}

func normalizeText(src *InterviewSource) error {
	if src.Ref != "" {
		return fmt.Errorf("%w: pasted text has no ref; send its label and body", apperrs.ErrInvalid)
	}
	label, err := checkLabel(src.Label)
	if err != nil {
		return err
	}
	src.Label, src.Body = label, strings.TrimSpace(src.Body)
	if src.Body == "" {
		return fmt.Errorf("%w: pasted text needs a body", apperrs.ErrInvalid)
	}
	if n := utf8.RuneCountInString(src.Body); n > MaxSourceTextChars {
		return fmt.Errorf("%w: the pasted text is %d characters, over the %d-character cap", apperrs.ErrInvalid, n, MaxSourceTextChars)
	}
	return nil
}

func checkLabel(label string) (string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", fmt.Errorf("%w: pasted text needs a label", apperrs.ErrInvalid)
	}
	if utf8.RuneCountInString(label) > maxSourceLabelChars {
		return "", fmt.Errorf("%w: the label is over %d characters", apperrs.ErrInvalid, maxSourceLabelChars)
	}
	return label, nil
}

func checkStance(stance string) error {
	if stance == StanceFollow || stance == StanceQuestion {
		return nil
	}
	return fmt.Errorf("%w: stance is follow or question, not %q", apperrs.ErrInvalid, stance)
}

// normalizeDraft checks a draft like an answer, and that it cites only the project's follow sources.
func normalizeDraft(d *InterviewDraft, projectID string, follow map[string]bool) error {
	a := InterviewAnswer{ProjectID: projectID, Question: d.Question, Selected: d.Selected, Text: d.Text}
	if err := normalizeAnswer(&a); err != nil {
		return err
	}
	d.Question, d.Selected, d.Text = a.Question, a.Selected, a.Text
	if len(d.SourceIDs) == 0 {
		return fmt.Errorf("%w: a draft names the follow sources it came from", apperrs.ErrInvalid)
	}
	for _, id := range d.SourceIDs {
		isFollow, ok := follow[id]
		if !ok {
			return fmt.Errorf("%w: %s is not a source of this interview; memory_get lists them", apperrs.ErrInvalid, id)
		}
		if !isFollow {
			return fmt.Errorf("%w: source %s is under question, so it is only asked about, never drafted from", apperrs.ErrInvalid, id)
		}
	}
	d.Where, d.TrailID = strings.TrimSpace(d.Where), strings.TrimSpace(d.TrailID)
	if utf8.RuneCountInString(d.Where) > MaxDraftWhereChars {
		return fmt.Errorf("%w: the where line is over %d characters", apperrs.ErrInvalid, MaxDraftWhereChars)
	}
	return nil
}

// sameAsAnswer reports a draft that says what the stored answer already says, picks compared in any order.
func sameAsAnswer(a *InterviewAnswer, d *InterviewDraft) bool {
	if a == nil || strings.TrimSpace(a.Text) != d.Text {
		return false
	}
	got, want := slices.Clone(a.Selected), slices.Clone(d.Selected)
	slices.Sort(got)
	slices.Sort(want)
	return slices.Equal(got, want)
}

func requireActor(ctx context.Context) (string, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return "", fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	return actor.ID, nil
}

func sourceEvent(topic string, src *InterviewSource, authorID string, at time.Time) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: SourceEvent{
		WorkspaceID: src.WorkspaceID, ProjectID: src.ProjectID, SourceID: src.ID, Kind: src.Kind, Stance: src.Stance, AuthorID: authorID, At: at,
	}}
}

func draftEvent(topic string, d *InterviewDraft, authorID string, at time.Time) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: DraftEvent{
		WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DraftID: d.ID, Question: d.Question, AuthorID: authorID, At: at,
	}}
}
