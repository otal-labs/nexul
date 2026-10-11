# 08 — The role editor as a dialog

**Status:** ready-for-agent

**Blocked by:** None — can start immediately (the look is locked in permission-rework ticket 08)

**Branch:** master, a normal PR, before the switch. Today's sections only; the new ones arrive in slice 15.

Read first: `practices/react-guide.md` (F1 to F7, State management, Forms, the self-review),
`practices/design-language.md`, `practices/testing.md`, ADR 0148, the spec (UI),
`.scratch/permission-rework/issues/08-the-role-editor-dialog.md` (all of it).

Per rework ticket 08's answer: the inline `RoleEditForm` and `CreateRoleForm` become one dialog, so editing a role
doesn't spread down the roles list. The list itself, Duplicate, Clone to workspace and Delete are unchanged.

## What to build

- The row menu's Edit and the card footer's New role open the same dialog: the house dialog at 40rem
  (`sm:max-w-[40rem]`), `h-[min(52rem,calc(100dvh-4rem))]`, in three parts like `TeamPersonDialog`, at the same width at
  768, 1024 and 1440px.
  - Header: "Edit <role>" or "New role", and on an edit the holders as overlapping 20px avatars with the muted line
    naming them ("Held by Bob and Lena. Saving changes what they both can do.", "Nobody holds this role yet."). No
    autofocus.
  - Body: the only part that scrolls.
  - Footer, always in view: the muted note ("No changes yet", "Name it to create it", or a dot and "Unsaved changes"
    with the existing `settle-in`), a ghost Cancel, and "Save" or "Create role", disabled until something changed and
    the name isn't blank.
- Name, and on a new role "Start from" ("Nothing allowed", or "A copy of <role>" for every role but Owner), beside it
  from a 32rem body and under it below; picking a copy replaces the draft and shows "Copied from <role>. Change
  anything below; <role> stays as it is."
- Today's two sections, Workspace and Every project, each a microheader with its mono tally ("12 of 24 allowed"), a
  muted line and one bordered list:
  - a bulk row per section on the `bg-muted/40` strip with the four-rung strip, selecting no rung once the rows differ,
    replacing "Every domain" and "Every area";
  - an area row per domain with the existing level strip;
  - each area's extra verbs as checkboxes on a second line under its name (Run, Threads, Clone, Lock, Logs, Create),
    the full label as tooltip and `aria-label`, disabled at None with "Needs Read first", and cleared by None as
    `withLevel` already does;
  - a domain with no ladder renders as switches.
- The instance areas stay in the Workspace section, after the workspace areas, in catalog order.
- Leaving with changes (Cancel, ×, Escape, scrim) asks "Discard changes?" with the role's or the new role's line;
  with nothing changed it closes at once. Save shows the usual toast and closes.
- The draft is one held value (name and actions) compared with what was loaded. A store selector's fallback for a
  closed dialog is a shared constant empty array, never a fresh `[]`, which loops React on close (found in the
  prototype).
- The guide page `people-and-access.md`, where it shows the role editor.

## Acceptance criteria

- [ ] A role is created, edited, copied from another and cloned through the dialog, over the existing routes.
- [ ] Save stays in view as the body scrolls, at the 60-character worst-case name too.
- [ ] Leaving with changes asks first; leaving with none doesn't.
- [ ] Closing the dialog renders no loop (a test that opens and closes it).
- [ ] `RoleSettingsSection.test.tsx` covers the dialog in place of the inline forms.
- [ ] Verified at 768, 1024 and 1440px in both modes, and the self-review passes.
