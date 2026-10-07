package botwebhook

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Discord's execute-webhook limits, in characters after trimming, as Discord measures them.
const (
	MaxBodyBytes       = 64 << 10
	maxContent         = 2000
	maxEmbeds          = 10
	maxEmbedText       = 6000
	maxTitle           = 256
	maxDescription     = 4096
	maxFields          = 25
	maxFieldName       = 256
	maxFieldValue      = 1024
	maxFooterText      = 2048
	maxEmbedAuthorName = 256
)

// Payload is Discord's execute-webhook JSON; fields Discord takes and Nexul does not, color among them, are dropped.
type Payload struct {
	Content         string           `json:"content"`
	Username        string           `json:"username"`
	AvatarURL       string           `json:"avatar_url"`
	Embeds          []Embed          `json:"embeds"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions"`
}

// Embed is one embed as a message stores it.
type Embed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	// Timestamp is kept as the sender wrote it: Uptime Kuma, among others, sends a time that is not RFC 3339.
	Timestamp string       `json:"timestamp,omitempty"`
	Footer    *EmbedFooter `json:"footer,omitempty"`
	Image     *EmbedImage  `json:"image,omitempty"`
	Thumbnail *EmbedImage  `json:"thumbnail,omitempty"`
	Author    *EmbedAuthor `json:"author,omitempty"`
	Fields    []EmbedField `json:"fields,omitempty"`
}

// EmbedFooter is an embed's footer line.
type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedImage is an embed's image or thumbnail.
type EmbedImage struct {
	URL string `json:"url"`
}

// EmbedAuthor is an embed's author line.
type EmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedField is one name and value of an embed's fields.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// AllowedMentions is Discord's allowed_mentions; roles and everyone are ignored, as Nexul mentions only people.
type AllowedMentions struct {
	Parse []string `json:"parse"`
	// Users are Nexul user ids or logins.
	Users []string `json:"users"`
}

// Message is the posted message as Discord answers wait=true.
type Message struct {
	ID        string        `json:"id"`
	ChannelID string        `json:"channel_id"`
	Content   string        `json:"content"`
	Embeds    []Embed       `json:"embeds"`
	Timestamp string        `json:"timestamp"`
	WebhookID string        `json:"webhook_id"`
	Author    MessageAuthor `json:"author"`
}

// MessageAuthor is the bot as a posted message names it.
type MessageAuthor struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Bot      bool   `json:"bot"`
}

// mentionPolicy is who a post may mention: everyone it names, only the listed people, or nobody.
type mentionPolicy struct {
	all   bool
	users []string
}

// policy reads allowed_mentions as Discord does: absent mentions everyone named, parse "users" too.
func (a *AllowedMentions) policy() mentionPolicy {
	if a == nil || slices.Contains(a.Parse, "users") {
		return mentionPolicy{all: true}
	}
	return mentionPolicy{users: a.Users}
}

// allows reports whether p lets a post mention the member with id and login.
func (p mentionPolicy) allows(id, login string) bool {
	return p.all || slices.ContainsFunc(p.users, func(ref string) bool {
		ref = strings.TrimSpace(ref)
		return ref == id || strings.EqualFold(ref, login)
	})
}

// checked is a payload trimmed and within Discord's limits, its empty embeds dropped.
type checked struct {
	content, username, avatarURL string
	embeds                       []Embed
}

// check holds p to Discord's limits, naming the one it breaks.
func (p Payload) check() (*checked, error) {
	c := &checked{content: strings.TrimSpace(p.Content), username: strings.TrimSpace(p.Username), avatarURL: strings.TrimSpace(p.AvatarURL)}
	if err := atMost("content", c.content, maxContent); err != nil {
		return nil, err
	}
	if err := atMost("username", c.username, MaxNameLength); err != nil {
		return nil, err
	}
	if err := httpURL("avatar_url", c.avatarURL); err != nil {
		return nil, err
	}
	if len(p.Embeds) > maxEmbeds {
		return nil, fmt.Errorf("%w: embeds holds %d, over the %d a post takes", apperrs.ErrInvalid, len(p.Embeds), maxEmbeds)
	}
	total := 0
	for i, e := range p.Embeds {
		e, n, err := e.check(fmt.Sprintf("embeds[%d]", i))
		if err != nil {
			return nil, err
		}
		total += n
		if e != nil {
			c.embeds = append(c.embeds, *e)
		}
	}
	if total > maxEmbedText {
		return nil, fmt.Errorf("%w: the embeds hold %d characters of text, over the %d a post takes", apperrs.ErrInvalid, total, maxEmbedText)
	}
	if c.content == "" && len(c.embeds) == 0 {
		return nil, fmt.Errorf("%w: a post needs content or an embed", apperrs.ErrInvalid)
	}
	return c, nil
}

// check trims e within Discord's per-field limits and counts its text; an embed with nothing in it is nil.
func (e Embed) check(at string) (*Embed, int, error) {
	out := Embed{
		Title: strings.TrimSpace(e.Title), Description: strings.TrimSpace(e.Description),
		URL: strings.TrimSpace(e.URL), Timestamp: strings.TrimSpace(e.Timestamp),
	}
	if err := firstErr(
		atMost(at+".title", out.Title, maxTitle), atMost(at+".description", out.Description, maxDescription),
		httpURL(at+".url", out.URL),
	); err != nil {
		return nil, 0, err
	}
	if len(e.Fields) > maxFields {
		return nil, 0, fmt.Errorf("%w: %s.fields holds %d, over the %d an embed takes", apperrs.ErrInvalid, at, len(e.Fields), maxFields)
	}
	for i, f := range e.Fields {
		f, err := f.check(fmt.Sprintf("%s.fields[%d]", at, i))
		if err != nil {
			return nil, 0, err
		}
		out.Fields = append(out.Fields, f)
	}
	var err error
	if out.Footer, err = e.Footer.check(at + ".footer"); err != nil {
		return nil, 0, err
	}
	if out.Author, err = e.Author.check(at + ".author"); err != nil {
		return nil, 0, err
	}
	if out.Image, err = e.Image.check(at + ".image"); err != nil {
		return nil, 0, err
	}
	if out.Thumbnail, err = e.Thumbnail.check(at + ".thumbnail"); err != nil {
		return nil, 0, err
	}
	if out.empty() {
		return nil, 0, nil
	}
	return &out, out.textLen(), nil
}

func (e Embed) empty() bool {
	return e.Title == "" && e.Description == "" && e.URL == "" && e.Timestamp == "" && len(e.Fields) == 0 &&
		e.Footer == nil && e.Image == nil && e.Thumbnail == nil && e.Author == nil
}

// textLen is what an embed spends of the post's 6000 characters.
func (e Embed) textLen() int {
	n := chars(e.Title) + chars(e.Description)
	for _, f := range e.Fields {
		n += chars(f.Name) + chars(f.Value)
	}
	if e.Footer != nil {
		n += chars(e.Footer.Text)
	}
	if e.Author != nil {
		n += chars(e.Author.Name)
	}
	return n
}

func (f EmbedField) check(at string) (EmbedField, error) {
	out := EmbedField{Name: strings.TrimSpace(f.Name), Value: strings.TrimSpace(f.Value), Inline: f.Inline}
	return out, firstErr(
		required(at+".name", out.Name), atMost(at+".name", out.Name, maxFieldName),
		required(at+".value", out.Value), atMost(at+".value", out.Value, maxFieldValue),
	)
}

func (f *EmbedFooter) check(at string) (*EmbedFooter, error) {
	if f == nil {
		return nil, nil
	}
	out := &EmbedFooter{Text: strings.TrimSpace(f.Text), IconURL: strings.TrimSpace(f.IconURL)}
	return out, firstErr(required(at+".text", out.Text), atMost(at+".text", out.Text, maxFooterText), httpURL(at+".icon_url", out.IconURL))
}

func (a *EmbedAuthor) check(at string) (*EmbedAuthor, error) {
	if a == nil {
		return nil, nil
	}
	out := &EmbedAuthor{Name: strings.TrimSpace(a.Name), URL: strings.TrimSpace(a.URL), IconURL: strings.TrimSpace(a.IconURL)}
	return out, firstErr(
		required(at+".name", out.Name), atMost(at+".name", out.Name, maxEmbedAuthorName),
		httpURL(at+".url", out.URL), httpURL(at+".icon_url", out.IconURL),
	)
}

func (i *EmbedImage) check(at string) (*EmbedImage, error) {
	if i == nil {
		return nil, nil
	}
	out := &EmbedImage{URL: strings.TrimSpace(i.URL)}
	return out, firstErr(required(at+".url", out.URL), httpURL(at+".url", out.URL))
}

func chars(s string) int { return utf8.RuneCountInString(s) }

func atMost(field, s string, limit int) error {
	if n := chars(s); n > limit {
		return fmt.Errorf("%w: %s is %d characters, over the %d it takes", apperrs.ErrInvalid, field, n, limit)
	}
	return nil
}

func required(field, s string) error {
	if s == "" {
		return fmt.Errorf("%w: %s is required", apperrs.ErrInvalid, field)
	}
	return nil
}

// httpURL takes an empty value or an absolute http(s) URL, the only links a bot message shows or loads.
func httpURL(field, s string) error {
	if s == "" {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%w: %s must be an http or https URL", apperrs.ErrInvalid, field)
	}
	return nil
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// storedEmbeds is embeds as chat stores them, nil for none.
func storedEmbeds(embeds []Embed) (json.RawMessage, error) {
	if len(embeds) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(embeds)
	if err != nil {
		return nil, fmt.Errorf("encode embeds: %w", err)
	}
	return raw, nil
}

// avatarPath is the bot's own avatar as a message shows it, versioned by the bot's last change.
func avatarPath(b *Bot) string {
	if b.Avatar == "" {
		return ""
	}
	return fmt.Sprintf("/api/botwebhooks/%s/avatar?v=%d", b.ID, b.UpdatedAt.Unix())
}
