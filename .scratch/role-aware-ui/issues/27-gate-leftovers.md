# 27: Three gates that still answer role-wide

Type: task
Status: open
Blocked by: None

Read first: `AGENTS.md`, `CONTEXT.md` (Machine, Locked action, Source), ADR 0151, ADR 0152, `practices/react-guide.md`.

## What was seen

Left over from tickets 11 and 24, found on 2026-10-14:

1. **Wizard deploy buttons skip the machine.** `WizardEnvStep` and `WizardServiceStep` read `deploys:write` role-wide. A person without Deploy to on the chosen machine can reach a live deploy button and be refused after the click. The stack page is fixed (ticket 24) because explain folds in the machine, but the wizard has no stack answer before the stack exists.
2. **A ticket's attachments follow the selected project.** On the ticket page, `AttachmentsSection` gates on the project selected in the sidebar, not the ticket's own project. A ticket opened from the Inbox in another project can show the wrong locked or live state.
3. **The per-role permissions panel ignores the machine.** `ExplainRole`, the panel's per-role view, doesn't fold Deploy to on the stack's machine into `deploys:write`. It can say a role may deploy a stack while the person's own answer says no.

## Acceptance criteria

- [ ] The wizard's deploy buttons lock with the machine's reason when the person holds no Deploy to on the chosen machine.
- [ ] A ticket's attachments gate on the ticket's own project.
- [ ] The per-role view agrees with the person's answer about a stack's machine.
