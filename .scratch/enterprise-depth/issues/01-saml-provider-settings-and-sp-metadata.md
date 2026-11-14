# SAML provider settings and the service provider metadata

**Status:** ready-for-agent
**Blocked by:** ADR 0189 decided (the library and the one-provider rule)

The Owner can save one SAML provider, and Nexul publishes the metadata the IdP registers. Nobody signs in yet.

## Done looks like

- `ee/sso` (or a sibling package the ADR names) holds the SAML settings: metadata URL or uploaded XML, the entity id,
  SSO URL and signing certificates read from it, the email and optional display-name attributes, display name, join role,
  `only`. A migration adds the table and the SP key pair, which the server makes on first save.
- `GET`, `PUT`, `DELETE /api/instance/saml`, `POST /api/instance/saml/verify`, and public `GET /auth/saml/metadata`
  returning the SP metadata XML. Verify joins `readOnlyRoutes`; each route names its permission in the catalog.
- Saving SAML while an OIDC provider is saved, or the reverse, is refused with a message naming what to remove first.
- `instance_update` takes a `saml` object and `instance_get` reads it back (never the private key); the advertised size
  budget is raised for the fields.
- `instance.saml_changed` is declared, written to the outbox in the settings transaction, `make event-schemas` run, with
  an `instanceScope` rule, `make live-topics` and a web follower stub that refreshes the card.
- The license check is `FeatureSSO`; the settings stay when the key lapses and sign-in is off.
- `GET /api/auth/sso` carries the protocol.

## Tests

Error paths first: metadata that is not XML, has no signing certificate or no SSO URL; a URL that is http, resolves
nowhere or times out; a response larger than the cap (metadata fetched by URL is size-limited); a patch that changes
nothing; saving with the key locked; saving SAML over OIDC and OIDC over SAML; SSO-only with no Owner passkey. Then the
happy paths: URL and upload both parse; the SP metadata names the entity id, the ACS URL and the certificate. Parity:
the audit classification test, the permission catalog test and the event contract test pass. The metadata fetch is
tested against `httptest`, never the network.
