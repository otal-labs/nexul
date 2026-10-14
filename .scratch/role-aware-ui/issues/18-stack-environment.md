# 18: Edit a stack's environment, with secrets masked for readers

Type: task
Status: resolved
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- The Environment section on the stack page: applies on the next deploy, with Deploy now.
- Reveal for writers.
- The server masks secret values for readers.
- Check the automations SDK and the phone for value reads.
- Update `stacks-and-deploys.md`.

## Acceptance criteria

- [ ] A reader's `GET` carries no secret values.
- [ ] A writer can reveal and edit.
- [ ] Removing or blanking a key works.

## Answer

- The stack page has an Environment section: rows of name and value, values dots until a writer clicks Reveal values, Add variable, remove a row or empty a value, then Save. Deploy now sits beside Save and waits for the save.
- `GET /api/stacks`, `GET /api/stacks/{id}` and a stack's branch deployments send everyone without `stacks:write` on that stack the keys with empty values and `env_masked: true`, branch rule overrides included. A stack's own rules decide it, in the list too.
- `service.created` and `service.updated` carried every value to every reader's live socket, webhook and automation. They now carry the keys only, with `env_masked` beside the old fields.
- MCP already returned keys only and `stack_update` sets and unsets single variables; nothing there changed beyond one description.
- Nothing in the SDK's default automations or the phone app reads a value.
- The web follows `service.updated`, so another person's save shows up.
