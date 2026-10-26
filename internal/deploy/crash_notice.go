package deploy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// NoticeDesk is the chat side of crash notices, wired in the composition root (ADR 0017).
type NoticeDesk interface {
	// Check refuses, as the caller, a conversation that is not a channel of workspaceID they may post in, or a botID
	// that is not one of its live bots.
	Check(ctx context.Context, workspaceID, conversationID, botID string) error
	// Post writes body into the conversation, as the bot when botID is set, under an id derived from key, so posting
	// the same key again changes nothing. A conversation or bot gone is ErrNotFound; a bot that may not post is ErrForbidden.
	Post(ctx context.Context, conversationID, botID, key, body string) error
	// TicketURL is the address of the ticket's page, which the clients show as the ticket.
	TicketURL(ctx context.Context, ticketID string) (string, error)
}

// SetCrashNotices wires the desk crash notices post through; unset, no stack may name a conversation and none is posted to.
func (s *Service) SetCrashNotices(desk NoticeDesk) { s.notices = desk }

func (s *Service) noticesOn(stack *Stack) bool {
	return s.notices != nil && stack.CrashNoticeConversationID != ""
}

// checkCrashNotice holds a new or changed notice target to the desk's check. One left as it was is not asked again, so
// someone who can't post there may still edit the rest of the stack.
func (s *Service) checkCrashNotice(ctx context.Context, existing, stack *Stack) error {
	if stack.CrashNoticeConversationID == "" {
		return nil
	}
	if existing != nil && existing.ProjectID == stack.ProjectID &&
		existing.CrashNoticeConversationID == stack.CrashNoticeConversationID && existing.CrashNoticeBotID == stack.CrashNoticeBotID {
		return nil
	}
	if s.notices == nil || s.projects == nil {
		return fmt.Errorf("%w: crash notices are not available on this server", apperrs.ErrInvalid)
	}
	workspaceID, err := s.projects.WorkspaceOf(ctx, stack.ProjectID)
	if err != nil {
		return fmt.Errorf("find workspace of project %s: %w", stack.ProjectID, err)
	}
	return s.notices.Check(ctx, workspaceID, stack.CrashNoticeConversationID, stack.CrashNoticeBotID)
}

// postCrashNotice posts text, with a link to the crash ticket when there is one, into the stack's notice conversation.
// A conversation or bot that is gone, or a bot that may no longer post, drops the notice: retrying would not change it.
func (s *Service) postCrashNotice(ctx context.Context, stack *Stack, key, ticketID, text string) error {
	if !s.noticesOn(stack) {
		return nil
	}
	if ticketID != "" {
		url, err := s.notices.TicketURL(ctx, ticketID)
		if err != nil {
			return fmt.Errorf("link ticket %s: %w", ticketID, err)
		}
		text += " " + url
	}
	err := s.notices.Post(ctx, stack.CrashNoticeConversationID, stack.CrashNoticeBotID, key, text)
	if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) {
		slog.WarnContext(ctx, "crash notice dropped", "stack_id", stack.ID, "conversation_id", stack.CrashNoticeConversationID, "bot_id", stack.CrashNoticeBotID, "error", err)
		return nil
	}
	if err != nil {
		return fmt.Errorf("post crash notice for stack %s: %w", stack.ID, err)
	}
	return nil
}

// crashNotice is one outage, however often its container restarts; "exit 137, out of memory" says how it died.
func crashNotice(p OutageEvent) string {
	return fmt.Sprintf("`%s` crashed on `%s`: %s.", p.Service, p.StackName, crashReason(p))
}

func recoveredNotice(p OutageEvent) string {
	down := ""
	if p.EndedAt != nil {
		down = " after " + p.EndedAt.Sub(p.StartedAt).Round(time.Second).String() + " down"
	}
	more := ""
	if p.RestartCount > 0 {
		more = fmt.Sprintf(" It crashed %d more times meanwhile.", p.RestartCount)
	}
	return fmt.Sprintf("`%s` came back on `%s`%s.%s", p.Service, p.StackName, down, more)
}

func errorRateNotice(p ErrorRateEvent) string {
	return fmt.Sprintf("`%s` answered %s of its requests with 5xx over the last %d minutes (%d of %d), above its %s limit.",
		p.StackName, percentText(p.PeakPercent), p.WindowMinutes, p.Errors, p.Requests, percentText(p.LimitPercent))
}

func errorRateRecoveredNotice(p ErrorRateEvent) string {
	return fmt.Sprintf("5xx on `%s` is back under %s; the worst window reached %s.", p.StackName, percentText(p.LimitPercent), percentText(p.PeakPercent))
}

// trimNoticeTarget drops the whitespace a typed id carries, so an id of spaces posts nowhere.
func trimNoticeTarget(stack Stack) Stack {
	stack.CrashNoticeConversationID = strings.TrimSpace(stack.CrashNoticeConversationID)
	stack.CrashNoticeBotID = strings.TrimSpace(stack.CrashNoticeBotID)
	return stack
}
