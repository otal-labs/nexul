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

Then, under **Private keys** on the same page, click **Generate a private key**. GitHub downloads a `.pem` file; keep it for step 3.

## 2. Install it

On the App's page, click **Install App**, pick your account, and choose **All repositories** or the ones you want to deploy. Nexul only sees repositories in accounts the App is installed on.

## 3. Connect it to Nexul

1. In the setup wizard, paste the client ID, client secret and App slug.
2. In the owner wizard's **Connect your tools** step, or later under **Settings → Connectors**, click **Connect** on GitHub and approve. If you already authorized the App while installing it, GitHub skips the approval screen.
3. Open **Settings → Connectors → GitHub App**, click **Add private key**, paste the whole `.pem` file, then **Verify** and **Save**. Nexul checks the key with GitHub first and never shows it again; **Replace private key** swaps it and **Remove key** takes it away.

To change the client ID or secret later, open **Settings → Connectors → GitHub App** and click **Edit**.

## Reading as the App

With the private key set, Nexul reads GitHub as the App itself, with a short-lived token for each account the App is installed on. Installing the App on an account is all it takes for its repositories to be readable: listing, scanning, cloning, webhooks and pull requests all work whoever clicked **Connect**.

Without the key, Nexul reads GitHub as the account that clicked **Connect**, and the GitHub App card and the project wizard say that only that account's repositories are visible. A repository then lists only when the App is installed on its owner and the connected account can open it, and adding its webhook needs admin rights on it. An instance that upgrades keeps working this way until a key is added.

## Adding an account or organisation

Each installation belongs to the workspaces that list its repositories, so one client's workspace never sees another's. **Settings → Connectors → GitHub App** lists every account the App is installed on under **Installations**, with whether it grants all repositories or a selection, and the workspaces it is assigned to.

- **From the project wizard.** **Install it on another account or organisation**, under the repository search, opens GitHub with this workspace attached. Once the owner of that account installs the App, GitHub sends them back to Nexul and the account joins the workspace. The link is good for a week, so you can send it to a client to install.
- **From Settings.** **Add account or organisation** installs the App with no workspace attached. The new installation shows as **Unassigned** until someone who manages connectors picks a workspace for it with **Assign to…**. The **×** next to a workspace takes the installation away from it again.

Upgrading assigns each account to every workspace whose projects already use one of its repositories. Agents see the installations, with their workspaces, through `repository_list` with `installations` set, and assign them with `workspace_update`.

## Changing permissions later

Adding or raising a permission on the App does not reach existing installations. GitHub asks each installation's owner to accept it: **Settings → Applications → Installed GitHub Apps → Configure → Review request**. Until they accept, anything that needs the new permission keeps failing.

## Which token does what

- Sign-in uses your own GitHub token, only to read your profile.
- With the private key set, the server reads repositories, pull requests and webhooks with an installation token of the App, and hands a runner one for its build. Each lasts an hour. If the key ever leaks, delete it on the App's page on GitHub, generate a new one, and **Replace private key**.
- Without the key, the connector token, stored when you click **Connect**, does all of that. If it ever leaks, click **Disconnect**, confirm, and then **Connect** for a fresh one.
