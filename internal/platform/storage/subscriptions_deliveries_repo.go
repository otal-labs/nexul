package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/integrations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ integrations.SubscriptionStore = (*IntegrationSubscriptionsRepo)(nil)
var _ integrations.DeliveryStore = (*IntegrationDeliveriesRepo)(nil)

// IntegrationSubscriptionsRepo persists per-topic webhook subscriptions.
type IntegrationSubscriptionsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *IntegrationSubscriptionsRepo) Add(ctx context.Context, sub integrations.Subscription) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).AddIntegrationSubscription(ctx, sqlcgen.AddIntegrationSubscriptionParams{
			InstallID: sub.InstallID, Topic: sub.Topic, CreatedAt: sub.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("add subscription %s/%s: %w", sub.InstallID, sub.Topic, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IntegrationSubscriptionsRepo) Remove(ctx context.Context, installID, topic string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RemoveIntegrationSubscription(ctx, sqlcgen.RemoveIntegrationSubscriptionParams{
			InstallID: installID, Topic: topic,
		})
		if err != nil {
			return fmt.Errorf("remove subscription %s/%s: %w", installID, topic, err)
		}
		if n == 0 {
			return fmt.Errorf("remove subscription %s/%s: %w", installID, topic, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *IntegrationSubscriptionsRepo) ListByInstall(ctx context.Context, installID string) ([]integrations.Subscription, error) {
	rows, err := r.q.ListIntegrationSubscriptionsByInstall(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	var out []integrations.Subscription
	for _, row := range rows {
		out = append(out, toSubscription(row))
	}
	return out, nil
}

func (r *IntegrationSubscriptionsRepo) ListByTopic(ctx context.Context, topic string) ([]integrations.Subscription, error) {
	rows, err := r.q.ListIntegrationSubscriptionsByTopic(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions for %s: %w", topic, err)
	}
	var out []integrations.Subscription
	for _, row := range rows {
		out = append(out, toSubscription(row))
	}
	return out, nil
}

func toSubscription(row sqlcgen.IntegrationSubscription) integrations.Subscription {
	return integrations.Subscription{
		InstallID: row.InstallID,
		Topic:     row.Topic,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}

// IntegrationDeliveriesRepo persists webhook deliveries and drives the relay.
type IntegrationDeliveriesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *IntegrationDeliveriesRepo) Create(ctx context.Context, d *integrations.Delivery) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateIntegrationDelivery(ctx, sqlcgen.CreateIntegrationDeliveryParams{
			ID: d.ID, InstallID: d.InstallID, Topic: d.Topic, EventID: d.EventID, Payload: d.Payload,
			Signature: d.Signature, Url: d.URL, Status: string(d.Status), Attempts: int64(d.Attempts),
			NextAttemptAt: d.NextAttemptAt.Unix(), CreatedAt: d.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert delivery %s: %w", d.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IntegrationDeliveriesRepo) Due(ctx context.Context, limit int, now int64) ([]*integrations.Delivery, error) {
	rows, err := r.q.ListDueIntegrationDeliveries(ctx, sqlcgen.ListDueIntegrationDeliveriesParams{
		NextAttemptAt: now, Limit: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("query due deliveries: %w", err)
	}
	var out []*integrations.Delivery
	for _, row := range rows {
		out = append(out, toDelivery(row))
	}
	return out, nil
}

func (r *IntegrationDeliveriesRepo) MarkDelivered(ctx context.Context, id string, at int64) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MarkIntegrationDeliveryDelivered(ctx, sqlcgen.MarkIntegrationDeliveryDeliveredParams{
			DeliveredAt: sql.NullInt64{Int64: at, Valid: true}, ID: id,
		})
		if err != nil {
			return fmt.Errorf("mark delivery %s delivered: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mark delivery %s delivered: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *IntegrationDeliveriesRepo) MarkFailed(ctx context.Context, id string, attempts int, nextAttemptAt int64) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MarkIntegrationDeliveryFailed(ctx, sqlcgen.MarkIntegrationDeliveryFailedParams{
			Attempts: int64(attempts), NextAttemptAt: nextAttemptAt, ID: id,
		})
		if err != nil {
			return fmt.Errorf("mark delivery %s failed: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mark delivery %s failed: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *IntegrationDeliveriesRepo) MarkDead(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MarkIntegrationDeliveryDead(ctx, id)
		if err != nil {
			return fmt.Errorf("mark delivery %s dead: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mark delivery %s dead: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *IntegrationDeliveriesRepo) ListByInstall(ctx context.Context, installID string) ([]*integrations.Delivery, error) {
	rows, err := r.q.ListIntegrationDeliveriesByInstall(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	var out []*integrations.Delivery
	for _, row := range rows {
		out = append(out, toDelivery(row))
	}
	return out, nil
}

func toDelivery(row sqlcgen.IntegrationDelivery) *integrations.Delivery {
	d := &integrations.Delivery{
		ID:            row.ID,
		InstallID:     row.InstallID,
		Topic:         row.Topic,
		EventID:       row.EventID,
		Payload:       row.Payload,
		Signature:     row.Signature,
		URL:           row.Url,
		Status:        integrations.DeliveryStatus(row.Status),
		Attempts:      int(row.Attempts),
		NextAttemptAt: time.Unix(row.NextAttemptAt, 0).UTC(),
		CreatedAt:     time.Unix(row.CreatedAt, 0).UTC(),
	}
	if row.DeliveredAt.Valid {
		t := time.Unix(row.DeliveredAt.Int64, 0).UTC()
		d.DeliveredAt = &t
	}
	return d
}
