# SAML is served by crewjam/saml, and an instance holds one provider of either protocol

Amends ADR 0186, which left SAML out. Single sign-on gains SAML 2.0 beside OpenID Connect, for companies whose identity
provider offers nothing else. It is the same enterprise feature, `sso`, behind the same key, permission and Owner-passkey
escape (ADR 0184, ADR 0186).

- **One provider, either protocol.** The instance saves one provider, OIDC or SAML. Saving one while the other is saved
  is refused, and the Owner removes the first. The sign-in page shows one button, SSO-only means the provider, and a second
  protocol would need a rule for which one an email belongs to. Removing a provider changes no account, so the switch
  between protocols is remove and set up; the person's first sign-in through the new one is matched by verified email.
- **Service-provider initiated, signed assertions only.** Nexul makes the AuthnRequest and keeps its id for 10 minutes in
  a table, because the IdP's POST to the assertion consumer service is cross-site and never carries the `Lax` state cookie
  OIDC uses; the id is deleted when a response uses it, which is what refuses a replay. An unsigned assertion is refused
  even over https. IdP-initiated sign-in is out, since it cannot be tied to a request Nexul made. Encrypted assertions are
  out of the first version: https carries the assertion, and encryption is where the library's xmlenc advisories are.
- **Who signs in.** The sign-in identity is the IdP's entity id with a persistent NameID, provider `saml`; a transient
  NameID is refused. A first sign-in links to an existing Google or Discord identity by the email in the attribute the
  Owner names, and otherwise makes an account with the join role, as ADR 0186 does. SAML has no `email_verified`: a signed
  assertion from the provider the Owner configured is taken as the provider vouching for the address, which is weaker than
  OIDC's flag, because an IdP that lets a person edit their own email attribute allows taking over a matched account. The
  settings card says so beside the attribute, and the Owner owns that choice.
- **The library.** `github.com/crewjam/saml`, used through `saml.ServiceProvider` and not its `samlsp` middleware, at
  `v0.6.3` or newer. Verifying an XML signature is cryptography the codebase must not write itself (`practices/go.md`,
  section 15), and this is the one dependency the change adds, with `goxmldsig` beneath it. On 2026-11-14 it was
  BSD-2-Clause, pushed the day before, with 12 advisories published on 2026-11-07 (one critical, CVE-2026-33487, several
  high) all patched by 0.6.1, so `v0.6.3` carries them; it is pre-1.0. Rejected: `russellhaering/gosaml2`, which is
  maintained, narrower and was patched in v0.12.0, but whose history is the same signature dependency with the same
  class of advisories and fewer users to find the next one; writing the XML checks ourselves; and a sidecar that speaks
  SAML and hands Nexul OIDC, which adds a service to run for a self-hosted install. The cost is a pre-1.0 API and a
  library whose advisories come in batches: Dependabot follows it weekly and `make vuln` runs in CI, and a test replays
  the crafted responses the advisories describe so a downgrade fails.
- **Metadata.** By URL (https, refreshed daily and on demand) or upload. Metadata fetched by URL is not
  signature-checked in the first version; the https certificate is its protection. Every certificate in it verifies a
  response, so an IdP rotation overlaps; Nexul's own certificate rotates by publishing two until the Owner retires one.

Decided: pending owner approval.
