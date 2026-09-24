package memories

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	decisionsLogTitle     = "Decisions log"
	decisionsLogWhenToUse = "Why the project works the way it does: decisions from tickets that changed it, reversed ones marked superseded."
)

// listItem matches the first line of a markdown list item, which starts a new decisions-log entry.
var listItem = regexp.MustCompile(`^([-*+]|\d+\.)\s`)

// CreateWithKind creates an ordinary memory (kind empty) or the project's decisions log; the interview has CreateInterview.
func (s *Service) CreateWithKind(ctx context.Context, kind, projectID, workspaceID, title, whenToUse, body string, alwaysIncluded bool, via string) (*Memory, error) {
	switch strings.TrimSpace(kind) {
	case "":
		return s.Create(ctx, projectID, workspaceID, title, whenToUse, body, alwaysIncluded, via)
	case KindDecisionsLog:
		return s.createDecisionsLog(ctx, projectID, title, whenToUse, body, via)
	}
	return nil, fmt.Errorf("%w: kind must be empty or %s; the interview memory is created with memory_create_interview", apperrs.ErrInvalid, KindDecisionsLog)
}

// createDecisionsLog creates the project's decisions log on its first entry; a project holds at most one.
func (s *Service) createDecisionsLog(ctx context.Context, projectID, title, whenToUse, body, via string) (*Memory, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: the decisions log belongs to a project; project id is required", apperrs.ErrInvalid)
	}
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	workspaceID, err := s.projects.WorkspaceForProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesWrite); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByProjectKind(ctx, projectID, KindDecisionsLog)
	if err == nil {
		return nil, fmt.Errorf("%w: the project already has a decisions log (%s); add to it with memory_update", apperrs.ErrConflict, existing.ID)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get decisions log for project %s: %w", projectID, err)
	}
	normalized, err := richtext.Normalize(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body is not valid document content", apperrs.ErrInvalid)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = decisionsLogTitle
	}
	whenToUse = capWhenToUse(whenToUse)
	if whenToUse == "" {
		whenToUse = decisionsLogWhenToUse
	}
	now := s.now().UTC()
	m := &Memory{
		ID: ids.New(), WorkspaceID: workspaceID, ProjectID: projectID, Kind: KindDecisionsLog,
		Title: title, WhenToUse: whenToUse, Body: normalized, Version: 1,
		CreatedBy: actor.ID, CreatedAt: now, UpdatedBy: actor.ID, UpdatedAt: now,
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCreated, Payload: CreatedEvent{Memory: toRef(m), AuthorID: actor.ID}}
	if err := s.repo.Create(ctx, m, via, evt); err != nil {
		return nil, fmt.Errorf("create decisions log for project %s: %w", projectID, err)
	}
	return m, nil
}

// DecisionEntriesCiting returns the decisions-log entries naming any ref (a ticket key or link), none without a log; needs memories:read.
func (s *Service) DecisionEntriesCiting(ctx context.Context, projectID string, refs []string) ([]string, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	workspaceID, err := s.projects.WorkspaceForProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace for project %s: %w", projectID, err)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	log, err := s.repo.GetByProjectKind(ctx, projectID, KindDecisionsLog)
	if errors.Is(err, apperrs.ErrNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get decisions log for project %s: %w", projectID, err)
	}
	md, err := richtext.ToMarkdown(log.Body)
	if err != nil {
		return nil, fmt.Errorf("export decisions log %s: %w", log.ID, err)
	}
	out := []string{}
	for _, entry := range decisionEntries(md) {
		if citesAny(entry, refs) {
			out = append(out, entry)
		}
	}
	return out, nil
}

// decisionEntries splits a decisions log into entries: a paragraph or a list item each, headings dropped.
func decisionEntries(md string) []string {
	var entries, current []string
	flush := func() {
		if len(current) > 0 {
			entries = append(entries, strings.Join(current, "\n"))
		}
		current = nil
	}
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			flush()
			continue
		}
		if listItem.MatchString(line) {
			flush()
		}
		current = append(current, line)
	}
	flush()
	return entries
}

func citesAny(entry string, refs []string) bool {
	for _, ref := range refs {
		if ref != "" && containsWord(entry, ref) {
			return true
		}
	}
	return false
}

// containsWord reports whether w occurs in s with no word character on either side, so NEX-1 never matches NEX-12.
func containsWord(s, w string) bool {
	for from := 0; ; {
		i := strings.Index(s[from:], w)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(w)
		if !wordBefore(s, start) && !wordAt(s, end) {
			return true
		}
		from = start + 1
	}
}

func wordBefore(s string, i int) bool { return i > 0 && isWordByte(s[i-1]) }

func wordAt(s string, i int) bool { return i < len(s) && isWordByte(s[i]) }

// isWordByte counts any non-ASCII byte as a word character, so a boundary is only ever punctuation or space.
func isWordByte(b byte) bool {
	return b == '_' || b == '-' || b >= 0x80 || ('0' <= b && b <= '9') || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}
