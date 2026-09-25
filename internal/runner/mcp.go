package runner

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the machine and instance tools: machines with their runners and queue, discovery, and upgrades.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		machineListTool(s),
		machineDiscoverTool(s),
		instanceGetTool(s),
		instanceUpgradeTool(s),
	}
}

type machineListIn struct {
	mcptool.PageArgs
}

// machineResult is a machine with the runners connected from it.
type machineResult struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	StackRoot        string       `json:"stack_root"`
	ReportedHostname string       `json:"reported_hostname,omitempty"`
	LastSeen         time.Time    `json:"last_seen"`
	Runners          []RunnerView `json:"runners"`
}

// machineList pages the machines; the queue and any runner not yet tied to a machine ride along unpaged.
type machineList struct {
	mcptool.Page[machineResult]
	UnassignedRunners []RunnerView `json:"unassigned_runners,omitempty"`
	Queue             []QueuedJob  `json:"queue"`
}

func machineListTool(s *Service) mcptool.Tool {
	return mcptool.New("machine_list", "List machines",
		"Lists the machines runners connect from, each with its stack root and its runners' connection state, "+
			"version, and running job, plus the deploys queued for a runner. Use a machine's name as stack_create's "+
			"machine and its id for machine_discover and machine_import. Returns at most 100 machines per page.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in machineListIn) (any, error) {
			machines, err := s.ListMachines(ctx)
			if err != nil {
				return nil, err
			}
			runners, err := s.ListRunners(ctx)
			if err != nil {
				return nil, err
			}
			queue, err := s.ListQueue(ctx)
			if err != nil {
				return nil, err
			}
			results, unassigned := groupRunners(machines, runners)
			if queue == nil {
				queue = []QueuedJob{}
			}
			return machineList{Page: mcptool.Paginate(results, in.PageArgs), UnassignedRunners: unassigned, Queue: queue}, nil
		})
}

// groupRunners files each runner under its machine by name; a runner whose machine is unresolved is returned apart.
func groupRunners(machines []*Machine, runners []RunnerView) ([]machineResult, []RunnerView) {
	byName := make(map[string][]RunnerView, len(machines))
	var unassigned []RunnerView
	known := make(map[string]bool, len(machines))
	for _, m := range machines {
		known[m.Name] = true
	}
	for _, r := range runners {
		if !known[r.Machine] {
			unassigned = append(unassigned, r)
			continue
		}
		byName[r.Machine] = append(byName[r.Machine], r)
	}
	out := make([]machineResult, 0, len(machines))
	for _, m := range machines {
		rs := byName[m.Name]
		if rs == nil {
			rs = []RunnerView{}
		}
		out = append(out, machineResult{
			ID: m.ID, Name: m.Name, StackRoot: m.StackRoot, ReportedHostname: m.ReportedHostname,
			LastSeen: m.LastSeen, Runners: rs,
		})
	}
	return out, unassigned
}

type machineDiscoverIn struct {
	ID string `json:"id" jsonschema:"The machine's id, from machine_list."`
}

func machineDiscoverTool(s *Service) mcptool.Tool {
	return mcptool.New("machine_discover", "Discover machine",
		"Asks an idle runner on the machine to list what is running there and returns the containers no stack "+
			"tracks yet, grouped by compose project, standalone containers, and recognised cloudflared gateways with "+
			"the hostnames their tunnels already route, plus the machine's docker networks. Use it before "+
			"machine_import and pass the names it returns. It changes nothing and fails if no runner on the machine "+
			"is connected and idle.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in machineDiscoverIn) (any, error) {
			grouped, err := s.DiscoverForImport(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			return withoutLabels(grouped), nil
		})
}

// withoutLabels drops docker labels, compose and image metadata the choice of what to import never needs.
func withoutLabels(g GroupedDiscovery) GroupedDiscovery {
	strip := func(cs []DiscoveredContainer) {
		for i := range cs {
			cs[i].Labels = nil
		}
	}
	for _, st := range g.Stacks {
		strip(st.Containers)
	}
	strip(g.Standalone)
	strip(g.Gateways)
	return g
}

func instanceGetTool(s *Service) mcptool.Tool {
	return mcptool.New("instance_get", "Get instance",
		"Returns the instance's running version and release channel, the channel's newest release, whether an "+
			"upgrade can start now and if not why, and the latest upgrade record. Check it before instance_upgrade "+
			"and poll it afterwards to see the upgrade complete or fail. Instance admins only.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, _ struct{}) (any, error) {
			return s.UpgradeStatus(ctx)
		})
}

func instanceUpgradeTool(s *Service) mcptool.Tool {
	return mcptool.New("instance_upgrade", "Upgrade instance",
		"Upgrades the instance to its channel's newest release, exactly as the settings page's Upgrade button "+
			"does: the bundled instance runner starts a helper that pulls the new images and restarts Nexul, so the "+
			"server goes away for a while. It is refused with the reason when instance_get says an upgrade cannot "+
			"start. Returns the pending upgrade record; poll instance_get for the outcome. Instance admins only.",
		mcptool.Hints{},
		func(ctx context.Context, _ struct{}) (any, error) {
			return s.RequestUpgrade(ctx, mcpUpgradeActor(ctx))
		})
}

// mcpUpgradeActor is upgrade provenance for an MCP call (ADR 0049): "<user id>:mcp", empty with no actor.
func mcpUpgradeActor(ctx context.Context) string {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return ""
	}
	return a.ID + ":mcp"
}
