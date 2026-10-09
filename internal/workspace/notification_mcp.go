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
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"Only notifications about tickets, docs, and memories in this workspace, from workspace_list. Omit for every workspace."`
	UnreadOnly  bool   `json:"unread_only,omitzero" jsonschema:"Only notifications you have not read yet. Defaults to false."`
	mcptool.PageArgs
}

type notificationUpdateIn struct {
	ID          string `json:"id,omitempty" jsonschema:"One notification's id, from notification_list, to mark read."`
	All         bool   `json:"all,omitzero" jsonschema:"true marks every one of your notifications read. Send instead of id."`
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"With all, marks only this workspace's notifications read, from workspace_list. Omit for every workspace."`
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
			"Lists your own notifications newest first: what happened (kind), to which ticket, doc, or memory (subject), and whether you have read it. "+
				"A doc's item also names the folder the doc is in now (folder_id, folder_name, folder_is_default for its project's default folder), so items can be grouped by folder. "+
				"Set workspace_id to see one workspace's inbox and unread_only to see only what is new, then mark items read with notification_update. "+
				"It never shows another user's inbox.",
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
		mcptool.New("notification_update", "Mark notifications read",
			"Marks one of your notifications read by id, or every one of them with all true; send exactly one of the two. "+
				"Use notification_list to find ids and what is unread. "+
				"It only ever touches your own inbox, and marking a read notification again changes nothing.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in notificationUpdateIn) (any, error) {
				userID, err := notificationCaller(ctx)
				if err != nil {
					return nil, err
				}
				if (in.ID == "") != in.All {
					return nil, fmt.Errorf("%w: send either id for one notification or all true for every one", apperrs.ErrInvalid)
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
