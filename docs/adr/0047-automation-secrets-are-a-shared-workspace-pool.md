# Automation secrets are one shared workspace pool, readable by every automation

Secrets are set once per workspace, encrypted at rest, write-only after save,
and delivered to every automation as `ctx.secrets.NAME` — the GitHub Actions
shape, not per-automation isolation.

The consequence was weighed and accepted: anyone holding `automations:write`
can read every secret in the workspace by writing an automation that prints
one. `automations:write` is therefore granted as repo-collaborator trust, the
same level of trust as being able to add a workflow to a repository.
Per-automation secret isolation stays possible later; it was not worth
blocking the first version on.

Decided: 2026-08-28.
