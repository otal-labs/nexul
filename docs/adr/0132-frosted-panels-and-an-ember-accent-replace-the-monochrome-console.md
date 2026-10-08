# Frosted panels and an ember accent replace the monochrome console

The web app was strictly monochrome: a near-black canvas, hairlines, no accent
hue anywhere in the chrome, colour reserved for status. After a pass that fixed
headers, widths and truncation page by page, the owner still found the product
flat and wanted it to stand out, and lifted the monochrome rules.

Decision: every page's content floats in a raised, frosted panel (12px radius,
8px from the canvas edges and from a neighbouring pane, the panel colour at 85%
with a backdrop blur, a hairline ring and a soft shadow) over a near-black
canvas that carries a slow, soft light field (an ember glow top right, blue
bottom left, a hint of pink). One ember accent, the `brand` token, marks the
primary action, focus, the active nav item, selection, your own chat messages,
checked controls and progress, and nothing else. People without a photo get a
seeded gradient avatar. Light mode is its own soft grey canvas with a pastel
field and white panels, not an inversion. `practices/design-language.md` holds
the values and the rules.

The trade-offs, accepted:

- Backdrop blur is the most expensive thing a browser paints, and the field
  moving behind the panels makes it repaint while it drifts. The field is
  gradients only, animated on transform alone, still under reduced motion,
  and the panels are opaque enough (85%) that dropping the blur on a slow
  machine changes little.
- An accent competes with status colour for attention, the reason the
  monochrome direction dropped it. It is held to the roles above; status
  stays a dot or icon beside plain text in its own hues, and the accent never
  fills a chip.
- Palettes now theme the accent: a palette's primary becomes its brand colour
  and tints the field, so a palette changes more of the app than before.

Decided 2026-10-08. Supersedes the "no accent" rule recorded in
`practices/design-language.md`; ADR 0003's monochrome resolution is history.
