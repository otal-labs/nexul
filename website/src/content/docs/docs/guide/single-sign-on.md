---
title: Single sign-on
description: Let people sign in through your company's identity provider over OpenID Connect or SAML, turn every other sign-in off, and keep the instance Owner's passkey as the way back in.
sidebar:
  order: 4
---

Single sign-on is an [enterprise feature](/docs/guide/enterprise-license/): it needs a license key that unlocks it. People then sign in with the identity provider your company already runs, such as Okta, Microsoft Entra, Google Workspace, Authentik or Keycloak, instead of GitHub, Google or Discord. One provider covers the whole instance, over OpenID Connect or SAML. Provisioning people from the provider is not part of it.

Without a key that unlocks it, **Settings → Sign-in providers** shows **Single sign-on needs a license key**, the button stays off the sign-in page, and anything you saved is kept for when a key is back.

## Set it up over OpenID Connect

1. At your provider, register Nexul as an application (a confidential, web, authorization-code client). Copy the **Redirect URI** from the **Single sign-on** card in **Settings → Sign-in providers** and register it there. It is the instance address followed by `/auth/oidc/callback`.
2. On the card, click **Set up OpenID Connect** and enter:
   - **Issuer URL**: the provider's address, such as `https://login.example.com`. It must be https; plain http is taken only for `localhost` while you try it out.
   - **Client ID** and **Client secret**. The secret can be a [secret reference](/docs/guide/external-secrets/) such as `op://vault/item/field`. It is never shown again; leave it blank when you edit to keep the saved one.
   - **Name on the button**, up to 40 characters: the button reads "Sign in with Acme SSO".
   - **New people join as**: the role a person gets in the default workspace when their first sign-in makes their account. Owner is not offered.
3. Click **Verify**. One check asks the issuer for its sign-in endpoints and one reads the secret; each shows a tick or the reason it failed. **Confirm** saves once both are green.

Only someone whose role holds the instance's write permission, the instance Owner by default, can change it. Anyone who can see Settings → Instance sees the card.

The provider must send the person's `email` and `email_verified` in the ID token; Nexul asks for the `openid`, `profile` and `email` scopes. A provider that sends no verified email is refused at sign-in.

## Set it up over SAML

Use SAML when your provider speaks nothing else. An instance holds one provider: while an OpenID Connect provider is saved the card offers no SAML setup, so remove that one first, and the same the other way round. Nobody's account changes, and each person's first sign-in through the new provider is matched as in Who signs in below: someone whose only sign-in was the old provider gets a second account, which you can join onto their own.

1. At your provider, add Nexul as a SAML application. It needs three values, all on the **Single sign-on** card once SAML is saved, and in the setup dialog before:
   - **Entity ID** (the audience): the instance address followed by `/auth/saml/metadata`.
   - **Assertion consumer service URL** (the reply or ACS URL), taking the HTTP-POST binding: the instance address followed by `/auth/saml/acs`.
   - After the first save, Nexul's **metadata** at that same `/auth/saml/metadata` address, which carries its signing certificate. Give the provider the link if it reads metadata, or the certificate if it wants requests signed.
2. Tell the provider to sign its assertions (or the whole response), to send a **persistent** NameID (Entra's default; in Okta, "Persistent" or the user's id, not "Transient"), and to send the person's email as an attribute.
3. On the card, click **Set up SAML** and enter:
   - The provider's **metadata**: its **Metadata URL** (https; Nexul reads it now and once a day), or the **Metadata XML** pasted or read from its file (kept as it is; upload again to replace it).
   - **Email attribute**: the attribute the address comes in, `email` unless you change it. Entra sends a long claim name such as `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress`; any name works.
   - **Name attribute**, if the provider sends one for the person's display name.
   - **Name on the button** and **New people join as**, as above.
4. Click **Verify**. One check reads the metadata and says which provider and how many signing certificates it names; one checks the email attribute is named. **Confirm** saves once both are green.

SAML has no "email verified" flag. A signed assertion from your provider is taken as the provider vouching for the address in the email attribute, so a provider that lets people type their own email there lets them sign in as someone else here. The card says so beside the attribute.

Matching is as in Who signs in below, with the provider's entity id and the person's NameID as their sign-in identity. A sign-in that sends no email, or a transient NameID, is refused and the sign-in page says single sign-on didn't work; the reason is in the server's log.

### Certificates

Every signing certificate in the provider's metadata verifies its responses. When the provider rotates its certificate, a metadata URL picks the new one up at its next daily read, or at once with **Refresh** on the **Provider metadata** row; while the provider publishes both, both work. If a read fails, the certificates from the last good read stay in use and the row shows **Refresh failing** with why. Uploaded metadata never changes until you upload again. A metadata URL that starts naming another entity id is refused, since that is another provider: remove this one and set that one up.

Nexul's own certificate is made at the first save and runs for five years. Each certificate has a row with its expiry, which turns to **Expires soon** 30 days before. **Rotate** makes Nexul a new one, signs its requests with it from then on, and keeps the old one in its metadata beside it, so a provider that cached the old metadata keeps working. Once your provider has the new certificate, **Retire** on **Nexul's old certificate** drops the old one; you can rotate again after that.

### What SAML does not do

- Sign-in started at the provider (IdP-initiated): a sign-in always starts at Nexul's button, so every response answers a request Nexul made. Requests wait 10 minutes and are used once.
- Encrypted assertions: https carries the assertion. Turn assertion encryption off at the provider.
- Checking a signature on the provider's metadata: a metadata URL is protected by its https certificate alone, so use one on the provider's own domain.
- Several providers at once, or OpenID Connect and SAML together.
- Provisioning or removing people from the provider: accounts are made at their first sign-in. SAML has no refresh token, so someone removed at the provider keeps their Nexul sessions until you deactivate them under **Team**.

## Who signs in

The first time someone signs in through the provider:

- if the provider's verified email is the email of an existing account's Google or Discord sign-in, that sign-in is linked to the account and they land in it. Accounts that only sign in with GitHub hold no email, so they are not matched; those people get a second account the first time, which you can join onto their own (below).
- otherwise their account is made, joined to the default workspace with the **New people join as** role, as an invitation would. They then see the same Welcome steps as anyone new.

After that the provider's identity is the account's sign-in, whatever email they use later. A disabled or removed account is refused, as at every other sign-in. Invitations still work as they did: the person picks the provider on the invitation page, and signs in through single sign-on afterwards.

### Join a second account onto the first

When someone's first sign-in made a second account, the instance Owner moves the sign-in over:

1. Open **Settings → Team** and click the person's own account.
2. Under **Single sign-on**, click **Move from another account** and pick the account the sign-in made; only accounts whose one way in is single sign-on are listed.
3. Confirm with **Move sign-in**.

From then on, signing in through the provider opens their own account. The second account is disabled and signed out, and anything it made stays; remove it from Team when you no longer need it. The row also says **Signs in through** your provider for anyone already linked.

## Only single sign-on

The **Only single sign-on** switch hides every other way to sign in. The sign-in page shows the one button and, under it, **Instance Owner? Use your passkey**; the server refuses GitHub, Google, Discord and everyone else's passkeys too, so hiding is not the only guard.

The instance Owner's passkey always still works, so a provider that breaks, or a client secret that expires, never locks everyone out. For that reason the switch turns on only while an instance Owner holds a passkey; otherwise it says to add one under **Profile → Passkeys** first.

Turning the switch on signs out everyone who didn't sign in through single sign-on, on every device, so nobody keeps a way in the switch now refuses. Instance Owners stay signed in. A phone connected from a single sign-on session counts as one. Personal access tokens are not sessions and keep working; deactivate the person under **Team** to stop theirs.

## When someone is removed at the provider

People who sign in through the provider are checked against it about once an hour, when the provider issues refresh tokens (Nexul asks for them with the `offline_access` scope when the provider lists it). Someone removed or disabled at the provider is signed out of their single sign-on sessions within about an hour. A provider that issues no refresh tokens never re-checks anyone, and deactivating the person under **Team** is the way to end their access then. If the provider is down or rejects the client secret, nobody is signed out.

## When the license lapses

When the key expires, is removed or stops fitting the instance's address, the switch goes and the other sign-in methods come back at once. The button stays for the people single sign-on already signs in, so nobody whose only way in is your provider is locked out, and someone removed at the provider is still signed out within the hour. It makes no new accounts and links none by email until the key is renewed. The card says it needs a license, keeps what you saved, and still lets you remove it. Paste a renewed key and single sign-on is back as it was.

## Remove it

**Remove provider** deletes the settings. The button goes, every other method comes back, and nobody's account changes: people who signed in through the provider keep their accounts and their other linked sign-ins.

## Over MCP

`instance_get` returns the single sign-on settings with the license, never the client secret. `instance_update` takes an `sso` object with `issuer_url`, `client_id`, `client_secret`, `display_name`, `join_role_id` and `only`; an omitted field keeps its value, and `remove: true` deletes the provider. The provider is reached, and a secret reference read, before anything is stored. `workspace_list` with the default workspace's id lists the roles for `join_role_id`. `account_update` with `sso_from` set to the second account's id joins it onto the account named by `id`, as above. The same permissions apply as on the web page, and the card updates on every open Settings page at once.

For SAML, `instance_get` returns a `saml` object beside it, with the values to register (`sp`), each certificate's expiry and whether the last metadata read failed, never Nexul's private key. `instance_update` takes a `saml` object with `metadata_url` or `metadata_xml`, `email_attribute`, `name_attribute`, `display_name`, `join_role_id` and `only`; sending the saved `metadata_url` again reads it again, `rotate: true` and `retire: true` rotate and retire Nexul's certificate, and `remove: true` deletes the provider. Saving one protocol while the other is saved is refused with what to remove first.
