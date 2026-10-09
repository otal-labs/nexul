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

The site speaks the one design language the web app and the phone app share:
[`practices/design-language.md`](https://github.com/otal-labs/nexul/blob/master/practices/design-language.md),
its shared core and its Public site section, is the spec for every visual
change here. `src/styles/tokens.css` carries the app's
token values under their own names (`--background`, `--card`, `--brand`,
`--focus`, the status hues, the panel and field values, shadows and curves)
and maps Fumadocs' `--color-fd-*` variables onto them; colours use
`light-dark()`, so they follow the system scheme until the header's theme
toggle sets `.light` or `.dark` on `<html>` (kept in `localStorage` under
`theme`). Every page loads it. Never add a colour that is not one of these
tokens.

`src/components/SiteHead.astro` is every page's shared head: the fonts and
their preloads, and an inline script that sets the theme class before first
paint. The one header (`src/components/SiteHeader.tsx`) is styled by
`src/styles/site-header.css` and shared by every page, with the theme toggle
and, below 768px, the phone menu; `src/lib/site-chrome.ts` runs both without a
framework, so the landing pages ship no React. On docs pages the header also
holds search, and the phone menu is the docs sidebar drawer, which ends with
the same links. The docs add `src/styles/docs.css` (Tailwind, the Fumadocs
preset and the overrides) and their components in `src/components/docs/`;
code blocks use the console theme in `src/lib/shiki-mono.ts`. The homepage,
roadmap and changelog use `src/styles/landing.css`; the homepage adds
`src/styles/home.css`, and the roadmap and changelog add
`src/styles/pages.css`. The 404 page carries its own styles.

Fonts are bundled locally. Inter and JetBrains Mono come from their
`@fontsource-variable` packages. The display face is one file,
`src/assets/fonts/fraunces-display-latin.woff2`: Fraunces' Latin build from
`@fontsource-variable/fraunces` with weight, SOFT and WONK pinned to the
values the design language uses, keeping optical sizing, which brings it from
121KB to 34KB. Regenerate it after upgrading that package:

```sh
pip install fonttools brotli
fonttools varLib.instancer \
  node_modules/@fontsource-variable/fraunces/files/fraunces-latin-full-normal.woff2 \
  wght=560 SOFT=50 WONK=1 -o src/assets/fonts/fraunces-display-latin.woff2
```

Verify docs pages at 320, 375, 414, 768, 1024 and 1440px in both color
schemes, with the phone menu open below 768px.

The homepage shows the product as screenshots of the app running the seeded
example workspace (Northwind), never a real instance. Every shot follows one
feature, passkey sign-in, in its Accounts project, and the step copy narrates
the same feature, so change both together. The sources are
`src/assets/home/<name>-<dark|light>.webp`, captured at 1440x900 and 2x in each
mode, then cropped; `src/components/ProductShot.astro` turns each pair into
AVIF and WebP at several widths and lets the page's theme pick one. Seeded
people have logins no GitHub account can have (`bob_nw`), so they show their
gradient avatars; never capture people whose login could be a real GitHub
account, because the app shows `github.com/<login>.png` for anyone without a
picture.

Recapture them when the screens they show change. Set the app's appearance
for each mode (the app keeps its own, it does not follow the system), dismiss
the board's interview banner, and keep the crops, given in CSS pixels on the
1440x900 page:

| Shot | Page | Crop (x, y, width, height) |
|---|---|---|
| `board` | the Accounts board | the whole page |
| `board-narrow` | the same board, for phones | 256, 24, 480, 600 |
| `doc` | the Passkey sign-in doc | 790, 176, 640, 480 (the Doc tab and the text) |
| `ticket` | ticket ACC-3 | 580, 25, 640, 480 |
| `deploy` | the latest accounts-api deploy | 345, 30, 640, 480 |
| `chat` | #engineering, scrolled to the end | 462, 339, 640, 480 (the bot card and the replies) |

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
