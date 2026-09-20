package identity

import (
	"context"
	"testing"
)

func TestActorRoundtrip(t *testing.T) {
	ctx := WithActor(context.Background(), Actor{ID: "u1", CanCreateWorkspace: true})
	a, ok := ActorFromCtx(ctx)
	if !ok || a.ID != "u1" || !a.CanCreateWorkspace {
		t.Fatalf("got %+v %v", a, ok)
	}
}
