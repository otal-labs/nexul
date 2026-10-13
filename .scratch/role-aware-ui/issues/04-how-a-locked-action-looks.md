# 04: How a locked action looks

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

Build it in auto design mode, on the design language, at 768, 1024 and 1440 on the web and on the emulator for the
phone. Cover:
- a primary button (Deploy), an outline button, an icon button, a menu item (Delete in a row menu), a
  drag handle, and an inline edit
- how the reason reads: tooltip, popover on tap for the phone, or a line under the button
- keyboard and screen reader behaviour
- how it differs from a disabled-while-loading button and from today's locked pill

The two reason sentences:
- "Your role can't deploy", the role-wide one
- "Denied on stack api for bob", the source sentence

## Answer

Decided in auto design mode, 2026-10-13. The prototype is on the pushed branch `proto/locked-actions`
(`web/proto-locked.html?v=a|b|c&theme=dark|light&phone=1`), and is never merged. The report and the references are
in `~/Code/Images/role-aware-ui/`.

**Picked: "Quiet lock and tooltip"** (11/11). The runner-up is "Reason written out" (10/11).

- **Primary button**:
  - It keeps its height, radius and padding.
  - The fill becomes `foreground` at 6%, the label `muted-foreground`, and the leading icon a 14px `Lock`.
  - No hover lift, no press scale, `cursor: default`.
  - Label contrast is 6.35:1 dark and 5.54:1 light.
- **Outline button**: keeps its hairline border, drops its fill and shadow, and takes a muted label with a leading
  lock.
- **Icon button**: keeps its own glyph in muted ink, with a 10px lock badge on a 14px `card` disc at the
  bottom-right. Its tooltip gives the action on line one and the reason on line two.
- **Menu item**:
  - It stays in place and order, and loses the destructive red.
  - It takes a muted label, a 14px lock at the end, and the reason as a 12px muted second line.
  - Picking it does nothing, and the menu stays open.
- **Drag handle**: the grip goes to `muted-foreground` at 45%, with a default cursor, no hover fill, and the reason in
  a tooltip.
- **Inline edit**: the value stays full `foreground`, with no hover fill and no pencil. A lock fades in where the
  pencil was, over 150ms, and the reason shows in a tooltip.
- **The reason**: both kinds take the same place and style. "Your role can't <verb>" and the rule's source sentence
  differ only in wording. The app's tooltip timing applies (500ms, then instant), below a button and above a row
  control.
- **Accessibility**:
  - It is a real `button` with `aria-disabled="true"`, never `disabled`, so it stays in the tab order.
  - `aria-describedby` points at the reason, and the tooltip opens on focus.
  - Enter and Space do nothing.
- **Phone**:
  - A tap opens a small popover: "Deploy is locked" over the reason.
  - In an action sheet, the reason is a second line.
  - The touch target stays at least 44pt.
- **Unlike a loading button**: loading keeps the ember at half strength with a spinner, and is truly `disabled` and
  `aria-busy`. Locked has no ember, has a lock, stays focusable, and explains itself.
- **Unlike the locked pill**: the pill is a reference inside text. A locked action keeps its control's shape.

  Today's pill isn't focusable, so keyboard users never get its tooltip. A locked action must not copy that.
- **Runner-up**: the reason written under the control instead of a tooltip, with one line per list. It lost because it
  makes headers taller and crowds them at 768.
