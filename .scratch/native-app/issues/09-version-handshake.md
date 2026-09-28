# 09 — Version handshake between app and server

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

How does the app know its server is new enough? Which endpoint and fields the server exposes (version only, or a capability list), how the app compares them given beta tags like `v0.2.0-beta.9`, where the app's minimum server version is declared and bumped, and what the refusal screen says and offers.

## Answer

Decided 2026-09-28 (the owner delegated the remaining phone-side decisions).

- **Public `GET /api/about`** returns only `{"product": "nexul", "version": "<tag>"}`.
  It also confirms that a typed address is a Nexul server. The signed-in
  `/api/version` stays as it is.
- **The app carries one constant, its minimum server version**, bumped only in
  the change where the app starts relying on a new server capability. No
  capability list: the owner's rule is that the server is always the newer side.
- **Comparison** is semantic-version precedence: numeric pre-release parts
  compare as numbers (`beta.10` > `beta.9`) and a release beats its betas. A
  non-release server version (`dev`) always passes.
- **When:** before a connect code is spent, on every launch, and on return to
  the foreground.
- **Refusal screen:** "<host> runs <version>. This app needs <min> or newer. Ask
  whoever runs it to upgrade." with Retry. A signed-in session is kept behind
  it.
- **The HTTP API is additive:** never remove or rename a route or field that a
  released app may call. Recorded as an ADR with the `/api/about` build.
