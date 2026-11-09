# A doc may run an HTML page in a sandbox

A doc's body is prose, tables, images and Mermaid diagrams. An agent or a person who wants a chart, a mockup or a styled
summary in it had nowhere to put one.

Decision: a code block in the `html-page` language runs its source as a page in the body. The source stays ordinary
body text, so markdown, MCP, history and search carry it unchanged.

- **Sandboxed on an opaque origin.** The web frame is `sandbox="allow-scripts"` over `srcdoc`, never
  `allow-same-origin`: a page can run script but cannot read the reader's session, storage or the doc around it. It can't
  submit forms, open popups or navigate the app; a link it posts opens in a new tab only while the reader is using the
  frame.
- **It may load from the internet.** Pages may pull a chart library, a font or an image from anywhere. The cost is that a page's author can see who opened the doc from the requests it makes; a doc's address is
  never sent (`no-referrer`). Blocking the network was rejected: agents reach for CDN libraries, and a self-contained
  rule would push every chart into hand-written SVG.
- **Theme through CSS variables.** The page gets the app's tokens on `:root` (`--foreground`, `--card`, `--border`,
  `--brand`, the status hues, `--radius`, `--font-sans`), resolved from the reader's palette and mode, and the app's own
  `@font-face` rules; the server allows fonts under `assets/` cross-origin for that. A theme change rebuilds the page
  rather than messaging it, so a page's own state resets when the reader switches theme.
- **Fitted to its content.** A bootstrap script in the page reports its height and the frame follows it, up to 20,000px,
  so the page never scrolls inside the doc.
