# nexul.io on Fumadocs

## Problem

The docs feel bolted on. Starlight's chrome (header, sidebar, search box) reads
as a stock theme next to the marketing home, and the guide pages read like a
spec: capitalised domain terms, a definition first, field lists, nothing to
scan. Several shipped features have no guide page.

## Decisions

- Fumadocs UI inside the existing Astro site, rendered as React islands. The
  home, roadmap and changelog stay Astro pages. If the Astro integration
  cannot cover search, the page tree or static output, fall back to a Next.js
  static export and record why in the PR.
- Tailwind v4 and React join `website/` as dependencies; both are what
  Fumadocs requires.
- Mono Console, not a new theme: the `--color-fd-*` tokens take the values
  from `practices/design-language.md`, dark default, light a true inversion,
  no accent hue, Inter and JetBrains Mono bundled locally.
- Look: docs home is a card grid per sidebar group (filled card one surface
  step above the canvas, no border, icon top-left, title and one line at the
  bottom). Article pages: section eyebrow, title, one-line lead from the
  page description, a copy-as-markdown control, right-hand table of contents
  from 1280px.
- One header across the whole site: wordmark, Docs, Roadmap, Changelog,
  GitHub, search on docs pages, theme toggle.
- Static search (Orama) built at build time; no server.
- `src/lib/sidebar.ts` stays the one place the guide's group order lives, and
  `test/sidebar.test.ts` keeps guarding it.
- Mobile first: 320, 375, 414, 768px, both schemes, keyboard reachable.
- Guide voice: what the reader does and sees, second person, short. Domain
  terms in lower case unless they are a UI label. Facts come from the code.

## Out of scope

Bots (not built). Versioned docs. i18n. Generated API reference pages.

## Done when

`bun run test` and `bun run build` pass in `website/`, every existing URL under
`/docs/` still resolves, and the pages render correctly at the four widths.
