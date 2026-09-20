---
title: GitHub App
description: Create and configure the one GitHub App your instance uses for sign-in, repositories, and runner downloads.
sidebar:
  order: 4
---

Nexul talks to GitHub through a single GitHub App. It signs people in,
connects your workspace to GitHub, and lets runners clone private application
repositories to build them. You create this App once, then paste its details
into the [setup wizard](/docs/guide/setup-wizard/).

## Why a GitHub App and not an OAuth App

It has to be a GitHub App, not an OAuth App: only a GitHub App has fine-grained repository permissions and installations, and an OAuth App can't be converted into one later.

## 1. Create the App

Go to GitHub → Settings → Developer settings → **GitHub Apps** → New GitHub App.

| Field | Value |
| --- | --- |
| GitHub App name | Anything — the URL slug it produces is what Nexul asks for. |
| Homepage URL | Your instance URL, e.g. `https://deploy.example.com`. |
| Callback URLs | `<instance>/auth/callback` (sign-in) and `<instance>/auth/connectors/github/callback` (connector). For local dev, add the same two paths under `http://localhost:5173` and `http://localhost`. |
| Expire user authorization tokens | On — Nexul refreshes tokens itself. |
| Request user authorization (OAuth) during installation | On. |
| Webhook | Leave inactive. Nexul registers per-repository webhooks itself. |

Set these repository permissions:

| Permission | Level |
| --- | --- |
| Metadata | Read (mandatory on every GitHub App) |
| Contents | Read |
| Pull requests | Read and write |
| Webhooks | Read and write |

Contents: Read matters more than it looks — without it, a runner cloning a private repository fails with "Write access to repository not granted". Grant no organization or account permissions, and no Actions permissions.

Generate a **client secret** on the App's page and save it. GitHub only shows it to you once.

## 2. Install it

On the App's page, click Install App, choose your account, and pick **All repositories**, or select the repositories you need. Every private application repository you want to deploy needs the App installed on it. Public repositories do not need an installation for cloning.

## 3. Paste it into Nexul

- **First boot** (`/setup`) — instance URL, OAuth client ID, client secret, App slug. This page shows the exact callback URL and verifies the slug against GitHub before letting you continue. See [Setup wizard](/docs/guide/setup-wizard/).
- **Owner wizard's "Connect your tools" step, or Settings → Connectors → GitHub → Connect** — the OAuth consent round trip. If you already authorized the App while installing it, there's no consent screen to click through.
- **Rotating credentials** — Settings' GitHub App card takes a new client ID and secret at any time. Nothing lives in environment variables.

## 4. When you change permissions later

Raising or adding a permission on the App doesn't apply to installations that already exist. GitHub sends the installation owner a request instead: Settings → Applications → Installed GitHub Apps → Configure → **Review request** → accept. Until you accept it, the token keeps its old permissions, and anything that needed the new one — a clone, a webhook call — keeps failing.

## Which token is used where

- Sign-in only uses the user's own token, and only to read their profile.
- The connector token — the one Settings → Connectors stores when you connect — is what the server uses for repositories, pull requests, webhooks, and release downloads, and what it hands a runner for a build job. Reconnecting in Settings issues a fresh token; do that if a token ever ends up somewhere it shouldn't.

## Next step

Once the App is installed and connected, add a [runner](/docs/guide/runners/) and deploy your first [stack](/docs/guide/stacks-and-deploys/).
