package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/topology"
)

// topologyStackCreatedHandler anchors one topology node per container the event carries — pending ones for a
// stack the wizard just created, observed ones for containers adopted by machine import — with the row's own
// status and address; a redelivered event on an already-present node is a no-op.
func topologyStackCreatedHandler(topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e deploy.StackEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicStackCreated, err))
		}
		for _, svc := range e.Services {
			name := containerNodeName(e.Stack.Name, svc.Name, len(e.Services))
			status, address := containerNodeStatus(&svc, deploy.StatusPending)
			_, err := topoSvc.AddServiceNode(ctx, topology.Node{
				ID: svc.ID,
				Data: topology.NodeData{
					Name:    name,
					Runtime: "docker",
					Status:  status,
					Address: address,
				},
			})
			if err != nil && !errors.Is(err, apperrs.ErrConflict) {
				return err
			}
		}
		return nil
	}
}

// topologyStackUpdatedHandler propagates a stack rename onto every one of its auto-managed container nodes.
func topologyStackUpdatedHandler(deploySvc *deploy.Service, topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e deploy.StackEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicStackUpdated, err))
		}
		services, err := deploySvc.ListServices(ctx, e.Stack.ID)
		if err != nil {
			return err
		}
		for _, svc := range services {
			name := containerNodeName(e.Stack.Name, svc.Name, len(services))
			if _, err := topoSvc.RenameServiceNode(ctx, svc.ID, name); err != nil {
				return err
			}
		}
		return nil
	}
}

// topologyStackDeletedHandler drops every one of the deleted stack's container nodes/edges; a redelivered
// event on an already-missing node is a no-op.
func topologyStackDeletedHandler(topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e deploy.StackDeletedEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicStackDeleted, err))
		}
		for _, id := range e.ServiceIDs {
			if _, err := topoSvc.RemoveServiceNode(ctx, id); err != nil && !errors.Is(err, apperrs.ErrNotFound) {
				return err
			}
		}
		return nil
	}
}

// topologyDeployStatusHandler lands the deploying stack's containers on the canvas. HandleStatusChanged reconciles the
// runner's observation report into the services table first, so a container the deploy first observed gets its node,
// one it dropped loses it, and each badge shows its own container's status and address; a container the report never
// reached (e.g. a failed deploy with no report at all) falls back to the deploy's own terminal status.
func topologyDeployStatusHandler(deploySvc *deploy.Service, topoSvc *topology.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e deploy.DeployStatusChangedEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", deploy.TopicDeployStatusChanged, err))
		}
		d, err := deploySvc.Get(ctx, e.ID)
		if err != nil {
			return err
		}
		before, err := deploySvc.ListServices(ctx, d.StackID)
		if err != nil {
			return err
		}
		if err := deploySvc.HandleStatusChanged(ctx, ev); err != nil {
			return err
		}
		if d, err = deploySvc.Get(ctx, e.ID); err != nil {
			return err
		}
		stack, err := deploySvc.GetStack(ctx, d.StackID)
		if err != nil {
			return err
		}
		services, err := deploySvc.ListServices(ctx, d.StackID)
		if err != nil {
			return err
		}
		kept := make(map[string]bool, len(services))
		nodes := make([]topology.Node, 0, len(services))
		for _, svc := range services {
			kept[svc.ID] = true
			status, address := containerNodeStatus(svc, deploy.Status(d.Status))
			nodes = append(nodes, topology.Node{ID: svc.ID, Data: topology.NodeData{
				Name: containerNodeName(stack.Name, svc.Name, len(services)), Runtime: "docker", Status: status, Address: address,
			}})
		}
		var dropped []string
		for _, svc := range before {
			if !kept[svc.ID] {
				dropped = append(dropped, svc.ID)
			}
		}
		_, err = topoSvc.SyncServiceNodes(ctx, nodes, dropped)
		return err
	}
}

// containerNodeName is a container's canvas label: the stack's name alone, or stack/service once it runs several.
func containerNodeName(stack, service string, containers int) string {
	if containers > 1 {
		return stack + "/" + service
	}
	return stack
}

// containerNodeStatus maps one observed container onto its node's badge and address.
func containerNodeStatus(svc *deploy.Container, deployStatus deploy.Status) (topology.ServiceStatus, string) {
	address := ""
	if len(svc.Networks) > 0 {
		address = svc.Networks[0].Address
	}
	switch svc.Status {
	case deploy.ServiceStatusHealthy:
		return topology.ServiceHealthy, address
	case deploy.ServiceStatusRunning:
		return topology.ServiceRunning, address
	case deploy.ServiceStatusStopped:
		return topology.ServiceStopped, address
	case deploy.ServiceStatusExited:
		return topology.ServiceFailed, address
	default: // pending: the report never reached this container, so reflect the deploy's own terminal status.
		if deployStatus == deploy.StatusFailed {
			return topology.ServiceFailed, address
		}
		return topology.ServiceRunning, address
	}
}
