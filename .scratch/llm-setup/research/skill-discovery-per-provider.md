# Skill discovery per T3 Code provider

## Summary

T3 Code ships six provider drivers: Codex, Claude (Claude Agent SDK), Cursor, Grok,
OpenCode, and Antigravity (`apps/server/src/provider/builtInDrivers.ts`).
**Corrected 2026-09-22**: an earlier pass over this question concluded Cursor and
Grok carry no skill-discovery code. That was wrong — see the addendum at the end of
this file. Five of the six report a `skills` list in the provider snapshot: Claude,
Codex, OpenCode, Cursor, and Grok. `CursorDriver.ts` wires `discoverCursorSkills` /
`probeCursorSkills` (`Drivers/CursorSkills.ts`) into both its command catalog and its
per-cwd snapshot; `GrokDriver.ts` wires `discoverGrokSkills` (`Drivers/GrokSkills.ts`)
into `snapshotForCwd`. Antigravity resolves its own Gemini-style skill directories
through a separate profile-isolation path, not the `skills` field the other five
share (see "Antigravity" below) — out of scope for this question but flagged for
completeness.

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
| Cursor CLI (`cursor-agent`), added 2026-09-22 | Own: `~/.cursor/skills/`, `.cursor/skills/` (project, nested subdirs included). Plus natively: `~/.agents/skills/`, `.agents/skills/`, `~/.claude/skills/`, `.claude/skills/`, `~/.codex/skills/`, `.codex/skills/` | Codex + opencode (`.agents/skills/`), Claude Code + opencode (`.claude/skills/`) |
| Grok Build (`grok`), added 2026-09-22 | Own: `~/.grok/skills/`, `./.grok/skills/` (walked to repo root), any enabled plugin's `skills/` dir, extra paths under `[skills] paths` in `~/.grok/config.toml`. Plus `~/.agents/skills/` via its AGENTS.md-format support | Codex + opencode + Cursor (`~/.agents/skills/`) |

Minimal set to cover all three original providers: **`~/.claude/skills/` +
`~/.agents/skills/`** (project-level `.claude/skills/` and `.agents/skills/` cover
the per-repo case the same way). opencode needs nothing separate as long as both of
those exist; Claude Code never reads `~/.agents/skills/`; Codex never reads
`~/.claude/skills/`. Once Cursor and Grok are in scope, the same two locations still
do the most work: `~/.claude/skills/` now also reaches Cursor, and `~/.agents/skills/`
now also reaches Cursor and (per its own docs) Grok — see the addendum for the
caveats on each.

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

**Corrected 2026-09-22**: an earlier pass here searched only for the literal string
`skills` inside the driver files themselves and missed the dedicated
`Drivers/CursorSkills.ts` and `Drivers/GrokSkills.ts` modules those two drivers
import and wire into their snapshots. Both Cursor and Grok do surface a skills list —
see the addendum at the end of this file for the full re-verification.

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

## Addendum: Cursor and Grok, re-verified (2026-09-22)

The original pass above searched the driver files for the literal string `skills`
and missed two dedicated modules the drivers import. Re-checked directly against
`apps/server/src/provider/Drivers/CursorSkills.ts`,
`apps/server/src/provider/Drivers/GrokSkills.ts`, the two drivers that wire them in,
and each vendor's own current docs. Both Cursor and Grok discover skills; the
earlier conclusion that they have no skill-discovery code was wrong.

### 1. Cursor

T3 drives Cursor's headless CLI, `cursor-agent`, over its ACP protocol
(`CursorDriver.ts`: "Cursor exposes an ACP-based CLI"). `CursorSkills.ts` performs a
recursive filesystem scan rather than asking the CLI, because "its ACP command
catalog only appears after opening a real session" and scanning avoids starting an
agent and its MCP servers just to populate a composer menu. It is wired into
`CursorDriver.ts` in two places: `onAvailableCommands` calls `discoverCursorSkills`
to populate the command catalog, and `snapshotForCwd` calls `probeCursorSkills` (the
same discovery, but surfacing scan-budget/filesystem errors instead of swallowing
them) whenever a workspace's snapshot is requested.

Paths scanned, per-root, both project (`cwd`) and user (`$HOME`) scope:

```
.cursor/skills/
.agents/skills/
.codex/skills/
.claude/skills/
```

This matches Cursor's own current docs (https://cursor.com/docs/skills, fetched
2026-09-22), which list the same four directories at both project and user scope —
project: `.agents/skills/` and `.cursor/skills/`, user: `~/.agents/skills/` and
`~/.cursor/skills/`, plus "compatibility paths" `.claude/skills/`, `.codex/skills/`,
`~/.claude/skills/`, and `~/.codex/skills/`. So Cursor's CLI does read
`~/.claude/skills/` and `~/.agents/skills/` — both are native discovery paths, not
something T3 adds.

**Plugins are a separate question from skills, and the marketplace does not (fully)
reach the CLI.** Cursor's plugin marketplace (github.com/cursor/plugins,
cursor.com/docs/plugins) lets a plugin bundle skills, rules, MCP servers, commands,
and hooks, installed from the editor's Customize panel at user or workspace scope.
Per a Cursor staff reply on the community forum
(https://forum.cursor.com/t/cursor-agent-cli-does-not-register-skills-from-plugins-ide-does-parity-gap/158947,
thread dated May 2026): the IDE agent surfaces plugin skills from the Cursor
marketplace, but as of the initial report the CLI did not — "I can confirm that
plugins are not currently working in the CLI" (Colin, Cursor staff, May 4). The team
"re-enabled the feature flag for plugins in the CLI" three days later (May 7). So
plugin-sourced skills reaching `cursor-agent` is a recently-fixed feature-flagged
path, not something to assume was always true — worth a live re-check
(`cursor-agent`'s own `--version` and installed-plugins state) before relying on it,
since T3's own `CursorSkills.ts` does not scan a plugins directory at all — it only
scans the four `skills/` roots above. Skills placed directly under those four roots
(personal, project, or the Claude/Codex-compatible aliases) are unaffected by the
plugin gap either way, since that path never went through the plugin subsystem.

### 2. Grok

T3 drives xAI's Grok Build CLI (binary `grok`, confirmed in `GrokDriver.ts` and
https://docs.x.ai/build/overview). Unlike Cursor, T3 does not scan the filesystem
itself for Grok — `GrokSkills.ts` shells out to `grok inspect --json` and parses its
`skills[]` array (`name`, `description`, `source.type`, `source.path`,
`userInvocable`), wired into `GrokDriver.ts`'s `snapshotForCwd`. The code comment
explains why: "the Grok CLI reports its full skill catalog itself... Asking the CLI
beats scanning the filesystem because the catalog honors Grok's own skill config
(ignore lists, disabled skills) and includes plugin skills, which live three levels
deep under `~/.grok/installed-plugins/` where a flat scan cannot see them."

xAI's own current docs
(https://docs.x.ai/build/features/skills-plugins-marketplaces, fetched 2026-09-22)
confirm Grok discovers skills from four scopes:

> "Grok discovers skills from: `./.grok/skills/` (walked up to the repo root),
> `~/.grok/skills/`, Any enabled plugin's `skills/` directory, [and] Extra paths
> under `[skills] paths` in `~/.grok/config.toml`."

The same docs note Grok also picks up user-level skills from `~/.agents/skills/`
through its AGENTS.md-format compatibility support. `~/.claude/skills/` is not
listed as a Grok discovery path in the current docs — unlike Cursor, Grok does not
appear to read the Claude-compatible location natively. `grok inspect --json`
(referenced in the `GrokSkills.ts` code comment but not found verbatim in the
fetched docs excerpt) is the mechanism that reports config sources, instructions,
skills, plugins, hooks, and MCP servers for the current directory — this is a
first-party CLI capability, not a T3 convention, and it is why T3 asks the CLI
instead of walking `~/.grok/skills/` and plugin directories by hand.

### 3. Why the earlier pass got it wrong

The driver capability does matter here, and both drivers implement it — the earlier
conclusion did not come from T3 actually lacking the capability, it came from an
incomplete search. `CursorDriver.ts` and `GrokDriver.ts` themselves contain no
literal occurrences of the word `skills` in a way a narrow grep would catch at a
glance unless the import lines and call sites are read — `discoverCursorSkills`,
`probeCursorSkills`, and `discoverGrokSkills` are all imported from separate
`*Skills.ts` files and invoked inside larger `Effect.gen` blocks. The fix here was
reading the imports and call sites, not just the driver file's direct RPC calls.

Confirmed live wiring:

- `CursorDriver.ts`: `onAvailableCommands: (commands, cwd) => discoverCursorSkills(cwd, processEnv)...` and `snapshotForCwd: (cwd) => ... probeCursorSkills(cwd, processEnv)...`
- `GrokDriver.ts`: `snapshotForCwd = (workspaceCwd) => ... discoverGrokSkills(effectiveConfig, processEnv, workspaceCwd)...`

Both feed into the same `ServerProviderSkill[]` shape used by Claude, Codex, and
OpenCode, and the same `dedupeProviderSkillsByName` / `isProviderSkillUserInvocable`
helpers in `packages/client-runtime/src/providerSkills.ts` apply uniformly across
all five drivers — Cursor and Grok are first-class citizens of the same `skills`
field, not a bolted-on special case.

### 4. Bottom line

| Driver | Skill support | Discovery mechanism | Discovery paths | Shared with other drivers |
|---|---|---|---|---|
| Claude Code (`claudeAgent`) | Yes | T3 filesystem scan, mirrors CLI precedence | `~/.claude/skills/`, `.claude/skills/` (+ plugin `skills/` dirs) | opencode (`~/.claude/skills/`, `.claude/skills/`), Cursor (`~/.claude/skills/`, `.claude/skills/` as a compatibility path) |
| Codex CLI (`codex`) | Yes | Asks the CLI (`skills/list` app-server RPC) | `~/.agents/skills/`, `.agents/skills/`, `/etc/codex/skills`, built-in | opencode, Cursor, and (per its docs) Grok all read `~/.agents/skills/` / `.agents/skills/` too |
| opencode (`opencode`) | Yes | Asks the CLI/SDK (`app.skills`) | Own `~/.config/opencode/skills/`, `.opencode/skills/`, plus natively all of Claude's and Codex's paths | Claude Code and Codex, both, natively |
| Cursor CLI (`cursor-agent`) | **Yes — corrected 2026-09-22** | T3 filesystem scan (`CursorSkills.ts`), matches Cursor's own documented paths | `~/.cursor/skills/`, `.cursor/skills/` (own) + `~/.agents/skills/`, `.agents/skills/`, `~/.claude/skills/`, `.claude/skills/`, `~/.codex/skills/`, `.codex/skills/` (compatibility) | Codex + opencode via `.agents/skills/`; Claude Code + opencode via `.claude/skills/`. Marketplace **plugin** skills are CLI-supported only as of a May 2026 feature-flag fix — not scanned by T3's own `CursorSkills.ts`, which reads only the four `skills/` roots directly |
| Grok Build (`grok`) | **Yes — corrected 2026-09-22** | Asks the CLI (`grok inspect --json`, `skills[]`) | `~/.grok/skills/`, `./.grok/skills/` (own, walked to repo root), enabled plugins' `skills/` dirs, `[skills] paths` in `~/.grok/config.toml`, plus `~/.agents/skills/` via AGENTS.md-format support | Codex + opencode + Cursor via `~/.agents/skills/`. Does **not** read `~/.claude/skills/` per current xAI docs — the one driver that does not share that location |
| Antigravity | Separate mechanism | Own profile-isolated `~/.gemini` links | Not the shared `skills` field (see section 6 above) | Out of scope, not comparable |

Practical takeaway for install locations: `~/.claude/skills/` now reaches Claude
Code, opencode, and Cursor; `~/.agents/skills/` now reaches Codex, opencode, Cursor,
and Grok. Grok is the outlier that needs its own `~/.grok/skills/` (or the
`~/.agents/skills/` compatibility path) rather than piggybacking on
`~/.claude/skills/`.

Sources for this addendum, all fetched 2026-09-22:

- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/CursorSkills.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/GrokSkills.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/CursorDriver.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/GrokDriver.ts
- https://github.com/pingdotgg/t3code/blob/master/apps/server/src/provider/Drivers/ClaudeSkills.ts
- https://github.com/pingdotgg/t3code/blob/master/packages/client-runtime/src/providerSkills.ts
- https://github.com/pingdotgg/t3code/blob/master/docs/internals/providers.md
- https://cursor.com/docs/skills
- https://cursor.com/docs/plugins
- https://github.com/cursor/plugins
- https://forum.cursor.com/t/cursor-agent-cli-does-not-register-skills-from-plugins-ide-does-parity-gap/158947
- https://docs.x.ai/build/overview
- https://docs.x.ai/build/features/skills-plugins-marketplaces
