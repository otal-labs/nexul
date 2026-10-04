package t3clientv2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
)

const persistAttachments = "assets.persistChatAttachments"

// supportedImages are the types T3's providers take natively; Claude fails the run on any other.
var supportedImages = []string{"image/gif", "image/jpeg", "image/png", "image/webp"}

type persistInput struct {
	ThreadID    string        `json:"threadId"`
	MessageID   string        `json:"messageId"`
	Attachments []uploadImage `json:"attachments"`
}

// uploadImage carries its bytes as a data URL whose mime T3 requires to equal the lowercased mimeType.
type uploadImage struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	MIMEType  string `json:"mimeType"`
	SizeBytes int    `json:"sizeBytes"`
	DataURL   string `json:"dataUrl"`
}

// TakesImage implements harness.ImageTaker: only the types T3's providers take go to the agent.
func (h *Harness) TakesImage(mime string) bool {
	return slices.Contains(supportedImages, strings.ToLower(mime))
}

var _ harness.ImageTaker = (*Harness)(nil)

// persistImages uploads the images T3 takes and returns the references it minted, plus a note for those left out.
func (t *turn) persistImages(ctx context.Context, images []harness.Attachment) ([]json.RawMessage, []harness.Update, error) {
	var upload []uploadImage
	var skipped []string
	for _, img := range images {
		mime := strings.ToLower(img.MIME)
		if !t.h.TakesImage(mime) {
			t.h.log().Warn("t3clientv2: image left out, T3 Code takes only gif, jpeg, png and webp", "name", img.Name, "mime", img.MIME)
			skipped = append(skipped, img.Name)
			continue
		}
		upload = append(upload, uploadImage{
			Type: "image", Name: img.Name, MIMEType: mime, SizeBytes: len(img.Bytes),
			DataURL: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(img.Bytes),
		})
	}
	var notes []harness.Update
	if len(skipped) > 0 {
		notes = append(notes, note(fmt.Sprintf("Not sent to T3 Code: %s. It takes only gif, jpeg, png and webp images.", strings.Join(skipped, ", "))))
	}
	if len(upload) == 0 {
		return []json.RawMessage{}, notes, nil
	}
	out, err := t.conn.Call(ctx, persistAttachments, persistInput{ThreadID: t.threadID, MessageID: t.messageID, Attachments: upload})
	if err != nil {
		return nil, nil, refused("upload the images to T3 Code", err)
	}
	var persisted struct {
		Attachments []json.RawMessage `json:"attachments"`
	}
	if err := json.Unmarshal(out, &persisted); err != nil {
		return nil, nil, fmt.Errorf("read the image references T3 Code returned: %w", err)
	}
	return persisted.Attachments, notes, nil
}
