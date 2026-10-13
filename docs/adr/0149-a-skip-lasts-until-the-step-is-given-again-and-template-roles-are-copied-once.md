# A skip lasts until the step is given again, and template roles are copied once

Status: accepted.

The Welcome wizard shows each person the steps ticked on any role they hold, and the account menu shows a dot while
one is neither done nor skipped. Two choices in it surprise a reader of the code.

**A skip lasts until the step is given again.** A skip is stored per person per step (`welcome_step_states`). It is
cleared, and the dot comes back, only when the step is given to the person again: a role they hold gains the tick
(`role.updated` carries `welcome_steps_added`), or they are given a role that ticks it. Nothing else clears it, not a
new sign-in, not the step being undone (a revoked GitHub link makes the step left, which a skip still silences).
A skip that lasted forever would leave an admin no way to hand a struggling person the step again; a skip that
expired on its own would bring the dot back for no reason the person could see. Giving the step is the one act that
means "do this now".

**Template roles are copied once.** A new workspace gets a copy of each template role beside its Owner role,
permissions and welcome ticks included, and the copy is never rewritten. Editing a template role, resetting a shipped
one, or deleting an added one changes only workspaces created afterwards. Following the template live would change
what existing members may do whenever the instance template moved, without anyone in the workspace editing a role;
copying once keeps a workspace's roles its own, as play instructions and ticket body templates already are
(`CONTEXT.md`, Instance template). Existing workspaces get no template roles.

The Owner role's steps (Connect GitHub and Your computer) are fixed in code and never stored, like the rest of the
Owner role, which stays uneditable.
