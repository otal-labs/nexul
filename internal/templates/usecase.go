package templates

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Service is the templates use-case layer: the instance layer itself, and clone and reset across every layer.
type Service struct {
	repo  Repo
	gate  Gate
	kinds []*Kind
	now   func() time.Time
}

// NewService wires the instance templates over their repo and the instance-level permission gate.
func NewService(repo Repo, gate Gate, kinds ...Kind) *Service {
	s := &Service{repo: repo, gate: gate, now: time.Now}
	for i := range kinds {
		s.kinds = append(s.kinds, &kinds[i])
	}
	return s
}

// List returns every instance template in registration order, the code default where nobody has edited one.
func (s *Service) List(ctx context.Context) ([]*Template, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list instance templates: %w", err)
	}
	stored := map[string]*Record{}
	for _, r := range records {
		stored[r.Kind+"\x00"+r.Key] = r
	}
	var out []*Template
	for _, k := range s.kinds {
		for _, d := range k.Defaults {
			out = append(out, instanceView(k, d, stored[k.Name+"\x00"+d.Key]))
		}
	}
	return out, nil
}

// Effective is the instance's text for kind and key, the code default until edited; the server's own read, unchecked.
func (s *Service) Effective(ctx context.Context, kind, key string) (string, error) {
	k, err := s.kind(kind)
	if err != nil {
		return "", err
	}
	d, err := instanceDefault(k, key)
	if err != nil {
		return "", err
	}
	rec, err := s.record(ctx, k.Name, d.Key)
	if err != nil {
		return "", err
	}
	return instanceView(k, d, rec).Body, nil
}

// Get returns the template at a location; below the instance it is read through the owning domain's permissions.
func (s *Service) Get(ctx context.Context, kind, key string, at Location) (*Template, error) {
	k, err := s.kind(kind)
	if err != nil {
		return nil, err
	}
	if err := checkLocation(k, at); err != nil {
		return nil, err
	}
	return s.view(ctx, k, key, at)
}

// Update replaces the template's text at a location; the instance takes templates:write, a lower layer its own write.
func (s *Service) Update(ctx context.Context, kind, key string, at Location, body string) (*Template, error) {
	k, err := s.kind(kind)
	if err != nil {
		return nil, err
	}
	if err := checkLocation(k, at); err != nil {
		return nil, err
	}
	if err := s.write(ctx, k, key, at, body); err != nil {
		return nil, err
	}
	return s.view(ctx, k, key, at)
}

// Reset returns a location to its default: the instance to the code default, a lower layer to the instance's text.
func (s *Service) Reset(ctx context.Context, kind, key string, at Location) (*Template, error) {
	k, err := s.kind(kind)
	if err != nil {
		return nil, err
	}
	if err := checkLocation(k, at); err != nil {
		return nil, err
	}
	if at.Scope == ScopeInstance {
		if err := s.resetInstance(ctx, k, key); err != nil {
			return nil, err
		}
		return s.view(ctx, k, key, at)
	}
	instanceBody, err := s.Effective(ctx, k.Name, key)
	if err != nil {
		return nil, err
	}
	if err := k.Layer.Reset(ctx, at, key, instanceBody); err != nil {
		return nil, err
	}
	return s.view(ctx, k, key, at)
}

// Clone copies the template's text from one location over another's; reading takes read where the source lives and
// writing the target's own write, or templates:write for the instance.
func (s *Service) Clone(ctx context.Context, kind, key string, from, to Location) (*Template, error) {
	k, err := s.kind(kind)
	if err != nil {
		return nil, err
	}
	if err := checkLocation(k, from); err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	if err := checkLocation(k, to); err != nil {
		return nil, fmt.Errorf("target: %w", err)
	}
	if from == to {
		return nil, fmt.Errorf("%w: the source and the target are the same place", apperrs.ErrInvalid)
	}
	body, err := s.read(ctx, k, key, from)
	if err != nil {
		return nil, err
	}
	if err := s.write(ctx, k, key, to, body); err != nil {
		return nil, err
	}
	return s.view(ctx, k, key, to)
}

func (s *Service) read(ctx context.Context, k *Kind, key string, at Location) (string, error) {
	if at.Scope == ScopeInstance {
		return s.Effective(ctx, k.Name, key)
	}
	body, _, err := k.Layer.Read(ctx, at, key)
	return body, err
}

func (s *Service) write(ctx context.Context, k *Kind, key string, at Location, body string) error {
	if at.Scope != ScopeInstance {
		return k.Layer.Write(ctx, at, key, body)
	}
	d, err := instanceDefault(k, key)
	if err != nil {
		return err
	}
	if err := s.requireWrite(ctx); err != nil {
		return err
	}
	if k.Check != nil {
		if err := k.Check(body); err != nil {
			return err
		}
	}
	rec := &Record{Kind: k.Name, Key: d.Key, Body: body, UpdatedBy: actorID(ctx), UpdatedAt: s.now().UTC()}
	if err := s.repo.Save(ctx, rec, s.event(rec.Kind, rec.Key, rec.UpdatedBy, rec.UpdatedAt, false)); err != nil {
		return fmt.Errorf("save instance template %s %s: %w", k.Name, d.Key, err)
	}
	return nil
}

func (s *Service) resetInstance(ctx context.Context, k *Kind, key string) error {
	d, err := instanceDefault(k, key)
	if err != nil {
		return err
	}
	if err := s.requireWrite(ctx); err != nil {
		return err
	}
	evt := s.event(k.Name, d.Key, actorID(ctx), s.now().UTC(), true)
	if err := s.repo.Delete(ctx, k.Name, d.Key, evt); err != nil {
		return fmt.Errorf("reset instance template %s %s: %w", k.Name, d.Key, err)
	}
	return nil
}

func (s *Service) view(ctx context.Context, k *Kind, key string, at Location) (*Template, error) {
	if at.Scope == ScopeInstance {
		d, err := instanceDefault(k, key)
		if err != nil {
			return nil, err
		}
		rec, err := s.record(ctx, k.Name, d.Key)
		if err != nil {
			return nil, err
		}
		return instanceView(k, d, rec), nil
	}
	body, own, err := k.Layer.Read(ctx, at, key)
	if err != nil {
		return nil, err
	}
	t := &Template{Kind: k.Name, Key: key, Name: key, Scope: at.Scope, WorkspaceID: at.WorkspaceID, ProjectID: at.ProjectID, Body: body, Follows: k.Follows}
	if d, err := instanceDefault(k, key); err == nil {
		t.Key, t.Name = d.Key, d.Name
		if t.DefaultBody, err = s.Effective(ctx, k.Name, d.Key); err != nil {
			return nil, err
		}
	}
	t.Edited = own
	if !k.Follows {
		t.Edited = body != t.DefaultBody
	}
	return withQuestions(k, t), nil
}

func withQuestions(k *Kind, t *Template) *Template {
	if k.Questions != nil {
		t.Questions = k.Questions(t.Body)
	}
	return t
}

func (s *Service) record(ctx context.Context, kind, key string) (*Record, error) {
	rec, err := s.repo.Get(ctx, kind, key)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get instance template %s %s: %w", kind, key, err)
	}
	return rec, nil
}

func (s *Service) kind(name string) (*Kind, error) {
	names := make([]string, len(s.kinds))
	for i, k := range s.kinds {
		if k.Name == strings.TrimSpace(name) {
			return k, nil
		}
		names[i] = k.Name
	}
	return nil, fmt.Errorf("%w: kind %q is not a template; use one of %s", apperrs.ErrInvalid, name, strings.Join(names, ", "))
}

func (s *Service) requireWrite(ctx context.Context) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.RequireAnywhere(ctx, permissions.TemplatesWrite)
}

func (s *Service) event(kind, key, authorID string, at time.Time, reset bool) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Kind: kind, Key: key, AuthorID: authorID, UpdatedAt: at, Reset: reset}}
}

// instanceDefault matches key against the kind's instance keys, ignoring case, so a project's "Bug" type finds "bug".
func instanceDefault(k *Kind, key string) (Default, error) {
	key = strings.TrimSpace(key)
	keys := make([]string, len(k.Defaults))
	for i, d := range k.Defaults {
		if strings.EqualFold(d.Key, key) {
			return d, nil
		}
		keys[i] = d.Key
	}
	if len(k.Defaults) == 1 && k.Defaults[0].Key == "" {
		return Default{}, fmt.Errorf("%w: %s templates have no key; leave it empty", apperrs.ErrInvalid, k.Name)
	}
	return Default{}, fmt.Errorf("%w: the instance has no %s template %q to match; its keys are %s", apperrs.ErrNotFound, k.Name, key, strings.Join(keys, ", "))
}

func instanceView(k *Kind, d Default, rec *Record) *Template {
	t := &Template{Kind: k.Name, Key: d.Key, Name: d.Name, Scope: ScopeInstance, Body: d.Body, DefaultBody: d.Body, Follows: k.Follows}
	if rec == nil {
		return withQuestions(k, t)
	}
	at := rec.UpdatedAt
	t.Body, t.Edited, t.UpdatedBy, t.UpdatedAt = rec.Body, true, rec.UpdatedBy, &at
	return withQuestions(k, t)
}

func checkLocation(k *Kind, at Location) error {
	switch {
	case at.Scope == ScopeInstance:
		return nil
	case k.Below == "":
		return fmt.Errorf("%w: %s templates live only at the instance", apperrs.ErrInvalid, k.Name)
	case at.Scope != k.Below:
		return fmt.Errorf("%w: %s templates live at the instance or a %s, not %q", apperrs.ErrInvalid, k.Name, k.Below, at.Scope)
	case at.Scope == ScopeWorkspace && strings.TrimSpace(at.WorkspaceID) == "":
		return fmt.Errorf("%w: a workspace location needs workspace_id", apperrs.ErrInvalid)
	case at.Scope == ScopeProject && strings.TrimSpace(at.ProjectID) == "":
		return fmt.Errorf("%w: a project location needs project_id", apperrs.ErrInvalid)
	}
	return nil
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
}
