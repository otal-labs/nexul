# Can a support workspace give each client only their own conversation

Type: task
Status: resolved
Blocked by: None — can start immediately

## Question

Priority support is meant to run on the owner's own instance: a support workspace, each client invited
into it, each seeing only their own conversation with the owner. Can today's workspaces, invitations,
roles and conversation access do that with no new code? Check it on a demo instance. If yes, file a
Nexul task (label `ee`) for the owner to set it up and decide the response time; if no, say what is
missing so it can become a ticket.

## Answer

Yes, with no new code, but as one workspace per client rather than one shared support workspace. A
workspace's people list and @mention search show everyone in it (mention search reads the People of
the workspaces the searcher shares), so clients sharing one would see each other's people. Per client:
a workspace named after them, a role with chat and no project (with no project a member sees no
project, ADR 0148), its `#general` as the support chat. Direct messages also stay between their
people. Setup and the response-time promise are the owner's task, NEXUL-123 (label `ee`).
