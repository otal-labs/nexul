---
title: Enterprise license
description: Paste the license key that unlocks the enterprise features on one instance, see what it covers, and renew it before it runs out.
sidebar:
  order: 3
---

Nexul's enterprise features, single sign-on and branding, run on an instance that holds a valid **license key**. Everything else in Nexul is free and stays free: a key only ever unlocks features that were built as enterprise features, never one that shipped without a key.

A key is a short block of text naming who it is for, the address of the one instance it covers, the features it unlocks and the date it expires. It is signed, and the server checks the signature against a public key built into Nexul. Nothing is sent anywhere to check it, so a key works on an instance with no internet access.

## Add a key

1. Open **Settings → Instance**. The **License** section says **No key**.
2. Click **Add license key**, paste the whole key (it starts with `nexul-license-1.`), and click **Save key**.

The section then shows the licensee, the instance address the key is for, what it unlocks and when it expires. Only someone whose role holds the instance's write permission, the instance Owner by default, can add, replace or remove the key. Anyone who can see Settings → Instance sees what it covers.

A key is refused, with the reason under the field, when:

- it was issued for another address. A key covers one instance: the address it names must match the **Instance URL** in Settings → Instance, ignoring a trailing slash and upper or lower case.
- its signature doesn't check out, because the text was changed or it wasn't issued for Nexul.
- it has already expired.

If you change the instance's address after adding a key, the key stops fitting: the section says **Not valid here** and the features stop until you paste a key for the new address.

## Expiry and renewal

Fourteen days before the key expires, everyone who can change it sees a notice at the top of every page, with the date. Paste the renewed key with **Replace key** and the notice goes.

When a key expires, its features stop. Everything they made stays: nothing is deleted, and nobody is locked out of the instance. The notice stays until a new key is pasted, and the features come back as they were.

## Remove a key

Click **Remove key**, then **Remove**. The features stop the same way they do at expiry, and their data stays.

## From an agent

`instance_get` returns the license with the rest of the instance's state. `instance_update` with `license_key` set to a key's text replaces the key, and an empty `license_key` removes it. Both need the same permissions as the web page, and the change shows on every open Settings page at once.
