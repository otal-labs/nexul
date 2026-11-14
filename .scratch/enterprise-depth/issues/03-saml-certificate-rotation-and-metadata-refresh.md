# SAML certificate rotation and metadata refresh

**Status:** ready-for-agent
**Blocked by:** 01 (settings), 02 (a signed response to test with)

Neither side's certificate expiry locks anyone out.

## Done looks like

- A metadata URL is refreshed every 24 hours by a background loop and on a **Refresh** action; every certificate in the
  metadata verifies a response, so both are accepted while the IdP publishes both. A failed refresh keeps the last good
  certificates and records the error for the card.
- `POST /api/instance/saml/rotate` makes a new SP pair and publishes both in the SP metadata; `POST
  /api/instance/saml/retire` drops the old one. Both audited, both publish `instance.saml_changed`.
- The view carries each certificate's expiry and a flag within 30 days of it.

## Tests

Refresh that returns an error, an empty body, metadata with no certificate (last good kept in each), a certificate that
changed (new one verifies a response signed with it, old one still does until the metadata drops it), a metadata URL
that changes entity id (refused, since it would be a different provider), rotate then sign with the old key still
accepted by an IdP that cached the old metadata (the AuthnRequest is signed with the new one; the metadata carries both),
retire with nothing to retire. Loop tests use `testing/synctest`, not sleeps, and `goleak` stays green.
