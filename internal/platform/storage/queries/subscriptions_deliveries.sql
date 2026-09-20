-- name: AddIntegrationSubscription :exec
INSERT INTO integration_subscriptions (install_id, topic, created_at) VALUES (?, ?, ?);

-- name: RemoveIntegrationSubscription :execrows
DELETE FROM integration_subscriptions WHERE install_id = ? AND topic = ?;

-- name: ListIntegrationSubscriptionsByInstall :many
SELECT * FROM integration_subscriptions WHERE install_id = ? ORDER BY topic;

-- name: ListIntegrationSubscriptionsByTopic :many
SELECT * FROM integration_subscriptions WHERE topic = ? ORDER BY created_at;

-- name: CreateIntegrationDelivery :exec
INSERT INTO integration_deliveries (id, install_id, topic, event_id, payload, signature, url, status, attempts, next_attempt_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(install_id, event_id, topic) DO NOTHING;

-- name: ListDueIntegrationDeliveries :many
SELECT * FROM integration_deliveries
WHERE status IN ('pending', 'failed') AND next_attempt_at <= ?
ORDER BY next_attempt_at LIMIT ?;

-- name: MarkIntegrationDeliveryDelivered :execrows
UPDATE integration_deliveries SET status = 'delivered', delivered_at = ? WHERE id = ?;

-- name: MarkIntegrationDeliveryFailed :execrows
UPDATE integration_deliveries SET status = 'failed', attempts = ?, next_attempt_at = ? WHERE id = ?;

-- name: MarkIntegrationDeliveryDead :execrows
UPDATE integration_deliveries SET status = 'dead', attempts = attempts + 1 WHERE id = ?;

-- name: ListIntegrationDeliveriesByInstall :many
SELECT * FROM integration_deliveries WHERE install_id = ? ORDER BY created_at DESC;
