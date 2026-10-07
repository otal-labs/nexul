package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ botwebhook.Repo = (*BotwebhooksRepo)(nil)

// BotwebhooksRepo stores bots with their token AES-256-GCM sealed; only this repo ever sees it sealed.
type BotwebhooksRepo struct {
	db     *sql.DB
	w      *Serializer
	q      *sqlcgen.Queries
	encKey []byte
}

func (r *BotwebhooksRepo) Create(ctx context.Context, b *botwebhook.Bot, evts ...eventbus.OutboxEvent) error {
	token, err := crypto.Encrypt(r.encKey, []byte(b.Token))
	if err != nil {
		return fmt.Errorf("seal token of bot %s: %w", b.ID, err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateBotwebhook(ctx, sqlcgen.CreateBotwebhookParams{
			ID: b.ID, ConversationID: b.ConversationID, Name: b.Name, Avatar: b.Avatar, Token: token,
			CreatedBy: b.CreatedBy, CreatedAt: b.CreatedAt.Unix(), UpdatedAt: b.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert bot %s: %w", b.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *BotwebhooksRepo) Get(ctx context.Context, id string) (*botwebhook.Bot, error) {
	row, err := r.q.GetBotwebhook(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get bot %s: %w", id, notFoundIfNoRows(err))
	}
	return r.toBot(row)
}

func (r *BotwebhooksRepo) List(ctx context.Context, conversationID string, deleted bool) ([]*botwebhook.Bot, error) {
	rows, err := r.q.ListBotwebhooks(ctx, sqlcgen.ListBotwebhooksParams{ConversationID: conversationID, Deleted: sql.NullInt64{Int64: int64(boolInt(deleted)), Valid: true}})
	if err != nil {
		return nil, fmt.Errorf("list bots of conversation %s: %w", conversationID, err)
	}
	out := make([]*botwebhook.Bot, 0, len(rows))
	for _, row := range rows {
		b, err := r.toBot(row)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *BotwebhooksRepo) Update(ctx context.Context, b *botwebhook.Bot, evts ...eventbus.OutboxEvent) error {
	token, err := crypto.Encrypt(r.encKey, []byte(b.Token))
	if err != nil {
		return fmt.Errorf("seal token of bot %s: %w", b.ID, err)
	}
	var deletedAt sql.NullInt64
	if b.DeletedAt != nil {
		deletedAt = sql.NullInt64{Int64: b.DeletedAt.Unix(), Valid: true}
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateBotwebhook(ctx, sqlcgen.UpdateBotwebhookParams{
			Name: b.Name, Avatar: b.Avatar, Token: token, UpdatedAt: b.UpdatedAt.Unix(), DeletedAt: deletedAt, DeletedBy: b.DeletedBy, ID: b.ID,
		})
		if err != nil {
			return fmt.Errorf("update bot %s: %w", b.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update bot %s: %w", b.ID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *BotwebhooksRepo) toBot(row sqlcgen.Botwebhook) (*botwebhook.Bot, error) {
	token, err := crypto.Decrypt(r.encKey, row.Token)
	if err != nil {
		return nil, fmt.Errorf("open token of bot %s: %w", row.ID, err)
	}
	return &botwebhook.Bot{
		ID: row.ID, ConversationID: row.ConversationID, Name: row.Name, Avatar: row.Avatar, CreatedBy: row.CreatedBy,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
		LastPostAt: unixPtrFromNull(row.LastPostAt), PostCount: int(row.PostCount),
		DeletedAt: unixPtrFromNull(row.DeletedAt), DeletedBy: row.DeletedBy, Token: string(token),
	}, nil
}
