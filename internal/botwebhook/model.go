// Package botwebhook implements bots: named posters bound to one conversation, driven from outside through a
// Discord-compatible webhook URL whose token is the credential.
package botwebhook

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// MaxNameLength is the longest name a bot takes, in characters, as Discord's webhook names.
const MaxNameLength = 80

// MaxLive is how many live bots one conversation holds; a deleted bot frees its place.
const MaxLive = 10

// maxAvatarBytes caps a decoded avatar the way a person's avatar override is capped.
const maxAvatarBytes = 10 << 20

// Bot is one conversation's poster; Token stays server-side and URL is filled only for a caller holding botwebhook:write.
type Bot struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Name           string `json:"name"`
	// Avatar is a base64 data: URI like a person's avatar override; empty shows the built-in bot glyph.
	Avatar     string     `json:"avatar,omitempty"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastPostAt *time.Time `json:"last_post_at,omitempty"`
	PostCount  int        `json:"post_count"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	DeletedBy  string     `json:"deleted_by,omitempty"`
	URL        string     `json:"url,omitempty"`
	Token      string     `json:"-"`
}

// Changes is a PATCH on a bot: a nil field keeps its value, an empty Avatar clears it, Deleted false restores.
type Changes struct {
	Name       *string `json:"name,omitempty"`
	Avatar     *string `json:"avatar,omitempty"`
	Regenerate bool    `json:"regenerate,omitempty"`
	Deleted    *bool   `json:"deleted,omitempty"`
}

// Change names one part of a bot botwebhook.updated reports as changed.
type Change string

const (
	ChangeRenamed     Change = "renamed"
	ChangeAvatar      Change = "avatar"
	ChangeRegenerated Change = "regenerated"
)

// Conversation is what bots need of the conversation they post into, read through chat's own read rule.
type Conversation struct {
	ID          string
	WorkspaceID string
	// MembersOnly marks a DM or private channel, whose bot events never reach integrations or automations.
	MembersOnly bool
}

// Person is a workspace member a bot's post may mention.
type Person struct {
	ID, Login string
}

// botName is the one validation a bot's name goes through, on create, rename, and restore.
func botName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", fmt.Errorf("%w: a bot needs a name", apperrs.ErrInvalid)
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		return "", fmt.Errorf("%w: a bot's name is at most %d characters", apperrs.ErrInvalid, MaxNameLength)
	}
	return name, nil
}

// checkAvatar takes an empty avatar, meaning the built-in glyph, or a base64 data: URI image of at most maxAvatarBytes.
func checkAvatar(raw string) error {
	if raw == "" {
		return nil
	}
	meta, data, ok := strings.Cut(strings.TrimPrefix(raw, "data:image/"), ",")
	if !ok || !strings.HasPrefix(raw, "data:image/") || !strings.HasSuffix(meta, ";base64") {
		return fmt.Errorf("%w: a bot's avatar must be a base64 data: URI image", apperrs.ErrInvalid)
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("%w: a bot's avatar has an invalid base64 payload", apperrs.ErrInvalid)
	}
	if len(decoded) > maxAvatarBytes {
		return fmt.Errorf("%w: a bot's avatar exceeds the %dMB limit", apperrs.ErrInvalid, maxAvatarBytes>>20)
	}
	return nil
}
