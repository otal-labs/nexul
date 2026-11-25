---
title: Enterprise license
description: Paste the license key that unlocks the enterprise features on one instance, see what it covers, and renew it before it runs out.
sidebar:
  order: 3
---

Nexul's enterprise features, [single sign-on](/docs/guide/single-sign-on/), [branding](/docs/guide/branding/) and a longer, streamed [audit log](/docs/guide/audit-log/), run on an instance that holds a valid **license key**. Everything else in Nexul is free and stays free: a key only ever unlocks features that were built as enterprise features, never one that shipped without a key.

A key is a short block of text naming who it is for, the address of the one instance it covers, the features it unlocks and the date it expires. It is signed, and the server checks the signature against a public key built into Nexul. Nothing is sent anywhere to check it, so a key works on an instance with no internet access.

## Get a key

A key covers one instance and every enterprise feature, with no limit on people. It runs monthly or yearly, and a key fits one address, so each instance you run needs its own.

To get one, write to [sales@nexul.io](mailto:sales@nexul.io) with the instance's full public address, as it appears under **Instance URL** in Settings → Instance, and the name the key should carry. We reply within 2 business days. The key arrives as one block of text starting with `nexul-license-1.`.

[//]: # "TODO(pre-release/07-launch-copy): the license portal at cloud.nexul.io, once it is live"

### Plans

| Plan | Covers |
| --- | --- |
| Simple | [Single sign-on](/docs/guide/single-sign-on/) and [branding](/docs/guide/branding/) |
| Enterprise | Arranged on a call |

For a price, a pilot (an 8 to 12 week paid trial), a hosting partner agreement or a done-for-you install, write to the same address.

### What to know before you buy

- **Grace.** A key keeps working for 7 days past the period you paid for.
- **Staging.** Staging instances get no key. A key fits one address, and a staging instance runs without the enterprise features.
- **Refunds.** A monthly key is not refunded: cancel any time and the key runs to its date. A yearly key is refunded in full within 7 days of the first purchase, and renewals are not refunded.

## Add a key

1. Open **Settings → Instance**. The **License** section says **No key**.
2. Click **Add license key**, paste the whole key (it starts with `nexul-license-1.`), and click **Save key**.

The section then shows the licensee, the instance address the key is for, what it unlocks and when it expires. Only someone whose role holds the instance's write permission, the instance Owner by default, can add, replace or remove the key. Anyone who can see Settings → Instance sees what it covers.

A key is refused, with the reason under the field, when:

- it was issued for another address. A key covers one instance: the address it names must match the **Instance URL** in Settings → Instance, ignoring a trailing slash and upper or lower case.
- its signature doesn't check out, because the text was changed or it wasn't issued for Nexul.
- it has already expired.

If you change the instance's address after adding a key, the key stops fitting: the section says **Not valid here** and the features stop until you paste a key for the new address. Ask for a key with the new address before you move.

## Expiry and renewal

A key stops working at 00:00 UTC on its expiry date, the date the License section shows. Fourteen days before then, everyone who can change it sees a notice at the top of every page, with the date. Each renewal is a new key; paste it with **Replace key** and the notice goes. Replacing a key changes nothing else, so there is no gap while you do it.

When a key expires, its features stop. Everything they made stays: nothing is deleted, and nobody is locked out of the instance. The notice stays until a new key is pasted, and the features come back as they were.

## Remove a key

Click **Remove key**, then **Remove**. The features stop the same way they do at expiry, and their data stays.

## Branding

A key that includes **Branding** unlocks the instance's own name, logo, accent and color theme: see [Branding](/docs/guide/branding/). Without that feature in the key, the instance keeps Nexul's look.

## Audit log

A key that includes **Audit log** keeps the audit log up to 1,095 days instead of 45 and streams each change, signed, to your own log store: see [Audit log](/docs/guide/audit-log/). The days you saved still hold after the key lapses.

## From an agent

`instance_get` returns the license with the rest of the instance's state. `instance_update` with `license_key` set to a key's text replaces the key, and an empty `license_key` removes it. Both need the same permissions as the web page, and the change shows on every open Settings page at once.
