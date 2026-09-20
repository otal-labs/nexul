package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Indexer consumes the re-index topics; v1's FTS5 index syncs via DB triggers, so handlers only validate and ack.
type Indexer struct {
	log *slog.Logger
}

// NewIndexer builds an Indexer with the given logger (slog.Default when nil).
func NewIndexer(log *slog.Logger) *Indexer {
	if log == nil {
		log = slog.Default()
	}
	return &Indexer{log: log}
}

// HandleDocEvent validates a doc.created/doc.updated payload; the wire shape is mirrored (ADR 0017).
func (i *Indexer) HandleDocEvent(ctx context.Context, ev eventbus.Event) error {
	var p struct {
		Doc struct {
			ID string `json:"id"`
		} `json:"doc"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.Doc.ID == "" {
		return apperrs.Fatal(fmt.Errorf("%w: %s payload missing doc id", apperrs.ErrInvalid, ev.Topic))
	}
	i.log.Debug("re-index doc", "id", p.Doc.ID, "topic", ev.Topic)
	return nil
}

// HandleTicketCreated validates a ticket.created payload.
func (i *Indexer) HandleTicketCreated(ctx context.Context, ev eventbus.Event) error {
	var p struct {
		Ticket struct {
			ID string `json:"id"`
		} `json:"ticket"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.Ticket.ID == "" {
		return apperrs.Fatal(fmt.Errorf("%w: %s payload missing ticket id", apperrs.ErrInvalid, ev.Topic))
	}
	i.log.Debug("re-index ticket", "id", p.Ticket.ID, "topic", ev.Topic)
	return nil
}
