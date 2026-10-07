package botwebhook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// TestPayloadCheck_DiscordLimits holds each of Discord's limits at its edge, inclusively and after trimming, and names
// the field a post breaks.
func TestPayloadCheck_DiscordLimits(t *testing.T) {
	x := strings.Repeat
	fields := func(n int) []EmbedField {
		out := make([]EmbedField, n)
		for i := range out {
			out[i] = EmbedField{Name: "Service", Value: "api"}
		}
		return out
	}
	embeds := func(n int) []Embed {
		out := make([]Embed, n)
		for i := range out {
			out[i] = Embed{Title: "deploy"}
		}
		return out
	}
	tests := []struct {
		name    string
		p       Payload
		wantErr string
	}{
		{"content at the limit, padded", Payload{Content: "  " + x("a", 2000) + "\n"}, ""},
		{"content over", Payload{Content: x("a", 2001)}, "content"},
		{"content counts characters, not bytes", Payload{Content: x("é", 2000)}, ""},
		{"username over", Payload{Content: "hi", Username: x("a", 81)}, "username"},
		{"avatar_url not http", Payload{Content: "hi", AvatarURL: "javascript:alert(1)"}, "avatar_url"},
		{"ten embeds", Payload{Embeds: embeds(10)}, ""},
		{"eleven embeds", Payload{Embeds: embeds(11)}, "embeds holds 11"},
		{"title over", Payload{Embeds: []Embed{{Title: x("a", 257)}}}, "embeds[0].title"},
		{"description at the limit", Payload{Embeds: []Embed{{Description: x("a", 4096)}}}, ""},
		{"description over", Payload{Embeds: []Embed{{Description: x("a", 4097)}}}, "embeds[0].description"},
		{"embed url not http", Payload{Embeds: []Embed{{Title: "t", URL: "ftp://example.com"}}}, "embeds[0].url"},
		{"25 fields", Payload{Embeds: []Embed{{Fields: fields(25)}}}, ""},
		{"26 fields", Payload{Embeds: []Embed{{Fields: fields(26)}}}, "embeds[0].fields holds 26"},
		{"field name over", Payload{Embeds: []Embed{{Fields: []EmbedField{{Name: x("a", 257), Value: "v"}}}}}, "embeds[0].fields[0].name"},
		{"field value over", Payload{Embeds: []Embed{{Fields: []EmbedField{{Name: "n", Value: x("a", 1025)}}}}}, "embeds[0].fields[0].value"},
		{"field value blank", Payload{Embeds: []Embed{{Fields: []EmbedField{{Name: "n", Value: " "}}}}}, "embeds[0].fields[0].value is required"},
		{"footer over", Payload{Embeds: []Embed{{Footer: &EmbedFooter{Text: x("a", 2049)}}}}, "embeds[0].footer.text"},
		{"footer icon not http", Payload{Embeds: []Embed{{Footer: &EmbedFooter{Text: "f", IconURL: "data:image/png;base64,AA"}}}}, "embeds[0].footer.icon_url"},
		{"author name over", Payload{Embeds: []Embed{{Author: &EmbedAuthor{Name: x("a", 257)}}}}, "embeds[0].author.name"},
		{"author url not http", Payload{Embeds: []Embed{{Author: &EmbedAuthor{Name: "a", URL: "/relative"}}}}, "embeds[0].author.url"},
		{"image not http", Payload{Embeds: []Embed{{Image: &EmbedImage{URL: "attachment://chart.png"}}}}, "embeds[0].image.url"},
		{"thumbnail without a url", Payload{Embeds: []Embed{{Thumbnail: &EmbedImage{}}}}, "embeds[0].thumbnail.url is required"},
		{"6000 characters across embeds", Payload{Embeds: []Embed{{Description: x("a", 4000)}, {Description: x("a", 1900), Title: x("a", 100)}}}, ""},
		{"over 6000 across embeds", Payload{Embeds: []Embed{{Description: x("a", 4000)}, {Description: x("a", 2000), Footer: &EmbedFooter{Text: "f"}}}}, "6001 characters"},
		{"nothing to post", Payload{Content: " \n "}, "content or an embed"},
		{"only empty embeds", Payload{Embeds: []Embed{{}, {Title: "  "}}}, "content or an embed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.p.check()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
