# Owner colors take any hex, not only the suggested hues

ADR 0005 limited the color of a category, a ticket type and a label to five suggested hues, so every color had a
light and a dark shade picked to hold contrast on either canvas. Owners asked for colors of their own.

Decision: a color field takes a suggested hue name (`cyan`, `emerald`, `orange`, `fuchsia`, `lime`) or a `#rrggbb`
hex, checked by `colors.Valid`. The web picker is a saturation and brightness square, a hue strip, a hex field and
quick hues; every new pick is stored as hex, and the five names stay valid so existing rows render as before.

- **A hex draws as picked in every theme.** It has no light and dark shade, so a color that disappears on one palette
  is the owner's choice. The web passes it to the class as `--hue`; a label pill's ink is the hue mixed toward
  `foreground` so its name stays readable.
- **Rejected: a larger fixed palette.** It keeps the contrast guarantee, but it still refuses the brand color an
  owner wants, which is what was asked for.
- **Saved once.** The label and ticket-type pickers save when their popover closes or a quick hue is picked, never on
  each step of a drag.

Supersedes the "fixed suggested palette (not freeform hex)" paragraph of ADR 0005.
