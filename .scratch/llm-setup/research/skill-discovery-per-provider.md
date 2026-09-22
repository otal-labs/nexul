# Skill discovery per T3 Code provider

## Summary

T3 Code ships six provider drivers: Codex, Claude (Claude Agent SDK), Cursor, Grok,
OpenCode, and Antigravity (`apps/server/src/provider/builtInDrivers.ts`). Of these,
three report a `skills` list in the provider snapshot: Claude, Codex, and OpenCode.
Cursor and Grok carry no skill-discovery code at all. Antigravity resolves its own
Gemini-style skill directories through a separate profile-isolation path, not the
`skills` field the other three share (see "Antigravity" below) — out of scope for
this question but flagged for completeness.

For the three that matter here, **skills are a native feature of all three
underlying CLIs**, not a T3-invented convention:

- **Claude Code** reads `~/.claude/skills/` (personal) and `.claude/skills/`
  (project, walked up to the repo root), plus plugin-provided skills.
- **Codex CLI** reads `~/.agents/skills/` (personal) and `.agents/skills/`
  (repository, walked from cwd up to repo root), plus admin (`/etc/codex/skills`)
  and built-in system skills.
- **opencode** natively scans *both* of the above conventions in addition to its
  own `~/.config/opencode/skills/` / `.opencode/skills/` — confirmed directly from
  opencode's own docs, not inferred.

**Bottom line on the owner's question ("is opencode the same thing as Claude
here?")**: yes, for the *read* path. Anything installed at `~/.claude/skills/`
is natively picked up by opencode as well, no shim, no symlink, no config flag —
opencode's own docs list `~/.claude/skills/*/SKILL.md` as one of its three global
scan roots. Codex does not read that path; it reads `~/.agents/skills/` instead.
So `~/.claude/skills/` covers Claude Code + opencode, and `~/.agents/skills/`
covers Codex + opencode. Two install locations, not three, cover all three
providers, and opencode is the only one of the three that reads both.

T3 Code itself adds no skills layer of its own: it queries each CLI's native
skill-discovery mechanism (a filesystem scan mirroring the CLI for Claude, the
Codex app-server's `skills/list` RPC for Codex, the OpenCode SDK's `app.skills`
endpoint for OpenCode) and republishes whatever that CLI reports. Installing a
skill for the underlying CLI is sufficient — T3 Code does not need separate
configuration, and does not gate or filter what the CLI already discovers.

## Table: minimal install locations

| Provider (T3 driver) | Discovery path(s) it reads | Shared with |
|---|---|---|
| Claude Code (`claudeAgent`) | `~/.claude/skills/` (personal), `.claude/skills/` (project, walked to repo root), plugin `skills/` dirs | opencode (personal path only) |
| Codex CLI (`codex`) | `~/.agents/skills/` (personal), `.agents/skills/` (repo, walked from cwd to repo root), `/etc/codex/skills` (admin), built-in system skills | opencode (personal + repo path) |
| opencode (`opencode`) | Own: `~/.config/opencode/skills/`, `.opencode/skills/`. Plus natively: `~/.claude/skills/`, `.claude/skills/`, `~/.agents/skills/`, `.agents/skills/` | Claude Code and Codex, both, natively |

Minimal set to cover all three: **`~/.claude/skills/` + `~/.agents/skills/`**
(project-level `.claude/skills/` and `.agents/skills/` cover the per-repo case the
same way). opencode needs nothing separate as long as both of those exist;
Claude Code never reads `~/.agents/skills/`; Codex never reads `~/.claude/skills/`.

## Per-provider detail

### 1. Providers T3 Code ships

`apps/server/src/provider/builtInDrivers.ts`:

```ts
export const BUILT_IN_DRIVERS: ReadonlyArray<AnyProviderDriver<BuiltInDriversEnv>> = [
  CodexDriver,
  ClaudeDriver,
  CursorDriver,
  GrokDriver,
  OpenCodeDriver,
  AntigravityDriver,
];
```

Source: https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/builtInDrivers.ts

A search of the driver files for `skills` turned up matches only in
`ClaudeDriver.ts`, `CodexDriver.ts`, and `OpenCodeDriver.ts`. `CursorDriver.ts` and
`GrokDriver.ts` have none — those two drivers do not surface a skills list in the
provider snapshot at all as of this build.

### 2. Claude Code

T3 Code's own comment in `apps/server/src/provider/Drivers/ClaudeSkills.ts` states
the discovery contract T3 verified against the CLI directly:

> "Claude Code loads skills from `<config dir>/skills` (user scope) and
> `<cwd>/.claude/skills` (project scope), one directory per skill with a
> `SKILL.md` carrying YAML frontmatter. The user root wins on name collisions,
> matching the CLI. `.agents/skills` is a Codex location: verified against the
> CLI, a skill that lives only there is answered with `Unknown command`, so it
> is not scanned here."

Source: https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/ClaudeSkills.ts

Confirmed independently against Anthropic's current docs
(https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview,
fetched 2026-09-22):

> "Custom Skills in Claude Code are filesystem-based and don't require API
> uploads: place them in `~/.claude/skills/` (personal) or `.claude/skills/`
> (project)."
>
> Sharing scope: "Claude Code: Personal (`~/.claude/skills/`) or project-based
> (`.claude/skills/`). Can also be shared through Claude Code Plugins."

And from https://code.claude.com/docs/en/skills (fetched 2026-09-22):

- Personal: `~/.claude/skills/<skill-name>/SKILL.md` — loads in all projects on
  that machine, not Cowork/cloud sessions.
- Project: `.claude/skills/<skill-name>/SKILL.md` — loaded from the directory
  Claude Code starts in and every parent directory up to the repository root.
- Plugin-provided: `<plugin>/skills/<skill-name>/SKILL.md`, namespaced as
  `/plugin-name:skill-name` so they never collide with local skills.
- Precedence on a name collision: **Enterprise > Personal > Project**.

No mention of `~/.agents/skills/` anywhere in Anthropic's docs, and T3 Code's own
CLI-verified note above confirms Claude Code actively rejects a skill that lives
only there (`Unknown command`).

### 3. Codex CLI

T3 Code queries this over Codex's own app-server protocol rather than scanning the
filesystem itself — `apps/server/src/provider/Layers/CodexProvider.ts` calls
`client.request("skills/list", { cwds: [cwd] })` against a spawned
`codex app-server` process
(https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Layers/CodexProvider.ts).
That RPC is part of Codex's own app-server protocol
(`codex-rs/app-server-protocol/schema/json/ClientRequest.json` in
https://github.com/openai/codex), confirming skills are a native, first-class
Codex CLI capability, not something layered on by an external tool.

OpenAI's own current docs (https://learn.chatgpt.com/docs/build-skills, the
current location of developers.openai.com/codex/skills after a 308 redirect,
fetched 2026-09-22) describe four discovery levels:

> "Codex reads skills from repository, user, admin, and system locations. For
> repositories, Codex scans `.agents/skills` in every directory from your current
> working directory up to the repository root."

- Repository: `$CWD/.agents/skills`, walked up through parent directories to
  `$REPO_ROOT/.agents/skills`.
- User: `$HOME/.agents/skills` — personal skills.
- Admin: `/etc/codex/skills` — system-wide shared location.
- System: built-in skills bundled by OpenAI.

Skills are enabled/disabled optionally via `~/.codex/config.toml`, but discovery
itself needs no config — Codex "detects skill changes automatically."

`AGENTS.md` is a separate, older mechanism (always-on repo instructions, no
frontmatter, no on-demand loading) and is not the skills path. Codex supports
both `AGENTS.md` and `.agents/skills/`-style skills; they are not the same thing.

**Uncertain / worth re-checking**: `learn.chatgpt.com/docs/build-skills` is a
ChatGPT-Learn-branded doc rather than the canonical `developers.openai.com` or
`platform.openai.com` docs tree; it is presumably the officially redirected
destination (a 308 from `developers.openai.com/codex/skills`) but treat the exact
admin/system tiers as secondary confirmation rather than gospel — the repository
and user tiers are corroborated independently by the Codex app-server protocol
existing at all and by T3 Code's own comment above naming `.agents/skills` (both
`$HOME` and repo-relative) as "a Codex location."

### 4. opencode

`apps/server/src/provider/opencodeRuntime.ts` shows T3 Code does not scan the
filesystem for OpenCode skills either — it calls the OpenCode SDK's
`client.app.skills(...)` against a running opencode server
(`loadOpenCodeSkills`), i.e. it asks opencode itself and republishes whatever
opencode's own discovery already found. A comment on the driver explains why the
SDK path is used over the CLI's `debug skill` command (a Bun stdout-buffering
issue truncates large JSON output over a pipe) — an implementation detail, not a
discovery-semantics one.

Confirmed directly from opencode's own docs
(https://opencode.ai/docs/skills/, fetched 2026-09-22), quoted verbatim:

> "Create one folder per skill name and put a `SKILL.md` inside it. OpenCode
> searches these locations:
> - Project config: `.opencode/skills/<name>/SKILL.md`
> - Global config: `~/.config/opencode/skills/<name>/SKILL.md`
> - Project Claude-compatible: `.claude/skills/<name>/SKILL.md`
> - Global Claude-compatible: `~/.claude/skills/<name>/SKILL.md`
> - Project agent-compatible: `.agents/skills/<name>/SKILL.md`
> - Global agent-compatible: `~/.agents/skills/<name>/SKILL.md`"
>
> "For project-local paths, OpenCode walks up from your current working
> directory until it reaches the git worktree. It loads any matching
> `skills/*/SKILL.md` in `.opencode/` and any matching `.claude/skills/*/SKILL.md`
> or `.agents/skills/*/SKILL.md` along the way. Global definitions are also
> loaded from `~/.config/opencode/skills/*/SKILL.md`, `~/.claude/skills/*/SKILL.md`,
> and `~/.agents/skills/*/SKILL.md`."

This is opencode's own native behavior, not a community shim — it is documented
on opencode's first-party docs site as core discovery logic, with no plugin or
extra configuration required.

**Uncertain**: the docs do not state what happens if the same skill *name*
exists under more than one of the three global roots at once (e.g. both
`~/.claude/skills/my-skill/` and `~/.config/opencode/skills/my-skill/`) — whether
one wins by a documented precedence order or whether both entries are exposed
side by side. Not verified against a live opencode instance for this ticket.

### 5. Does T3 Code add its own layer?

No, for these three. In every case (`ClaudeSkills.ts`'s filesystem scan,
`CodexProvider.ts`'s app-server `skills/list` call, `opencodeRuntime.ts`'s SDK
`app.skills` call), T3 Code is reading what the underlying CLI itself would
report — it mirrors the CLI's own discovery precedence rather than defining a
new one. Skills installed for the CLI directly (by hand, or by whatever
convention the CLI documents) show up in T3 Code's `$` picker without any
T3-specific installation step. The one caveat: for Claude, T3 Code performs its
own filesystem scan (rather than asking the Claude Agent SDK, which "surfaces
skills only as slash commands without their filesystem paths" per the code
comment) — but the scan is written to match the CLI's own two roots and
precedence exactly, verified against the CLI's behavior, not an independent
policy.

### 6. Antigravity (out of scope, flagged for completeness)

`docs/internals/providers.md` in the T3 Code repo notes Antigravity (Google's
Gemini-based driver) "resolves its user-global skill directories under that
[isolated] profile, so the profile links those two directories back to the
user's real `~/.gemini`." This is a distinct mechanism from the `skills` field
used by Claude/Codex/OpenCode above, tied to Antigravity's account-profile
isolation rather than the shared provider-snapshot skills list, and was not
investigated further since it falls outside Claude Code / Codex / opencode.

## Sources checked

- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/builtInDrivers.ts
- https://github.com/pingdotgg/t3code/blob/master/docs/internals/providers.md
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/ClaudeDriver.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/CodexDriver.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/OpenCodeDriver.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/ClaudeSkills.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Layers/CodexProvider.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/opencodeRuntime.ts
- https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview
- https://code.claude.com/docs/en/skills
- https://learn.chatgpt.com/docs/build-skills (redirect target of developers.openai.com/codex/skills)
- https://github.com/openai/codex (codex-rs/app-server-protocol — confirms `skills/list` as a native app-server RPC)
- https://opencode.ai/docs/skills/

All fetched 2026-09-22.
