# An online instance can opt in to fetching its next license key

Status: proposed.

Amends ADR 0184 for opted-in instances only: the server never contacts anyone, except that an owner can turn on key auto-refresh, and
then the instance asks the license portal at cloud.nexul.io for its next key. Everything else in ADR 0184 holds: the key is still
checked offline against the public key built into the binary, one key covers one instance, and nothing locks when a key lapses.

The setting is off by default. When on, about 7 days before the current key's expiry the instance asks for the next key for the same
URL, checks it exactly as a pasted key is checked, and stores it. A cancelled subscription gets no new key, so the key lapses on its
date. A failed request changes nothing: the instance keeps its key and the 14-day expiry notice shows as it does today, and the
request is retried until expiry. An instance with no internet access, or one whose owner leaves the setting off, keeps pasting keys.

What authenticates the request. Two options:

- **The current key as the credential.** The instance sends the key it holds; the portal checks its signature and that the
  subscription behind it is still paid. Nothing new is stored or issued, and the key already names the one URL it fits. The cost is
  that a leaked key can fetch the next one, but a leaked key is already usable on its own URL, and a key fits only that URL's
  instance, so it adds little. An expired key cannot refresh, which the 7-day lead and the portal's grace of 7 days past the
  paid period cover.
- **A refresh token issued with the key.** The portal issues a secret beside each key and the instance stores it. It can be revoked
  without touching the key, and it survives a key that has already expired. The cost is a second secret to issue, store, rotate and
  paste alongside every key, and an instance with a pasted key and no token cannot refresh until the owner pastes both.

Recommended: the current key. It adds no secret to manage, works for every instance that already holds a key, and a refresh token
guards nothing the key does not already show.

Privacy: the request carries the key, which names the licensee and the instance URL, and nothing else: no users, content, usage or
version. The portal learns that this instance is online and when. It is sent only after the owner turns the setting on, and the
settings page says what it sends.

Why not make it the default: ADR 0184's reason stands, that people self-host to avoid calling home. Why not a long-lived key instead:
it cannot end with a cancelled subscription without a check, and one key valid for years is what the monthly and yearly plans avoid.

To be built after the portal launches: `.scratch/key-auto-refresh/issues/01-fetch-the-next-key-when-opted-in.md`.

Proposed: 2026-11-18.
