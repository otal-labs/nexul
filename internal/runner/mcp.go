package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/logstore"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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

type machineResult struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	StackRoot        string    `json:"stack_root"`
	ReportedHostname string    `json:"reported_hostname,omitempty"`
	LastSeen         time.Time `json:"last_seen"`
	// Actions are what you hold on this machine: machines:see, machines:deploy, machines:write (Manage), machines:delete.
	Actions permissions.Set `json:"actions"`
	Runners []runnerResult  `json:"runners"`
	// AutomationsHosts are the automations hosts filed under this machine; their ids are host_delete's, as is the machine's own.
	AutomationsHosts []automationsHostResult `json:"automations_hosts"`
}

type automationsHostResult struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Connected bool      `json:"connected"`
	LastSeen  time.Time `json:"last_seen"`
}

// runnerResult and jobResult name a job's stack and machine in CONTEXT.md terms, not the wire's service and target.
type runnerResult struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Version    string     `json:"version"`
	Connected  bool       `json:"connected"`
	LastSeen   time.Time  `json:"last_seen"`
	RunningJob *jobResult `json:"running_job"`
}

type jobResult struct {
	ID      string      `json:"id"`
	Kind    RequestKind `json:"kind"`
	Stack   string      `json:"stack,omitempty"`
	Machine string      `json:"machine,omitempty"`
}

// machineList pages the machines; the queue and any runner not yet tied to a machine ride along unpaged.
type machineList struct {
	mcptool.Page[machineResult]
	UnassignedRunners []runnerResult `json:"unassigned_runners,omitempty"`
	// UnassignedAutomationsHosts are hosts filed under a machine no runner has joined.
	UnassignedAutomationsHosts []automationsHostResult `json:"unassigned_automations_hosts,omitempty"`
	// OpenEnrollments are the unused codes the caller may revoke; their ids are host_delete's code_id.
	OpenEnrollments []OpenEnrollment `json:"open_enrollments,omitempty"`
	Queue           []jobResult      `json:"queue"`
}

func machineListTool(s *Service) mcptool.Tool {
	return mcptool.New("machine_list", "List machines",
		"Lists the machines you can see, each with its stack root, its runners' connection state, "+
			"version, and running job, and its automations hosts, plus the deploys queued for a runner on them; a "+
			"build or deploy job's id is its "+
			"deploy's id for deploy_get. A machine you are not on is left out, as it is everywhere else. Use a machine's "+
			"name as stack_create's machine and its id for machine_discover, machine_import and, with kind machine, host_delete. open_enrollments lists "+
			"the unused enrollment codes you may revoke, never the codes themselves; host_delete revokes one by its id. "+
			"Returns at most 100 machines per page.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in machineListIn) (any, error) {
			machines, err := s.ListMachineViews(ctx)
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
			r, err := s.reach(ctx)
			if err != nil {
				return nil, err
			}
			hosts, err := s.listAutomationsHosts(ctx, r, machines)
			if err != nil {
				return nil, err
			}
			codes, err := s.ListEnrollments(ctx)
			if err != nil {
				return nil, err
			}
			results, unassigned := groupRunners(machines, runners)
			results, unassignedHosts := groupAutomationsHosts(results, hosts)
			jobs := make([]jobResult, 0, len(queue))
			for _, q := range queue {
				jobs = append(jobs, jobResult{ID: q.ID, Kind: q.Kind, Stack: q.Service, Machine: q.Target})
			}
			return machineList{
				Page: mcptool.Paginate(results, in.PageArgs), UnassignedRunners: unassigned,
				UnassignedAutomationsHosts: unassignedHosts, OpenEnrollments: codes, Queue: jobs,
			}, nil
		})
}

// groupRunners files each runner under its machine by name; a runner whose machine is unresolved is returned apart.
func groupRunners(machines []MachineView, runners []RunnerView) ([]machineResult, []runnerResult) {
	byName := make(map[string][]runnerResult, len(machines))
	var unassigned []runnerResult
	known := make(map[string]bool, len(machines))
	for _, m := range machines {
		known[m.Name] = true
	}
	for _, r := range runners {
		if !known[r.Machine] {
			unassigned = append(unassigned, toRunnerResult(r))
			continue
		}
		byName[r.Machine] = append(byName[r.Machine], toRunnerResult(r))
	}
	out := make([]machineResult, 0, len(machines))
	for _, m := range machines {
		rs := byName[m.Name]
		out = append(out, machineResult{
			ID: m.ID, Name: m.Name, StackRoot: m.StackRoot, ReportedHostname: m.ReportedHostname,
			LastSeen: m.LastSeen, Actions: m.Actions, Runners: rs,
		})
	}
	return out, unassigned
}

// groupAutomationsHosts files each host under the machine of its name; a host on no known machine is returned apart.
func groupAutomationsHosts(machines []machineResult, hosts []AutomationsHost) ([]machineResult, []automationsHostResult) {
	index := make(map[string]int, len(machines))
	for i := range machines {
		index[machines[i].Name] = i
		machines[i].AutomationsHosts = []automationsHostResult{}
	}
	var unassigned []automationsHostResult
	for _, h := range hosts {
		r := automationsHostResult{ID: h.ID, Name: h.Name, Version: h.Version, Connected: h.Connected, LastSeen: h.LastSeen}
		i, ok := index[h.Machine]
		if !ok {
			unassigned = append(unassigned, r)
			continue
		}
		machines[i].AutomationsHosts = append(machines[i].AutomationsHosts, r)
	}
	return machines, unassigned
}

func toRunnerResult(r RunnerView) runnerResult {
	out := runnerResult{ID: r.ID, Name: r.Name, Version: r.Version, Connected: r.Connected, LastSeen: r.LastSeen}
	if j := r.RunningJob; j != nil {
		out.RunningJob = &jobResult{ID: j.ID, Kind: j.Kind, Stack: j.Service}
	}
	return out
}

type machineDiscoverIn struct {
	ID string `json:"id" jsonschema:"The machine's id, from machine_list."`
}

func machineDiscoverTool(s *Service) mcptool.Tool {
	return mcptool.New("machine_discover", "Discover machine",
		"Asks an idle runner on the machine to list what is running there and returns the containers no stack "+
			"tracks yet, grouped by compose project, standalone containers, and recognised cloudflared gateways with "+
			"the hostnames their tunnels already route, plus the machine's docker networks. Use it before "+
			"machine_import and pass the names it returns. It needs Manage on the machine and answers not found for "+
			"one you can't see. It changes nothing and fails if no runner on the machine is connected and idle.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in machineDiscoverIn) (any, error) {
			grouped, err := s.DiscoverForImport(ctx, in.ID)
			if errors.Is(err, apperrs.ErrNotFound) {
				return nil, fmt.Errorf("%w; machine_list lists machines", err)
			}
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
			"and poll it afterwards to see the upgrade complete or fail; refresh asks GitHub for a release published "+
			"in the last few minutes. Pass logs to also read the instance's own logs, the server's, the browser's, "+
			"and the runners' and automations hosts', newest first, 100 lines by default and at most 500, in the "+
			"list shape with the until the read is pinned to: pass that until back with next_offset for the next "+
			"page. That needs logs:read besides instance:read, and never reaches a container's output, which "+
			"stack_get with logs reads. Needs instance:read in any workspace.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in instanceGetIn) (any, error) {
			if in.Refresh {
				s.RefreshReleases()
			}
			status, err := s.UpgradeStatus(ctx)
			if err != nil || in.Logs == nil {
				return status, err
			}
			page, err := s.InstanceLogs(ctx, *in.Logs)
			return instanceDetail{UpgradeStatus: status, Logs: &page}, err
		})
}

// instanceDetail is instance_get's answer: the upgrade facts plus the logs read when asked for.
type instanceDetail struct {
	UpgradeStatus
	Logs *logstore.Page `json:"logs,omitempty"`
}

func instanceUpgradeTool(s *Service) mcptool.Tool {
	return mcptool.New("instance_upgrade", "Upgrade instance",
		"Upgrades the instance to its channel's newest release, exactly as the settings page's Upgrade button "+
			"does: the instance runner starts `nexul upgrade` on the host, which downloads the new release's binaries and restarts every Nexul service, so the "+
			"server goes away for a while. It is refused with the reason when instance_get says an upgrade cannot "+
			"start. Returns the pending upgrade record; poll instance_get for the outcome. Needs instance:write in any workspace.",
		mcptool.Hints{},
		func(ctx context.Context, _ struct{}) (any, error) {
			return s.RequestUpgrade(ctx, mcpUpgradeActor(ctx))
		})
}

type instanceGetIn struct {
	Refresh bool             `json:"refresh,omitempty" jsonschema:"Skip the few-minute release cache and ask GitHub for the newest release now. Defaults to false."`
	Logs    *logstore.Params `json:"logs,omitzero" jsonschema:"Also read the instance's own logs, filtered by these fields; all of them are optional. Needs logs:read. Omit it for the upgrade facts alone."`
}

// mcpUpgradeActor is upgrade provenance for an MCP call (ADR 0049): "<user id>:mcp", empty with no actor.
func mcpUpgradeActor(ctx context.Context) string {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return ""
	}
	return a.ID + ":mcp"
}
