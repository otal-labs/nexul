package agent

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
)

// attachmentRefPattern matches both shapes an attachment's URL renders as in markdown: an image
// (richtext.JSONToMarkdown, the ticket/doc editors) and a bare link (a non-image attachment reference).
var attachmentRefPattern = regexp.MustCompile(`!?\[([^\]]*)\]\(/api/attachments/([^\s")]+)(?:\s+"[^"]*")?\)`)

// StoredAttachment is one attachment's bytes and metadata, read from the attachments domain.
type StoredAttachment struct {
	Name  string
	MIME  string
	Bytes []byte
}

// AttachmentReader is the agent pipeline's seam onto stored attachment bytes (ADR 0017: agent never imports attachments).
type AttachmentReader interface {
	Get(ctx context.Context, id string) (StoredAttachment, error)
}

// bodyImage is one image a ticket or doc body embeds: attached to the turn, or left for the agent to open by its link.
type bodyImage struct {
	name string
	link string
	// attachment is nil for an image the harness does not take or that does not fit the size caps.
	attachment *harness.Attachment
}

// bodyImages reads each image markdown embeds once, attaching those takes accepts that fit the size cap and what is left of budget.
func bodyImages(ctx context.Context, markdown string, reader AttachmentReader, takes func(mime string) bool, budget *int) []bodyImage {
	seen := map[string]bool{}
	var images []bodyImage
	for _, ref := range attachmentRefPattern.FindAllStringSubmatch(markdown, -1) {
		id := ref[2]
		if seen[id] {
			continue
		}
		seen[id] = true
		stored, err := reader.Get(ctx, id)
		if err != nil {
			slog.Default().Warn("agent: fetch attachment for turn failed", "attachment", id, "error", err)
			continue
		}
		if !strings.HasPrefix(stored.MIME, "image/") {
			continue
		}
		img := bodyImage{name: stored.Name, link: "/api/attachments/" + id}
		if takes(stored.MIME) && len(stored.Bytes) <= MaxAttachmentBytes && len(stored.Bytes) <= *budget {
			*budget -= len(stored.Bytes)
			img.attachment = &harness.Attachment{Name: stored.Name, MIME: stored.MIME, Bytes: stored.Bytes}
		}
		images = append(images, img)
	}
	return images
}

// nameAttached puts each attached image's name in place of its markdown, since the harness shows the file and cannot open the link.
func nameAttached(markdown string, images []bodyImage) string {
	names := map[string]string{}
	for _, img := range images {
		if img.attachment != nil {
			names[img.link] = img.name
		}
	}
	return attachmentRefPattern.ReplaceAllStringFunc(markdown, func(ref string) string {
		name, ok := names["/api/attachments/"+attachmentRefPattern.FindStringSubmatch(ref)[2]]
		if !ok {
			return ref
		}
		return "[Attached image: " + name + "]"
	})
}

// nameAttachedIn is nameAttached over each message's body.
func nameAttachedIn(msgs []ContextMessage, images []bodyImage) []ContextMessage {
	out := make([]ContextMessage, len(msgs))
	for i, m := range msgs {
		m.Body = nameAttached(m.Body, images)
		out[i] = m
	}
	return out
}
