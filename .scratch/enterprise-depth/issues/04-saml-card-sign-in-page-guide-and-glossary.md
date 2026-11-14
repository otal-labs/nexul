# SAML in the Settings card, the sign-in page, the guide and the glossary

**Status:** ready-for-agent
**Blocked by:** 01, 02 (03 for the rotation controls)

## Done looks like

- The Single sign-on card asks OpenID Connect or SAML at setup, shows the SAML form (metadata URL or upload, attributes,
  name, join role, SSO-only), the SP values with copy buttons and the metadata link, the certificate rows with Rotate and
  Retire, and the one line about the email attribute. It follows the design language (`practices/design-language.md`,
  the web app section) and the Frontend Commandments.
- The sign-in page shows the one button for either protocol, and the SSO-only state unchanged.
- `single-sign-on.md` gains a SAML section (registering Nexul at the IdP, what to send, matching, rotation, what is not
  supported: IdP-initiated sign-in, encrypted assertions, signed-metadata checking, several providers, provisioning).
- `CONTEXT.md`: Single sign-on, its `_Avoid_` line, and the new Service provider metadata term.
- The MCP guide page lists the `saml` object, and the SAML line in ADR 0186 points to ADR 0189.

## Tests

The card's states first: key locked, not configured, metadata fetch failing, a certificate within 30 days of expiry, the
other protocol already saved (the refusal's message shows). Then the form's verify, save and remove, the follower refresh
on `instance.saml_changed`, and the sign-in page with each protocol and with SSO-only. Verified by hand at 768, 1024 and
1440px for the app and 320, 375, 414 and 768px for the guide page, with screenshots in the PR.
