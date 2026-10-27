package botwebhook

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// botResult is a bot as the model reads it: no avatar bytes, and a URL only when the use-cases filled one in for a
// caller holding botwebhook:write.
type botResult struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	Name           string     `json:"name"`
	HasAvatar      bool       `json:"has_avatar"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	LastPostAt     *time.Time `json:"last_post_at,omitempty"`
	PostCount      int        `json:"post_count"`
	Deleted        bool       `json:"deleted"`
	URL            string     `json:"url,omitempty"`
	Stopped        string     `json:"stopped,omitempty"`
}

func toBotResult(b *Bot) botResult {
	return botResult{
		ID: b.ID, ConversationID: b.ConversationID, Name: b.Name, HasAvatar: b.Avatar != "", CreatedBy: b.CreatedBy,
		CreatedAt: b.CreatedAt, LastPostAt: b.LastPostAt, PostCount: b.PostCount, Deleted: b.DeletedAt != nil, URL: b.URL,
		Stopped: b.Stopped,
	}
}

type botListIn struct {
	ConversationID string `json:"conversation_id" jsonschema:"The conversation, from conversation_list."`
	Deleted        bool   `json:"deleted,omitzero" jsonschema:"true lists the deleted bots instead, the restore list."`
	mcptool.PageArgs
}

type botCreateIn struct {
	ConversationID string `json:"conversation_id" jsonschema:"The conversation, from conversation_list."`
	Name           string `json:"name" jsonschema:"The bot's name, at most 80 characters and unique among the conversation's live bots."`
	Avatar         string `json:"avatar,omitempty" jsonschema:"A base64 data URL image of at most 10MB. Omit for the built-in bot glyph."`
}

type botUpdateIn struct {
	ID         string  `json:"id" jsonschema:"The bot id, from botwebhook_list."`
	Name       *string `json:"name,omitempty" jsonschema:"New name, at most 80 characters and unique among the conversation's live bots."`
	Avatar     *string `json:"avatar,omitempty" jsonschema:"New avatar as a base64 data URL image of at most 10MB; an empty string clears it back to the built-in glyph."`
	Regenerate bool    `json:"regenerate,omitzero" jsonschema:"true issues a new URL and ends the old one at once."`
	Deleted    *bool   `json:"deleted,omitempty" jsonschema:"true deletes the bot and takes no other field in the same call; false restores a deleted bot under a fresh URL."`
}

// MCPTools returns the bot tools; none posts as a bot, a sender uses the bot's URL like any other client.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{botListTool(s), botCreateTool(s), botUpdateTool(s)}
}

func botListTool(s *Service) mcptool.Tool {
	return mcptool.New("botwebhook_list", "List bots",
		"Lists the bots of one conversation, each with its name, who created it, when it last posted, and how many posts it has made. "+
			"A bot is a named poster that outside systems drive through a webhook URL, up to ten live ones per conversation; use this to find a bot's id before botwebhook_update. "+
			"stopped says why a live bot can't post, and url, the bot's credential, appears only with botwebhook:write. "+
			"deleted true lists the deleted bots instead, the ones botwebhook_update deleted false restores.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in botListIn) (any, error) {
			bots, err := s.List(ctx, in.ConversationID, in.Deleted)
			if err != nil {
				return nil, err
			}
			items := make([]botResult, len(bots))
			for i, b := range bots {
				items[i] = toBotResult(b)
			}
			return mcptool.Paginate(items, in.PageArgs), nil
		})
}

func botCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("botwebhook_create", "Create bot",
		"Creates a bot in a conversation, so an outside system such as a CI job or a monitor can post there through the bot's webhook URL. "+
			"The bot's name shows on each post unless the sender overrides it. "+
			"Returns the new bot with its url, which is the credential to hand to the sender; botwebhook_list shows it again to anyone holding botwebhook:write.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in botCreateIn) (any, error) {
			b, err := s.Create(ctx, in.ConversationID, in.Name, in.Avatar)
			if err != nil {
				return nil, err
			}
			return toBotResult(b), nil
		})
}

func botUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("botwebhook_update", "Update bot",
		"Changes a bot's name or avatar, issues it a new URL, or deletes or restores it. "+
			"regenerate true ends the old URL at once and returns the new one, and it is not idempotent, so each repeat ends the previous URL. "+
			"deleted true ends the URL for good until deleted false restores the bot under a fresh one, and a deleted bot takes no other change until restored. "+
			"A bot's posts stay in the conversation after a rename, an avatar change, or a delete, because each keeps the name and avatar it showed. "+
			"Returns the bot, with its url when you hold botwebhook:write and it is live.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in botUpdateIn) (any, error) {
			b, err := s.Update(ctx, in.ID, Changes{Name: in.Name, Avatar: in.Avatar, Regenerate: in.Regenerate, Deleted: in.Deleted})
			if err != nil {
				return nil, err
			}
			return toBotResult(b), nil
		})
}
