# 12: Sweep: stacks, deploys, runners, machines, DNS and topology

Type: task
Status: resolved
Blocked by: 10

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

The same for research 01's stacks and deploys, runners and machines, DNS, gateways and exposures, and topology
tables. The Runners page opens for every member and lists the machines they see; its buttons follow the
`actions` each machine carries: Add runner, Import and removing a runner need `machines:write`, so a member who
only sees a machine gets them hidden or locked.

## Acceptance criteria

- [x] As ticket 11, for these areas.

## Answer

Swept: stack branch deployments (Tear down, by the base stack's rules), the deploy page's Cancel and Retry, a stack's
Attach repository (needs both `stacks:write` and `projects:write`), the Runners page (Add runner on the header and on
each machine, Import, Remove runner, the stack root, the machine menu's Delete), DNS (Set up DNS, Create gateway,
Delete gateway, Expose, Unexpose, Delete tunnel, Delete this record, the setup stepper) and topology (the External
button, the empty canvas's Add a node, the node dialog's Save and Delete, and what a reader's canvas may drag, connect,
delete or save).

A machine answers from the `actions` its card carries, through `machineAnswer` in `client-core/actionState.ts`.

Skipped: stack list rows and deploy list rows carry no write (Permissions… already follows the rule check), unused
runner codes are listed only for people who may revoke them, and research 01's runner permissions became machine
permissions (ADR 0151).
