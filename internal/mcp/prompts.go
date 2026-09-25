package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// prompt is a user-invoked workflow template; render only runs once every required argument is present.
type prompt struct {
	name, title, description string
	args                     []promptArg
	render                   func(args map[string]string) string
}

type promptArg struct {
	name, description string
	required          bool
}

func (p prompt) sdkPrompt() *sdk.Prompt {
	out := &sdk.Prompt{Name: p.name, Title: p.title, Description: p.description}
	for _, a := range p.args {
		out.Arguments = append(out.Arguments, &sdk.PromptArgument{Name: a.name, Description: a.description, Required: a.required})
	}
	return out
}

func (p prompt) handler() sdk.PromptHandler {
	return func(_ context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
		args := req.Params.Arguments
		for _, a := range p.args {
			if a.required && args[a.name] == "" {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: fmt.Sprintf("prompt %s needs the %s argument", p.name, a.name)}
			}
		}
		return &sdk.GetPromptResult{
			Description: p.description,
			Messages:    []*sdk.PromptMessage{{Role: "user", Content: &sdk.TextContent{Text: p.render(args)}}},
		}, nil
	}
}

func workflowPrompts() []prompt {
	return []prompt{
		{
			name:        "create_ticket_from_doc",
			title:       "Ticket from a doc",
			description: "Turn a doc into one scoped ticket.",
			args:        []promptArg{{name: "doc_id", description: "The source doc's id.", required: true}},
			render: func(a map[string]string) string {
				return fmt.Sprintf("Read doc %s with doc_get. Then create one ticket scoped to the work it describes with "+
					"ticket_create: propose a title and a body that summarizes the task, and keep the ticket limited to that "+
					"doc. project_get lists the project's ticket types and statuses.", a["doc_id"])
			},
		},
		{
			name:        "deploy_and_watch_stack",
			title:       "Deploy and watch a stack",
			description: "Deploy a stack and watch its services until they are healthy or the deploy fails.",
			args:        []promptArg{{name: "stack_id", description: "The stack's id, from stack_list.", required: true}},
			render: func(a map[string]string) string {
				return fmt.Sprintf("Deploy stack %s with stack_deploy, which builds from the stack's ref or redeploys its "+
					"last image. Then poll deploy_get with the returned deploy id, and stack_get for its services, until "+
					"every service is healthy or the deploy fails. If it fails, read the log tail deploy_get returns and "+
					"say what went wrong.", a["stack_id"])
			},
		},
		{
			name:        "investigate_failure",
			title:       "Investigate a failed deploy",
			description: "Investigate a failed deploy end to end and propose a fix as a ticket.",
			args:        []promptArg{{name: "deploy_id", description: "The failed deploy's id.", required: true}},
			render: func(a map[string]string) string {
				return fmt.Sprintf("Investigate failed deploy %s. Read it and its log with deploy_get, check the stack's "+
					"services with stack_get and the topology with topology_get, and, if you are an instance admin, look "+
					"for related failed events with dead_letter_list. Then propose a fix as a new ticket with "+
					"ticket_create.", a["deploy_id"])
			},
		},
		{
			name:        "ship_repository",
			title:       "Ship a repository",
			description: "Take a repository from zero to a deployed, reachable stack: the project wizard's own steps, in order.",
			args: []promptArg{
				{name: "owner", description: "The repository's owner.", required: true},
				{name: "repo", description: "The repository's name.", required: true},
				{name: "project_id", description: "The project to create the stack in.", required: true},
				{name: "machine", description: "The machine to deploy the stack on.", required: true},
				{name: "hostname", description: "A hostname to expose the stack at, if it should be reachable."},
			},
			render: func(a map[string]string) string {
				text := fmt.Sprintf("Ship %[1]s/%[2]s to project %[3]s on machine %[4]s. Call repository_list to confirm "+
					"the repository is visible, then repository_scan with owner %[1]q and repo %[2]q to get its deploy "+
					"candidates and default branch. Pick the best candidate (compose over a standalone Dockerfile when both "+
					"exist) and call stack_create with project_id %[3]q, machine %[4]q, that candidate, this repository and "+
					"branch as the build source, and deploy set to true. Poll deploy_get and stack_get until every service is "+
					"healthy.", a["owner"], a["repo"], a["project_id"], a["machine"])
				if a["hostname"] == "" {
					return text
				}
				return text + fmt.Sprintf(" Once healthy, call exposure_create to route %s to the reachable service's "+
					"container and port from the scan result.", a["hostname"])
			},
		},
	}
}
