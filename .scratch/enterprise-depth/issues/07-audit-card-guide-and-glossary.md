# The Audit log card, its guide page and glossary terms

**Status:** ready-for-agent
**Blocked by:** 05, 06

## Done looks like

- An **Audit log** card in Settings, Instance: days kept, the destination form (URL, secret shown once), Test, the
  pending and dead counts, the last delivery, Retry failed, Rotate secret, Remove. It follows the design language and the
  Frontend Commandments, and refreshes on `instance.audit_settings_changed` and on window focus.
- A new guide page `audit-log.md` (what is recorded, what is not, the body and signature with a four-line verification
  example, retry and dedupe on `id`, that a lapsed key keeps the saved days), the sentence in `api-and-tokens.md` updated,
  and the page linked from the license page's feature list.
- `CONTEXT.md`: Audit log and the new Audit stream term, with an `_Avoid_` line.

## Tests

The card's states first: key locked, not configured, destination failing with dead rows, a delivery pending, the secret
shown once and not again. Then Test, save, retry and remove, and the follower. A web test beside the others, no snapshot.
Verified by hand at 768, 1024 and 1440px for the app and 320, 375, 414 and 768px for the guide page, screenshots in the PR.
