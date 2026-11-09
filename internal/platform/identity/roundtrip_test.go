package identity

import (
	"context"
	"testing"
)

func TestActorRoundtrip(t *testing.T) {
	ctx := WithActor(context.Background(), Actor{ID: "u1"})
	a, ok := ActorFromCtx(ctx)
	if !ok || a.ID != "u1" {
		t.Fatalf("got %+v %v", a, ok)
	}
}

func TestUserID_AttachedOrEmpty(t *testing.T) {
	if got := UserID(context.Background()); got != "" {
		t.Fatalf("bare context gave %q", got)
	}
	if got := UserID(WithUserID(context.Background(), "u1")); got != "u1" {
		t.Fatalf("got %q", got)
	}
}
