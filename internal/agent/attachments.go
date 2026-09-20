package agent

import (
	"context"
	"fmt"
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

// AttachmentBudget bounds the total attachment bytes collected across every ExtractAttachments call sharing it.
type AttachmentBudget struct {
	Remaining int
}

// NewAttachmentBudget returns a budget capped at MaxTurnAttachmentBytes for one turn.
func NewAttachmentBudget() *AttachmentBudget {
	return &AttachmentBudget{Remaining: MaxTurnAttachmentBytes}
}

// ExtractAttachments rewrites every attachment reference in markdown to "[image: name, attached to this turn]" and returns the
// images among them as harness attachments, fetching each id once. A nil reader leaves markdown untouched.
// A non-image attachment, one over MaxAttachmentBytes, or one that would break budget's remaining bytes
// becomes "[attachment omitted: name]" instead; so does a fetch error, which also logs a warning — a bad
// attachment never fails the turn.
func ExtractAttachments(ctx context.Context, markdown string, reader AttachmentReader, budget *AttachmentBudget) (string, []harness.Attachment) {
	if reader == nil {
		return markdown, nil
	}
	if budget == nil {
		budget = NewAttachmentBudget()
	}
	fetched := map[string]string{} // id -> the note already resolved for it this call
	var atts []harness.Attachment
	rewritten := attachmentRefPattern.ReplaceAllStringFunc(markdown, func(match string) string {
		groups := attachmentRefPattern.FindStringSubmatch(match)
		label, id := groups[1], groups[2]
		if note, ok := fetched[id]; ok {
			return note
		}
		note, att := resolveAttachment(ctx, id, label, reader, budget)
		fetched[id] = note
		if att != nil {
			atts = append(atts, *att)
		}
		return note
	})
	return rewritten, atts
}

// resolveAttachment fetches one attachment and decides whether it fits within the per-attachment and
// per-turn caps; the caller rewrites every occurrence of id in the markdown to the string returned here.
func resolveAttachment(ctx context.Context, id, label string, reader AttachmentReader, budget *AttachmentBudget) (string, *harness.Attachment) {
	fallback := label
	if fallback == "" {
		fallback = id
	}
	stored, err := reader.Get(ctx, id)
	if err != nil {
		slog.Default().Warn("agent: fetch attachment for turn failed", "attachment", id, "error", err)
		return omittedNote(fallback), nil
	}
	if !strings.HasPrefix(stored.MIME, "image/") || len(stored.Bytes) > MaxAttachmentBytes || len(stored.Bytes) > budget.Remaining {
		return omittedNote(stored.Name), nil
	}
	budget.Remaining -= len(stored.Bytes)
	return attachmentNote(stored.Name), &harness.Attachment{Name: stored.Name, MIME: stored.MIME, Bytes: stored.Bytes}
}

func omittedNote(name string) string {
	return fmt.Sprintf("[attachment omitted: %s]", name)
}

func attachmentNote(name string) string {
	return fmt.Sprintf("[image: %s, attached to this turn]", name)
}
