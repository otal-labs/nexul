# A member may view Nexul as another, read only, in one workspace

Whether a role gives a person too much or too little is only visible from where they stand. An Owner setting up the
Access page could read the grids and the explain sentences, but could not see the board, the docs list or the chat a
Client actually gets, and had to sign in as someone else to find out.

Decision: a member who holds **View Nexul as another member** (`members:view_as`, in the `members` domain, workspace
area) can view Nexul as another member of one workspace, read only. The web sends `X-View-As-User` and
`X-View-As-Workspace` on every HTTP request while viewing, and the server answers each as that member, so every access
check, SQL list filter and `/api/auth/me` answer runs as the target. There is no second evaluator that could disagree
with enforcement.

- **The check** is one function, used by the start route and by every request in a view. The viewer holds the bit in
  that workspace. The target is a member of it, is not the viewer, and does not hold the Owner role, which sees
  everything and leaves nothing to check. The union of the target's role permissions there is within what the viewer
  could give (ADR 0088, `access.GrantablePermissions`), so a viewer never sees through a role they could not hand out.
- **Starting** is `POST /api/workspaces/{id}/members/{userId}/view-as`, run as the real caller. Its audit row is the
  record of who viewed as whom, and the only one: nothing else in a view writes.
- **Headers** are honoured only on a browser session. A personal access token, an automation or an integration that
  sends them gets a 400. Sending one of the two is a 400. A request the check refuses is a 403 with `X-View-As:
  refused`, never a 401, so the web leaves the mode instead of signing the viewer out.
- **Writes are off.** Every request that is not a GET, HEAD or OPTIONS is a 403 ("You're viewing as <name>; changes
  are off"), except the POSTs that only read: mentions resolve, ticket label colours and the browser log relay.
- **Personal things are not found.** Sign-in state, tokens and sessions (`/api/auth/` but `GET /api/auth/me`), personal
  computers (`/api/pairing/`, ADR 0146: no bypass reaches them, and `access` answers no computer permission to an
  actor with a viewer either), the inbox, and a connector's sign-in.
- **One workspace.** Another workspace's routes are not found. In `access`, membership of any other workspace is not
  found and the person's workspace list holds only the viewed one, applied beside the request memo (ADR 0135), never
  written into it, because that memo also holds the check's own reads as the viewer. The switcher lists one
  workspace, and `/api/auth/me` reports instance permissions from the target's roles in that workspace alone.
- **Members-only conversations** (DMs and private channels) show in a view only when the viewer could read them too,
  in the list, in a direct read and in unread counts. A DM between the target and someone else stays hidden.
- **Nothing live.** The sockets do not take the headers and the web keeps them closed in a view: a socket would mark
  the target online and dial their computers.
- **No template role carries the bit.** Owners hold it and can give it to a role.

The trade-offs: a holder reads what the target reads in that workspace, including what a rule naming the target opens
beyond the holder's own reach, which holding-what-you-give over roles cannot catch. The view is a snapshot, so a change
elsewhere shows after a reload. Writes are off rather than performed as the target, so a flow that needs a write
(opening a form that saves a draft) cannot be previewed. Each request in a view pays the check's reads again, memoised
within that request only.

Rejected: viewing as a role, which misses per-person rules; performing writes as the target, which is actions taken in
someone else's name; an Owner-only feature with no bit, which could not be handed to an Administrator; and live push in
a view, for the reason above.

Builds on ADR 0088, ADR 0135, ADR 0146, ADR 0148 and ADR 0152. Decided 2026-10-19.
