# 15 — The role editor's new sections, and the web's last Restricted reads

**Status:** ready-for-agent

**Blocked by:** 08, 11

**Branch:** `permission-switch`.

Read first: `practices/react-guide.md` (F1 to F7, the self-review), `practices/design-language.md`,
`practices/testing.md`, ADR 0148, the spec (New permissions, UI), `.scratch/permission-rework/issues/08-the-role-editor-dialog.md`.

## What to build

- The dialog from slice 08, per rework ticket 08's answer:
  - the "Every project" tab becomes **In projects**, "Only inside the projects a holder is in. Each project lists who's in
    it.", with its bulk row "All areas in projects"; Workspace reads "Across Acme, whatever project someone is in."
    with "All workspace areas";
  - **Create projects** leads the Workspace list and **Manage this project's access** leads In projects, each an
    ability row with a `Switch` and its muted line ("A new project lists its creator and the roles they pick.",
    "Choose who's in a project and what they can do there."); the bulk rows never touch the switches, and the tallies
    count switches that are on;
  - plays' first rung reads **See**, and its extras read "Needs See first" at None;
  - In this project never appears in the editor.
- The Instance tab stays as slice 08 built it (owner decision 18).
- `useNewConversationDialogs` and `ChannelSettingsCard` stop reading `restricted`.
- `RoleSettingsSection.test.tsx` and `ChannelSettings.test.tsx` are rewritten.

## Acceptance criteria

- [ ] `rg -w restricted web/src` finds nothing.
- [ ] Verified at 768, 1024 and 1440px, and the self-review passes.
