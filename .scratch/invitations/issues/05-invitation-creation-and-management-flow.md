# 05 — What should invitation creation and management feel like?

**Type:** prototype
**Status:** open
**Blocked by:** 02, 03

## Question

Build a throwaway prototype of the inviter-facing flow and use it to choose
the final interaction model:

- Where invitation management lives when one link may span several
  workspaces: instance settings, a workspace Members page, or a shared dialog
  reachable from both.
- How the inviter adds workspaces, picks one role per workspace, and optionally
  sets workspace-wide permission overwrites without turning the dialog into a
  permission matrix.
- How the one-day and seven-day choices are presented, with seven days as the
  default.
- The creation result that reveals the link once, makes copying unmistakable,
  and explains that a lost link must be revoked and replaced.
- The active-invitation list: enough non-secret information to identify,
  inspect, and revoke a link without ever showing its token again.
- How automatic deletion appears when the creator loses authority in one of
  the included workspaces.

Use real workspace, role, and permission names from Nexul. Explore genuinely
different flows, let the owner choose by using them, then record the chosen
behavior rather than production code.
