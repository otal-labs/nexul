package deploy

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// defaultLogLines and maxLogLines bound the log tail deploy_get returns, so a long build never floods the context.
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
		deployGetTool(s),
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
	StackID string `json:"stack_id,omitempty" jsonschema:"Only this stack's deploys, by the stack's id from stack_list."`
	Status  string `json:"status,omitempty" jsonschema:"Only deploys in this state: pending, running, healthy, or failed."`
	mcptool.PageArgs
}

func deployListTool(s *Service) mcptool.Tool {
	return mcptool.New("deploy_list", "List deploys",
		"Lists deploys newest first, as summaries without their logs. Filter by stack_id to see one stack's "+
			"history or by status to find what is running or failed; use deploy_get for one deploy with its log. "+
			"Returns at most 100 per page.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in deployListIn) (any, error) {
			ds, err := listDeploys(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return mcptool.Paginate(toDeployResults(ds), in.PageArgs), nil
		})
}

func listDeploys(ctx context.Context, s *Service, in deployListIn) ([]*Deploy, error) {
	if in.Status != "" {
		ds, err := s.ListByStatus(ctx, Status(in.Status))
		if errors.Is(err, apperrs.ErrInvalid) {
			return nil, fmt.Errorf("%w; status is one of pending, running, healthy, failed", err)
		}
		if err != nil {
			return nil, err
		}
		return slices.DeleteFunc(ds, func(d *Deploy) bool { return in.StackID != "" && d.StackID != in.StackID }), nil
	}
	if in.StackID != "" {
		return s.ListByStackID(ctx, in.StackID)
	}
	ds, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	slices.Reverse(ds)
	return ds, nil
}

type deployGetIn struct {
	ID       string `json:"id" jsonschema:"The deploy's id, from deploy_list or from the result of stack_deploy."`
	LogLines int    `json:"log_lines,omitzero" jsonschema:"How many of the log's last lines to return, 1 to 1000. Defaults to 200."`
}

// deployDetail is a deploy with the tail of its log; log_total is the full log's length, so a cut tail shows.
type deployDetail struct {
	deployResult
	Log      []LogLine `json:"log"`
	LogTotal int       `json:"log_total"`
}

func deployGetTool(s *Service) mcptool.Tool {
	return mcptool.New("deploy_get", "Get deploy",
		"Returns one deploy with its status, what triggered it, and the last lines of its checkout, build, and "+
			"deploy log (each line has a unix-millisecond ts and a phase). Poll it after stack_deploy until the status "+
			"is healthy or failed, and read the log to diagnose a failure. The log tail defaults to 200 lines; "+
			"log_total is the full length.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in deployGetIn) (any, error) {
			d, err := s.Get(ctx, in.ID)
			if err != nil {
				return nil, withHint(err, "deploy_list lists deploys")
			}
			lines, err := s.Log(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			n := in.LogLines
			if n < 1 {
				n = defaultLogLines
			}
			n = min(n, maxLogLines)
			return deployDetail{deployResult: toDeployResult(d), Log: lines[max(0, len(lines)-n):], LogTotal: len(lines)}, nil
		})
}

type deployCancelIn struct {
	ID string `json:"id" jsonschema:"The deploy's id, from deploy_list."`
}

func deployCancelTool(s *Service) mcptool.Tool {
	return mcptool.New("deploy_cancel", "Cancel deploy",
		"Asks the runner to stop a pending or running deploy; the deploy then lands failed with reason cancelled. "+
			"Use deploy_get afterwards to see it land. A deploy that already finished cannot be cancelled and returns "+
			"a conflict.",
		mcptool.Hints{},
		func(ctx context.Context, in deployCancelIn) (any, error) {
			if err := s.Cancel(ctx, in.ID); err != nil {
				return nil, withHint(err, "deploy_list lists deploys")
			}
			return map[string]string{"id": in.ID, "status": "cancelling"}, nil
		})
}

type stackListIn struct {
	ProjectID string `json:"project_id,omitempty" jsonschema:"Only this project's stacks, by the project's id from project_list. Omit for every project."`
	mcptool.PageArgs
}

func stackListTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_list", "List stacks",
		"Lists stacks, the deployable units of a project, with their machine, strategy, and repository. Branch "+
			"deployments are not listed here; stack_get shows them under their base stack, along with the stack's "+
			"services and recent deploys. Returns at most 100 per page.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in stackListIn) (any, error) {
			stacks, err := s.ListStacks(ctx, in.ProjectID)
			if err != nil {
				return nil, err
			}
			return mcptool.Paginate(toStackSummaries(stacks), in.PageArgs), nil
		})
}

type stackGetIn struct {
	ID string `json:"id" jsonschema:"The stack's id, from stack_list."`
}

// stackDetail is stack_get's answer: the stack plus what an agent reads next about it.
type stackDetail struct {
	stackResult
	Services          []*Container   `json:"services"`
	RecentDeploys     []deployResult `json:"recent_deploys"`
	BranchDeployments []stackSummary `json:"branch_deployments,omitempty"`
}

func stackGetTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_get", "Get stack",
		"Returns one stack's definition with its services (the containers it runs, with image, status, networks, "+
			"and ports), its ten most recent deploys, and its branch deployments. Environment variables are listed "+
			"by key only; values never leave the server. Use it before stack_update or stack_deploy, and deploy_list "+
			"for older deploys.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in stackGetIn) (any, error) {
			stack, err := s.GetStack(ctx, in.ID)
			if err != nil {
				return nil, withHint(err, "stack_list lists stacks")
			}
			return stackDetailFor(ctx, s, stack)
		})
}

func stackDetailFor(ctx context.Context, s *Service, stack *Stack) (stackDetail, error) {
	services, err := s.ListServices(ctx, stack.ID)
	if err != nil {
		return stackDetail{}, err
	}
	deploys, err := s.ListByStackID(ctx, stack.ID)
	if err != nil {
		return stackDetail{}, err
	}
	out := stackDetail{
		stackResult:   toStackResult(stack),
		Services:      nonNil(services),
		RecentDeploys: toDeployResults(deploys[:min(len(deploys), recentDeploys)]),
	}
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
	RepoOwner   string `json:"repo_owner" jsonschema:"The repository's owner, for example acme."`
	RepoName    string `json:"repo_name" jsonschema:"The repository's name, for example api."`
	Branch      string `json:"branch,omitempty" jsonschema:"The branch deploys build from by default, for example main."`
	Dockerfile  string `json:"dockerfile,omitempty" jsonschema:"Path of the Dockerfile to build, relative to the repository root."`
	ComposePath string `json:"compose_path,omitempty" jsonschema:"Path of the compose file to build, relative to the repository root."`
}

type branchRuleIn struct {
	Pattern          string            `json:"pattern" jsonschema:"An exact branch name, or one trailing wildcard such as feature/*."`
	DockerNetwork    string            `json:"docker_network" jsonschema:"The docker network this branch's deployment joins."`
	HostnameTemplate string            `json:"hostname_template,omitempty" jsonschema:"Hostname to expose the deployment on, where {branch} becomes the branch as a hostname label, for example {branch}.preview.example.com. Needs port and a gateway on the network."`
	Port             int               `json:"port,omitzero" jsonschema:"The container port the hostname template exposes."`
	NameSuffix       string            `json:"name_suffix,omitempty" jsonschema:"Exact patterns only: deploy a clone named <stack>-<suffix> instead of redeploying the stack in place, for example qa."`
	Overrides        map[string]string `json:"overrides,omitempty" jsonschema:"Env values that replace the stack's own in this rule's deployments; clone rules only. On stack_update, omit it to keep the current rule's overrides and send {} to clear them."`
}

type candidateBuildIn struct {
	Context    string `json:"context,omitempty" jsonschema:"The build context directory."`
	Dockerfile string `json:"dockerfile,omitempty" jsonschema:"The Dockerfile path."`
}

type candidateServiceIn struct {
	Name    string            `json:"name" jsonschema:"The service's name."`
	Image   string            `json:"image,omitempty" jsonschema:"The image the service runs, when it pulls instead of building."`
	Build   *candidateBuildIn `json:"build,omitempty" jsonschema:"How the service builds, when it builds."`
	Ports   []int             `json:"ports,omitempty" jsonschema:"Ports the service publishes."`
	Expose  []int             `json:"expose,omitempty" jsonschema:"Ignored; accepted so a repository_scan candidate passes unchanged."`
	EnvKeys []string          `json:"env_keys,omitempty" jsonschema:"Environment keys the service reads."`
}

type candidateReachableIn struct {
	Service string `json:"service,omitempty" jsonschema:"Ignored."`
	Port    int    `json:"port,omitzero" jsonschema:"Ignored."`
}

type candidateIn struct {
	Kind      string                `json:"kind" jsonschema:"compose or dockerfile."`
	Path      string                `json:"path" jsonschema:"The compose file or Dockerfile path within the repository."`
	Name      string                `json:"name,omitempty" jsonschema:"Ignored; accepted so a repository_scan candidate passes unchanged."`
	Services  []candidateServiceIn  `json:"services,omitempty" jsonschema:"The candidate's declared services."`
	Reachable *candidateReachableIn `json:"reachable,omitempty" jsonschema:"Ignored; accepted so a repository_scan candidate passes unchanged."`
}

type stackCreateIn struct {
	ProjectID         string            `json:"project_id" jsonschema:"The project the stack belongs to, by id from project_list."`
	Machine           string            `json:"machine" jsonschema:"The name of the machine the stack runs on, from machine_list, for example prod-1."`
	Name              string            `json:"name,omitempty" jsonschema:"The stack's name; its slug is derived from it once. With a candidate it defaults to build_source.repo_name."`
	Strategy          Strategy          `json:"strategy,omitempty" jsonschema:"compose or run. Required without a candidate; a candidate sets it from its kind."`
	ComposePath       string            `json:"compose_path,omitempty" jsonschema:"The compose file's path for the compose strategy. Defaults to docker-compose.yml."`
	DockerNetwork     string            `json:"docker_network,omitempty" jsonschema:"The docker network a run stack's container joins; required for the run strategy."`
	Ports             []string          `json:"ports,omitempty" jsonschema:"Host-to-container port mappings for a run stack, for example 8080:80."`
	Mounts            []string          `json:"mounts,omitempty" jsonschema:"Host-to-container bind mounts for a run stack, for example /srv/data:/data."`
	Command           []string          `json:"command,omitempty" jsonschema:"Replaces the image's command for a run stack, one argument per item."`
	Env               map[string]string `json:"env,omitempty" jsonschema:"Environment variables for the stack's containers."`
	BuildSource       *buildSourceIn    `json:"build_source,omitempty" jsonschema:"The repository the stack builds from; without it the stack can only redeploy images."`
	BranchDeployRules []branchRuleIn    `json:"branch_deploy_rules,omitempty" jsonschema:"Rules that deploy pushed branches, each mapping a branch pattern to a network."`
	Candidate         *candidateIn      `json:"candidate,omitempty" jsonschema:"A deployable candidate from repository_scan; it sets the strategy, compose path, and declared services."`
	LinkRepository    bool              `json:"link_repository,omitzero" jsonschema:"Attach build_source's repository to the project first, so a repository not yet in the project is accepted."`
	Deploy            bool              `json:"deploy,omitzero" jsonschema:"Start the first deploy right after creating the stack."`
	Ref               string            `json:"ref,omitempty" jsonschema:"The branch, tag, or commit the first deploy builds, for example main. Defaults to build_source.branch."`
}

// stackCreated is the new stack, plus its first deploy when one was asked for.
type stackCreated struct {
	Stack  stackResult   `json:"stack"`
	Deploy *deployResult `json:"deploy,omitempty"`
}

func stackCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_create", "Create stack",
		"Creates a stack on a machine, from explicit fields or from a repository_scan candidate plus a build "+
			"source, exactly as the project wizard does. Set deploy to also start its first deploy, then poll "+
			"deploy_get; use stack_deploy for later deploys and stack_update to change it. Returns the stack and the "+
			"started deploy; if the stack is created but its first deploy cannot start, the error says so.",
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
	Set   map[string]string `json:"set,omitempty" jsonschema:"Keys to add or overwrite, with their values."`
	Unset []string          `json:"unset,omitempty" jsonschema:"Keys to remove."`
}

type buildSourcePatchIn struct {
	RepoOwner   *string `json:"repo_owner,omitempty" jsonschema:"The repository's owner, for example acme."`
	RepoName    *string `json:"repo_name,omitempty" jsonschema:"The repository's name, for example api."`
	Branch      *string `json:"branch,omitempty" jsonschema:"The branch deploys build from by default, for example main."`
	Dockerfile  *string `json:"dockerfile,omitempty" jsonschema:"Path of the Dockerfile to build; an empty string clears it."`
	ComposePath *string `json:"compose_path,omitempty" jsonschema:"Path of the compose file to build; an empty string clears it."`
}

type stackUpdateIn struct {
	ID                string              `json:"id" jsonschema:"The stack's id, from stack_list."`
	ProjectID         *string             `json:"project_id,omitempty" jsonschema:"Move the stack to this project, by id from project_list."`
	Name              *string             `json:"name,omitempty" jsonschema:"A new display name; the slug never changes."`
	Machine           *string             `json:"machine,omitempty" jsonschema:"Move the stack to the machine with this name, from machine_list; the next deploy runs there."`
	Strategy          *Strategy           `json:"strategy,omitempty" jsonschema:"compose or run."`
	ComposePath       *string             `json:"compose_path,omitempty" jsonschema:"The compose file's path for the compose strategy."`
	DockerNetwork     *string             `json:"docker_network,omitempty" jsonschema:"The docker network a run stack's container joins."`
	Ports             []string            `json:"ports,omitempty" jsonschema:"Replaces the run stack's port mappings, for example 8080:80; send [] to clear them."`
	Mounts            []string            `json:"mounts,omitempty" jsonschema:"Replaces the run stack's bind mounts, for example /srv/data:/data; send [] to clear them."`
	Command           []string            `json:"command,omitempty" jsonschema:"Replaces the run stack's command override; send [] to use the image's own."`
	Env               *envPatchIn         `json:"env,omitempty" jsonschema:"Changes to the environment variables; keys not named keep their values."`
	BuildSource       *buildSourcePatchIn `json:"build_source,omitempty" jsonschema:"Changes to the repository the stack builds from; fields not sent keep their values."`
	BranchDeployRules []branchRuleIn      `json:"branch_deploy_rules,omitempty" jsonschema:"Replaces the whole list of branch deploy rules; send [] to remove them all."`
}

func stackUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_update", "Update stack",
		"Changes a stack's definition; only the fields you send change, and every other field keeps its value. "+
			"Env changes go through set and unset, so values you never saw are kept; branch_deploy_rules replaces "+
			"the whole list, and a rule sent without overrides keeps the current overrides of the rule with the same "+
			"pattern. It does not redeploy: use stack_deploy for that. Returns the updated stack.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in stackUpdateIn) (any, error) {
			cur, err := s.GetStack(ctx, in.ID)
			if err != nil {
				return nil, withHint(err, "stack_list lists stacks")
			}
			updated, err := s.UpdateStack(ctx, patchStack(*cur, in))
			if err != nil {
				return nil, err
			}
			return toStackResult(updated), nil
		})
}

// patchStack overlays only the fields the caller sent, so an omitted field keeps its stored value.
func patchStack(cur Stack, in stackUpdateIn) Stack {
	set(&cur.ProjectID, in.ProjectID)
	set(&cur.Name, in.Name)
	set(&cur.Machine, in.Machine)
	set(&cur.Strategy, in.Strategy)
	set(&cur.ComposePath, in.ComposePath)
	set(&cur.DockerNetwork, in.DockerNetwork)
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
		"Deletes a stack and its service records, releases every hostname routed to it from the gateway and DNS, "+
			"and drops its nodes from the topology. Deploy history is kept. It does not stop containers already "+
			"running on the machine. Returns the deleted id.",
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
	Ref      string `json:"ref,omitempty" jsonschema:"Build and deploy this branch, tag, or commit of the stack's repository, for example main."`
	Image    string `json:"image,omitempty" jsonschema:"Redeploy this already-built image instead of building, for example ghcr.io/acme/api:1.4.0."`
	Rollback bool   `json:"rollback,omitzero" jsonschema:"Redeploy the stack's last healthy image; send it without ref or image."`
}

func stackDeployTool(s *Service) mcptool.Tool {
	return mcptool.New("stack_deploy", "Deploy stack",
		"Starts a deploy of a stack on its machine, exactly as the deploy button does: build a ref of its repository, "+
			"redeploy an image, or roll back to the last healthy image. It is rejected while the stack already has a "+
			"deploy pending or running. Returns the new deploy; poll deploy_get until it is healthy or failed.",
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
	if in.Rollback {
		d, err := s.Rollback(ctx, in.ID, mcpTriggeredBy(ctx))
		return d, withHint(err, "a rollback needs a healthy deploy of an existing stack; deploy_list with stack_id shows its deploys")
	}
	if in.Ref == "" && in.Image == "" {
		return nil, fmt.Errorf("%w: send ref to build, image to redeploy, or rollback: true", apperrs.ErrInvalid)
	}
	d, err := s.Deploy(ctx, DeployRequest{StackID: in.ID, Ref: in.Ref, Image: in.Image, TriggeredBy: mcpTriggeredBy(ctx)})
	return d, withHint(err, "stack_list lists stacks")
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
		"Adopts containers already running on a machine as unmanaged stacks, using what machine_discover found: "+
			"compose projects, standalone containers, and cloudflared gateways with their hostnames. Run "+
			"machine_discover first and pass the names it returned. Re-importing updates the observed facts by "+
			"container name and never deletes; a gateway that cannot be adopted carries its reason in the result.",
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

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
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
		Image: d.Image, Address: d.Address, TriggeredBy: d.TriggeredBy, RuleName: d.RuleName,
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
	ProjectID   string       `json:"project_id"`
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
	ComposePath       string             `json:"compose_path,omitempty"`
	DockerNetwork     string             `json:"docker_network,omitempty"`
	Ports             []string           `json:"ports,omitempty"`
	Mounts            []string           `json:"mounts,omitempty"`
	Command           []string           `json:"command,omitempty"`
	EnvKeys           []string           `json:"env_keys"`
	BranchDeployRules []branchRuleResult `json:"branch_deploy_rules,omitempty"`
	DerivedFrom       string             `json:"derived_from,omitempty"`
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
		EnvKeys:           nonNil(slices.Sorted(maps.Keys(st.Env))),
		BranchDeployRules: rules,
		DerivedFrom:       st.DerivedFrom,
	}
}
