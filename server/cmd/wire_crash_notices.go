package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

var _ deploy.NoticeDesk = crashNoticeDesk{}

// crashNoticeDesk adapts chat and bots to deploy's NoticeDesk seam (ADR 0017). Without a bot a notice is a system
// message posted for the workspace's Owner, as a crash ticket's comment is.
type crashNoticeDesk struct {
	chat       *chat.Service
	bots       *botwebhook.Service
	tickets    *tickets.Service
	projects   *storage.ProjectsRepo
	workspaces *storage.WorkspacesRepo
	users      *storage.UsersRepo
	instance   dnsSettingsAdapter
}

func (d crashNoticeDesk) Check(ctx context.Context, workspaceID, conversationID, botID string) error {
	c, err := d.chat.GetConversation(ctx, conversationID)
	if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) {
		return fmt.Errorf("%w: crash notices need a conversation you can read", apperrs.ErrInvalid)
	}
	if err != nil {
		return err
	}
	if c.WorkspaceID != workspaceID || c.Kind != chat.KindChannel {
		return fmt.Errorf("%w: crash notices go to a channel of the stack's workspace", apperrs.ErrInvalid)
	}
	if actor, ok := identity.ActorFromCtx(ctx); ok {
		stopped, _, err := d.chat.BotStopped(ctx, c.ID, actor.ID)
		if err != nil {
			return err
		}
		if stopped {
			return fmt.Errorf("%w: you can't post in that conversation, so a stack can't post there for you", apperrs.ErrForbidden)
		}
	}
	if botID == "" {
		return nil
	}
	bots, err := d.bots.List(ctx, c.ID, false)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(bots, func(b *botwebhook.Bot) bool { return b.ID == botID }) {
		return fmt.Errorf("%w: that bot isn't one of the conversation's", apperrs.ErrInvalid)
	}
	return nil
}

func (d crashNoticeDesk) Post(ctx context.Context, conversationID, botID, key, body string) error {
	if botID != "" {
		return d.bots.Notify(ctx, botID, ids.From("crash-notice\x00"+conversationID+"\x00"+key), body)
	}
	c, err := d.chat.GetConversation(ctx, conversationID)
	if err != nil {
		return err
	}
	owner, err := d.users.FirstWorkspaceOwner(ctx, c.WorkspaceID)
	if err != nil {
		return fmt.Errorf("find an owner of workspace %s to post for: %w", c.WorkspaceID, err)
	}
	return d.chat.PostSystemNotice(ctx, c.ID, owner, key, body)
}

// TicketURL is absolute once the instance has an address, which the web app shows as the ticket's pill.
func (d crashNoticeDesk) TicketURL(ctx context.Context, ticketID string) (string, error) {
	t, err := d.tickets.Get(ctx, ticketID)
	if err != nil {
		return "", err
	}
	project, err := d.projects.Get(ctx, t.ProjectID)
	if err != nil {
		return "", err
	}
	ws, err := d.workspaces.Get(ctx, project.WorkspaceID)
	if err != nil {
		return "", err
	}
	base, err := d.instance.GetInstanceURL(ctx)
	if err != nil {
		return "", fmt.Errorf("read the instance URL: %w", err)
	}
	return strings.TrimRight(base, "/") + "/" + ws.Slug + "/tickets/" + t.ID, nil
}
