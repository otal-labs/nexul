---
title: Setup Wizard
description: What happens the first time you open a freshly installed Nexul instance.
sidebar:
  order: 3
---

The first time you open your instance, nobody exists yet, so there's nothing to sign in to. Nexul walks you through instance bootstrap, then an owner wizard, before handing you the normal app.

## 1. Instance bootstrap (`/setup`)

Before anyone can sign in, the instance needs a [GitHub App](/docs/guide/github-app/) to authenticate through. This page asks for:

- **Instance URL** — defaults to the URL you're loading the page from.
- **GitHub OAuth client ID** and **client secret** — from the App's page.
- **GitHub App slug** — the name in the App's own URL, `github.com/apps/<slug>`.

The page shows you the exact OAuth callback URL to register on the App (`<instance-url>/auth/callback`), then verifies each value live, one row at a time:

1. The instance URL reaches this server.
2. The App slug resolves on GitHub.
3. The client secret is accepted.

Once every row is green, **Set up instance** hands off to GitHub's OAuth screen to sign you in as the first user.

If your instance is already configured, `/setup` refuses a second bootstrap — reload and sign in instead. You can still reach `/setup` later (a link, not hidden) if you ever need to point the instance at a different GitHub App.

## 2. Owner wizard

The very first person to sign in becomes the workspace owner and lands in a four-step wizard:

1. **Introduce yourself** — set the name and avatar other members will see, or keep the GitHub defaults.
2. **Set up your workspace** — name the workspace and its default project (and the project's ticket prefix).
3. **Connect your tools** — the same connectors list as Settings, so you can connect GitHub and Cloudflare right away.
4. **Set up DNS** — optional. Choose how traffic reaches this instance and point a hostname at it. See [Topology and DNS](/docs/guide/topology-and-dns/) for what each entry path does. You can skip this step and come back to it later.

## 3. Everyone else: first-login wizard

Teammates who sign in after the owner don't see the workspace setup — the workspace already exists. They get a single lightweight step: confirm the name and avatar they want to use, then land straight in the app.

## Next step

Once the owner wizard finishes, connect your first repository from the Runners page's onboarding or the project wizard — see [Runners](/docs/guide/runners/) and [Stacks and deploys](/docs/guide/stacks-and-deploys/).
