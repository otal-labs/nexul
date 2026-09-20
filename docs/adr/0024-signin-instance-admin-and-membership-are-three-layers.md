# Sign-in, instance admin, and workspace membership are three separate layers

An instance hosts many workspaces, and three different things decide what a
person can reach:

1. **Sign-in is instance-wide.** The GitHub-login allowlist is the only
   sign-in boundary; there is no per-workspace allowlist. Inviting somebody
   to a workspace therefore never grants them sign-in access — an invite for
   a login that isn't allowlisted is rejected outright and points at the
   allowlist screen. Getting a new person onto the instance and getting them
   into a workspace stay two deliberate steps, because merging them would
   make every workspace admin an instance admin by accident.
2. **Instance-level capability is a bit on `User`** (`can_create_workspace`),
   deliberately *outside* the `<domain>:<action>` grid of ADR 0010. That grid
   is scoped to a workspace, and creating a workspace happens when the actor
   holds no membership anywhere yet, so it cannot be expressed in it. This
   replaced the old global `is_owner` flag.
3. **Role lives per membership**, not on the user: the same person can be
   Owner of one workspace and an ordinary member of another.

One consequence worth stating because it breaks a house rule: pending
invites for a login that has never signed in are resolved **synchronously**,
inside the request that creates the `User` row, not over the event bus. The
browser's very next call after login is the workspace list, and an async
fan-out would race it into showing an empty switcher. This is the one place
two domains' write paths meet in a single request, and the seam must stay a
thin synchronous call in both directions.

The Owner role is protected in the use-case layer — it cannot be invited,
assigned, removed, or demoted — and not only by leaving it out of the role
picker. The check looks redundant next to a UI that already hides the option,
which is exactly why it gets deleted by accident: the HTTP API is a separate
trust boundary from the browser, and without the server-side check any
member holding `members:write` could demote the Owner with a hand-written
request and leave the workspace with nobody able to bypass its own
permission overwrites.
