package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/templates"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

// wireTemplates builds the instance layer over the four template kinds and points each owning domain at it
// (ADR 0103); the domains seed and resolve, the templates domain only stores the instance text and clones.
func wireTemplates(store *storage.Store, gate templates.Gate, mem *memories.Service, ten *tenancy.Service, pl *plays.Service, ws *workspace.Service) *templates.Service {
	builtins := plays.Builtins()
	playDefaults := make([]templates.Default, len(builtins))
	for i, b := range builtins {
		playDefaults[i] = templates.Default{Key: b.Key, Name: b.Label, Body: b.Instructions}
	}
	typeDefaults := make([]templates.Default, len(workspace.DefaultTicketTypes))
	for i, t := range workspace.DefaultTicketTypes {
		typeDefaults[i] = templates.Default{Key: t.Name, Name: t.Name, Body: t.BodyTemplate}
	}
	svc := templates.NewService(store.InstanceTemplates, gate,
		templates.Kind{
			Name: memories.TemplateKind, Below: templates.ScopeWorkspace, Follows: true,
			Defaults: []templates.Default{{Name: "Interview", Body: memories.DefaultInterviewTemplate}},
			Check:    memories.CheckInterviewTemplate, Layer: interviewLayer{svc: mem},
		},
		templates.Kind{
			Name: tenancy.TemplateKind, Below: templates.ScopeWorkspace, Follows: true,
			Defaults: []templates.Default{{Name: "Mention chip", Body: tenancy.DefaultMentionChipTemplate}},
			Check:    checkChipTemplate, Layer: chipLayer{svc: ten},
		},
		templates.Kind{Name: plays.TemplateKind, Below: templates.ScopeWorkspace, Defaults: playDefaults, Layer: playLayer{svc: pl}},
		templates.Kind{Name: workspace.TemplateKind, Below: templates.ScopeProject, Defaults: typeDefaults, Layer: ticketBodyLayer{svc: ws}},
	)
	mem.SetInstanceTemplates(svc)
	ten.SetInstanceTemplates(svc)
	pl.SetInstanceTemplates(svc)
	ws.SetInstanceTemplates(svc)
	return svc
}

func checkChipTemplate(body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("%w: a mention chip template needs at least one {ticket.Field} token, such as {ticket.Ticket}", apperrs.ErrInvalid)
	}
	return nil
}

// interviewLayer is a workspace's Interview template, stored only once edited.
type interviewLayer struct{ svc *memories.Service }

func (l interviewLayer) Read(ctx context.Context, at templates.Location, _ string) (string, bool, error) {
	t, err := l.svc.InterviewTemplate(ctx, at.WorkspaceID)
	if err != nil {
		return "", false, err
	}
	return t.Body, t.Edited, nil
}

func (l interviewLayer) Write(ctx context.Context, at templates.Location, _, body string) error {
	_, err := l.svc.SaveInterviewTemplate(ctx, at.WorkspaceID, body)
	return err
}

func (l interviewLayer) Reset(ctx context.Context, at templates.Location, _, _ string) error {
	_, err := l.svc.ResetInterviewTemplate(ctx, at.WorkspaceID)
	return err
}

// chipLayer is a workspace's mention chip template; empty means it follows the instance's.
type chipLayer struct{ svc *tenancy.Service }

func (l chipLayer) Read(ctx context.Context, at templates.Location, _ string) (string, bool, error) {
	return l.svc.MentionChipTemplate(ctx, actorID(ctx), at.WorkspaceID)
}

func (l chipLayer) Write(ctx context.Context, at templates.Location, _, body string) error {
	if err := checkChipTemplate(body); err != nil {
		return err
	}
	_, err := l.svc.SetMentionChipTemplate(ctx, actorID(ctx), at.WorkspaceID, body)
	return err
}

func (l chipLayer) Reset(ctx context.Context, at templates.Location, _, _ string) error {
	_, err := l.svc.SetMentionChipTemplate(ctx, actorID(ctx), at.WorkspaceID, "")
	return err
}

// playLayer is a workspace's copy of a built-in play, found by its key.
type playLayer struct{ svc *plays.Service }

func (l playLayer) Read(ctx context.Context, at templates.Location, key string) (string, bool, error) {
	p, err := l.svc.BuiltinPlay(ctx, at.WorkspaceID, key)
	if err != nil {
		return "", false, err
	}
	return p.Instructions, true, nil
}

func (l playLayer) Write(ctx context.Context, at templates.Location, key, body string) error {
	_, err := l.svc.SetBuiltinInstructions(ctx, at.WorkspaceID, key, body)
	return err
}

func (l playLayer) Reset(ctx context.Context, at templates.Location, key, instanceBody string) error {
	return l.Write(ctx, at, key, instanceBody)
}

// ticketBodyLayer is a project's ticket type body template, found by the type's name.
type ticketBodyLayer struct{ svc *workspace.Service }

func (l ticketBodyLayer) Read(ctx context.Context, at templates.Location, key string) (string, bool, error) {
	t, err := l.svc.TicketTypeNamed(ctx, at.ProjectID, key)
	if err != nil {
		return "", false, err
	}
	return t.BodyTemplate, true, nil
}

func (l ticketBodyLayer) Write(ctx context.Context, at templates.Location, key, body string) error {
	t, err := l.svc.TicketTypeNamed(ctx, at.ProjectID, key)
	if err != nil {
		return err
	}
	_, err = l.svc.SetTicketTypeTemplate(ctx, actorID(ctx), t.ID, body)
	return err
}

func (l ticketBodyLayer) Reset(ctx context.Context, at templates.Location, key, instanceBody string) error {
	return l.Write(ctx, at, key, instanceBody)
}
