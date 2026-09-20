package collab

import (
	"encoding/json"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// MaxFrameBytes is generous but finite, since full-state commits for large docs dwarf incremental updates.
const MaxFrameBytes = 16 << 20

// message type constants (protocol v1) are additive-only: never renamed, so older clients degrade gracefully.
const (
	msgHello    = "hello"
	msgInit     = "init"
	msgUpdate   = "update"
	msgCommit   = "commit"
	msgPresence = "presence"
	msgLeave    = "leave"
)

// ClientMsg is the wire shape of a client→server frame.
type ClientMsg struct {
	Type     string `json:"type"`
	ClientID int    `json:"client_id,omitempty"` // y client id (hello, presence)
	Update   string `json:"update,omitempty"`    // base64 Y.js update (update/commit)
	Payload  string `json:"payload,omitempty"`   // base64 awareness state (presence)
	BaseSeq  int64  `json:"base_seq,omitempty"`  // commit: highest seq already applied
	Title    string `json:"title,omitempty"`     // commit: converged title (LWW)
	Body     string `json:"body,omitempty"`      // commit: converged canonical body
}

// ServerMsg is the wire shape of a server→client frame.
type ServerMsg struct {
	Type     string         `json:"type"`
	From     string         `json:"from,omitempty"`      // actor id that authored the relay
	ClientID int            `json:"client_id,omitempty"` // y client id (presence/leave routing)
	Seq      int64          `json:"seq,omitempty"`       // stored seq of the payload/state
	Snapshot *StoredUpdate  `json:"snapshot,omitempty"`  // init: newest full-state commit
	Updates  []StoredUpdate `json:"updates,omitempty"`   // init: increments after the snapshot
	Payload  string         `json:"payload,omitempty"`   // relayed update/awareness payload
	Presence []PresenceMsg  `json:"presence,omitempty"`  // init: current roster
	Title    string         `json:"title,omitempty"`     // commit: new canonical title, when it changed
}

// PresenceMsg is one participant's awareness state, relayed to a new joiner so it sees who is already present.
type PresenceMsg struct {
	ClientID int    `json:"client_id"`
	Payload  string `json:"payload"`
}

// parseClientMsg drops unknown types for forward compatibility, but rejects malformed frames.
func parseClientMsg(raw []byte) (ClientMsg, error) {
	var m ClientMsg
	if err := json.Unmarshal(raw, &m); err != nil {
		return ClientMsg{}, fmt.Errorf("parse collab frame: %w", err)
	}
	switch m.Type {
	case msgHello:
		if m.ClientID <= 0 {
			return ClientMsg{}, fmt.Errorf("%w: hello needs a positive client_id", apperrs.ErrInvalid)
		}
	case msgUpdate:
		if strings.TrimSpace(m.Update) == "" {
			return ClientMsg{}, fmt.Errorf("%w: update needs a payload", apperrs.ErrInvalid)
		}
	case msgCommit:
		if strings.TrimSpace(m.Update) == "" {
			return ClientMsg{}, fmt.Errorf("%w: commit needs a full-state update", apperrs.ErrInvalid)
		}
	case msgPresence:
		if strings.TrimSpace(m.Payload) == "" {
			return ClientMsg{}, fmt.Errorf("%w: presence needs a payload", apperrs.ErrInvalid)
		}
	default:
		return ClientMsg{}, fmt.Errorf("%w: unknown collab frame type %q", apperrs.ErrInvalid, m.Type)
	}
	return m, nil
}
