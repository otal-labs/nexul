// Package identity carries the acting user through a request context, so use-cases enforce access without auth.
package identity

import "context"

// Actor is the identity performing a use-case call.
type Actor struct {
	ID string
	// CanCreateWorkspace lets access checks short-circuit without a user lookup.
	CanCreateWorkspace bool
	// Automation is set for an automation-token request: domains record the automation, not its creator, as the actor.
	Automation *AutomationRef
}

// AutomationRef names the automation behind an automation-token request.
type AutomationRef struct {
	ID   string
	Name string
}

type actorKey struct{}

// WithActor attaches an actor to ctx.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// ActorFromCtx returns the acting actor, or zero when none is attached.
func ActorFromCtx(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(actorKey{}).(Actor)
	return a, ok
}
