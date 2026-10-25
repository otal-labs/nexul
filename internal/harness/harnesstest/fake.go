// Package harnesstest is the one in-memory harness.Client every domain test uses instead of its own fake.
package harnesstest

import (
	"context"

	"github.com/otal-labs/nexul/internal/harness"
)

// Client implements harness.Client with per-method function fields; an unset field returns the zero value and nil.
type Client struct {
	KindValue       harness.Kind
	PairFn          func(ctx context.Context, serverURL, secret string) (harness.PairResult, error)
	VersionFn       func(ctx context.Context, serverURL string) (string, error)
	ListProjectsFn  func(ctx context.Context, s harness.Session) ([]harness.Project, error)
	CloneProjectFn  func(ctx context.Context, s harness.Session, title, remoteURL, path string) (harness.Project, error)
	CanCloneFn      func(ctx context.Context, s harness.Session) (bool, error)
	CreateProjectFn func(ctx context.Context, s harness.Session, title, path string) (harness.Project, error)
	ListProvidersFn func(ctx context.Context, s harness.Session) ([]harness.Provider, error)
	HoldFn          func(ctx context.Context, s harness.Session) (harness.Conn, error)
	StartTurnFn     func(ctx context.Context, t harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error)
	WatchFn         func(ctx context.Context, t harness.Target) (harness.StartResult, error)
	InterruptFn     func(ctx context.Context, t harness.Target) error
	AnswerFn        func(ctx context.Context, t harness.Target, requestID string, answer harness.QuestionAnswer) error
	AnswerSecretFn  func(ctx context.Context, t harness.Target, requestID, secret string) error
	SettleFn        func(ctx context.Context, t harness.Target) error
	SteerFn         func(ctx context.Context, t harness.Target) error
}

// Registry wraps c as the sole client of its kind (KindT3Code when unset).
func Registry(c *Client) harness.Registry {
	if c.KindValue == "" {
		c.KindValue = harness.KindT3Code
	}
	return harness.Registry{c.KindValue: c}
}

func (c *Client) Kind() harness.Kind { return c.KindValue }

func (c *Client) Pair(ctx context.Context, serverURL, secret string) (harness.PairResult, error) {
	if c.PairFn == nil {
		return harness.PairResult{}, nil
	}
	return c.PairFn(ctx, serverURL, secret)
}

func (c *Client) Version(ctx context.Context, serverURL string) (string, error) {
	if c.VersionFn == nil {
		return "", nil
	}
	return c.VersionFn(ctx, serverURL)
}

func (c *Client) ListProjects(ctx context.Context, s harness.Session) ([]harness.Project, error) {
	if c.ListProjectsFn == nil {
		return nil, nil
	}
	return c.ListProjectsFn(ctx, s)
}

func (c *Client) CloneProject(ctx context.Context, s harness.Session, title, remoteURL, path string) (harness.Project, error) {
	if c.CloneProjectFn == nil {
		return harness.Project{}, nil
	}
	return c.CloneProjectFn(ctx, s, title, remoteURL, path)
}

func (c *Client) CreateProject(ctx context.Context, s harness.Session, title, path string) (harness.Project, error) {
	if c.CreateProjectFn == nil {
		return harness.Project{}, nil
	}
	return c.CreateProjectFn(ctx, s, title, path)
}

func (c *Client) CanClone(ctx context.Context, s harness.Session) (bool, error) {
	if c.CanCloneFn == nil {
		return true, nil
	}
	return c.CanCloneFn(ctx, s)
}

func (c *Client) ListProviders(ctx context.Context, s harness.Session) ([]harness.Provider, error) {
	if c.ListProvidersFn == nil {
		return nil, nil
	}
	return c.ListProvidersFn(ctx, s)
}

func (c *Client) Hold(ctx context.Context, s harness.Session) (harness.Conn, error) {
	if c.HoldFn == nil {
		return nil, nil
	}
	return c.HoldFn(ctx, s)
}

func (c *Client) StartTurn(ctx context.Context, t harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
	if c.StartTurnFn == nil {
		return harness.StartResult{}, nil
	}
	return c.StartTurnFn(ctx, t, title, prompts)
}

func (c *Client) Watch(ctx context.Context, t harness.Target) (harness.StartResult, error) {
	if c.WatchFn == nil {
		return harness.StartResult{}, nil
	}
	return c.WatchFn(ctx, t)
}

func (c *Client) Interrupt(ctx context.Context, t harness.Target) error {
	if c.InterruptFn == nil {
		return nil
	}
	return c.InterruptFn(ctx, t)
}

var _ harness.Client = (*Client)(nil)

func (c *Client) Answer(ctx context.Context, t harness.Target, requestID string, answer harness.QuestionAnswer) error {
	if c.AnswerFn == nil {
		return nil
	}
	return c.AnswerFn(ctx, t, requestID, answer)
}

func (c *Client) AnswerSecret(ctx context.Context, t harness.Target, requestID, secret string) error {
	if c.AnswerSecretFn == nil {
		return nil
	}
	return c.AnswerSecretFn(ctx, t, requestID, secret)
}

func (c *Client) Settle(ctx context.Context, t harness.Target) error {
	if c.SettleFn == nil {
		return nil
	}
	return c.SettleFn(ctx, t)
}

func (c *Client) Steer(ctx context.Context, t harness.Target) error {
	if c.SteerFn == nil {
		return nil
	}
	return c.SteerFn(ctx, t)
}
