# Each phone query declares the topics that refresh it

How a cached read on the phone stays current was spread over three places: the
hooks file held the key, the fetcher and the stale time; a table in
`useLiveEvents` mapped topic strings to keys; and a set of "detail keys" with a
payload sniffer decided which record a frame named. A query cached forever
(`staleTime: Infinity`) needed a row in that table, and the only guard was a
test that scanned the source for the pattern. Topics were plain strings, so a
typo was silent.

Decision: every phone query is one `defineQuery` call in its hooks file
(`native/src/lib/liveQuery.ts`) naming its key, its fetcher and `refreshes`,
the topics that refresh it and how each frame finds its entry:

- `"all"`: every cached entry under the key.
- `{ key: (payload) => … }`: the entry whose first key argument the payload
  names (a workspace, project or conversation).
- `{ record: (payload) => … }`: the record the payload names, matched by the
  id in the key or by the cached record's own id, so a ticket a link opened by
  its key is reached too (`recordQueries`).
- `{ patch: (client, payload) => … }`: the frame carries the change and is
  written into the cache with no request (chat messages).

Topic names and payload shapes are the generated event catalog's
(`@nexul/sdk/events`, ADR 0045), imported as types only: a misspelled topic or
a payload field the catalog does not carry is a type error. `tsconfig.json`
maps the specifier to `sdk/src/events.generated.ts`; Metro never resolves it,
because Babel erases a type import and `no-restricted-imports` rejects a value
import, so no SDK code reaches the app. Jest maps the same path, so tests can
dispatch the catalog's own fixtures.

Two things are derived from the definitions instead of kept by hand:

- The live dispatcher. Each definition registers as its module loads, which is
  before anything can cache an entry under its key; a frame reaches exactly the
  queries that list its topic. A frame whose payload lacks the field falls back
  to every entry under the key. The one rule outside the definitions stays in
  `useLiveEvents`: a change to the viewer's own access refetches everything.
- The stale time. `untilPushed: true` caches a query until a topic refreshes it
  and refetches it only on foreground and reconnect, because the socket is
  closed in the background and frames sent then are lost. A definition with
  `untilPushed` and no topic is a type error, and throws if cast past it.
  Everything else keeps TanStack's default and refetches when a screen mounts.

`untilPushed` is a claim that the listed topics cover every change to the
data, which only a person can make, so it stays opt-in rather than following
from having any topic at all. Hidden tabs still refresh on foreground: making
them wait until shown would need TanStack's `subscribed`, which turns every tab
switch into a remount and refetches each stale-time-zero query on it, so it
waits until more of the tabs' queries are cached until pushed.

Decided 2026-10-09.
