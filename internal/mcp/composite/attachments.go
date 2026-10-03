package composite

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/attachments"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/tickets"
)

// AttachmentTools upload and read the files people attach to docs, tickets, chat, and memories; they take a ticket
// key, so they reach into the tickets domain.
func AttachmentTools(a *attachments.Service, t *tickets.Service) []mcptool.Tool {
	return []mcptool.Tool{attachmentCreateTool(a, t), attachmentGetTool(a)}
}

const attachmentPath = "/api/attachments/"

// attachmentResult carries the markdown the browser itself writes for an upload: an image renders inline, any other
// file is a download link.
type attachmentResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	Markdown    string `json:"markdown"`
}

func toAttachmentResult(a *attachments.Attachment) attachmentResult {
	url := attachmentPath + a.ID
	markdown := "[" + a.Name + "](" + url + ")"
	if a.Inline() {
		markdown = "!" + markdown
	}
	return attachmentResult{ID: a.ID, Name: a.Name, ContentType: a.ContentType, Size: a.Size, URL: url, Markdown: markdown}
}

// attachmentID accepts the id or any URL or path that embeds it, since agents meet attachments as links in bodies.
func attachmentID(ref string) string {
	ref = strings.TrimSpace(ref)
	if i := strings.LastIndex(ref, attachmentPath); i >= 0 {
		return ref[i+len(attachmentPath):]
	}
	return ref
}

func getAttachment(ctx context.Context, a *attachments.Service, ref string) (*attachments.Attachment, error) {
	got, err := a.Get(ctx, attachmentID(ref))
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w; an attachment id comes from attachment_create or the /api/attachments/<id> link in a body or message", err)
	}
	return got, err
}

type attachmentCreateInput struct {
	DocID            string `json:"doc_id,omitempty" jsonschema:"The doc to attach the file to, from doc_list. Name exactly one of doc_id, ticket_id, conversation_id, or memory_id."`
	TicketID         string `json:"ticket_id,omitempty" jsonschema:"The ticket to attach the file to, by id or key, for example REF-102."`
	Workspace        string `json:"workspace,omitempty" jsonschema:"The workspace to look a ticket key up in, by its id or slug from workspace_list, for example otal. Needed only when the key exists in more than one of your workspaces; an id needs none."`
	ConversationID   string `json:"conversation_id,omitempty" jsonschema:"The conversation to attach the file to, from conversation_list, or the conversation_id that message_list and message_post return for a ticket's or doc's thread."`
	MemoryID         string `json:"memory_id,omitempty" jsonschema:"The memory to attach the file to, from memory_list."`
	Name             string `json:"name,omitempty" jsonschema:"The file name people see, for example logo.png. Required with content; with from_attachment_id it defaults to the copied file's name."`
	Content          string `json:"content,omitempty" jsonschema:"The file's bytes, base64-encoded. Send this or from_attachment_id, not both."`
	FromAttachmentID string `json:"from_attachment_id,omitempty" jsonschema:"An attachment to copy, by its id or its /api/attachments/<id> link, for example an image posted in chat; the bytes are copied on the server."`
}

func attachmentCreateTool(a *attachments.Service, t *tickets.Service) mcptool.Tool {
	return mcptool.New("attachment_create", "Upload an attachment",
		"Uploads a file, or copies an existing attachment, onto a doc, ticket, conversation, or memory, the same as a person's upload: the same permission to change it, the same 10 MiB cap, and the content type read from the bytes. "+
			"It returns the attachment's id, url, and markdown; put the markdown on a line of its own in a body (doc_update, ticket_update, memory_update) or a message (message_post) and an image shows inline, any other file as a download link. "+
			"To read a file people attached, use attachment_get.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in attachmentCreateInput) (any, error) {
			data, name, err := attachmentSource(ctx, a, in)
			if err != nil {
				return nil, err
			}
			owner := attachments.Owner{DocID: in.DocID, ConversationID: in.ConversationID, MemoryID: in.MemoryID}
			if in.TicketID != "" {
				tk, err := resolveTicket(ctx, t, in.Workspace, in.TicketID)
				if err != nil {
					return nil, err
				}
				owner.TicketID = tk.ID
			}
			created, err := a.Upload(ctx, owner, name, data)
			if err != nil {
				return nil, err
			}
			return toAttachmentResult(created), nil
		})
}

func attachmentSource(ctx context.Context, a *attachments.Service, in attachmentCreateInput) ([]byte, string, error) {
	if (in.Content == "") == (in.FromAttachmentID == "") {
		return nil, "", fmt.Errorf("%w: send exactly one of content or from_attachment_id", apperrs.ErrInvalid)
	}
	if in.FromAttachmentID != "" {
		src, err := getAttachment(ctx, a, in.FromAttachmentID)
		if err != nil {
			return nil, "", err
		}
		if in.Name == "" {
			return src.Data, src.Name, nil
		}
		return src.Data, in.Name, nil
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, "", fmt.Errorf("%w: name is required with content, for example logo.png", apperrs.ErrInvalid)
	}
	data, err := base64.StdEncoding.DecodeString(in.Content)
	if err != nil {
		return nil, "", fmt.Errorf("%w: content is not standard base64: %v", apperrs.ErrInvalid, err)
	}
	return data, in.Name, nil
}

type attachmentGetInput struct {
	ID string `json:"id" jsonschema:"The attachment's id, or the /api/attachments/<id> link a body or message embeds it by."`
}

func attachmentGetTool(a *attachments.Service) mcptool.Tool {
	return mcptool.New("attachment_get", "Get an attachment",
		"Returns one attachment's name, content type, and size, followed by its contents: an image you can look at, a text file as text, or any other file as a base64 resource. "+
			"Use it to read a screenshot or file people attached to a doc, ticket, conversation, or memory; it needs read access to that owner, the same as opening it in the browser. "+
			"To put a copy of the file somewhere else, use attachment_create with from_attachment_id instead of reading the bytes.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in attachmentGetInput) (any, error) {
			got, err := getAttachment(ctx, a, in.ID)
			if err != nil {
				return nil, err
			}
			return mcptool.File{Meta: toAttachmentResult(got), URI: "attachments://" + got.ID, MIMEType: got.ContentType, Data: got.Data}, nil
		})
}
