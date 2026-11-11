# Enterprise features are new, and unlocked by an offline key per instance

Only capabilities that have never shipped free become enterprise features; nothing in the AGPL core
ever moves behind the key. A license key is a signed file naming the licensee, the instance's public
URL, the features it unlocks and an expiry. The server checks it against a public key built into the
binary and never contacts anyone. One key covers one instance and every enterprise feature, at one price
per instance, with no user count. When it expires the features stop and their data stays; nothing locks.

Why: taking back a free feature costs the trust of everyone self-hosting, and the AGPL copies already out
keep it anyway. People self-host to avoid calling home, so the check is offline, and binding the key to
the URL is what makes one key per instance mean something without a phone-home. Considered: a partner key
for many instances, which cannot count them offline; a key per feature, which reads as a wall on each
one; a user cap, which locks a team out where a notice would do.

Decided: 2026-11-11. Map: `.scratch/enterprise-edition/map.md`.
