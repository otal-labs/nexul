package crashintake

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Outcome is what the intake did with one report.
type Outcome string

const (
	// OutcomeIgnored is a post that is not a crash-report/1 report; nothing changed.
	OutcomeIgnored Outcome = "ignored"
	// OutcomeFiled is a crash no ticket had yet, filed as a new bug.
	OutcomeFiled Outcome = "filed"
	// OutcomeSeenAgain is a repeat of an open crash, or of one closed without a fix, added to its running note.
	OutcomeSeenAgain Outcome = "seen_again"
	// OutcomeRegression is a crash back in a build holding every merged fix, filed as a new high-priority bug.
	OutcomeRegression Outcome = "regression"
	// OutcomeOldBuild is a fixed crash reported by a build missing a fix; only a note.
	OutcomeOldBuild Outcome = "old_build"
	// OutcomeCantTell is a fixed crash GitHub couldn't place against the build; only a note naming why.
	OutcomeCantTell Outcome = "cant_tell"
)

// DefaultBugType and DefaultLabel fill a config that leaves them empty.
const (
	DefaultBugType = "bug"
	DefaultLabel   = "high-priority"
)

// Config is where one automation's reports land.
type Config struct {
	ProjectID string `json:"project_id"`
	// BugType is the name of the project's ticket type a new crash is filed as.
	BugType string `json:"bug_type"`
	// Label marks a returned crash as high priority.
	Label string `json:"label"`
}

// Post is a bot's message as chat.bot_message.posted carries it.
type Post struct {
	Content string          `json:"content"`
	Embeds  json.RawMessage `json:"embeds,omitempty"`
}

// Result says what the intake did, on which ticket, and why when it did nothing or couldn't tell.
type Result struct {
	Outcome  Outcome `json:"outcome"`
	TicketID string  `json:"ticket_id,omitempty"`
	Reason   string  `json:"reason,omitempty"`
}

// Match is a ticket that carries a crash's fingerprint line.
type Match struct {
	ID        string
	CreatedAt time.Time
	// Done is a ticket in a done-stage column.
	Done bool
	// Merged are its merged pull requests, the fixes a returning crash is checked against.
	Merged []PR
}

// PR names one merged pull request.
type PR struct {
	Owner  string
	Repo   string
	Number int
}

// Bug is a ticket the intake files.
type Bug struct {
	ProjectID string
	TypeName  string
	Title     string
	Body      string
	// FoundIn is the done ticket a returning crash was found in; empty marks the origin unknown.
	FoundIn string
	// Label, when set, is added and the bug moves to the top of its column.
	Label string
}

// Desk is the tickets side of the intake (ADR 0017).
type Desk interface {
	// Find lists the project's tickets whose body carries the fingerprint line.
	Find(ctx context.Context, projectID, fingerprint string) ([]Match, error)
	// File creates the bug and returns its id.
	File(ctx context.Context, bug Bug) (string, error)
}

// Note is one note on a ticket's thread.
type Note struct {
	ID       string
	Markdown string
}

// Notes keeps the intake's running notes on a ticket's thread.
type Notes interface {
	// Find returns the ticket's note whose file is name, if it has one.
	Find(ctx context.Context, ticketID, name string) (Note, bool, error)
	Post(ctx context.Context, ticketID, name, markdown string) error
	Replace(ctx context.Context, ticketID, noteID, markdown string) error
}

// Fixes answers whether a build holds a merged pull request.
type Fixes interface {
	MergedInto(ctx context.Context, owner, repo string, number int, sha string) (bool, error)
}

// Service is the crash intake.
type Service struct {
	desk  Desk
	notes Notes
	fixes Fixes
	log   *slog.Logger
	now   func() time.Time
	// ponytail: one lock for every report, so two posts of a new crash file one ticket; per-fingerprint locks if volume grows.
	mu sync.Mutex
}

// NewService wires the intake.
func NewService(desk Desk, notes Notes, fixes Fixes, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{desk: desk, notes: notes, fixes: fixes, log: log, now: time.Now}
}

// Intake files, notes, or ignores one bot post in cfg's project.
func (s *Service) Intake(ctx context.Context, cfg Config, post Post) (Result, error) {
	cfg, err := cfg.normalized()
	if err != nil {
		return Result{}, err
	}
	r, err := ParseReport(post.Content, post.Embeds)
	if errors.Is(err, errNotAReport) {
		s.log.InfoContext(ctx, "crash intake: ignored a post", "reason", err.Error())
		return Result{Outcome: OutcomeIgnored, Reason: err.Error()}, nil
	}
	if err != nil {
		return Result{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	matches, err := s.desk.Find(ctx, cfg.ProjectID, r.Fingerprint)
	if err != nil {
		return Result{}, fmt.Errorf("find crash %s: %w", r.Fingerprint, err)
	}
	if len(matches) == 0 {
		return s.file(ctx, cfg, r, "", OutcomeFiled)
	}
	m := slices.MaxFunc(matches, func(a, b Match) int { return a.CreatedAt.Compare(b.CreatedAt) })
	if !m.Done || len(m.Merged) == 0 {
		return s.tally(ctx, m.ID, seenAgain, r, "")
	}
	fixed, reason := s.fixedIn(ctx, m.Merged, r.BuildCommit)
	if reason != "" {
		return s.tally(ctx, m.ID, cantTell, r, reason)
	}
	if !fixed {
		return s.tally(ctx, m.ID, oldBuild, r, "")
	}
	return s.file(ctx, cfg, r, m.ID, OutcomeRegression)
}

func (c Config) normalized() (Config, error) {
	c.ProjectID = strings.TrimSpace(c.ProjectID)
	if c.ProjectID == "" {
		return c, fmt.Errorf("%w: project_id is required", apperrs.ErrInvalid)
	}
	c.BugType = cmp.Or(strings.TrimSpace(c.BugType), DefaultBugType)
	c.Label = cmp.Or(strings.TrimSpace(c.Label), DefaultLabel)
	return c, nil
}

// fixedIn reports whether build holds every fix; one fix it lacks decides, and otherwise any unanswered one is the reason.
func (s *Service) fixedIn(ctx context.Context, fixes []PR, build string) (bool, string) {
	var reasons []string
	for _, pr := range fixes {
		in, err := s.fixes.MergedInto(ctx, pr.Owner, pr.Repo, pr.Number, build)
		if err != nil {
			s.log.WarnContext(ctx, "crash intake: GitHub couldn't place a fix", "owner", pr.Owner, "repo", pr.Repo, "pr", pr.Number, "error", err)
			reasons = append(reasons, cantTellReason(pr, build, err))
			continue
		}
		if !in {
			return false, ""
		}
	}
	return true, strings.Join(reasons, "; ")
}

func cantTellReason(pr PR, build string, err error) string {
	where := fmt.Sprintf("%s/%s#%d", pr.Owner, pr.Repo, pr.Number)
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Sprintf("Nexul couldn't find build commit %s or %s on GitHub", short(build), where)
	}
	if errors.Is(err, apperrs.ErrUnauthorized) {
		return fmt.Sprintf("Nexul can't read %s/%s on GitHub", pr.Owner, pr.Repo)
	}
	if errors.Is(err, apperrs.ErrConflict) {
		return fmt.Sprintf("%s has not merged", where)
	}
	return fmt.Sprintf("GitHub didn't answer for %s", where)
}

func (s *Service) file(ctx context.Context, cfg Config, r Report, foundIn string, outcome Outcome) (Result, error) {
	bug := Bug{ProjectID: cfg.ProjectID, TypeName: cfg.BugType, Title: bugTitle(r), Body: bugBody(r, foundIn != ""), FoundIn: foundIn}
	if foundIn != "" {
		bug.Label = cfg.Label
	}
	id, err := s.desk.File(ctx, bug)
	if err != nil {
		return Result{}, fmt.Errorf("file crash %s: %w", r.Fingerprint, err)
	}
	return Result{Outcome: outcome, TicketID: id}, nil
}

// tally adds r to the ticket's running note of kind k, creating the note on the first report.
func (s *Service) tally(ctx context.Context, ticketID string, k noteKind, r Report, reason string) (Result, error) {
	note, ok, err := s.notes.Find(ctx, ticketID, k.file)
	if err != nil {
		return Result{}, fmt.Errorf("read %s on ticket %s: %w", k.file, ticketID, err)
	}
	t := parseTally(note.Markdown)
	t.add(r, reason, s.now().UTC())
	markdown := t.render(k)
	if ok {
		err = s.notes.Replace(ctx, ticketID, note.ID, markdown)
	}
	if !ok {
		err = s.notes.Post(ctx, ticketID, k.file, markdown)
	}
	if err != nil {
		return Result{}, fmt.Errorf("write %s on ticket %s: %w", k.file, ticketID, err)
	}
	return Result{Outcome: k.outcome, TicketID: ticketID, Reason: reason}, nil
}

func bugTitle(r Report) string {
	title := "Crash: " + cmp.Or(r.Signature, "[redacted]")
	if frame := r.TopFrame(); frame != "" {
		title += " in " + frame
	}
	return title + " (" + r.Kind + ")"
}

func bugBody(r Report, returned bool) string {
	var b strings.Builder
	b.WriteString(FingerprintLine + r.Fingerprint + "\n\n")
	if returned {
		b.WriteString("This crash came back in a build that contains every merged fix of the ticket it was found in.\n\n")
	}
	fmt.Fprintf(&b, "- Kind: %s\n- Process type: %s\n- First version: %s\n- Build commit: %s\n- OS: %s %s\n- First seen: %s\n- Signature: %s\n",
		r.Kind, cmp.Or(r.ProcessType, "unknown"), r.AppVersion, r.BuildCommit, cmp.Or(r.OS, "unknown"), r.Arch, cmp.Or(r.FirstSeen, "unknown"), cmp.Or(r.Signature, "[redacted]"))
	if r.Truncated {
		b.WriteString("- The sender cut this report to its size cap.\n")
	}
	fence := codeFence(r.Stack)
	fmt.Fprintf(&b, "\nStack, already redacted by the sender:\n\n%stext\n%s\n%s\n", fence, cmp.Or(r.Stack, "none"), fence)
	return b.String()
}

// codeFence is a backtick fence longer than any backtick run in text, so a posted stack can't close it early.
func codeFence(text string) string {
	longest, run := 0, 0
	for _, c := range text {
		if c != '`' {
			run = 0
			continue
		}
		run++
		longest = max(longest, run)
	}
	return strings.Repeat("`", max(3, longest+1))
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
