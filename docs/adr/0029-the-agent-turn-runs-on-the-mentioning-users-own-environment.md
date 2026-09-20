# An `@Agent` turn runs on the mentioning user's own paired environment, with only that user's permissions

`@Agent` is not a service account and Nexul holds no model credentials.
Mentioning the agent starts a turn on the T3 Code environment that the
mentioning user paired to their own account, and that turn reaches back into
Nexul through the MCP server authenticated by that same user's personal
access token. Every action the agent takes is therefore attributable to, and
bounded by, the person who invoked it — there is no shared identity that
accumulates permissions and no instance-wide API key to leak or bill. The
cost is that the feature is unavailable until a user pairs a computer, and a
mention with no usable pairing answers with an inline system note saying what
to fix rather than failing silently.

The backend is a deliberately minimal, T3-free interface — start a turn
(streamed cumulative snapshots plus a terminal result) and interrupt — so a
raw-model-API implementation can be added later without the chat pipeline
learning anything new. The integration pins no T3 release: it records the
harness version at pairing time and only warns, never blocks, when that
base release has moved since.

In-progress replies stream as **ephemeral** live-hub frames keyed by message
id and bypass the outbox entirely; only the final reply is persisted. The
outbox exists so a change and its event cannot disagree, and a half-written
sentence is neither.

Decided 2026-08-26.
