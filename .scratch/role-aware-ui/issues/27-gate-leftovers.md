# 27: Three gates that still answer role-wide

Type: task
Status: resolved
Blocked by: None

Read first: `AGENTS.md`, `CONTEXT.md` (Machine, Locked action, Source), ADR 0151, ADR 0152, `practices/react-guide.md`.

## What was seen

Left over from tickets 11 and 24, found on 2026-10-14:

1. **Wizard deploy buttons skip the machine.** `WizardEnvStep` and `WizardServiceStep` read `deploys:write` role-wide. A person without Deploy to on the chosen machine can reach a live deploy button and be refused after the click. The stack page is fixed (ticket 24) because explain folds in the machine, but the wizard has no stack answer before the stack exists.
2. **A ticket's attachments follow the selected project.** On the ticket page, `AttachmentsSection` gates on the project selected in the sidebar, not the ticket's own project. A ticket opened from the Inbox in another project can show the wrong locked or live state.
3. **The per-role permissions panel ignores the machine.** `ExplainRole`, the panel's per-role view, doesn't fold Deploy to on the stack's machine into `deploys:write`. It can say a role may deploy a stack while the person's own answer says no.

## Acceptance criteria

- [x] The wizard's deploy buttons lock with the machine's reason when the person holds no Deploy to on the chosen machine.
- [x] A ticket's attachments gate on the ticket's own project.
- [x] The per-role view agrees with the person's answer about a stack's machine.

## Answer

1. The wizard's Save & deploy (Environment) and Attach (Service, when no Environment step follows) gate
   `machines:deploy` first, with the machine list's own answer from `useDeployToAnswer`, so a person without Deploy to on
   the stack's machine, or who can't see it, sees them locked with `No Deploy to on machine <name>`. Create keeps its
   plain gate because its machine picker already offers only the machines the viewer may deploy to.
2. A ticket's attachments take `ticket.project_id`, so a ticket opened from the Inbox answers for its own project.
3. `ExplainRole` folds Deploy to on the stack's machine into `deploys:write` through the same `noDeployTo` as explain,
   reading the role alone: its bits and its own rules on the machine, never a rule naming a person.
4. Add machines is read from the auth `/me` through `useCanAddMachines`, for the picker's Add a runner link and the import
   wizard's Runners crumb; the `runners` area is gone.
