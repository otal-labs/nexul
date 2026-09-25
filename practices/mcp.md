# MCP practices (Nexul)

The binding standard for the MCP server: the adapter in `internal/mcp/`, the
tool contract in `internal/platform/mcptool/`, and every domain's `mcp.go`.
`AGENTS.md` routes MCP work here; read it before the first edit. Each rule
carries the failure it prevents, so it extends to cases this file does not
list.

The protocol moves faster than memory. The current revision is 2026-07-28,
and field names and error codes changed in it, so a protocol question is
answered from the specification (Sources, at the end), never from recall.
Before any protocol-level change, check the specification's draft changelog
for a newer revision.

---

## 1. Where MCP code lives

- `internal/mcp/` is the adapter, and the only package that imports
  `github.com/modelcontextprotocol/go-sdk` (ADR 0067). It owns the transport,
  argument validation, the translation of errors into tool results, the
  server instructions, resources, prompts, and the tools that cross domains.
- `internal/platform/mcptool/` is the contract every domain declares its
  tools with. Domains import it, never the SDK, so an SDK upgrade touches one
  package and a domain stays free of protocol types.
- A domain's `mcp.go` declares that domain's tools: a typed input, one or more
  use-case calls, and a shaped output. It holds no business rule; a rule in
  `mcp.go` is a rule the HTTP gateway does not have (ADR 0019).
- A tool that composes two domains lives in `internal/mcp/composite/`,
  because domains never import each other; the package imports only the
  domains it composes, so it builds and tests on its own.

---

## 2. Protocol and transport

- **Stateless Streamable HTTP, one endpoint.** `POST /mcp` on the main
  listener, served by the SDK's handler in stateless mode with JSON
  responses. No session ids, no `GET` stream, no Server-Sent Events: nothing
  the server sends is a notification in flight, so a stream buys nothing and
  costs a code path. `GET` and `DELETE` answer `405`.
- **No stdio transport.** Every client the product documents connects over
  Streamable HTTP with a static `Authorization` header. A stdio bridge would
  need a credential on the user's machine and a second code path to keep
  conformant.
- **Revisions.** The server speaks 2026-07-28 per request, and the earlier
  revisions, which open with an `initialize` handshake, through the SDK's own
  negotiation. The SDK's default version list is used as-is; narrowing it
  needs an ADR, because a client that cannot negotiate simply stops working.
- **Conformance is the SDK's job, wiring is ours.** The SDK validates
  `MCP-Protocol-Version`, `Mcp-Method`, and `Mcp-Name`, maps error codes to
  HTTP statuses, and serves `server/discover`. Do not re-implement or patch
  around it; a wire-level bug is fixed by an SDK upgrade or an upstream
  issue. Three things are the adapter's job, because the SDK's defaults are
  wrong for this server: it sets the capabilities explicitly (tools,
  resources, and prompts, with no `listChanged` and no `logging`, which the
  nil default advertises, and `listChanged` would invite a subscription
  stream), it sets cache fields through `SetCacheable`, and its resource and
  prompt handlers return only JSON-RPC errors (section 8).
- **Server identity.** The implementation info carries the real build
  version from `internal/platform/version`, so a client's logs name the
  build it talked to.
- **Upgrading the SDK.** Read the release notes on every minor bump.
  Spec-compliance fixes ship behind `MCPGODEBUG` escape hatches that are
  usually removed one minor later, so a skipped note becomes a silent break.
  Nexul never sets `MCPGODEBUG`.
- **Cancellation.** The handler sets `PropagateRequestCancellation`, so a
  closed 2026-07-28 request cancels the tool's context; a legacy-revision
  request runs to completion. Every tool passes its `ctx` to the use-case,
  and every use-case honors it (`practices/go.md`, section 7).
- **Cache hints.** Lists (`tools/list`, `resources/list`,
  `resources/templates/list`, `prompts/list`) and `server/discover` are the
  same for every caller and fixed per process, so they carry a short public
  TTL. `resources/read` returns live content and carries `ttlMs: 0`,
  `cacheScope: "private"`, because a client honoring a long TTL would serve a
  doc for an hour after someone edited it. Both are set through
  `SetCacheable`, and a list that ever becomes filtered by the caller turns
  `private` in the same change.

---

## 3. Authentication, authorization, and security

- **Every request carries a bearer token.** A session token or a personal
  access token, read only from the `Authorization` header, checked on every
  request. There is no OAuth authorization server: MCP authorization is
  optional, and every documented client sends a static header. A `401`
  carries `WWW-Authenticate: Bearer`, and the OAuth discovery paths under
  `/.well-known/` answer `404`, so a client that probes for OAuth stops
  cleanly instead of parsing the web app's HTML.
- **The tool runs as the caller.** The adapter puts the authenticated actor on
  the context; the use-case decides what that actor may do, through the
  permission table. Authorization never lives in the adapter, in a tool
  description, or in annotations.
- **MCP is never weaker than the browser.** When an HTTP route gates an
  action (instance administration, "only the caller's own inbox"), the same
  gate applies to the tool. The durable fix is moving the gate into the
  use-case so both adapters inherit it; a gate that exists only in an HTTP
  handler is a defect.
- **An argument never names the actor.** A tool acts on "the caller's"
  notifications, drafts, or tokens from the context, never from a `user_id`
  argument, or any caller can act as anyone.
- **Foreign origins are refused.** Middleware answers `403` when a request
  carries an `Origin` header that is not the configured instance URL's
  origin, as the specification requires of every Streamable HTTP server; the
  SDK checks nothing by default, and comparing `Origin` with `Host` would
  pass a DNS-rebinding page. A request without `Origin` (every non-browser
  client) goes on to the bearer check. The SDK's loopback guard is off,
  because it would reject a reverse proxy that forwards over `127.0.0.1`
  with the public host name, and bearer authentication on every request
  already makes rebinding useless.
- **Rate limits.** Tool calls are rate-limited per actor in the adapter, and
  a limited call returns a tool error that says how long to wait, so an agent
  stuck in a loop cannot monopolize the single SQLite writer.
- **Secrets never enter the model's context.** A tool result carries no
  secret, ciphertext, or credential-bearing URL. A tool whose purpose is to
  reveal or mint a credential is its own tool, returns it once, and says so
  in its description; every other result for the same object has no field
  for it. A domain type that holds a secret also tags it `json:"-"`, so a
  raw type can never leak it by accident.
- **Tool output is untrusted data.** Docs, tickets, chat, memories, pull
  request text, and logs are authored by people or systems other than the
  caller. Never splice user-authored text into server instructions, tool
  descriptions, or prompt templates.
- **Never forward the caller's token.** Upstream calls (the git provider,
  Cloudflare, a harness) use Nexul's own connector credentials.
- **Log the call, not the payload.** A tool call logs its name, the actor, the
  outcome, and the duration, under the request's `trace_id`. Arguments and
  results are never logged; they carry document bodies and secrets.

---

## 4. Shaping the tool surface

The tool list is the product's interface for agents. Its size and naming
decide whether an agent picks the right tool.

- **Tools are a budget.** The server stays well under 100 tools. At least
  one client caps an agent at 100 tools across all its servers, so 100 is a
  ceiling, not a target; every tool definition costs context in every
  session, and past a few dozen, agents start choosing the wrong tool. A new capability extends an existing tool (a filter, an
  optional field, a patch field) before it earns a new tool, and a pull
  request that adds a tool says why no existing tool could carry it.
- **One tool per task, not per use-case or endpoint.** Group by what an agent
  is trying to do (ADR 0068):
  - One-field setters fold into a patch-style `update` on their object
    (an omitted field keeps its value; section 6).
  - List variants fold into one `list` with optional filters.
  - A child collection (a project's repositories, a ticket's labels) is read
    on its parent's `get` and changed through `add_*` and `remove_*` fields on
    the parent's `update`.
  - A reversible toggle (archive and restore, enable and disable) is a field
    on `update`, so the way back is as visible as the way in.
  - Small related reads return together: a project's `get` includes its
    statuses, categories, and ticket types, because an agent needs all three
    before it can file a ticket.
- **Never mix reading and changing in one tool.** Annotations are per tool, so
  a tool that both reads and deletes is annotated as destructive and every
  read of it needs confirmation. Keep `list` and `get` read-only, and give
  `delete` its own tool so a client can confirm exactly that.
- **Starting work is its own tool.** A tool that starts an execution (a
  deploy, a play run, an upgrade) is never a side effect of an update,
  because an execution is neither idempotent nor local and needs its own
  hints and its own confirmation. Its name says so, and the execution
  records that MCP started it (ADR 0049).
- **Every capability stays reachable.** Parity with the web app (ADR 0019)
  means every capability the UI offers is reachable through some tool, not
  that each one has a dedicated tool. Adding a UI capability without a way
  to reach it over MCP is half shipped (`AGENTS.md`, Hit every surface).

---

## 5. Naming

- **`<object>_<verb>`, snake_case.** `ticket_update`, `stack_deploy`,
  `memory_list`. The object comes first so a sorted list or a tool search
  keeps every operation on one object together.
- **The object is a `CONTEXT.md` noun.** `trail`, not `run_history`; `stage`,
  not `kind`; `machine`, not `target`. Two-word objects join with `_`
  (`dns_record`, `ticket_type`).
- **The verb comes from a small set.** `list`, `get`, `create`, `update`,
  `delete`, plus a domain verb only for an action that is not one of those
  (`deploy`, `cancel`, `run`, `upgrade`, `scan`, `discover`, `import`,
  `replay`, `search`, `post`, `pair`, `report`). Two tools never use
  different verbs for the same operation.
- **Characters and length.** `[a-z0-9_]`, unique, and at most 52
  characters, so the client's `mcp__nexul__` prefix plus the name fits a
  64-character limit. The specification allows dots and dashes; some model
  APIs reject dots, and one style keeps names predictable. No `nexul_`
  prefix: the client already adds one.
- **Parameters.** A tool's own object is `id`; any other object is
  `<object>_id` (`project_id`, `status_id`), so an agent never guesses which
  object an id belongs to. Where a human key exists, the
  tool accepts it wherever it accepts the id (`REF-102` for a ticket).
  Parameter names use the same `CONTEXT.md` vocabulary as tool names.
- **Prompts are not tools.** Prompt names describe a workflow
  (`investigate_failure`) and never read like a tool's name, because clients
  list prompts as slash commands next to tools, and a look-alike makes the
  user's command read as the agent's tool.

---

## 6. Declaring a tool

- **Typed input.** A tool declares its arguments as a Go struct passed to
  `mcptool.New`. The input schema is inferred from the struct: a field
  without `omitempty` or `omitzero` is required, and unknown keys are
  rejected. Before a
  tool runs, the adapter validates the arguments against that schema, so a
  tool never hand-checks types or required keys; the use-case still
  validates business rules.
- **Every parameter is described.** A `jsonschema:"..."` tag on every field
  states its meaning, its format, and a short example value where the format
  is not obvious (`"REF-102"`, `"2026-09-25T10:00:00Z"`). The schema is the
  only documentation the model has for a parameter.
- **Patch semantics on update.** Every field except `id` is optional, and an
  omitted field keeps its current value; an update that wipes the body
  because the caller only sent a title is a data-loss bug. When a field can
  be cleared, the description says how (an empty string clears a category).
  Required fields in the schema are exactly the fields the call cannot run
  without.
- **Descriptions are contracts.** At least three sentences, more for a
  consolidated tool: what the tool does,
  when to use it and which sibling to use instead for nearby cases, what it
  returns, and any side effect or limit (it starts a deploy; it returns at
  most 50 items). No worked examples, no step-by-step procedures (those
  belong in a prompt or the server instructions), no capitals for emphasis.
- **Hints on every tool.** `mcptool.Hints` maps onto the specification's
  annotations, and its zero value is the conservative default: may overwrite
  or delete, not idempotent, reaches outside Nexul. Set `ReadOnly` on every
  `list`, `get`, and `search`; `Additive` on a create that never overwrites;
  `Idempotent` on an update or delete that is safe to repeat; `Local` when
  the tool touches only Nexul's own database, never GitHub, Cloudflare, or a
  machine. Clients decide what to confirm from these hints, and treat them
  as hints only; enforcement stays in the use-case.
- **A title, and no output schema.** Every tool has a short human `title`
  (sent as `annotations.title` too, for 2025-03-26 clients), because clients
  show it when they ask the user to approve a call. Results are one JSON
  text block: the adapter registers tools untyped, so the SDK publishes no
  output schema that would have to be kept true for every shape. Add
  structured output when a client needs it.

---

## 7. Shaping results

- **Return what the next step needs.** A tool returns a shaped struct with
  snake_case `json` tags, not the raw domain type. Keep ids the next call
  needs, put human names and keys beside them, and drop fields that inform
  nothing (internal timestamps, rich-text JSON, hashes). A document body is
  markdown, never the stored rich-text tree.
- **Every list is bounded.** A list or search takes `limit` (default 50,
  maximum 100) and `offset` through `mcptool.PageArgs`, and returns
  `{items, total, has_more, next_offset}` through `mcptool.Paginate`. An
  unbounded list grows until it breaks every client at once.
- **Large text returns a slice.** Logs and similar growing text return a tail
  by default, with a parameter for more. Keep typical results under 10,000
  tokens; a client may cut or divert a result past 25,000 tokens.
- **A delete returns what it deleted.** `{id, deleted: true}`, never `null`
  or an empty body, so the model can confirm the effect.

---

## 8. Errors

- **A tool's failure is a tool result, not a protocol error.** The adapter
  returns every error a tool produces as a result with `isError: true`, which
  the client passes to the model so it can correct itself. JSON-RPC errors
  are the SDK's, for an unknown tool or a malformed request.
- **Messages say how to recover.** Return the domain's sentinel errors
  (`internal/platform/errors`, `practices/go.md` section 2) wrapped with what was wrong, what the valid
  values are, and which tool returns them: "status_id is not a status of
  project REF; project_get lists them". The adapter passes the message of a
  known sentinel through.
- **Internal failures are hidden.** An error that is not a known sentinel
  reaches the model as "internal error" with the request's `trace_id`, and
  the full error goes to the log. SQL text, file paths, and provider response
  bodies never reach the client, the same rule the HTTP gateway follows.
- **Resource and prompt errors are JSON-RPC errors the adapter builds.** A
  missing entity is the SDK's resource-not-found error carrying the URI, a
  bad prompt argument is `-32602`, and anything else is `-32603` "internal
  error" with the `trace_id`. A plain Go error returned from these handlers
  would reach the client verbatim.
- **Composite tools say what was applied.** A tool that calls several
  use-cases in sequence stops at the first failure and says which parts took
  effect, because the use-cases commit separately.

---

## 9. Server instructions, resources, and prompts

- **Tools are the only required path.** Clients surface resources and
  prompts on user action, not on their own, so anything an agent must reach
  unaided is a tool.
- **Instructions are a map, not a manual.** The server's `instructions` are
  under 2,048 characters, most important first: what Nexul is in one line,
  the tool families and how they relate, the cross-tool workflows (deploy,
  then poll the deploy), and how ids and keys work. Clients that defer tool
  schemas show the instructions and the bare names at session start, so the
  instructions are how an agent learns which tool to search for. They never
  repeat a tool description and never carry behavior rules or security
  claims.
- **Resources mirror a read tool.** `docs://{id}`, `tickets://{id}`, and
  `topology://current` exist so a user can attach an entity to a
  conversation. Each is served by the same use-case as its `get` tool.
- **Prompts are user-invoked templates.** A prompt enforces its required
  arguments, renders text only, and names only tools that exist; a test
  checks the last rule, so a tool rename cannot strand a prompt.
- **Not implemented, on purpose.** Subscriptions and `listChanged` (the
  registries are fixed per process), completion, elicitation and
  multi-round-trip input, tasks, progress, and the deprecated sampling,
  roots, and logging. Each waits for a client the product supports to need
  it.

---

## 10. Testing

`practices/testing.md` applies; these are the MCP-specific rules.

- **Domain tools are tested through `Call`.** Table-driven, against a fake or
  a real SQLite use-case, error paths first: an invalid argument, a missing
  entity, a forbidden actor, and the patch rule (an omitted field survives an
  update).
- **The adapter is tested over HTTP.** Through `httptest` and the SDK's own
  client, so a test sees what a real client sees: an error arrives as an
  `isError` result, an unknown tool as a protocol error, a cross-origin
  request as `403`.
- **The surface is linted by a test.** One test walks `tools/list` and fails
  on a name outside `[a-z0-9_]{1,52}` or not ending in a known verb, a
  missing title, a description shorter than three sentences, a parameter
  without a description, a read-shaped name (`_list`, `_get`, `_search`)
  without `ReadOnly`, a count at or over the budget, and a prompt, server
  instruction, or built-in play that names a missing tool. A rule a test can
  check is checked by a test.
- **Wiring changes get a live check.** Before merging a change to the
  adapter's wiring (capabilities, cache fields, errors, the origin check),
  run the specification's conformance suite
  (`npx @modelcontextprotocol/conformance server`) against a dev server for
  2026-07-28 and 2025-11-25, and a real client against it: list the tools,
  call one read, and call one failing tool.
- **Reshaping is judged by the tasks.** A change that merges or splits tools
  walks the tasks it touches end to end, as an agent would, and says in the
  pull request how many calls each takes before and after. When selection
  accuracy is in doubt, a small set of read-only questions with checkable
  answers, run before and after, settles it.

---

## 11. Changing the surface

Tool names and parameters are a public contract. Agents' saved prompts, the
built-in play instructions, the harness setup text, the @Agent turn prompt,
the decisions check, the docs site's MCP guide, and end-to-end specs all
name tools. A pull request that renames, merges, or removes a tool updates
every one of them in the same change: `rg -w` the old name across the whole
repository before calling it done. Two copies live outside the repository
and need more than an edit:

- Play instructions are stored in the database when a workspace is created,
  and owners edit them, so a rename ships with a migration that rewrites the
  old names in stored instructions.
- The `nexul-memory` skill is written to users' machines once and never
  overwritten, so the memory tools it names (`memory_list`, `memory_get`,
  `memory_create`, `memory_update`) keep their names and the arguments it
  describes.

---

## Sources

The specification, revision 2026-07-28, at
`modelcontextprotocol.io/specification/2026-07-28`: the base protocol and
versioning, Streamable HTTP, `server/discover`, tools, resources, prompts,
caching, security best practices, and authorization. Its changelog lists what
changed from 2025-11-25. The TypeScript schema in the specification
repository is the reference for field names.

The official Go SDK, `github.com/modelcontextprotocol/go-sdk`, and its
package documentation, for the adapter's wiring.

Anthropic's guidance on tools for agents: the tool-use documentation on
defining tools, the engineering article "Writing effective tools for
agents", and the `mcp-builder` skill's best-practices reference, for tool
count, consolidation, naming, descriptions, response shaping, and error
text.
