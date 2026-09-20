package deadletter

import (
	"context"
	"encoding/json"
	"fmt"
)

// Publisher republishes a stored payload back onto the bus. The bus satisfies
// this with Publish and PublishWithID.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
	PublishWithID(ctx context.Context, id, topic string, payload any) error
}

// Replay republishes a dead letter under its original ID/topic, then removes it from the store on success.
func Replay(ctx context.Context, store Storer, p Publisher, id string) error {
	dl, err := store.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := p.PublishWithID(ctx, dl.ID, dl.Topic, json.RawMessage(dl.Payload)); err != nil {
		return fmt.Errorf("replay %s: %w", id, err)
	}
	if err := store.Delete(ctx, id); err != nil {
		return fmt.Errorf("replay delete %s: %w", id, err)
	}
	return nil
}
