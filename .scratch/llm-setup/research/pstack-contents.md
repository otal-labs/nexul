# pstack (cursor/plugins/pstack) — what it is, and does it fit beside mattpocock/skills

Verified against `cursor/plugins` (commit `53e579f1481697931fc44f5445171397cfa2b24b`) and
`mattpocock/skills` (commit `c55ee46073ed923f86ce59a5eb3b6d895095d1b7`) via the GitHub API, 2026-09-22.

## Summary

`pstack` is Lauren Tan's (poteto, React core team / ex-Meta / ex-Netflix, now Cursor) personal
engineering-discipline pack, shipped as one plugin inside Cursor's official multi-plugin
marketplace repo `cursor/plugins`. It bundles 38 skills (23 of them one-line "principle" skills),
2 subagents, and a dormant Slack-triage automation, all routed through one entry point,
`poteto-mode`, which picks from 23 playbooks. The individual skill files use the same `SKILL.md`
YAML-frontmatter convention (`name` + `description`, optionally `disable-model-invocation`) that
Claude Code and opencode read — so the content is portable in principle. But the plugin as a whole
is packaged for Cursor only: it ships a `.cursor-plugin/plugin.json` manifest and installs via
Cursor's `/add-plugin pstack` command against the marketplace's `.cursor-plugin/marketplace.json`.
It has no `.claude-plugin/` manifest and is not published to the `npx skills` registry, so Claude
Code, Codex, and opencode users have no first-party install path — only a manual copy of individual
`SKILL.md` files.

It can sit beside `mattpocock/skills` (also MIT, also `name`+`description` frontmatter), but not
without curation: both repos ship a skill literally named `tdd` and both ship one literally named
`teach`, with materially different behavior in each case. A flat, uncurated merge of both skill
directories overwrites or shadows one with the other depending on load order. Selective adoption
(the principle skills, `how`/`why`/`interrogate`/`architect`, skip `tdd`/`teach`) is the fit that
works.

## 1. Inventory

Top-level layout of `pstack/` (verified via `github_get_file_contents` on `cursor/plugins`):

```
pstack/
├── .cursor-plugin/plugin.json   # Cursor plugin manifest
├── .gitignore
├── LICENSE                      # MIT, Copyright (c) 2026 Lauren Tan
├── README.md
├── agents/                      # 2 Cursor subagent definitions
├── assets/                      # logo etc.
├── automations/                 # dormant benny/ Slack-triage pack, not registered as skills
├── docs/                        # guide/README.md walkthrough
└── skills/                      # 38 skill directories
```

Skills (`pstack/skills/*`), one phrase each, from the README and each `SKILL.md`:

| Skill | Purpose |
|---|---|
| `poteto-mode` | Entry point; matches a task to 1 of 23 playbooks and routes to the other skills as steps fire |
| `how` | Walkthrough of how a subsystem works |
| `why` | Why something was built this way; queries every available MCP source in parallel |
| `recall` | Rebuilds a tight current-state brief from the user's own chat history + shared record |
| `blast-radius` | What a small-looking change could break, proven by running code |
| `architect` | Settles a function boundary's usage/types/module shape before writing code |
| `arena` | N parallel attempts at the same task, then merges the best parts |
| `swarm` | N parallel workers across different slices, one aggregated report |
| `interrogate` | Several models try to break a diff, including a code-quality lens |
| `automate-me` | Drafts a personal `-mode` skill from the user's own transcripts |
| `make-bot-ui` | A dashboard whose buttons wake a Grok Bot over webhook |
| `setup-pstack` | Picks which models pstack uses per role; writes a config rule |
| `reflect` | Captures a long task's recipe as a skill edit |
| `teach` | Explains a subsystem/change plainly, weaving `how` + `why` into one account |
| `tdd` | Failing-test-first regression fix, only when explicitly asked or the bug has a cheap local test target (`disable-model-invocation: true`) |
| `no-comments` | Strips comments before review via the Comment Sicko subagent |
| `typescript-best-practices` | Grounds the type-system-discipline principle in TS syntax |
| `figure-it-out` | Designs a rigorous playbook when no bundled one fits |
| `show-me-your-work` | Logs a reviewable decision trail to a committable TSV |
| `create-verification-skill` | Generates a project-local "prove it runs" skill with a feature map |
| `maintain-verification-skill` | Keeps that feature map from drifting |
| `unslop` | Removes AI tells from writing |
| `bro` | Restates the last message in plain human language |
| `technical-writing` | Diátaxis + Google style + STE doc standard for docs/RFCs/PRs |
| `principle-*` (23 skills) | One principle each — laziness-protocol, foundational-thinking, redesign-from-first-principles, attack-the-premise, subtract-before-you-add, minimize-reader-load, outcome-oriented-execution, experience-first, exhaust-the-design-space, build-the-lever, model-the-domain, boundary-discipline, type-system-discipline, make-operations-idempotent, migrate-callers-then-delete-legacy-apis, separate-before-serializing-shared-state, prove-it-works, fix-root-causes, sequence-verifiable-units, test-behavior-not-implementation, guard-the-context-window, never-block-on-the-human, encode-lessons-in-structure |

Agents (`pstack/agents/*.md`):
- `poteto-agent.md` — runs the pstack style end-to-end as a subagent (`subagent_type: "poteto-agent"`); reads `poteto-mode` in full including its inline principles index before working.
- `comment-sicko.md` — read-only comment reviewer (`subagent_type: "Comment Sicko"`), normally invoked through `/no-comments`.

Not bundled in `pstack` itself (README's own disclosure): `/deslop`, `control-cli`, `control-ui` ship
in the separate `cursor-team-kit` plugin in the same marketplace; `/create-skill` and `/babysit` are
Cursor built-ins.

Sources: https://github.com/cursor/plugins/tree/main/pstack/skills ,
https://github.com/cursor/plugins/blob/main/pstack/README.md ,
https://github.com/cursor/plugins/tree/main/pstack/agents

## 2. Format

Two layers, both verified directly:

**Per-skill files** use the same convention Claude Code/opencode read — YAML frontmatter with
`name` + `description`, optionally `disable-model-invocation: true` to make a skill user-invoked
only. Example, `pstack/skills/tdd/SKILL.md`:

```yaml
---
name: tdd
description: "Use only when the user explicitly asks for TDD, a failing test, or a regression test, OR when the bug has an obvious cheap local test target. Skip when the test path is unclear, expensive, integration-heavy, or not requested."
disable-model-invocation: true
---
```

This is identical in shape to `mattpocock/skills/skills/engineering/tdd/SKILL.md`:

```yaml
---
name: tdd
description: Test-driven development. Use when the user wants to build features or fix bugs test-first, mentions "red-green-refactor", or wants integration tests.
---
```

**Plugin-level manifest** is Cursor-specific and has no equivalent in the SKILL.md convention:
`pstack/.cursor-plugin/plugin.json`:

```json
{
	"name": "pstack",
	"displayName": "pstack",
	"version": "0.15.2",
	"description": "if you want to go fast, go deep first. ...",
	"author": { "name": "Lauren Tan" },
	"homepage": "https://github.com/cursor/plugins/tree/main/pstack",
	"repository": "https://github.com/cursor/plugins",
	"license": "MIT",
	"logo": "assets/logo.png",
	"keywords": ["pstack", "poteto-mode", "workflow", "principles", "agent-style", "subagents", "unslop"],
	"category": "developer-tools",
	"tags": ["workflow", "principles", "review", "planning"],
	"skills": "./skills/",
	"agents": "./agents/"
}
```

The repo root's own README documents this two-layer structure explicitly:

```
plugins/
├── .cursor-plugin/
│   └── marketplace.json       # Marketplace manifest (lists all plugins)
├── plugin-name/
│   ├── .cursor-plugin/
│   │   └── plugin.json        # Per-plugin manifest
│   ├── skills/                # Agent skills (SKILL.md with frontmatter)
│   ├── rules/                 # Cursor rules (.mdc files)
│   ├── mcp.json               # MCP server definitions
│   ├── README.md
│   ├── CHANGELOG.md
│   └── LICENSE
└── ...
```

So: the skill bodies are the shared SKILL.md convention; the packaging that makes them installable
as one unit is Cursor's own plugin/marketplace format, not the Claude Code plugin format (there is
no `.claude-plugin/plugin.json` or `.claude-plugin/marketplace.json` anywhere under `pstack/`).

Sources: https://github.com/cursor/plugins/blob/main/pstack/skills/tdd/SKILL.md ,
https://github.com/cursor/plugins/blob/main/pstack/.cursor-plugin/plugin.json ,
https://github.com/cursor/plugins/blob/main/README.md

## 3. Install

Documented install (`pstack/README.md`):

```bash
/add-plugin pstack
```

This is a Cursor-only command against the marketplace listing in the repo root's
`.cursor-plugin/marketplace.json`, which pins `pstack` as one of ~80 plugins (most of the rest are
first- and third-party SaaS integrations — Gmail, Salesforce, GitHub, etc.). There is no `npx
skills add cursor/plugins --skill pstack`-style installer documented or implied anywhere in the
repo, and no `.claude-plugin/` manifest that Claude Code's plugin marketplace mechanism could read.

For non-Cursor harnesses (Claude Code, Codex, opencode), the only route is a manual copy: because
each `pstack/skills/<name>/SKILL.md` already uses the shared frontmatter convention, a user could
copy individual skill directories into `~/.claude/skills/<name>/` (Claude Code's own convention, per
this repo's global setup) or the Codex/opencode equivalents. pstack does not document, script, or
support this path — it is inferred from the file format matching, not offered.

By contrast, `mattpocock/skills` ships both: a real Claude Code plugin
(`claude plugins install mattpocock-skills` / `/plugin install mattpocock-skills`, backed by its own
`.claude-plugin/plugin.json` + `.claude-plugin/marketplace.json`) and an agent-agnostic installer,
`npx skills@latest add mattpocock/skills`, which copies editable files into the target repo for any
of Claude Code, Codex, or other agents and lets the user pick which skills to take.

Sources: https://github.com/cursor/plugins/blob/main/pstack/README.md#install ,
https://github.com/cursor/plugins/blob/main/.cursor-plugin/marketplace.json ,
https://github.com/mattpocock/skills/blob/main/.claude-plugin/plugin.json ,
https://github.com/mattpocock/skills/blob/main/README.md#installation-30-second-setup

## 4. License

`pstack/LICENSE`: MIT, `Copyright (c) 2026 Lauren Tan`. Standard MIT terms — the only obligation is
to keep the copyright notice and permission notice in copies or substantial portions of the
software. The README additionally invites forking ("fork it. improve it. make it yours. PRs are
welcome!") with no extra attribution clause beyond the license text itself. The repo root
`cursor/plugins/README.md` and its own `LICENSE` are also MIT, and credit each plugin's author by
name in the marketplace table (pstack is listed as authored by "Lauren Tan").

`mattpocock/skills/LICENSE` is also MIT (`Copyright (c) 2026 Matt Pocock`) — same terms, so no
license conflict in running both side by side.

Sources: https://github.com/cursor/plugins/blob/main/pstack/LICENSE ,
https://github.com/cursor/plugins/blob/main/LICENSE ,
https://github.com/mattpocock/skills/blob/main/LICENSE

## 5. Overlap with mattpocock/skills

`mattpocock/skills` ships 25 skills across `engineering/` and `productivity/` (per
`.claude-plugin/plugin.json`'s `skills` array): `ask-matt`, `diagnosing-bugs`, `grill-with-docs`,
`triage`, `improve-codebase-architecture`, `setup-matt-pocock-skills`, `tdd`, `to-spec`,
`to-tickets`, `wayfinder`, `implement`, `prototype`, `research`, `domain-modeling`,
`codebase-design`, `code-review`, `resolving-merge-conflicts`, `wizard`, `grill-me`, `grilling`,
`handoff`, `teach`, `to-questionnaire`, `wait-what`, `writing-for-agents`.

**Direct name collisions** (same skill name in both repos, so a flat merge of the two `skills/`
directories has one overwrite or shadow the other):

- **`tdd`** — both repos ship a skill literally named `tdd`. Different scope: pstack's `tdd`
  (`disable-model-invocation: true`) fires only when explicitly asked or the bug has "an obvious
  cheap local test target," and is scoped to bug-fix regression tests. mattpocock's `tdd` is broader
  — model-invocable, covers building features test-first as well as bug fixes, red-green-refactor
  with seam/anti-pattern guidance (`tests.md`, `mocking.md`). Installing both flat means whichever
  loads last wins the `/tdd` slash command and the `tdd`-named skill; the two have materially
  different trigger conditions and content, not interchangeable copies of the same idea.
- **`teach`** — both repos ship a skill literally named `teach`, and here the behaviors don't even
  overlap conceptually: pstack's `teach` (`disable-model-invocation: true`) is a one-shot plain-
  English explanation of an existing subsystem or change, built by running `how` + `why` and
  weaving the results. mattpocock's `teach` (`disable-model-invocation: true`, has an
  `argument-hint`) is a stateful, multi-session curriculum builder that maintains a teaching
  workspace on disk (`MISSION.md`, `RESOURCES.md`, `learning-records/`, `lessons/*.html`,
  `assets/`). A user typing `/teach` gets one of two unrelated experiences depending on install
  order, with no graceful degradation either way.

**Same domain, different name, no literal collision — worth flagging when picking which to keep:**

- `architect` (pstack — settle a function boundary's types/shape before crossing it) vs.
  `codebase-design` (mattpocock — deep-module vocabulary: seam, interface, depth, adapter, leverage,
  locality). Same job, different vocabulary; `tdd`'s own text in mattpocock/skills explicitly
  references `codebase-design` for this.
- `interrogate` / `no-comments` (pstack — multi-model adversarial diff review, comment stripping)
  vs. `code-review` (mattpocock — two-axis Standards + Spec review via parallel sub-agents). Both
  gate a diff before merge; different mechanics, no naming clash.
- `how` / `why` / `recall` (pstack — investigation, provenance, session-context rebuild) vs.
  `research` (mattpocock — investigate a question against primary sources, write a cited Markdown
  file, runnable as a background agent — this is the exact mechanism resolving this ticket). Same
  domain (answer a question with evidence), no naming clash, but redundant if both are active.
- `setup-pstack` (pstack — pick per-role models, write a config rule) vs.
  `setup-matt-pocock-skills` (mattpocock — pick issue tracker + triage labels + doc layout). Same
  first-run-once pattern and naming convention (`setup-*`), different config surface — harmless to
  run both, but a new user has two "run this first" onboarding skills to reconcile.
- pstack's `poteto-mode` `prototype` playbook vs. mattpocock's standalone `prototype` skill — both
  "build a throwaway sketch to settle a design/behavior question cheaply." pstack's version is a
  playbook fired from inside `poteto-mode`, not its own top-level skill/slash command, so no literal
  registry collision, only conceptual duplication.
- pstack's `poteto-mode` `multi-phase-plan` / `orchestrate` playbooks vs. mattpocock's `wayfinder`
  (map huge work as decision tickets on the tracker) — same "work spans more than one session"
  problem, different mechanics, no naming clash.

**Not overlapping**: pstack's 23 `principle-*` skills (laziness-protocol, subtract-before-you-add,
type-system-discipline, prove-it-works, etc.) have no equivalent skill-per-principle structure in
mattpocock/skills — that repo folds equivalent judgment calls into the prose of `implement`,
`code-review`, and `codebase-design` rather than shipping one skill per rule. pstack's `swarm`,
`arena`, `blast-radius`, `make-bot-ui`, `automate-me`, `reflect`, `show-me-your-work`,
`create-verification-skill` / `maintain-verification-skill`, `unslop`, `bro`, and
`technical-writing` also have no mattpocock counterpart by name or clear functional overlap.

Sources: https://github.com/mattpocock/skills/blob/main/.claude-plugin/plugin.json ,
https://github.com/mattpocock/skills/blob/main/skills/engineering/tdd/SKILL.md ,
https://github.com/cursor/plugins/blob/main/pstack/skills/tdd/SKILL.md ,
https://github.com/mattpocock/skills/blob/main/skills/productivity/teach/SKILL.md ,
https://github.com/cursor/plugins/blob/main/pstack/skills/teach/SKILL.md ,
https://github.com/mattpocock/skills/blob/main/README.md#reference

## 6. Bottom line

Usable side by side, but not as an uncurated dump of both `skills/` directories into one flat
folder — `tdd` and `teach` collide by name with incompatible behavior in both cases, and several
other skills duplicate the same job under a different name (`architect`/`codebase-design`,
`interrogate`/`code-review`, `how`+`why`/`research`). The clean way to combine them: install
`mattpocock/skills` as documented (Claude Code plugin or `npx skills add`, both of which keep it
namespaced/self-contained), then hand-pick the pstack skills that don't already have a mattpocock
equivalent — the 23 `principle-*` skills, `blast-radius`, `swarm`, `arena`, `show-me-your-work`,
`unslop`, `technical-writing` are the ones with no collision — and skip or rename pstack's `tdd` and
`teach` rather than installing them as-is. Because pstack itself only installs through Cursor's
`/add-plugin`, that curation has to happen by hand-copying the chosen `SKILL.md` files into the
other harness's skills directory; there's no tooling on pstack's side to do it for you.

One-line credit for a README inspiration section: "pstack (Lauren Tan / Cursor) — a 23-principle
engineering-discipline pack and a `poteto-mode` playbook router, MIT-licensed inside Cursor's
official plugin marketplace."
