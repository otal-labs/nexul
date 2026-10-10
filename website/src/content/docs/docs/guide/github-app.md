---
title: GitHub App
description: Create the GitHub App your instance signs people in with and reads repositories through.
sidebar:
  order: 4
---

You create one GitHub App per instance, then paste its details into the [setup wizard](/docs/guide/setup-wizard/). It has to be a GitHub App, not an OAuth App: only a GitHub App has per-repository permissions and installations, and an OAuth App cannot be converted later.

## 1. Create the App

On GitHub, go to **Settings → Developer settings → GitHub Apps → New GitHub App** and fill in:

| Field | Value |
| --- | --- |
| GitHub App name | Anything. Nexul asks for the slug it produces. |
| Homepage URL | Your instance URL, such as `https://nexul.example.com` |
| Callback URLs | `<instance-url>/auth/callback` for sign-in and `<instance-url>/auth/connectors/github/callback` for the connector |
| Expire user authorization tokens | On |
| Request user authorization (OAuth) during installation | On |
| Webhook | Off. Nexul adds a webhook to each repository you attach to a project. |

On a Mac or Windows install the instance URL is `http://localhost:5123`.

Give it these repository permissions and nothing else:

| Permission | Level |
| --- | --- |
| Contents | Read |
| Pull requests | Read and write |
| Webhooks | Read and write |
| Metadata | Read (GitHub adds it) |

Without Contents: Read, a runner cloning a private repository fails with "Write access to repository not granted".

Generate a client secret on the App's page and copy it. GitHub shows it once.

## 2. Install it

On the App's page, click **Install App**, pick your account, and choose **All repositories** or the ones you want to deploy. Nexul only sees repositories in accounts the App is installed on.

## 3. Connect it to Nexul

1. In the setup wizard, paste the client ID, client secret and App slug.
2. In the owner wizard's **Connect your tools** step, or later under **Settings → Connectors**, click **Connect** on GitHub and approve. If you already authorized the App while installing it, GitHub skips the approval screen.

To change the client ID or secret later, open **Settings → Connectors → GitHub App** and click **Edit**.

## Adding an account or organisation

**Settings → Connectors → GitHub App** lists every account the App is installed on under **Installations**, with whether it grants all repositories or a selection. Click **Add account or organisation** to install it somewhere else; the list updates when you come back to the tab.

Nexul reads GitHub as the account that clicked **Connect**, not as each person signed in. So it lists a repository only when the App is installed on the account that owns it and the connected account can open it. For a repository in someone else's account, such as a client's, its owner does both:

1. Installs the App there, from `https://github.com/apps/<slug>/installations/new`, with that repository selected.
2. Gives the connected account access to the repository: a collaborator with admin rights, or an organisation member with admin rights on it. Admin is what lets Nexul add its webhook; with less, the repository lists and deploys but pull request changes don't arrive as they happen.

Installing the App alone is not enough: the repository stays out of the list until the connected account can open it. Agents see the same list through `repository_list` with `installations` set.

## Changing permissions later

Adding or raising a permission on the App does not reach existing installations. GitHub asks each installation's owner to accept it: **Settings → Applications → Installed GitHub Apps → Configure → Review request**. Until they accept, anything that needs the new permission keeps failing.

## Which token does what

- Sign-in uses your own GitHub token, only to read your profile.
- The connector token, stored when you click **Connect**, is what the server uses for repositories, pull requests and webhooks, and what it hands a runner for a build. If it ever leaks, click **Disconnect**, confirm, and then **Connect** for a fresh one.
