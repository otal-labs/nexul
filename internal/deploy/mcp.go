package deploy

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// defaultLogLines and maxLogLines bound the log tail deploy_list returns for one deploy, so a long build never floods the context.
const (
	defaultLogLines = 200
	maxLogLines     = 1000
)

// recentDeploys is how many of a stack's newest deploys stack_get carries.
const recentDeploys = 10

// MCPTools returns the deploy tools: deploy history, stack management, starting deploys, and machine import.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		deployListTool(s),
		deployCancelTool(s),
		stackListTool(s),
		stackGetTool(s),
		stackCreateTool(s),
		stackUpdateTool(s),
		stackDeleteTool(s),
		stackDeployTool(s),
		machineImportTool(s),
	}
}

type deployListIn struct {
	ID       string `json:"id,omitempty" jsonschema:"One deploy, from this list or stack_deploy's result, returned with its log instead of a page."`
	LogLines int    `json:"log_lines,omitzero" jsonschema:"With an id, how many of the log's last lines to return, 1 to 1000, default 200."`
	StackID  string `json:"stack_id,omitempty" jsonschema:"Only this stack's deploys, by its id from stack_list."`
	Status   string `json:"status,omitempty" jsonschema:"Only deploys in this state: pending, running, healthy, or failed."`
	mcptool.PageArgs
}

// deployDetail is a deploy with the tail of its log; log_total is the full log's length, so a cut tail shows.
type deployDetail struct {
	deployResult
	Log      []LogLine `json:"log"`
	LogTotal int       `json:"log_total"`
}

func deployListTool(s *Service) mcptool.Tool {
	return mcptool.New("deploy_list", "List deploys",
		"Lists deploys newest first, filtered by stack_id or status. With an id it returns that one deploy with what "+
			"triggered it and the tail of its log (log_total is the full length). Poll it with the id stack_deploy "+
			"returned until the status is healthy or failed, and read the log to diagnose a failure.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in deployListIn) (any, error) {
			if in.ID != "" {
				return deployDetailOf(ctx, s, in.ID, in.LogLines)
			}
			ds, total, err := s.PageDeploys(ctx, DeployFilter{StackID: in.StackID, Status: Status(in.Status)}, in.Window())
			if errors.Is(err, apperrs.ErrInvalid) {
				return nil, fmt.Errorf("%w; status is one of pending, running, healthy, failed", err)
			}
			if err != nil {
				return nil, err
			}
			return mcptool.PageOf(toDeployResults(ds), total, in.Window()), nil
		})
}

func deployDetailOf(ctx context.Context, s *Service, id string, logLines int) (deployDetail, error) {
	d, err := s.Get(ctx, id)
	if err != nil {
		return deployDetail{}, withHint(err, "deploy_list without an id lists deploys")
	}
	lines, err := s.Log(ctx, id)
	if err != nil {
		return deployDetail{}, err
	}
	n := logLines
	if n < 1 {
		n = defaultLogLines
	}
	n = min(n, maxLogLines)
	return deployDetail{deployResult: toDeployResult(d), Log: lines[max(0, len(lines)-n):], LogTotal: len(lines)}, nil
}

type deployCancelIn struct {
	ID string `json:"id" jsonschema:"The deploy's id, from deploy_list."`
}

func deployCancelTool(s *Service) mcptool.Tool {
	return mcptool.New("deploy_cancel", "Cancel deploy",
		"Asks the runner to stop a pending or running deploy; it then lands failed with reason cancelled. Poll "+
			"deploy_list with its id to see it land. A finished deploy cannot be cancelled.",
		mcptool.Hints{},
		func(ctx context.Context, in deployCancelIn) (any, error) {
			if err := s.Cancel(ctx, in.ID); err != nil {
				return nil, withHint(err, "deploy_list lists deploys")
			}
			return map[string]string{"id": in.ID, "status": "cancelling"}, nil
		})
}

type stackListIn struct {
	ProjectID   string `json:"project_id,omitempty" jsonschema:"Only this project's stacks, by the project's id from project_list. Omit for every stack, the instance's own gateways included."`
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"Only the stacks of this workspace's projects, by its id from workspace_list; project_id wins when both are given."`
	mcptool.PageArgs
}

func stackListTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_list", "List stacks",
		"Lists stacks, the deployable units of a project, with their machine, strategy, and repository. Branch "+
			"deployments appear under their base stack in stack_get, not here. Use stack_get for a stack's services and "+
			"recent deploys.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in stackListIn) (any, error) {
			stacks, err := s.ListScopedStacks(ctx, in.ProjectID, in.WorkspaceID)
			if err != nil {
				return nil, err
			}
			return mcptool.Paginate(toStackSummaries(stacks), in.PageArgs), nil
		})
}

type stackGetIn struct {
	ID   string       `json:"id" jsonschema:"The stack's id, from stack_list."`
	Logs *stackLogsIn `json:"logs,omitzero" jsonschema:"Also return the last lines a service's container printed. Needs stacks:logs."`
}

type stackLogsIn struct {
	Service string `json:"service" jsonschema:"The service's name, from this stack's services."`
	Lines   int    `json:"lines,omitzero" jsonschema:"How many of the last lines to return, 1 to 1000. Defaults to 200."`
}

// stackDetail is stack_get's answer: the stack plus what an agent reads next about it.
type stackDetail struct {
	stackResult
	Services    []*Container    `json:"services"`
	OpenOutages []outageResult  `json:"open_outages"`
	Requests    *requestsResult `json:"requests,omitempty"`
	// OpenErrorRateEpisode is set while the stack's 5xx share is over its rule; ErrorRateRecoveredAt, when the last
	// episode ended within the hour; ErrorRateUnavailable, why the rule cannot watch this stack.
	OpenErrorRateEpisode *errorRateResult `json:"open_error_rate_episode,omitempty"`
	ErrorRateRecoveredAt *time.Time       `json:"error_rate_recovered_at,omitempty"`
	ErrorRateUnavailable string           `json:"error_rate_unavailable,omitempty"`
	Watching             bool             `json:"watching"`
	RecentDeploys        []deployResult   `json:"recent_deploys"`
	BranchDeployments    []stackSummary   `json:"branch_deployments,omitempty"`
	Logs                 *serviceLogs     `json:"logs,omitempty"`
}

// errorRateResult is an open error-rate episode: since when, and the worst window it has seen.
type errorRateResult struct {
	Since         time.Time `json:"since"`
	PeakPercent   float64   `json:"peak_percent"`
	Requests      int64     `json:"requests"`
	Errors5xx     int64     `json:"errors_5xx"`
	LimitPercent  float64   `json:"limit_percent"`
	WindowMinutes int       `json:"window_minutes"`
}

func toErrorRateResult(e *ErrorRateEpisode) *errorRateResult {
	if e == nil {
		return nil
	}
	return &errorRateResult{Since: e.StartedAt, PeakPercent: e.PeakPercent, Requests: e.Requests, Errors5xx: e.Errors, LimitPercent: e.LimitPercent, WindowMinutes: e.WindowMinutes}
}

// outageResult is a service that is down: since when, how it last exited, and how often it crashed again.
type outageResult struct {
	Service      string    `json:"service"`
	Since        time.Time `json:"since"`
	ExitCode     int       `json:"exit_code"`
	OOM          bool      `json:"oom"`
	RestartCount int       `json:"restart_count"`
}

func toOutageResults(open []*Outage) []outageResult {
	out := make([]outageResult, 0, len(open))
	for _, o := range open {
		out = append(out, outageResult{Service: o.Service, Since: o.StartedAt, ExitCode: o.ExitCode, OOM: o.OOM, RestartCount: o.RestartCount})
	}
	return out
}

// requestsResult is a stack's last hour of requests for an agent: one entry per minute from From, null for a minute
// nobody could count, with the totals of the counted ones.
type requestsResult struct {
	From           time.Time `json:"from"`
	Requests       []*int64  `json:"requests_per_minute"`
	Errors5xx      []*int64  `json:"errors_5xx_per_minute"`
	TotalRequests  int64     `json:"total_requests"`
	Total5xx       int64     `json:"total_5xx"`
	MissingMinutes int       `json:"missing_minutes"`
}

func toRequestsResult(w *RequestWindow) *requestsResult {
	if !w.Counted {
		return nil
	}
	out := &requestsResult{From: w.From, Requests: w.Requests, Errors5xx: w.Errors5xx}
	for i, r := range w.Requests {
		if r == nil {
			out.MissingMinutes++
			continue
		}
		out.TotalRequests += *r
		out.Total5xx += *w.Errors5xx[i]
	}
	return out
}

type serviceLogs struct {
	Service string             `json:"service"`
	Lines   []ContainerLogLine `json:"lines"`
}

func stackGetTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_get", "Get stack",
		"Returns one stack's definition, its services, open_outages (services down now), watching, the last hour's "+
			"requests for a stack with a hostname, the error_rate rule and its episode fields, its ten latest deploys, "+
			"and its branch deployments. Environment variables are listed by key only; values never leave the server. "+
			"Pass logs with a service name to read that container's last lines (200 by default, at most 1000, each cut at "+
			"2000 characters, environment values masked); that needs a connected runner on the stack's machine. Use it "+
			"before stack_update or stack_deploy; deploy_list has older deploys.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in stackGetIn) (any, error) {
			stack, err := s.GetStack(ctx, in.ID)
			if err != nil {
				return nil, withHint(err, "stack_list lists stacks")
			}
			detail, err := stackDetailFor(ctx, s, stack)
			if err != nil || in.Logs == nil {
				return detail, err
			}
			detail.Logs, err = serviceLogsFor(ctx, s, stack, *in.Logs)
			return detail, err
		})
}

func serviceLogsFor(ctx context.Context, s *Service, stack *Stack, in stackLogsIn) (*serviceLogs, error) {
	lines := in.Lines
	if lines < 1 {
		lines = DefaultLogTail
	}
	got, err := s.ServiceLogSnapshot(ctx, stack.ID, in.Service, min(lines, maxLogLines))
	if errors.Is(err, apperrs.ErrRetryable) {
		// Retryable is hidden from the model as an internal error; the offline case is the one it can act on.
		return nil, fmt.Errorf("%w: the logs of %s are unavailable, the runner on machine %q is offline or did not answer; retry once it reconnects",
			apperrs.ErrConflict, in.Service, stack.Machine)
	}
	if err != nil {
		return nil, err
	}
	containerlog.CutForAgent(got)
	return &serviceLogs{Service: in.Service, Lines: got}, nil
}

func stackDetailFor(ctx context.Context, s *Service, stack *Stack) (stackDetail, error) {
	services, err := s.ListServices(ctx, stack.ID)
	if err != nil {
		return stackDetail{}, err
	}
	open, err := s.OpenOutages(ctx, stack.ID)
	if err != nil {
		return stackDetail{}, err
	}
	deploys, err := s.ListByStackID(ctx, stack.ID)
	if err != nil {
		return stackDetail{}, err
	}
	out := stackDetail{
		stackResult:   toStackResult(stack),
		Services:      services,
		OpenOutages:   toOutageResults(open),
		RecentDeploys: toDeployResults(deploys[:min(len(deploys), recentDeploys)]),
	}
	if out.Watching, err = s.Watching(ctx, stack.ID); err != nil {
		return stackDetail{}, err
	}
	window, err := s.RequestWindow(ctx, stack)
	if err != nil {
		return stackDetail{}, err
	}
	out.Requests = toRequestsResult(window)
	out.ErrorRateUnavailable = window.ErrorRateUnavailable
	episode, recovered, err := s.ErrorRateState(ctx, stack)
	if err != nil {
		return stackDetail{}, err
	}
	out.OpenErrorRateEpisode, out.ErrorRateRecoveredAt = toErrorRateResult(episode), recovered
	if stack.DerivedFrom != "" {
		return out, nil
	}
	branches, err := s.ListBranchDeployments(ctx, stack.ID)
	if err != nil {
		return stackDetail{}, err
	}
	out.BranchDeployments = toStackSummaries(branches)
	return out, nil
}

type buildSourceIn struct {
	RepoOwner   string `json:"repo_owner" jsonschema:"The repository's owner."`
	RepoName    string `json:"repo_name" jsonschema:"The repository's name."`
	Branch      string `json:"branch,omitempty" jsonschema:"The branch deploys build from by default."`
	Dockerfile  string `json:"dockerfile,omitempty" jsonschema:"Path of the Dockerfile to build, relative to the repository root."`
	ComposePath string `json:"compose_path,omitempty" jsonschema:"Path of the compose file to build, relative to the repository root."`
}

type branchRuleIn struct {
	Pattern          string            `json:"pattern" jsonschema:"An exact branch name, or one trailing wildcard such as feature/*."`
	DockerNetwork    string            `json:"docker_network" jsonschema:"The docker network this branch's deployment joins."`
	HostnameTemplate string            `json:"hostname_template,omitempty" jsonschema:"Hostname to expose the deployment on; {branch} becomes the branch as a hostname label, for example {branch}.preview.example.com. Needs port and a gateway on the network."`
	Port             int               `json:"port,omitzero" jsonschema:"The container port the hostname template exposes."`
	NameSuffix       string            `json:"name_suffix,omitempty" jsonschema:"Exact patterns only: deploy a clone named <stack>-<suffix> instead of redeploying in place."`
	Overrides        map[string]string `json:"overrides,omitempty" jsonschema:"Env values that replace the stack's own in this rule's deployments; clone rules only. On stack_update, omit to keep the current rule's overrides, send {} to clear them."`
}

type candidateBuildIn struct {
	Context    string `json:"context,omitempty" jsonschema:"The build context directory."`
	Dockerfile string `json:"dockerfile,omitempty" jsonschema:"The Dockerfile path."`
}

type candidateServiceIn struct {
	Name    string            `json:"name" jsonschema:"The service's name."`
	Image   string            `json:"image,omitempty" jsonschema:"The image, when the service pulls instead of building."`
	Build   *candidateBuildIn `json:"build,omitempty" jsonschema:"How the service builds, when it builds."`
	Ports   []int             `json:"ports,omitempty" jsonschema:"Ports the service publishes."`
	Expose  []int             `json:"expose,omitempty" jsonschema:"Ignored."`
	EnvKeys []string          `json:"env_keys,omitempty" jsonschema:"Environment keys the service reads."`
}

type candidateReachableIn struct {
	Service string `json:"service,omitempty" jsonschema:"Ignored."`
	Port    int    `json:"port,omitzero" jsonschema:"Ignored."`
}

type candidateIn struct {
	Kind      string                `json:"kind" jsonschema:"compose or dockerfile."`
	Path      string                `json:"path" jsonschema:"The compose file or Dockerfile path within the repository."`
	Name      string                `json:"name,omitempty" jsonschema:"Ignored."`
	Services  []candidateServiceIn  `json:"services,omitempty" jsonschema:"The candidate's declared services."`
	Reachable *candidateReachableIn `json:"reachable,omitempty" jsonschema:"Ignored."`
}

type stackCreateIn struct {
	ProjectID         string            `json:"project_id,omitempty" jsonschema:"The project the stack belongs to, by id from project_list. Required with a build_source; omit only for the instance's own infrastructure."`
	Machine           string            `json:"machine" jsonschema:"The machine's name, from machine_list, for example prod-1. Needs Deploy to on it."`
	Name              string            `json:"name,omitempty" jsonschema:"The stack's name; its slug is derived from it once. Defaults to build_source.repo_name with a candidate."`
	Strategy          Strategy          `json:"strategy,omitempty" jsonschema:"compose or run. Required without a candidate; a candidate sets it from its kind."`
	ComposePath       string            `json:"compose_path,omitempty" jsonschema:"The compose file's path for the compose strategy. Defaults to docker-compose.yml."`
	DockerNetwork     string            `json:"docker_network,omitempty" jsonschema:"The docker network a run stack's container joins; required for run."`
	Ports             []string          `json:"ports,omitempty" jsonschema:"Host-to-container port mappings for a run stack, for example 8080:80."`
	Mounts            []string          `json:"mounts,omitempty" jsonschema:"Host-to-container bind mounts for a run stack, for example /srv/data:/data."`
	Command           []string          `json:"command,omitempty" jsonschema:"Replaces the image's command for a run stack, one argument per item."`
	Env               map[string]string `json:"env,omitempty" jsonschema:"Environment variables for the stack's containers."`
	BuildSource       *buildSourceIn    `json:"build_source,omitempty" jsonschema:"The repository to build from; without it the stack can only redeploy images."`
	BranchDeployRules []branchRuleIn    `json:"branch_deploy_rules,omitempty" jsonschema:"Rules that deploy pushed branches, each mapping a branch pattern to a network."`
	Candidate         *candidateIn      `json:"candidate,omitempty" jsonschema:"A candidate from repository_scan, passed unchanged; it sets the strategy, compose path, and services."`
	LinkRepository    bool              `json:"link_repository,omitzero" jsonschema:"Attach build_source's repository to the project first if it is not in it."`
	Deploy            bool              `json:"deploy,omitzero" jsonschema:"Also start the first deploy."`
	Ref               string            `json:"ref,omitempty" jsonschema:"The branch, tag, or commit the first deploy builds. Defaults to build_source.branch."`
}

// stackCreated is the new stack, plus its first deploy when one was asked for.
type stackCreated struct {
	Stack  stackResult   `json:"stack"`
	Deploy *deployResult `json:"deploy,omitempty"`
}

func stackCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_create", "Create stack",
		"Creates a stack on a machine, from explicit fields or from a repository_scan candidate plus a build source, "+
			"as the project wizard does. Set deploy to also start its first deploy, then poll deploy_list with its id; "+
			"use stack_deploy for later deploys and stack_update to change the stack. Returns the stack and the started "+
			"deploy.",
		mcptool.Hints{},
		func(ctx context.Context, in stackCreateIn) (any, error) {
			stack, declared, err := stackFromCreate(in)
			if err != nil {
				return nil, err
			}
			opts := CreateStackOptions{LinkRepository: in.LinkRepository, Deploy: in.Deploy, Ref: in.Ref, TriggeredBy: mcpTriggeredBy(ctx)}
			created, d, err := s.CreateStackWithOptions(ctx, stack, declared, opts)
			if err != nil && created != nil {
				return nil, &mcptool.PartialError{
					Applied: []string{fmt.Sprintf("created stack %s (id %s)", created.Name, created.ID)},
					Err:     fmt.Errorf("its first deploy did not start (retry with stack_deploy): %w", err),
				}
			}
			if err != nil {
				return nil, err
			}
			out := stackCreated{Stack: toStackResult(created)}
			if d != nil {
				dr := toDeployResult(d)
				out.Deploy = &dr
			}
			return out, nil
		})
}

// stackFromCreate builds the stack and its declared services from explicit fields or, when given, a candidate.
func stackFromCreate(in stackCreateIn) (Stack, map[string]Declared, error) {
	stack := Stack{
		ProjectID: in.ProjectID, Name: in.Name, Machine: in.Machine, Strategy: in.Strategy,
		ComposePath: in.ComposePath, DockerNetwork: in.DockerNetwork,
		Ports: in.Ports, Mounts: in.Mounts, Command: in.Command, Env: in.Env,
		BranchDeployRules: toBranchRules(in.BranchDeployRules, nil),
	}
	if bs := in.BuildSource; bs != nil {
		stack.BuildSource = &BuildSource{RepoOwner: bs.RepoOwner, RepoName: bs.RepoName, Branch: bs.Branch, Dockerfile: bs.Dockerfile, ComposePath: bs.ComposePath}
	}
	if in.Candidate == nil {
		if stack.Strategy == "" {
			return stack, nil, fmt.Errorf("%w: strategy is required without a candidate: compose or run", apperrs.ErrInvalid)
		}
		return stack, nil, nil
	}
	if stack.Name == "" && stack.BuildSource != nil {
		stack.Name = stack.BuildSource.RepoName
	}
	if stack.Name == "" {
		return stack, nil, fmt.Errorf("%w: name is required (the candidate has none of its own; set name or build_source.repo_name)", apperrs.ErrInvalid)
	}
	declared, err := applyCandidate(&stack, *in.Candidate)
	return stack, declared, err
}

// applyCandidate sets the strategy fields from the candidate's kind and returns the services CreateStack records.
func applyCandidate(stack *Stack, c candidateIn) (map[string]Declared, error) {
	declared := make(map[string]Declared, len(c.Services))
	switch c.Kind {
	case "compose":
		stack.Strategy = StrategyCompose
		stack.ComposePath = c.Path
		for _, svc := range c.Services {
			declared[svc.Name] = declaredFromCandidate(svc)
		}
		return declared, nil
	case "dockerfile":
		stack.Strategy = StrategyRun
		if len(c.Services) > 0 {
			// A dockerfile candidate is a stack of one, named after the stack's slug as CreateStack derives it.
			declared[Slug(stack.Name, dnsLabelMaxLen)] = declaredFromCandidate(c.Services[0])
		}
		return declared, nil
	}
	return nil, fmt.Errorf("%w: candidate kind %q must be compose or dockerfile", apperrs.ErrInvalid, c.Kind)
}

func declaredFromCandidate(svc candidateServiceIn) Declared {
	d := Declared{Image: svc.Image, EnvKeys: svc.EnvKeys}
	if svc.Build != nil {
		d.Build = svc.Build.Dockerfile
		if d.Build == "" {
			d.Build = svc.Build.Context
		}
	}
	for _, p := range svc.Ports {
		d.Ports = append(d.Ports, strconv.Itoa(p))
	}
	return d
}

type envPatchIn struct {
	Set   map[string]string `json:"set,omitempty" jsonschema:"Keys to add or overwrite, with their values; an empty string blanks a value."`
	Unset []string          `json:"unset,omitempty" jsonschema:"Keys to remove."`
}

type buildSourcePatchIn struct {
	RepoOwner   *string `json:"repo_owner,omitempty" jsonschema:"The repository's owner."`
	RepoName    *string `json:"repo_name,omitempty" jsonschema:"The repository's name."`
	Branch      *string `json:"branch,omitempty" jsonschema:"The branch deploys build from by default."`
	Dockerfile  *string `json:"dockerfile,omitempty" jsonschema:"Path of the Dockerfile to build; an empty string clears it."`
	ComposePath *string `json:"compose_path,omitempty" jsonschema:"Path of the compose file to build; an empty string clears it."`
}

type stackUpdateIn struct {
	ID                        string              `json:"id" jsonschema:"The stack's id, from stack_list."`
	ProjectID                 *string             `json:"project_id,omitempty" jsonschema:"Move the stack to this project, by id from project_list. An empty string leaves it with no project, which only a stack without a build source may be."`
	Name                      *string             `json:"name,omitempty" jsonschema:"A new display name; the slug never changes."`
	Machine                   *string             `json:"machine,omitempty" jsonschema:"Move the stack to the machine with this name, from machine_list; the next deploy runs there. Needs Deploy to on it."`
	Strategy                  *Strategy           `json:"strategy,omitempty" jsonschema:"compose or run."`
	ComposePath               *string             `json:"compose_path,omitempty" jsonschema:"The compose file's path for the compose strategy."`
	DockerNetwork             *string             `json:"docker_network,omitempty" jsonschema:"The docker network a run stack's container joins."`
	Ports                     []string            `json:"ports,omitempty" jsonschema:"Replaces the run stack's port mappings, for example 8080:80; send [] to clear them."`
	Mounts                    []string            `json:"mounts,omitempty" jsonschema:"Replaces the run stack's bind mounts, for example /srv/data:/data; send [] to clear them."`
	Command                   []string            `json:"command,omitempty" jsonschema:"Replaces the run stack's command override; send [] to use the image's own."`
	Env                       *envPatchIn         `json:"env,omitempty" jsonschema:"Changes to the environment variables; keys not named keep their values."`
	BuildSource               *buildSourcePatchIn `json:"build_source,omitempty" jsonschema:"Changes to the repository the stack builds from; fields not sent keep their values."`
	BranchDeployRules         []branchRuleIn      `json:"branch_deploy_rules,omitempty" jsonschema:"Replaces the whole list of branch deploy rules; send [] to remove them all."`
	CrashTickets              *bool               `json:"crash_tickets,omitempty" jsonschema:"True opens one ticket in the stack's project when a service crashes, and a repeat crash or recovery comments on it; false stops. Only a stack with a project may switch it on."`
	ErrorRate                 *errorRatePatchIn   `json:"error_rate,omitempty" jsonschema:"Changes to the error-rate rule; fields not sent keep their values."`
	CrashNoticeConversationID *string             `json:"crash_notice_conversation_id,omitempty" jsonschema:"Posts each crash, error-rate episode and recovery, with a link to its crash ticket, into this channel, by id from conversation_list; you must be able to post there. An empty string stops the notices. Changing it without crash_notice_bot_id clears the bot."`
	CrashNoticeBotID          *string             `json:"crash_notice_bot_id,omitempty" jsonschema:"Posts the notices as this bot of that conversation, by id from botwebhook_list; an empty string posts them as the instance."`
	Watching                  *bool               `json:"watching,omitempty" jsonschema:"true makes you a watcher, notified when a service crashes and when it comes back; false stops that for you alone. Everyone who can read a stack watches until they stop."`
}

// errorRatePatchIn changes a stack's error-rate rule: it trips when the share of 5xx answers over the last minutes is
// above percent with at least min_requests requests, and ends once each of those minutes has stayed under the limit.
type errorRatePatchIn struct {
	Enabled     *bool    `json:"enabled,omitempty" jsonschema:"True switches the rule on; false switches it off and ends an open episode. Needs a hostname on the stack, and for a Cloudflare tunnel hostname a token that can read analytics."`
	Percent     *float64 `json:"percent,omitempty" jsonschema:"The 5xx share, in percent, above which the rule trips, 0.1 to 100. Default 5."`
	Minutes     *int     `json:"minutes,omitempty" jsonschema:"Minutes the share is read over, and how long it must stay under the limit to recover, 1 to 60. Default 5."`
	MinRequests *int     `json:"min_requests,omitempty" jsonschema:"The fewest requests in those minutes for the rule to trip, 1 or more. Default 20."`
}

func patchErrorRate(cur ErrorRateRule, p *errorRatePatchIn) ErrorRateRule {
	if p == nil {
		return cur
	}
	set(&cur.Enabled, p.Enabled)
	set(&cur.Percent, p.Percent)
	set(&cur.Minutes, p.Minutes)
	set(&cur.MinRequests, p.MinRequests)
	return cur
}

// stackUpdated is stack_update's answer: the stack, and whether the caller watches it now.
type stackUpdated struct {
	stackResult
	Watching bool `json:"watching"`
}

func stackUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_update", "Update stack",
		"Changes a stack's definition. Env changes go through set and unset; branch_deploy_rules replaces the whole "+
			"list, and a rule sent without overrides keeps the overrides of the rule with the same pattern. "+
			"crash_tickets, crash_notice_conversation_id and error_rate set up crash tickets, channel notices, and the "+
			"5xx-rate rule; watching starts or stops your own notices and alone needs only read access. It does not "+
			"redeploy: use stack_deploy.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in stackUpdateIn) (any, error) {
			cur, err := s.GetStack(ctx, in.ID)
			if err != nil {
				return nil, withHint(err, "stack_list lists stacks")
			}
			updated := cur
			var applied []string
			if !onlyWatching(in) {
				if updated, err = s.UpdateStack(ctx, patchStack(*cur, in)); err != nil {
					return nil, err
				}
				applied = append(applied, "stack")
			}
			watching, err := watchingAfter(ctx, s, updated.ID, in.Watching)
			if err != nil && len(applied) > 0 {
				return nil, &mcptool.PartialError{Applied: applied, Err: fmt.Errorf("watching: %w", err)}
			}
			if err != nil {
				return nil, err
			}
			return stackUpdated{stackResult: toStackResult(updated), Watching: watching}, nil
		})
}

// watchingAfter applies the caller's watching choice when they sent one, and reads the state either way.
func watchingAfter(ctx context.Context, s *Service, stackID string, choice *bool) (bool, error) {
	if choice == nil {
		return s.Watching(ctx, stackID)
	}
	return s.SetWatching(ctx, stackID, *choice)
}

// onlyWatching is a call that sends nothing but watching, which needs stacks:read and not stacks:write.
func onlyWatching(in stackUpdateIn) bool {
	return in.Watching != nil && reflect.DeepEqual(in, stackUpdateIn{ID: in.ID, Watching: in.Watching})
}

// patchStack overlays only the fields the caller sent, so an omitted field keeps its stored value.
func patchStack(cur Stack, in stackUpdateIn) Stack {
	set(&cur.ProjectID, in.ProjectID)
	set(&cur.Name, in.Name)
	set(&cur.Machine, in.Machine)
	set(&cur.Strategy, in.Strategy)
	set(&cur.ComposePath, in.ComposePath)
	set(&cur.DockerNetwork, in.DockerNetwork)
	set(&cur.CrashTickets, in.CrashTickets)
	if in.CrashNoticeConversationID != nil && *in.CrashNoticeConversationID != cur.CrashNoticeConversationID {
		cur.CrashNoticeBotID = ""
	}
	set(&cur.CrashNoticeConversationID, in.CrashNoticeConversationID)
	set(&cur.CrashNoticeBotID, in.CrashNoticeBotID)
	cur.ErrorRate = patchErrorRate(cur.ErrorRate, in.ErrorRate)
	cur.Ports = replaced(cur.Ports, in.Ports)
	cur.Mounts = replaced(cur.Mounts, in.Mounts)
	cur.Command = replaced(cur.Command, in.Command)
	cur.Env = patchEnv(cur.Env, in.Env)
	cur.BuildSource = patchBuildSource(cur.BuildSource, in.BuildSource)
	if in.BranchDeployRules != nil {
		cur.BranchDeployRules = toBranchRules(in.BranchDeployRules, cur.BranchDeployRules)
	}
	return cur
}

func set[T any](dst *T, v *T) {
	if v != nil {
		*dst = *v
	}
}

func replaced(cur, next []string) []string {
	if next == nil {
		return cur
	}
	return next
}

func patchEnv(cur map[string]string, p *envPatchIn) map[string]string {
	if p == nil {
		return cur
	}
	env := maps.Clone(cur)
	if env == nil {
		env = map[string]string{}
	}
	maps.Copy(env, p.Set)
	for _, k := range p.Unset {
		delete(env, k)
	}
	return env
}

func patchBuildSource(cur *BuildSource, p *buildSourcePatchIn) *BuildSource {
	if p == nil {
		return cur
	}
	var bs BuildSource
	if cur != nil {
		bs = *cur
	}
	set(&bs.RepoOwner, p.RepoOwner)
	set(&bs.RepoName, p.RepoName)
	set(&bs.Branch, p.Branch)
	set(&bs.Dockerfile, p.Dockerfile)
	set(&bs.ComposePath, p.ComposePath)
	return &bs
}

// toBranchRules converts rule inputs; a rule sent without overrides keeps those of the current rule with its pattern.
func toBranchRules(in []branchRuleIn, cur []BranchDeployRule) []BranchDeployRule {
	if in == nil {
		return nil
	}
	kept := make(map[string]map[string]string, len(cur))
	for _, r := range cur {
		kept[r.Pattern] = r.Overrides
	}
	out := make([]BranchDeployRule, 0, len(in))
	for _, r := range in {
		rule := BranchDeployRule{
			Pattern: r.Pattern, DockerNetwork: r.DockerNetwork, HostnameTemplate: r.HostnameTemplate,
			Port: r.Port, NameSuffix: r.NameSuffix, Overrides: r.Overrides,
		}
		// Only a rule that deploys its own copy can carry overrides, so an in-place rule drops the stored ones.
		if rule.Overrides == nil && rule.DerivesClone() {
			rule.Overrides = kept[r.Pattern]
		}
		out = append(out, rule)
	}
	return out
}

type stackDeleteIn struct {
	ID string `json:"id" jsonschema:"The stack's id, from stack_list."`
}

func stackDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_delete", "Delete stack",
		"Deletes a stack and its service records, releases its hostnames from the gateway and DNS, and drops its "+
			"nodes from the topology. Deploy history is kept. Containers already running on the machine are not stopped.",
		mcptool.Hints{Idempotent: true},
		func(ctx context.Context, in stackDeleteIn) (any, error) {
			if err := s.DeleteStack(ctx, in.ID); err != nil {
				return nil, withHint(err, "stack_list lists stacks")
			}
			return mcptool.Gone(in.ID), nil
		})
}

type stackDeployIn struct {
	ID       string `json:"id" jsonschema:"The stack's id, from stack_list."`
	Ref      string `json:"ref,omitempty" jsonschema:"Build and deploy this branch, tag, or commit of the stack's repository."`
	Image    string `json:"image,omitempty" jsonschema:"An already-built image to redeploy instead of building, for example ghcr.io/acme/api:1.4.0."`
	Rollback bool   `json:"rollback,omitzero" jsonschema:"Redeploy the last healthy image; send it without ref or image."`
	Retry    string `json:"retry,omitempty" jsonschema:"A failed deploy of this stack, from deploy_list. Runs it again on the commit it built, with the stack's current settings; send it without ref, image or rollback."`
}

func stackDeployTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_deploy", "Deploy stack",
		"Starts a deploy of a stack on its machine, as the deploy button does. Choose one source: ref builds a "+
			"branch, tag, or commit, image redeploys an already-built image, rollback reuses the last healthy image, "+
			"retry runs a failed deploy again. Returns the new deploy; poll deploy_list with its id until it is healthy "+
			"or failed.",
		mcptool.Hints{},
		func(ctx context.Context, in stackDeployIn) (any, error) {
			d, err := startDeploy(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return toDeployResult(d), nil
		})
}

func startDeploy(ctx context.Context, s *Service, in stackDeployIn) (*Deploy, error) {
	if in.Rollback && (in.Ref != "" || in.Image != "") {
		return nil, fmt.Errorf("%w: rollback takes neither ref nor image", apperrs.ErrInvalid)
	}
	if in.Retry != "" {
		return retryDeploy(ctx, s, in)
	}
	if in.Rollback {
		d, err := s.Rollback(ctx, in.ID, mcpTriggeredBy(ctx))
		return d, withHint(err, "a rollback needs a healthy deploy of an existing stack; deploy_list with stack_id shows its deploys")
	}
	if in.Ref == "" && in.Image == "" {
		return nil, fmt.Errorf("%w: send ref to build, image to redeploy, rollback: true, or retry with a failed deploy's id", apperrs.ErrInvalid)
	}
	d, err := s.Deploy(ctx, DeployRequest{StackID: in.ID, Ref: in.Ref, Image: in.Image, TriggeredBy: mcpTriggeredBy(ctx)})
	return d, withHint(err, "stack_list lists stacks")
}

// retryDeploy retries in.Retry, which must be a deploy of the stack in.ID.
func retryDeploy(ctx context.Context, s *Service, in stackDeployIn) (*Deploy, error) {
	if in.Rollback || in.Ref != "" || in.Image != "" {
		return nil, fmt.Errorf("%w: retry takes none of ref, image or rollback", apperrs.ErrInvalid)
	}
	failed, err := s.Get(ctx, in.Retry)
	if err != nil {
		return nil, withHint(err, "deploy_list with stack_id lists a stack's deploys")
	}
	if failed.StackID != in.ID {
		return nil, fmt.Errorf("%w: deploy %s is not a deploy of stack %s", apperrs.ErrInvalid, in.Retry, in.ID)
	}
	return s.Retry(ctx, in.Retry, mcpTriggeredBy(ctx))
}

type importStackIn struct {
	Project    string   `json:"project" jsonschema:"The compose project's name, from machine_discover."`
	Containers []string `json:"containers" jsonschema:"The names of the project's containers to adopt."`
}

type machineImportIn struct {
	ID         string          `json:"id" jsonschema:"The machine's id, from machine_list."`
	ProjectID  string          `json:"project_id" jsonschema:"The project the adopted stacks belong to, by id from project_list."`
	Stacks     []importStackIn `json:"stacks,omitempty" jsonschema:"Compose projects to adopt, each as one stack, as machine_discover grouped them."`
	Standalone []string        `json:"standalone,omitempty" jsonschema:"Names of standalone containers to adopt, each as a stack of one."`
	Gateways   []string        `json:"gateways,omitempty" jsonschema:"Names of cloudflared containers to adopt as stacks and as gateways with their routed hostnames."`
}

type importResult struct {
	Stacks   []stackSummary    `json:"stacks"`
	Gateways []GatewayAdoption `json:"gateways"`
}

func machineImportTool(s *Service) mcptool.Tool {
	return mcptool.New("machine_import", "Import from machine",
		"Adopts containers already running on a machine as unmanaged stacks, from what machine_discover found. Run "+
			"machine_discover first and pass the names it returned. Re-importing updates the observed facts by container "+
			"name and never deletes; a gateway that cannot be adopted carries its reason in the result.",
		mcptool.Hints{Idempotent: true},
		func(ctx context.Context, in machineImportIn) (any, error) {
			req := ImportRequest{ProjectID: in.ProjectID, Standalone: in.Standalone, Gateways: in.Gateways}
			for _, g := range in.Stacks {
				req.Stacks = append(req.Stacks, ImportStackGroup(g))
			}
			result, err := s.Import(ctx, in.ID, req)
			if err != nil {
				return nil, withHint(err, "machine_list lists machines")
			}
			return importResult{Stacks: toStackSummaries(result.Stacks), Gateways: result.Gateways}, nil
		})
}

// mcpTriggeredBy is deploy provenance for an MCP call (ADR 0049): "<user id>:mcp", empty with no actor.
func mcpTriggeredBy(ctx context.Context) string {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return ""
	}
	return a.ID + ":mcp"
}

// withHint names the tool that lists valid ids on a not-found error, so the model can recover.
func withHint(err error, hint string) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w; %s", err, hint)
	}
	return err
}

// deployResult is a deploy as an agent reads it: the deprecated service_id alias and rule id left out.
type deployResult struct {
	ID          string    `json:"id"`
	Kind        Kind      `json:"kind"`
	Status      Status    `json:"status"`
	StackID     string    `json:"stack_id"`
	Stack       string    `json:"stack"`
	Machine     string    `json:"machine"`
	Image       string    `json:"image,omitempty"`
	Ref         string    `json:"ref,omitempty"`
	Commit      string    `json:"commit,omitempty"`
	Address     string    `json:"address,omitempty"`
	TriggeredBy string    `json:"triggered_by,omitempty"`
	RuleName    string    `json:"rule_name,omitempty"`
	TicketID    string    `json:"ticket_id,omitempty"`
	PRNumber    int       `json:"pr_number,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toDeployResult(d *Deploy) deployResult {
	return deployResult{
		ID: d.ID, Kind: d.Kind, Status: d.Status, StackID: d.StackID, Stack: d.Service, Machine: d.Target,
		Image: d.Image, Ref: d.Ref, Commit: d.Commit, Address: d.Address, TriggeredBy: d.TriggeredBy, RuleName: d.RuleName,
		TicketID: d.TicketID, PRNumber: d.PRNumber, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

func toDeployResults(ds []*Deploy) []deployResult {
	out := make([]deployResult, 0, len(ds))
	for _, d := range ds {
		out = append(out, toDeployResult(d))
	}
	return out
}

// stackSummary is a stack in a list: enough to pick one and call stack_get.
type stackSummary struct {
	ID          string       `json:"id"`
	ProjectID   string       `json:"project_id,omitempty"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Machine     string       `json:"machine"`
	Strategy    Strategy     `json:"strategy"`
	Managed     bool         `json:"managed"`
	BuildSource *BuildSource `json:"build_source,omitempty"`
	Branch      string       `json:"branch,omitempty"`
}

func toStackSummary(st *Stack) stackSummary {
	return stackSummary{
		ID: st.ID, ProjectID: st.ProjectID, Name: st.Name, Slug: st.Slug, Machine: st.Machine,
		Strategy: st.Strategy, Managed: st.Managed, BuildSource: st.BuildSource, Branch: st.Branch,
	}
}

func toStackSummaries(stacks []*Stack) []stackSummary {
	out := make([]stackSummary, 0, len(stacks))
	for _, st := range stacks {
		out = append(out, toStackSummary(st))
	}
	return out
}

// branchRuleResult is a branch deploy rule with its override values withheld; they are env values.
type branchRuleResult struct {
	Pattern          string   `json:"pattern"`
	DockerNetwork    string   `json:"docker_network"`
	HostnameTemplate string   `json:"hostname_template,omitempty"`
	Port             int      `json:"port,omitempty"`
	NameSuffix       string   `json:"name_suffix,omitempty"`
	OverrideKeys     []string `json:"override_keys,omitempty"`
}

// stackResult is a stack's definition with env values withheld: a stack's env holds secrets such as TUNNEL_TOKEN.
type stackResult struct {
	stackSummary
	ComposePath               string             `json:"compose_path,omitempty"`
	DockerNetwork             string             `json:"docker_network,omitempty"`
	Ports                     []string           `json:"ports,omitempty"`
	Mounts                    []string           `json:"mounts,omitempty"`
	Command                   []string           `json:"command,omitempty"`
	EnvKeys                   []string           `json:"env_keys"`
	BranchDeployRules         []branchRuleResult `json:"branch_deploy_rules,omitempty"`
	DerivedFrom               string             `json:"derived_from,omitempty"`
	CrashTickets              bool               `json:"crash_tickets"`
	ErrorRate                 ErrorRateRule      `json:"error_rate"`
	CrashNoticeConversationID string             `json:"crash_notice_conversation_id,omitempty"`
	CrashNoticeBotID          string             `json:"crash_notice_bot_id,omitempty"`
}

func toStackResult(st *Stack) stackResult {
	rules := make([]branchRuleResult, 0, len(st.BranchDeployRules))
	for _, r := range st.BranchDeployRules {
		rules = append(rules, branchRuleResult{
			Pattern: r.Pattern, DockerNetwork: r.DockerNetwork, HostnameTemplate: r.HostnameTemplate,
			Port: r.Port, NameSuffix: r.NameSuffix, OverrideKeys: slices.Sorted(maps.Keys(r.Overrides)),
		})
	}
	return stackResult{
		stackSummary: toStackSummary(st),
		ComposePath:  st.ComposePath, DockerNetwork: st.DockerNetwork,
		Ports: st.Ports, Mounts: st.Mounts, Command: st.Command,
		EnvKeys:                   slices.Sorted(maps.Keys(st.Env)),
		BranchDeployRules:         rules,
		DerivedFrom:               st.DerivedFrom,
		CrashTickets:              st.CrashTickets,
		ErrorRate:                 st.ErrorRate,
		CrashNoticeConversationID: st.CrashNoticeConversationID,
		CrashNoticeBotID:          st.CrashNoticeBotID,
	}
}
