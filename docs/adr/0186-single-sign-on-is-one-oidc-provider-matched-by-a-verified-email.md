# Single sign-on is one OIDC provider, matched by a verified email

Single sign-on is an enterprise feature (ADR 0184): the instance Owner sets one OpenID Connect provider for the whole
instance, with its issuer URL, client id, client secret (a secret reference is allowed) and the name its button shows.
Without a key that unlocks `sso` the settings say so, the button is off, and the saved settings stay for a renewed key.
SAML, provisioning people from the provider and several providers at once are out.

- **The provider is checked, not trusted.** The issuer's discovery document gives the endpoints; the authorization code
  flow runs with PKCE and a nonce, both derived from the state under the server's secret so only the state cookie crosses the
  redirect. The ID token's signature, issuer, audience, expiry and nonce are checked by `github.com/coreos/go-oidc/v3`,
  the one dependency this adds: verifying a signed token against a key set is cryptography the codebase must not write
  itself (`practices/go.md`, section 15).
- **Who signs in.** The provider's subject, scoped by its issuer, is the sign-in identity (provider `oidc`). A first
  sign-in whose `email_verified` is true and whose email is the login of an existing Google or Discord sign-in
  identity, which are verified emails already, links to that account. Otherwise it makes an account, which joins the
  default workspace with the role the Owner chose, as an invitation's redemption would, with the same events. A
  provider that sends no verified email is refused. A GitHub-only account has no email on record, so it is never matched.
- **SSO-only.** The switch hides every other method on the sign-in page and refuses them on the server too: the GitHub,
  Google and Discord callbacks, and passkeys other than an instance Owner's. It can be turned on only while an instance
  Owner holds a passkey, so a provider that breaks never locks the instance out. Invitations keep the provider the person
  picks to create their account; their later sign-ins go through the provider, matched by email.
- **Settings over MCP.** `instance_update` gains an `sso` object and `instance_get` reads it back (ADR 0185). The tool
  moves from `ee/license` to `internal/mcp/composite`, which exists for a tool that composes two domains, so the license
  package does not import SSO. It reaches the provider, so it is no longer marked local. The advertised size budget
  rises by 1,000 characters for the new fields.

Why not a standing invitation for new accounts: an invitation is single-use and names its grants; a setting that names
one role is what an instance with an identity provider means by "everyone in the company".

Decided: 2026-11-11.
