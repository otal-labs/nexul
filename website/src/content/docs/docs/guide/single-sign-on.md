---
title: Single sign-on
description: Let people sign in through your company's identity provider over OpenID Connect, turn every other sign-in off, and keep the instance Owner's passkey as the way back in.
sidebar:
  order: 4
---

Single sign-on is an [enterprise feature](/docs/guide/enterprise-license/): it needs a license key that unlocks it. People then sign in with the identity provider your company already runs, such as Okta, Microsoft Entra, Google Workspace, Authentik or Keycloak, instead of GitHub, Google or Discord. One provider covers the whole instance, over OpenID Connect. SAML and provisioning people from the provider are not part of it.

Without a key that unlocks it, **Settings → Sign-in providers** shows **Single sign-on needs a license key**, the button stays off the sign-in page, and anything you saved is kept for when a key is back.

## Set it up

1. At your provider, register Nexul as an application (a confidential, web, authorization-code client). Copy the **Redirect URI** from the **Single sign-on** card in **Settings → Sign-in providers** and register it there. It is the instance address followed by `/auth/oidc/callback`.
2. On the card, click **Set up provider** and enter:
   - **Issuer URL**: the provider's address, such as `https://login.example.com`. It must be https; plain http is taken only for `localhost` while you try it out.
   - **Client ID** and **Client secret**. The secret can be a [secret reference](/docs/guide/external-secrets/) such as `op://vault/item/field`. It is never shown again; leave it blank when you edit to keep the saved one.
   - **Name on the button**, up to 40 characters: the button reads "Sign in with Acme SSO".
   - **New people join as**: the role a person gets in the default workspace when their first sign-in makes their account. Owner is not offered.
3. Click **Verify**. One check asks the issuer for its sign-in endpoints and one reads the secret; each shows a tick or the reason it failed. **Confirm** saves once both are green.

Only someone whose role holds the instance's write permission, the instance Owner by default, can change it. Anyone who can see Settings → Instance sees the card.

The provider must send the person's `email` and `email_verified` in the ID token; Nexul asks for the `openid`, `profile` and `email` scopes. A provider that sends no verified email is refused at sign-in.

## Who signs in

The first time someone signs in through the provider:

- if the provider's verified email is the email of an existing account's Google or Discord sign-in, that sign-in is linked to the account and they land in it. Accounts that only sign in with GitHub hold no email, so they are not matched; those people get a new account the first time, or can be removed from Team beforehand.
- otherwise their account is made, joined to the default workspace with the **New people join as** role, as an invitation would. They then see the same Welcome steps as anyone new.

After that the provider's identity is the account's sign-in, whatever email they use later. A disabled or removed account is refused, as at every other sign-in. Invitations still work as they did: the person picks the provider on the invitation page, and signs in through single sign-on afterwards.

## Only single sign-on

The **Only single sign-on** switch hides every other way to sign in. The sign-in page shows the one button and, under it, **Instance Owner? Use your passkey**; the server refuses GitHub, Google, Discord and everyone else's passkeys too, so hiding is not the only guard.

The instance Owner's passkey always still works, so a provider that breaks, or a client secret that expires, never locks everyone out. For that reason the switch turns on only while an instance Owner holds a passkey; otherwise it says to add one under **Profile → Passkeys** first.

Turning the switch on signs out everyone who didn't sign in through single sign-on, on every device, so nobody keeps a way in the switch now refuses. Instance Owners stay signed in. A phone connected from a single sign-on session counts as one. Personal access tokens are not sessions and keep working; deactivate the person under **Team** to stop theirs.

## When someone is removed at the provider

People who sign in through the provider are checked against it about once an hour, when the provider issues refresh tokens (Nexul asks for them with the `offline_access` scope when the provider lists it). Someone removed or disabled at the provider is signed out of their single sign-on sessions within about an hour. A provider that issues no refresh tokens never re-checks anyone, and deactivating the person under **Team** is the way to end their access then. If the provider is down or rejects the client secret, nobody is signed out.

## When the license lapses

When the key expires, is removed or stops fitting the instance's address, the button and the switch go with it, so the other sign-in methods come back at once and nobody is locked out. The card says it needs a license, keeps what you saved, and still lets you remove it. Paste a renewed key and single sign-on is back as it was.

## Remove it

**Remove provider** deletes the settings. The button goes, every other method comes back, and nobody's account changes: people who signed in through the provider keep their accounts and their other linked sign-ins.

## Over MCP

`instance_get` returns the single sign-on settings with the license, never the client secret. `instance_update` takes an `sso` object with `issuer_url`, `client_id`, `client_secret`, `display_name`, `join_role_id` and `only`; an omitted field keeps its value, and `remove: true` deletes the provider. The provider is reached, and a secret reference read, before anything is stored. `workspace_list` with the default workspace's id lists the roles for `join_role_id`. The same permissions apply as on the web page, and the card updates on every open Settings page at once.
