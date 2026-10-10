# 19 — Share a computer: Run agents

**Status:** ready-for-agent

**Blocked by:** permission-overrides 12, 15, 18

Read first: `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/react-guide.md`,
`practices/testing.md`, ADRs 0063, 0102, 0111, 0146, 0148, the spec (Later: sharing a computer, Access and
privacy), `.scratch/permission-overrides/spec.md` (Computers) and its ticket 12.

The Run agents permission, its storage and its checks come from the computer rules of permission-overrides
ticket 12. This ticket is the runner-side work for it: the starter's own identity for each run, the attribution
line, the warning, and revoking a running turn.

## Design (owner, 2026-10-10)

"Run agents" runs with the **starter's own** Nexul identity. Each run a grantee (Bob) starts on the owner's
(Alice's) computer carries a short-lived Nexul token minted for Bob, expiring with the run. The run's calls to
Nexul over MCP use that token, never the owner's token that setup wrote into the providers. Bob's agent sees
only what Bob can see in Nexul, and never holds Alice's login. Alice's own runs on her computer are unchanged.

Every place the run appears shows **"Run by Bob using Alice's <computer name>"**: the trail, the thread it
lands in, and the computer owner's run log (ticket 18).

## What to build

- First, confirm against T3 Code how one thread can be given its own MCP credential (per-thread configuration,
  environment or header for the Nexul MCP server) and record the answer in an ADR. If T3 Code cannot give one
  thread its own credential, stop and report to the owner; do not ship "Run agents" with the owner's token.
- Mint the starter's per-run token at the start of a grantee's run, bound to the starter's identity and the
  run, expiring when the run ends or at the run's longest allowed time, and revoked at the end. It is never
  written into the providers' config files or kept on the computer after the run.
- Honour the Run agents permission of ticket 12's computer check in target resolution, the run dialog and
  project links (amends ADR 0102): a grantee picks "<owner>'s <computer> (shared)" and, with this permission,
  its T3 projects and models. An offline computer fails at once, as for the owner (spec, decision 18).
- Show the attribution line in the trail, the thread and the owner's run log. The owner's run log lists who,
  when, what and how it ended, and still never shows the transcript (spec, sharing rules).
- Revoking Run agents, through the computer rule change, interrupts a grantee's running turn at once (ticket
  18's revoke path).
- The sharing dialog (ticket 18) says plainly what Run agents hands over, including that the agent runs as the
  owner's OS user on the owner's computer.

Open technical detail: an agent running as the owner's OS user can still read the owner's stored Nexul token
from the providers' config files, since it is a file in the owner's account. Decide in this ticket whether to
close that (for example, setup stops leaving a long-lived token where an agent can read it, so even the owner's
runs use per-run tokens), or to keep it and say so in the dialog. Either way record the choice in the ADR.

## Acceptance criteria

- [ ] `TestRunAgents_GranteesAgentCannotReadAProjectOnlyTheOwnerCanOpen`: Bob's run on Alice's computer, over
      MCP, gets not found for a project only Alice can open, and lists the projects Bob can open.
- [ ] `TestRunAgents_RunTokenIsTheStartersAndExpires`: the token names Bob, stops working when the run ends,
      and is never written to the computer's provider config.
- [ ] `TestRunAgents_NeedsThePermission`: a rule with See and Run commands but not Run agents cannot start an
      agent run, and a workspace Owner without a rule naming them gets not found.
- [ ] The attribution line "Run by Bob using Alice's <computer name>" shows in the trail, the thread and the
      owner's run log, and an owner's own run shows no such line.
- [ ] Revoking Run agents interrupts a running grantee turn and refuses the next start, with no cache wait.
- [ ] `TestRunAgents_DoesNotShowTheOwnerTheTranscript`.
- [ ] The ADR for the decision is written, and the sharing dialog matches the open technical detail's outcome.
