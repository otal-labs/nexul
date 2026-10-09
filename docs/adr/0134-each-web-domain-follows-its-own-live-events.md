# Each web domain follows its own live events

The web client routed live frames through two central tables: one of payload handlers, and one mapping each topic to
the query keys it invalidated. The live layer had to import thirty domains' cache keys, a domain could not follow its
own events without exporting its keys to it, and most topics invalidated a key prefix, so a play changed in one
workspace refetched every workspace's plays and a deploy anywhere refetched the open deploy and its log.

Decision: a domain's hooks file exports one live follower beside its keys and queries, and `useLiveEvents` only
connects, routes each frame to every follower of its topic, and keeps the rule for the viewer's own permissions.

- A follower is a `LiveFollower` (`lib/live.ts`): per topic, a function of the frame's payload and a `Live` holding the
  query client and the router as it stands when the frame lands. It may return a promise for its work.
- Several domains may follow one topic, each for its own cache: `doc.moved` moves the doc in the doc views and
  refetches only the inboxes with a row about it. A follower calls another domain's cache operation
  (`ticketChanged`, `stageMoved`) when only it knows what changed, never another domain's keys.
- Scoping: a frame that carries the entity patches it into the views holding it; otherwise a follower invalidates
  only the queries named by the ids in the payload and the cached lists that hold that entity, read from their data.
  A list a row would move in, or a row no list holds yet, refetches for the server's order.
- When a frame names no id a view is keyed by (a deploy names no stack, a message no workspace, a new notice
  nothing), the follower falls back to every view of that kind. That gap is the server's to close, additively
  (ADR 0044).
- The set of followed topics is the union of the followers' topics, which `hooks/liveTopics.test.tsx` checks against
  the topics the server pushes.

The trade-off: the list of followers in `useLiveEvents` is one more place a new domain registers, and a follower
reading cached data to decide what holds an entity can miss a view it has not loaded, which is the view that is not on
screen. In exchange a domain's live behaviour sits beside its cache, and the main topics send none or one request per
frame instead of one per cached view.

Rejected: keeping the central tables and adding ids to them, which leaves the live layer knowing every cache layout;
and each domain opening its own socket subscription from a hook, which ties following to what is mounted.

Decided 2026-10-09.
