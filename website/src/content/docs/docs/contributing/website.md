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

## Installer downloads

`public/install.sh` and `public/install.ps1` are served directly at the site root.
They bootstrap a fresh clone and delegate setup to the repository installers.
Keep provisioning logic in the root scripts. Run `bun run test` before changing
the download entry points.

## Styling

The documentation theme lives in `website/src/styles/mono-console.css`.
The homepage uses `website/src/styles/landing.css`, with its example workflow
in `website/src/components/WorkflowPreview.astro`. Both use the product's
monochrome surfaces and locally bundled fonts (see
[Coding Standards](/docs/contributing/coding-standards/)).

The workflow is an illustration with example data, not a live instance.
Keep its documents, ticket states, and deployment details consistent with
the product when editing it. Verify the homepage at 320, 375, 414, and 768px
in both color schemes, including keyboard navigation and installation-command
copy feedback.

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
