---
title: This website
description: How the documentation site itself is built and deployed.
sidebar:
  order: 7
---

`website/` is its own Astro project. The home, roadmap and changelog are
Astro pages; the docs render with Fumadocs UI as a React island, styled with
Tailwind. It is not part of the Go module and has its own `bun` toolchain,
independent of `web/`.

## Running it

```sh
cd website
bun install
bun run dev     # local dev server
bun run build   # static build to dist/
```

## Where pages live

Content pages are markdown files under
`website/src/content/docs/docs/`, split by section:

- `guide/` — the user-facing guide.
- `contributing/` — this section.

Every page needs frontmatter:

```yaml
---
title: <Sentence case, short>
description: <one sentence>
sidebar:
  order: <position within its directory>
---
```

Pages are markdown only; a page that needs a component is a change to the
site, not to the page.

The sidebar is built from `website/src/lib/sidebar.ts`. The guide's groups
(Get started, Deploy, Work together, Agents, Reference) and their order are
listed there, so a new guide page goes into the group it belongs to; `bun run
test` fails while a guide page is missing from it, and the build fails if
the file names a page that does not exist. Contributing pages are every page
in `contributing/`, ordered by `sidebar.order`. `src/lib/docs-tree.ts` turns
both into the page tree, and the docs home shows one card grid per group from
the same list.

Every page is also served as markdown at its URL with `.md` in place of the
trailing slash (`/docs/guide/install.md`), which is what the Copy page button
copies. Search is a static index built into `/api/search` and searched in the
browser.

## Installer downloads

`public/install.sh` (Linux and macOS) and `public/install.ps1` (Windows) are
served directly at the site root. They only download the `nexul` binary, check
its checksum and run `nexul install`; everything else lives in the binary
(`internal/install/`). Run `bun run test` before changing `install.sh`.

## Styling

`website/src/styles/tokens.css` maps a monochrome palette onto
Fumadocs' `--color-fd-*` variables for both color schemes; every page loads
it. The one header (`src/components/SiteHeader.tsx`) is styled by
`src/styles/site-header.css` and shared by every page; on docs pages it also
holds search, the theme toggle and, below 768px, the menu. The docs add
`src/styles/docs.css` (Tailwind, the Fumadocs preset and the overrides) and
their components in `src/components/docs/`; code blocks use the monochrome
theme pair in `src/lib/shiki-mono.ts`. The homepage, roadmap and changelog
use `src/styles/landing.css`; the homepage adds `src/styles/home.css`, and the
roadmap, changelog and 404 page add `src/styles/pages.css`. Fonts are bundled
locally (see [Coding standards](/docs/contributing/coding-standards/)).

The homepage shows the product as screenshots of the app running the seeded
example workspace (Northwind), never a real instance. The sources are
`src/assets/home/<name>-<dark|light>.webp`, captured at 1440x900 and 2x in each
mode with avatar photos blocked so people show their gradient avatars, then
cropped; `src/components/ProductShot.astro` turns each pair into AVIF and WebP
at several widths and lets the page's theme pick one. Recapture them when the
screens they show change, keeping the same crops: the full board for the
hero, its top-left 480x600 for phones, and 640x480 of the doc, ticket, deploy
and channel pages for the steps.

Verify the homepage at 320, 375, 414, 768, 1024 and 1440px in both color
schemes and with the header's theme toggle, including keyboard navigation and
the install command's copy feedback. The same install block appears in the
hero and the closing install section. Keep the command on one line and the
copy control inline; a command wider than its box scrolls with a fading edge.

## Roadmap and changelog

The roadmap's items live in `src/pages/roadmap.astro`, grouped on the page by
status, newest first within each; `ROADMAP.md` at the repository root carries
the same items, so change both together. The changelog is built from the
repository's GitHub releases at build time (`src/lib/releases.ts`, every page
of the API, tags starting with `v`), one row per release with the pull
requests its notes list. Each new release rebuilds the site (Hosting, below). Set `GITHUB_TOKEN` when building locally
to stay clear of the API's rate limit.

## Hosting

The site deploys via Cloudflare Pages, connected directly to the GitHub
repository:

| Setting | Value |
|---|---|
| Root directory | `website` |
| Build command | `bun run build` |
| Output directory | `dist` |
| Production branch | `master` |

A push to `master` that changes anything under `website/` redeploys the
site, and so does every release, through the deploy hook in the
`CLOUDFLARE_PAGES_DEPLOY_HOOK` secret; nothing else triggers a Pages build.
