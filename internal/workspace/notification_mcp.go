package workspace

import (
	"context"
	"fmt"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// ponytail: pages in memory over the newest 1,000 notifications; add an offset query if inboxes grow past that.
const notificationScan = 1000

type notificationListIn struct {
	UnreadOnly bool `json:"unread_only,omitzero" jsonschema:"Only notifications you have not read yet. Defaults to false."`
	mcptool.PageArgs
}

type notificationUpdateIn struct {
	ID  string `json:"id,omitempty" jsonschema:"One notification's id, from notification_list, to mark read."`
	All bool   `json:"all,omitzero" jsonschema:"true marks every one of your notifications read. Send instead of id."`
}

type notificationResult struct {
	ID           string      `json:"id"`
	Kind         Kind        `json:"kind"`
	SubjectType  SubjectType `json:"subject_type"`
	SubjectID    string      `json:"subject_id"`
	SubjectTitle string      `json:"subject_title"`
	Read         bool        `json:"read"`
	CreatedAt    time.Time   `json:"created_at"`
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
				"Set unread_only to see only what is new, then mark items read with notification_update. "+
				"It never shows another user's inbox.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in notificationListIn) (any, error) {
				userID, err := notificationCaller(ctx)
				if err != nil {
					return nil, err
				}
				ns, err := s.List(ctx, userID, notificationScan)
				if err != nil {
					return nil, err
				}
				if in.UnreadOnly {
					ns = slices.DeleteFunc(ns, func(n *Notification) bool { return n.Read })
				}
				out := make([]notificationResult, 0, len(ns))
				for _, n := range ns {
					out = append(out, notificationResult{
						ID: n.ID, Kind: n.Kind, SubjectType: n.SubjectType, SubjectID: n.SubjectID, SubjectTitle: n.SubjectTitle,
						Read: n.Read, CreatedAt: n.CreatedAt,
					})
				}
				return mcptool.Paginate(out, in.PageArgs), nil
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
					if err := s.MarkAllRead(ctx, userID); err != nil {
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
