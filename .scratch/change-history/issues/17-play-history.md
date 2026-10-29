# 17 — Play history, and which version a run used

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, decisions 2, 5, 10, Storage.

`play_instruction_versions` and `play_changes`; each save writes a version
when the instructions change and a change per other field; the migration
seeds version 1 per play, "Built-in default" for an unedited built-in.
`play_trails.instructions_version` is set when a run starts. History and
restore routes; `play_list` with an id takes `version`, `play_update` takes
`revert_to_version`.

Done when: tests cover a save with and without instruction changes, the
seed, a trail recording its version, prunes, restore, and the MCP paths.
