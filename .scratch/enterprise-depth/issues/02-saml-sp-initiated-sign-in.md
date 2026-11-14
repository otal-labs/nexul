# SAML sign-in, started by Nexul

**Status:** needs-triage
**Blocked by:** 01

A person signs in through the SAML provider and lands in an account, in a session marked as single sign-on.

## Done looks like

- `GET /auth/saml` stores an AuthnRequest id (a table, 10 minutes) and redirects to the IdP; `POST /auth/saml/acs`
  validates the response with the library, finds the request by `InResponseTo`, deletes it, and accepts only a signed
  assertion for this issuer and audience, within 60 seconds of skew.
- Account matching is ADR 0186's: identity is entity id plus a persistent NameID (provider `saml`), a first sign-in links by
  the email attribute to a Google or Discord sign-in, otherwise it makes an account in the default workspace with the join
  role, with the same events as an invitation's redemption. A missing email, a transient NameID, a disabled or removed
  account are refused with reason codes the sign-in page already shows.
- The session carries the `sso` mark. SSO-only hides and refuses every other method exactly as with OIDC, including every
  passkey but an instance Owner's; `OnlySSO` reads whichever provider is active.
- The sign-in page's `Public` answer says which protocol, and the button goes to the right route.
- Failures redirect to `/login?error=<code>` and log the real reason, never sending it.

## Tests

Error paths first, each a response built and signed by a test IdP key pair: unsigned assertion, assertion signed by a
certificate not in the metadata, tampered after signing, a signature wrapping attack (a second unsigned assertion beside
the signed one), expired and not-yet-valid conditions, wrong audience, wrong issuer, unknown or reused `InResponseTo`, an
expired pending request, a transient NameID, no email attribute, a disabled account, an account that is the Owner role as
join role. Then: first sign-in creates, second signs in, email links to an existing Google identity, GitHub-only accounts
are not matched, the session carries `sso`, SSO-only refuses another provider's callback with SAML as the provider.
A test pins the library's fixed behaviours the advisories named (multi-reference signature, namespace-blind attribute
values) with a crafted response, so a downgrade fails.
