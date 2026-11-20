# Fetch the next license key when the owner opts in

**Status:** needs-triage
**Blocked by:** the license portal at cloud.nexul.io is live; ADR 0197 accepted

An online instance whose owner turns on auto-refresh fetches its next key from the license portal shortly before the current key
expires, as ADR 0197 describes. Offline instances and instances with the setting off behave as today.

## Done looks like

- A setting in Settings → Instance → License, off by default, owner-only like adding a key. Its text says exactly what the request
  sends (the key, nothing else).
- About 7 days before expiry the instance asks the portal for the next key for its Instance URL, authenticated by the current key
  (the option ADR 0197 recommends), checks the answer with the same parse and URL check a pasted key goes through, and stores it.
- A cancelled subscription returns no key; the instance keeps its key and it lapses on its date.
- A failed request (offline, portal down, refused) changes nothing and is retried; the 14-day notice shows as it does today and
  nothing locks (ADR 0184).
- A key that does not fit the instance's URL or fails the signature check is discarded, never stored.
- Tests cover success, cancelled, offline, a wrong-URL answer, a bad signature, and the setting being off making no request.
- The portal side (the refresh endpoint in nexul-cloud) has its own ticket there.
- The guide page `enterprise-license.md` gains a short section on the setting.
