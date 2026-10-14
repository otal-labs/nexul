# 12: The owner wizard hands over to the Welcome wizard

Type: task
Status: resolved
Blocked by: 10

Read first: `AGENTS.md`, `CONTEXT.md` (Welcome wizard, Template role), `practices/react-guide.md`, ticket 05's answer (Owner hand-over), `setup-wizard.md` in the guide.

## What to build

- The owner wizard's "Set up T3 Code" step becomes "Workspace ready", which opens the Welcome wizard on the first
  step that isn't done.
- The hard block on a paired computer goes.
- The Welcome wizard's end button reads "Create your first project" for an owner who has none.

## Acceptance criteria

- [ ] A fresh owner can finish setup with no computer. The `onboarding.spec.ts` end-to-end flow is updated.
- [ ] Verified at 768, 1024, 1440.

## Answer

Built in #642: the owner wizard ends on Workspace ready and hands over to the Welcome wizard; no paired computer is needed to finish.
