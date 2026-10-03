package harness

import (
	"context"
	"errors"
)

// Forward is from until from returns a *MovedError: a session call then runs moved once and retries on to, while
// Pair and Version, which have no session, retry on to alone. A failed moved is returned and to is not called.
func Forward(from, to Client, moved func(ctx context.Context, s Session, to Kind) error) Client {
	return forward{from: from, to: to, moved: moved}
}

type forward struct {
	from, to Client
	moved    func(ctx context.Context, s Session, to Kind) error
}

func follow[T any](ctx context.Context, f forward, s Session, call func(Client) (T, error)) (T, error) {
	out, err := call(f.from)
	var moved *MovedError
	if !errors.As(err, &moved) {
		return out, err
	}
	if err := f.moved(ctx, s, moved.To); err != nil {
		var zero T
		return zero, err
	}
	return call(f.to)
}

func retry[T any](f forward, call func(Client) (T, error)) (T, error) {
	out, err := call(f.from)
	var moved *MovedError
	if !errors.As(err, &moved) {
		return out, err
	}
	return call(f.to)
}

func (f forward) Kind() Kind { return f.from.Kind() }

func (f forward) Pair(ctx context.Context, serverURL, secret string) (PairResult, error) {
	return retry(f, func(c Client) (PairResult, error) { return c.Pair(ctx, serverURL, secret) })
}

func (f forward) Version(ctx context.Context, serverURL string) (string, error) {
	return retry(f, func(c Client) (string, error) { return c.Version(ctx, serverURL) })
}

func (f forward) ListProjects(ctx context.Context, s Session) ([]Project, error) {
	return follow(ctx, f, s, func(c Client) ([]Project, error) { return c.ListProjects(ctx, s) })
}

func (f forward) ListProviders(ctx context.Context, s Session) ([]Provider, error) {
	return follow(ctx, f, s, func(c Client) ([]Provider, error) { return c.ListProviders(ctx, s) })
}

func (f forward) Hold(ctx context.Context, s Session) (Conn, error) {
	return follow(ctx, f, s, func(c Client) (Conn, error) { return c.Hold(ctx, s) })
}

func (f forward) StartTurn(ctx context.Context, t Target, title string, prompts TurnPrompts) (StartResult, error) {
	return follow(ctx, f, t.Session, func(c Client) (StartResult, error) { return c.StartTurn(ctx, t, title, prompts) })
}

func (f forward) Interrupt(ctx context.Context, t Target) error {
	_, err := follow(ctx, f, t.Session, func(c Client) (struct{}, error) { return struct{}{}, c.Interrupt(ctx, t) })
	return err
}

func (f forward) Answer(ctx context.Context, t Target, requestID string, answer QuestionAnswer) error {
	_, err := follow(ctx, f, t.Session, func(c Client) (struct{}, error) { return struct{}{}, c.Answer(ctx, t, requestID, answer) })
	return err
}

func (f forward) Settle(ctx context.Context, t Target) error {
	_, err := follow(ctx, f, t.Session, func(c Client) (struct{}, error) { return struct{}{}, c.Settle(ctx, t) })
	return err
}
