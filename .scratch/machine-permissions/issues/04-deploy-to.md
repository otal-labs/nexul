# 04: Deploy to, on everything aimed at a machine and every deploy a person starts

Type: task
Status: open
Blocked by: 02

Read first: `AGENTS.md`, `CONTEXT.md`, `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2,
3, 8), `practices/mcp.md` (sections 4, 6, 7), `practices/react-guide.md` (F1 to F7 and the self-review),
`practices/testing.md` (sections 3, 8, 10), ADRs 0079, 0148, 0150, the spec (decisions 6, 8 and 9).

## What to build

- A stack or gateway's machine resolves by name to a machine; an unknown name is invalid input.
- **Aiming** takes Deploy to on the machine, beside the checks each path has now: `CreateStack` and
  `CreateStackWithOptions`, `UpdateStack` when the machine changes, branch deployments, Import, `CreateGateway`, the
  proxy and tunnel provisioning paths (`server/cmd/wire_dns.go`), and their MCP tools `stack_create`, `stack_update`,
  `machine_import` and `gateway_create`.
- **Deploying:** `Deploy`, retry and `stack_deploy` take Deploy to on the stack's machine when a person starts them.
  Deploys the server starts (a push, a merge, a branch deployment's update) pass as the server.
- **Stacks with no project:** seeing one takes See on its machine and `dns:read`; editing or deploying one takes
  Deploy to and `dns:write`. `stack_list` and the live stack frames follow.
- `MachinePicker` lists the machines the viewer may Deploy to. A stack page shows its machine's name as plain text,
  with no link, to someone who can't See it.
- Guide: `stacks-and-deploys.md`, `topology-and-dns.md`.

## Acceptance criteria

- [ ] `TestMachines_StackCreateRefusesAMachineYouCantDeployTo`, by name over MCP included.
- [ ] `TestMachines_TakenOffStopsTheNextDeploy` and `TestMachines_APushDeployStillRuns`.
- [ ] `TestMachines_AGatewayTakesDeployToAndDNSWrite`; an instance stack is not found without See and `dns:read`.
- [ ] Verified at 768, 1024 and 1440px: the picker, and a stack whose machine the viewer can't See.
