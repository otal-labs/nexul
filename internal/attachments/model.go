package attachments

import (
	"strings"
	"time"
)

// Owner names the doc, ticket, conversation, or memory an attachment belongs to; exactly one field is set
// (ADR 0027; memory added ticket 18).
type Owner struct {
	DocID          string
	TicketID       string
	ConversationID string
	MemoryID       string
}

// Attachment is one uploaded file; Data is only populated by GetByID, never by list views.
type Attachment struct {
	ID             string    `json:"id"`
	DocID          string    `json:"doc_id,omitempty"`
	TicketID       string    `json:"ticket_id,omitempty"`
	ConversationID string    `json:"conversation_id,omitempty"`
	MemoryID       string    `json:"memory_id,omitempty"`
	Name           string    `json:"name"`
	ContentType    string    `json:"content_type"`
	Size           int64     `json:"size"`
	UploadedBy     string    `json:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at"`
	Data           []byte    `json:"-"`
}

// Owner returns the attachment's owning doc, ticket, conversation, or memory.
func (a *Attachment) Owner() Owner {
	return Owner{DocID: a.DocID, TicketID: a.TicketID, ConversationID: a.ConversationID, MemoryID: a.MemoryID}
}

// Inline allows only raster images inline, since SVG can carry scripts.
func (a *Attachment) Inline() bool {
	return strings.HasPrefix(a.ContentType, "image/") && !strings.Contains(a.ContentType, "svg")
}
