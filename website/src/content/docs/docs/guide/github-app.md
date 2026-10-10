---
title: GitHub App
description: Create the GitHub App your instance signs people in with and reads repositories through.
sidebar:
  order: 4
---

You create one GitHub App per instance, then paste its details into the [setup wizard](/docs/guide/setup-wizard/). It has to be a GitHub App, not an OAuth App: only a GitHub App has per-repository permissions and installations, and an OAuth App cannot be converted later.

## Create an App during setup

On the GitHub step, choose **Create App on GitHub**. GitHub opens a registration with the callback addresses and required permissions filled in. Choose a unique name and create it. Back in Nexul, press **Finish setup**. Nexul receives and stores the private key with the App credentials, then opens sign-in. Use the same browser and setup pass throughout; the return expires after fifteen minutes and can be used once. If registration fails or expires, start it again.

The manual form below remains available for an App you already registered. Its private key can be added in Settings after signing in.

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

Without Contents: Read, a runner cloning a private repository fails with "Write access to repository not granted". **Create App on GitHub** asks for all of these. An App made before this version also asked for the organisation's Members: Read; Nexul no longer uses it, and you can take it off on the App's page.

Generate a client secret on the App's page and copy it. GitHub shows it once.

Then, under **Private keys** on the same page, click **Generate a private key**. GitHub downloads a `.pem` file; keep it for step 3.

## 2. Install it

On the App's page, click **Install App**, pick your account, and choose **All repositories** or the ones you want to deploy. Nexul only lists repositories in accounts the App is installed on, and to each person only the ones their own GitHub account can open.

## 3. Connect it to Nexul

1. In the setup wizard, paste the client ID, client secret and App slug.
2. In the owner wizard's **Connect your tools** step, or later under **Settings → Connectors**, click **Connect** on GitHub and approve. If you already authorized the App while installing it, GitHub skips the approval screen.
3. Open **Settings → Connectors → GitHub App**, click **Add private key**, paste the whole `.pem` file, then **Verify** and **Save**. Nexul checks the key with GitHub first and never shows it again; **Replace private key** swaps it and **Remove key** takes it away.

To change the client ID or secret later, open **Settings → Connectors → GitHub App** and click **Edit**. A new client ID is another App, so saving one removes the stored private key; add the new App's key afterwards.

## Whose repositories you see

Every person finds repositories through their own GitHub sign-in. The project wizard's search, its scan, the **Installations** list and an agent's `repository_list` and `repository_scan` read GitHub with that person's token, so they show exactly what that person can open wherever the App is installed, and nobody else's private repositories. Signing in with GitHub connects it; someone who signs in with Google or Discord clicks **Connect GitHub** in the wizard or under **Settings → Profile → GitHub**. Until then the wizard shows **Connect GitHub to see your repositories** instead of a list. See [Projects and repositories](/docs/guide/projects-and-repositories/).

The App's own access is only for work on repositories already attached to a project. With the private key set, Nexul clones, scans, adds webhooks and reads pull requests of an attached repository as the App, with a short-lived token for its account. Without the key it does that work as the account that clicked **Connect** under **Settings → Connectors**, which then needs access to the attached repositories, and admin rights on them for webhooks. That connected account never decides what anyone can find.

## Adding an account or organisation

**Settings → Connectors → GitHub App** lists, under **Installations**, the accounts and organisations the App is installed on that your own GitHub account can open, with whether each grants all repositories or a selection, and **Manage** to change that on GitHub. Without your GitHub connected, it shows the **Connect GitHub** prompt instead.

To add one, click **Add account or organisation**, or **Install it on an account or organisation you manage** under the wizard's search, and install the App there on GitHub. Its repositories then list for everyone whose GitHub account can open them. Installing links the account to no workspace. A client who wants you to deploy their repository installs the App on their account and gives your GitHub account access to the repository.

A workspace uses an account once someone who can open one of its repositories attaches that repository to one of its projects; the list shows those workspaces as **Used by**. The **×** next to a workspace, where you can manage its projects, detaches the account after you confirm: deploys, webhooks and pull request reads there stop reading its repositories as the App until someone attaches one again. Nobody assigns an account to a workspace by hand.

An account renamed on GitHub keeps its workspaces. If its owner uninstalls the App, it leaves the list and its workspaces stop using it; installing again links it only once a repository is attached again. Agents see the installations, with their workspaces, through `repository_list` with `installations` set, and detach one with `workspace_update`'s `remove_github_accounts`.

## Changing permissions later

Adding or raising a permission on the App does not reach existing installations. GitHub asks each installation's owner to accept it: **Settings → Applications → Installed GitHub Apps → Configure → Review request**. Until they accept, anything that needs the new permission keeps failing.

## Which token does what

- Your GitHub sign-in token reads your profile and lists your repositories. Nexul keeps it encrypted and refreshes it every eight hours while you use it; if GitHub refuses a refresh, you are asked to reconnect. **Disconnect** under **Settings → Profile → GitHub** forgets it.
- With the private key set, the server reads attached repositories, their pull requests and webhooks with an installation token of the App. A runner gets its own token for each build, minted for that one repository with read access to its contents and nothing else. Each lasts an hour. If the key ever leaks, delete it on the App's page on GitHub, generate a new one, and **Replace private key**.
- Without the key, the connector token, stored when you click **Connect**, does that work on attached repositories. If it ever leaks, click **Disconnect**, confirm, and then **Connect** for a fresh one.

Once a private key is set, repository reads and builds stay within accounts linked to the project's workspace by an attached repository. Detaching the account prevents new builds from being dispatched, even if the runner has its own GitHub credential or the repository is public. Pending builds fail with the authorization reason.
