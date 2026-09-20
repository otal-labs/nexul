---
title: This Website
description: How the documentation site itself is built and deployed.
sidebar:
  order: 7
---

`website/` is its own Astro project using the Starlight documentation theme.
It is not part of the Go module and has its own `bun` toolchain, independent
of `web/`.

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
title: <Title Case, short>
description: <one sentence>
sidebar:
  order: <position within its directory>
---
```

Starlight autogenerates a sidebar group per directory under
`content/docs/docs/`, ordered by each page's `sidebar.order`. There is no
separate sidebar config file to keep in sync — add a markdown file with
frontmatter and it appears.

## Styling

The site's CSS lives in `website/src/styles/mono-console.css` — it carries
the same visual language as the product itself (see
[Coding Standards](/docs/contributing/coding-standards/) → Mono Console),
not a separate theme.

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
site; nothing else in the repository triggers a Pages build.
