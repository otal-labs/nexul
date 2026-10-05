package main

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/topology"
)

// topologyScope answers which service nodes a workspace's canvas shows (ADR 0125): the containers of its projects'
// stacks, and those of every gateway routing a hostname to one of them. It reads the stores directly, since the
// person looking may hold topology:read without stacks:read or dns:read.
type topologyScope struct {
	stacks   *storage.StacksRepo
	services *storage.ServicesRepo
	dns      *storage.DNSRepo
}

var _ topology.Scope = topologyScope{}

func (t topologyScope) Services(ctx context.Context, workspaceID string) (map[string]bool, error) {
	ids := map[string]bool{}
	stacks, err := t.stacks.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, s := range stacks {
		if err := t.addStack(ctx, ids, s.ID); err != nil {
			return nil, err
		}
	}
	exposures, err := t.dns.ListExposures(ctx)
	if err != nil {
		return nil, err
	}
	routed := map[string]bool{}
	for _, e := range exposures {
		if e.ServiceID != "" && ids[e.ServiceID] {
			routed[e.GatewayID] = true
		}
	}
	gateways, err := t.dns.ListGateways(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range gateways {
		if !routed[g.ID] || g.ServiceID == "" {
			continue
		}
		if err := t.addStack(ctx, ids, g.ServiceID); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

// addStack adds a stack's id, which older canvases anchor a node to, and the ids of its containers.
func (t topologyScope) addStack(ctx context.Context, ids map[string]bool, stackID string) error {
	ids[stackID] = true
	containers, err := t.services.ListByStack(ctx, stackID)
	if err != nil {
		return fmt.Errorf("list containers of stack %s: %w", stackID, err)
	}
	for _, c := range containers {
		ids[c.ID] = true
	}
	return nil
}

// topologyFrame is the live push of one workspace's canvas, so a browser applies only the workspace it shows.
type topologyFrame struct {
	WorkspaceID string `json:"workspace_id"`
	topology.Canvas
}

// topologyLiveHandler pushes a workspace's changed canvas to its browsers. A registry change can reach every
// workspace's canvas, so each gets its own view pushed.
func topologyLiveHandler(topo *topology.Service, workspaces *storage.WorkspacesRepo, publish func(context.Context, string, any) error) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e topology.UpdatedEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", topology.TopicUpdated, err))
		}
		if e.WorkspaceID != "" {
			return publish(ctx, topicTopologyCanvas, topologyFrame{WorkspaceID: e.WorkspaceID, Canvas: e.Canvas})
		}
		all, err := workspaces.ListWithRoles(ctx)
		if err != nil {
			return err
		}
		for _, w := range all {
			c, err := topo.Get(ctx, w.ID)
			if err != nil {
				return err
			}
			if err := publish(ctx, topicTopologyCanvas, topologyFrame{WorkspaceID: w.ID, Canvas: *c}); err != nil {
				return err
			}
		}
		return nil
	}
}
