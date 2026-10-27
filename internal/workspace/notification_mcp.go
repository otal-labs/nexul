package workspace

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type notificationListIn struct {
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"Only notifications in this workspace, from workspace_list."`
	UnreadOnly  bool   `json:"unread_only,omitzero" jsonschema:"Only notifications you have not read yet."`
	mcptool.PageArgs
}

type notificationUpdateIn struct {
	ID          string `json:"id,omitempty" jsonschema:"One notification's id, from notification_list."`
	All         bool   `json:"all,omitzero" jsonschema:"true marks every one of your notifications read. Send instead of id."`
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"With all, only this workspace's notifications, from workspace_list."`
	Unread      bool   `json:"unread,omitzero" jsonschema:"true with id puts that notification back to unread. Not allowed with all."`
}

type notificationResult struct {
	ID              string      `json:"id"`
	WorkspaceID     string      `json:"workspace_id"`
	Kind            Kind        `json:"kind"`
	SubjectType     SubjectType `json:"subject_type"`
	SubjectID       string      `json:"subject_id"`
	SubjectTitle    string      `json:"subject_title"`
	Read            bool        `json:"read"`
	CreatedAt       time.Time   `json:"created_at"`
	FolderID        string      `json:"folder_id,omitempty"`
	FolderName      string      `json:"folder_name,omitempty"`
	FolderIsDefault bool        `json:"folder_is_default,omitempty"`
}

type notificationUpdated struct {
	ID   string `json:"id,omitempty"`
	All  bool   `json:"all,omitempty"`
	Read bool   `json:"read"`
}

// NotificationMCPTools act on the caller's own inbox, the same one the web app's bell shows.
func NotificationMCPTools(s *NotificationService) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("notification_list", "List notifications",
			"Lists your own notifications newest first: what happened (kind), to which ticket, doc, memory, or stack (subject), and whether you have read it. "+
				"A stack's item is a service that crashed (service.container_stopped) or came back (service.container_recovered), or its 5xx share went over its error-rate rule (stack.error_rate_crossed) or back under it (stack.error_rate_recovered), worded in subject_title, and its subject_id is the stack to read with stack_get. "+
				"A doc's item also names its current folder (folder_id, folder_name, folder_is_default). "+
				"Mark items read with notification_update.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in notificationListIn) (any, error) {
				userID, err := notificationCaller(ctx)
				if err != nil {
					return nil, err
				}
				ns, total, err := s.Page(ctx, userID, InboxFilter{WorkspaceID: in.WorkspaceID, UnreadOnly: in.UnreadOnly}, in.Window())
				if err != nil {
					return nil, err
				}
				out := make([]notificationResult, 0, len(ns))
				for _, n := range ns {
					out = append(out, notificationResult{
						ID: n.ID, WorkspaceID: n.WorkspaceID, Kind: n.Kind, SubjectType: n.SubjectType, SubjectID: n.SubjectID, SubjectTitle: n.SubjectTitle,
						Read: n.Read, CreatedAt: n.CreatedAt, FolderID: n.FolderID, FolderName: n.FolderName, FolderIsDefault: n.FolderIsDefault,
					})
				}
				return mcptool.PageOf(out, total, in.Window()), nil
			}),
		mcptool.New("notification_update", "Mark notifications read or unread",
			"Marks one of your notifications read by id, or every one of them with all true. "+
				"With id and unread true it puts that notification back to unread instead, so it counts in your badge again. "+
				"Use notification_list to find ids and what is unread.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in notificationUpdateIn) (any, error) {
				userID, err := notificationCaller(ctx)
				if err != nil {
					return nil, err
				}
				if (in.ID == "") != in.All {
					return nil, fmt.Errorf("%w: send either id for one notification or all true for every one", apperrs.ErrInvalid)
				}
				if in.Unread && in.All {
					return nil, fmt.Errorf("%w: unread works on one notification, so send id and not all", apperrs.ErrInvalid)
				}
				if in.Unread {
					if err := s.MarkUnread(ctx, userID, in.ID); err != nil {
						return nil, err
					}
					return notificationUpdated{ID: in.ID}, nil
				}
				if in.All {
					if err := s.MarkAllRead(ctx, userID, in.WorkspaceID); err != nil {
						return nil, err
					}
					return notificationUpdated{All: true, Read: true}, nil
				}
				if err := s.MarkRead(ctx, userID, in.ID); err != nil {
					return nil, err
				}
				return notificationUpdated{ID: in.ID, Read: true}, nil
			}),
	}
}

// notificationCaller is the authenticated user; an argument never names whose inbox a tool touches.
func notificationCaller(ctx context.Context) (string, error) {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return "", fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	return a.ID, nil
}
