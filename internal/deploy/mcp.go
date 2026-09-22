package deploy

import (
	"context"
	"fmt"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the deploy tool definitions: stack management, reads, and rollback.
func MCPTools(s *Service) []mcptool.Tool {
	tools := []mcptool.Tool{
		{
			Name:        "deploy_get",
			Description: "Fetch a single deploy record by id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Get(ctx, id)
			},
		},
		{
			Name:        "deploy_log",
			Description: "Fetch a deploy's streamed output as lines (seq, ts in unix milliseconds, phase, text), oldest first. Empty for a deploy that has not produced output yet.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Log(ctx, id)
			},
		},
		{
			Name:        "deploy_list",
			Description: "List all deploys, oldest first.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.List(ctx)
			},
		},
		{
			Name:        "deploy_list_by_service",
			Description: "List deploy history for one stack, newest first.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service": map[string]any{"type": "string"},
				},
				"required": []string{"service"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				service, err := mcptool.RequiredString(args, "service")
				if err != nil {
					return nil, err
				}
				return s.ListByService(ctx, service)
			},
		},
		{
			Name:        "deploy_list_by_status",
			Description: "List deploys in one state (pending, running, healthy, failed).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "string", "enum": []string{"pending", "running", "healthy", "failed"}},
				},
				"required": []string{"status"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				status, err := mcptool.RequiredString(args, "status")
				if err != nil {
					return nil, err
				}
				return s.ListByStatus(ctx, Status(status))
			},
		},
		{
			Name:        "deploy_cancel",
			Description: "Cancel a queued or running deploy; the record lands failed with reason 'cancelled'.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				if err := s.Cancel(ctx, id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "cancelling"}, nil
			},
		},
		{
			Name:        "service_list",
			Description: "List a stack's observed containers (name, image, status, networks, ports).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stack_id": map[string]any{"type": "string"},
				},
				"required": []string{"stack_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				stackID, err := mcptool.RequiredString(args, "stack_id")
				if err != nil {
					return nil, err
				}
				return s.ListServices(ctx, stackID)
			},
		},
	}
	tools = append(tools, stackTools(s)...)
	tools = append(tools, machineTools(s)...)
	return tools
}

// machineTools are the machine-import tools; list/discover live in the runner domain, which owns machines,
// but import writes stacks and so stays where its use-case lives: deploy.
func machineTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "machine_import",
			Description: "Adopt ticked machine-discovery selections (compose projects, standalone containers) as unmanaged stacks and their containers. Re-import updates by container name; never deletes.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"machine_id": map[string]any{"type": "string"},
					"project_id": map[string]any{"type": "string"},
					"stacks": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"project":    map[string]any{"type": "string"},
								"containers": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
							},
						},
					},
					"standalone": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"gateways":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"machine_id", "project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				machineID, err := mcptool.RequiredString(args, "machine_id")
				if err != nil {
					return nil, err
				}
				return s.Import(ctx, machineID, importRequestFromArgs(args))
			},
		},
	}
}

// importRequestFromArgs builds an ImportRequest from stack_import's args; malformed entries are skipped rather
// than rejected outright, matching the tolerant parsing the rest of this file uses for nested arrays.
func importRequestFromArgs(args map[string]any) ImportRequest {
	req := ImportRequest{
		ProjectID:  mcptool.OptionalString(args["project_id"]),
		Standalone: stringSliceFromArgs(anySlice(args["standalone"])),
		Gateways:   stringSliceFromArgs(anySlice(args["gateways"])),
	}
	for _, item := range anySlice(args["stacks"]) {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		req.Stacks = append(req.Stacks, ImportStackGroup{
			Project:    mcptool.OptionalString(m["project"]),
			Containers: stringSliceFromArgs(anySlice(m["containers"])),
		})
	}
	return req
}

func anySlice(v any) []any {
	raw, _ := v.([]any)
	return raw
}

// mcpTriggeredBy resolves the acting user for deploy provenance and marks the mcp source in the established
// TriggeredBy string (spec §9): "<user id>:mcp"; empty when no actor is attached to the call.
func mcpTriggeredBy(ctx context.Context) string {
	a, ok := actorFromArgs(ctx)
	if !ok || a.ID == "" {
		return ""
	}
	return a.ID + ":mcp"
}

// stackTools are the stack CRUD, deploy, and rollback tools.
func stackTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name: "stack_create",
			Description: "Create a stack, either from explicit fields or from a repository_scan candidate " +
				"(candidate: {kind, path, services}) plus machine and project_id, mirroring POST /api/stacks. " +
				"With a candidate, name defaults to the build source's repo name when omitted. Set deploy: true " +
				"to enqueue the first deploy immediately (needs a ref, from the candidate's build_source.branch " +
				"or the ref argument).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id":     map[string]any{"type": "string"},
					"name":           map[string]any{"type": "string"},
					"machine":        map[string]any{"type": "string"},
					"strategy":       map[string]any{"type": "string", "enum": []string{"compose", "run"}},
					"compose_path":   map[string]any{"type": "string"},
					"env":            map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
					"docker_network": map[string]any{"type": "string"},
					"ports":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"build_source":   buildSourceSchema(),
					"candidate":      candidateSchema(),
					"deploy":         map[string]any{"type": "boolean"},
					"ref":            map[string]any{"type": "string"},
				},
				"required": []string{"project_id", "machine"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				stack, declared, err := stackFromCreateArgs(args)
				if err != nil {
					return nil, err
				}
				created, err := s.CreateStack(ctx, stack, declared)
				if err != nil {
					return nil, err
				}
				if err := deployIfRequested(ctx, s, created, args); err != nil {
					return nil, err
				}
				return created, nil
			},
		},
		{
			Name:        "stack_deploy",
			Description: "Deploy a stack: build from a ref (repo-backed stacks) or redeploy a specific image. Enqueues exactly as the UI's deploy button does; rejects a second active deploy on the same stack.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stack_id": map[string]any{"type": "string"},
					"ref":      map[string]any{"type": "string"},
					"image":    map[string]any{"type": "string"},
				},
				"required": []string{"stack_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				stackID, err := mcptool.RequiredString(args, "stack_id")
				if err != nil {
					return nil, err
				}
				return s.Deploy(ctx, DeployRequest{
					StackID:     stackID,
					Image:       mcptool.OptionalString(args["image"]),
					Ref:         mcptool.OptionalString(args["ref"]),
					TriggeredBy: mcpTriggeredBy(ctx),
				})
			},
		},
		{
			Name:        "stack_get",
			Description: "Fetch a stack by id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.GetStack(ctx, id)
			},
		},
		{
			Name:        "stack_list",
			Description: "List stacks for a project.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
				},
				"required": []string{"project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				return s.ListStacks(ctx, projectID)
			},
		},
		{
			Name:        "stack_update",
			Description: "Update a stack by id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":             map[string]any{"type": "string"},
					"project_id":     map[string]any{"type": "string"},
					"name":           map[string]any{"type": "string"},
					"machine":        map[string]any{"type": "string"},
					"strategy":       map[string]any{"type": "string", "enum": []string{"compose", "run"}},
					"compose_path":   map[string]any{"type": "string"},
					"env":            map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
					"docker_network": map[string]any{"type": "string"},
					"ports":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"build_source":   buildSourceSchema(),
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				stack, err := stackFromArgs(args)
				if err != nil {
					return nil, err
				}
				stack.ID = id
				return s.UpdateStack(ctx, stack)
			},
		},
		{
			Name:        "stack_delete",
			Description: "Delete a stack. Deploy history is kept; topology drops the stack's service nodes.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				if err := s.DeleteStack(ctx, id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "deleted"}, nil
			},
		},
		{
			Name:        "stack_rollback",
			Description: "Roll a stack back to its last healthy image (one-click rollback).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stack_id": map[string]any{"type": "string"},
				},
				"required": []string{"stack_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				stackID, err := mcptool.RequiredString(args, "stack_id")
				if err != nil {
					return nil, err
				}
				return s.Rollback(ctx, stackID, mcpTriggeredBy(ctx))
			},
		},
	}
}

func buildSourceSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"repo_owner":   map[string]any{"type": "string"},
			"repo_name":    map[string]any{"type": "string"},
			"branch":       map[string]any{"type": "string"},
			"dockerfile":   map[string]any{"type": "string"},
			"compose_path": map[string]any{"type": "string"},
		},
	}
}

func stackFromArgs(args map[string]any) (Stack, error) {
	var stack Stack
	var err error
	if stack.ProjectID, err = mcptool.RequiredString(args, "project_id"); err != nil {
		return stack, err
	}
	if stack.Name, err = mcptool.RequiredString(args, "name"); err != nil {
		return stack, err
	}
	if stack.Machine, err = mcptool.RequiredString(args, "machine"); err != nil {
		return stack, err
	}
	stack.ComposePath = mcptool.OptionalString(args["compose_path"])
	stack.DockerNetwork = mcptool.OptionalString(args["docker_network"])
	if raw, ok := args["ports"].([]any); ok {
		stack.Ports = stringSliceFromArgs(raw)
	}
	if raw, ok := args["env"].(map[string]any); ok {
		env := make(map[string]string, len(raw))
		for k, v := range raw {
			env[k] = fmt.Sprint(v)
		}
		stack.Env = env
	}
	if raw, ok := args["build_source"].(map[string]any); ok {
		stack.BuildSource = buildSourceFromArgs(raw)
	}
	strat, err := mcptool.RequiredString(args, "strategy")
	if err != nil {
		return stack, err
	}
	stack.Strategy = Strategy(strat)
	return stack, nil
}

func buildSourceFromArgs(raw map[string]any) *BuildSource {
	return &BuildSource{
		RepoOwner:   mcptool.OptionalString(raw["repo_owner"]),
		RepoName:    mcptool.OptionalString(raw["repo_name"]),
		Branch:      mcptool.OptionalString(raw["branch"]),
		Dockerfile:  mcptool.OptionalString(raw["dockerfile"]),
		ComposePath: mcptool.OptionalString(raw["compose_path"]),
	}
}

func stringSliceFromArgs(raw []any) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// stackFromCreateArgs builds stack_create's input: a candidate (spec §6) when args carries one, otherwise the
// explicit-fields shape stack_create has always accepted.
func stackFromCreateArgs(args map[string]any) (Stack, map[string]Declared, error) {
	raw, ok := args["candidate"].(map[string]any)
	if !ok {
		stack, err := stackFromArgs(args)
		return stack, nil, err
	}
	return stackFromCandidateArgs(args, raw)
}

// stackFromCandidateArgs converts a repository_scan candidate (kind, path, services) into a Stack plus the
// per-container Declared map CreateStack persists services from (spec §6). Unlike the explicit-fields path,
// strategy and compose_path/docker_network come from the candidate's kind, not caller-supplied fields.
func stackFromCandidateArgs(args, candidate map[string]any) (Stack, map[string]Declared, error) {
	stack, err := stackBaseFromCandidateArgs(args)
	if err != nil {
		return stack, nil, err
	}
	declared, err := applyCandidateKind(&stack, args, candidate)
	if err != nil {
		return stack, nil, err
	}
	return stack, declared, nil
}

// stackBaseFromCandidateArgs fills the fields common to every candidate kind: identity, name (falling back to
// the build source's repo name), ports, and env. Strategy-specific fields are set by applyCandidateKind.
func stackBaseFromCandidateArgs(args map[string]any) (Stack, error) {
	var stack Stack
	var err error
	if stack.ProjectID, err = mcptool.RequiredString(args, "project_id"); err != nil {
		return stack, err
	}
	if stack.Machine, err = mcptool.RequiredString(args, "machine"); err != nil {
		return stack, err
	}
	if raw, ok := args["build_source"].(map[string]any); ok {
		stack.BuildSource = buildSourceFromArgs(raw)
	}
	stack.Name = mcptool.OptionalString(args["name"])
	if stack.Name == "" && stack.BuildSource != nil {
		stack.Name = stack.BuildSource.RepoName
	}
	if stack.Name == "" {
		return stack, fmt.Errorf("%w: name is required (the candidate has none of its own; set name or build_source.repo_name)", apperrs.ErrInvalid)
	}
	if raw, ok := args["ports"].([]any); ok {
		stack.Ports = stringSliceFromArgs(raw)
	}
	if raw, ok := args["env"].(map[string]any); ok {
		env := make(map[string]string, len(raw))
		for k, v := range raw {
			env[k] = fmt.Sprint(v)
		}
		stack.Env = env
	}
	return stack, nil
}

// applyCandidateKind sets stack's strategy fields from candidate's kind (compose or dockerfile) and returns the
// per-container Declared map CreateStack persists services from.
func applyCandidateKind(stack *Stack, args, candidate map[string]any) (map[string]Declared, error) {
	services, _ := candidate["services"].([]any)
	declared := make(map[string]Declared, len(services))
	switch mcptool.OptionalString(candidate["kind"]) {
	case "compose":
		stack.Strategy = StrategyCompose
		stack.ComposePath = mcptool.OptionalString(candidate["path"])
		for _, raw := range services {
			svc, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			name := mcptool.OptionalString(svc["name"])
			if name == "" {
				continue
			}
			declared[name] = declaredFromCandidateService(svc)
		}
	case "dockerfile":
		stack.Strategy = StrategyRun
		stack.DockerNetwork = mcptool.OptionalString(args["docker_network"])
		if len(services) > 0 {
			if svc, ok := services[0].(map[string]any); ok {
				// A dockerfile candidate is a stack of one; its single container is named after the stack's
				// slug (model.go), computed here the same deterministic way CreateStack derives it.
				declared[Slug(stack.Name, dnsLabelMaxLen)] = declaredFromCandidateService(svc)
			}
		}
	default:
		return nil, fmt.Errorf("%w: candidate kind %q must be compose or dockerfile", apperrs.ErrInvalid, candidate["kind"])
	}
	return declared, nil
}

// declaredFromCandidateService maps a repository_scan candidate service onto the deploy domain's Declared shape.
func declaredFromCandidateService(svc map[string]any) Declared {
	d := Declared{
		Image:   mcptool.OptionalString(svc["image"]),
		EnvKeys: stringSliceFromArgs(anySlice(svc["env_keys"])),
	}
	if build, ok := svc["build"].(map[string]any); ok {
		d.Build = mcptool.OptionalString(build["dockerfile"])
		if d.Build == "" {
			d.Build = mcptool.OptionalString(build["context"])
		}
	}
	for _, p := range anySlice(svc["ports"]) {
		switch v := p.(type) {
		case float64:
			d.Ports = append(d.Ports, strconv.Itoa(int(v)))
		case string:
			d.Ports = append(d.Ports, v)
		}
	}
	return d
}

func candidateSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind": map[string]any{"type": "string", "enum": []string{"compose", "dockerfile"}},
			"path": map[string]any{"type": "string"},
			"services": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":  map[string]any{"type": "string"},
						"image": map[string]any{"type": "string"},
						"build": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"context":    map[string]any{"type": "string"},
								"dockerfile": map[string]any{"type": "string"},
							},
						},
						"ports":    map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
						"env_keys": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
				},
			},
		},
	}
}

// deployIfRequested enqueues stack_create's optional first deploy (spec §6's deploy: true); the ref comes from
// the top-level ref argument or, failing that, the stack's own build source branch.
func deployIfRequested(ctx context.Context, s *Service, created *Stack, args map[string]any) error {
	if !boolArg(args["deploy"]) {
		return nil
	}
	ref := mcptool.OptionalString(args["ref"])
	if ref == "" && created.BuildSource != nil {
		ref = created.BuildSource.Branch
	}
	if ref == "" {
		return fmt.Errorf("%w: ref (or build_source.branch) is required to deploy", apperrs.ErrInvalid)
	}
	_, err := s.Deploy(ctx, DeployRequest{StackID: created.ID, Ref: ref, TriggeredBy: mcpTriggeredBy(ctx)})
	return err
}

func boolArg(v any) bool {
	b, _ := v.(bool)
	return b
}
