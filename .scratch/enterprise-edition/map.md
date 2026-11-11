# Enterprise edition

## Destination

A way to sell Nexul, built: a license key the instance checks, and the first enterprise features behind
it (SSO and branding), so a host can buy one key per client instance. The map is done when those build
tickets have merged and nothing is left to decide for them.

## Notes

- This map carries execution: once a decision ticket resolves, its build ticket in Nexul (label `ee`)
  is worked from the same map. The build tickets are NEXUL-120 (license key), NEXUL-121 (SSO, blocked by
  120) and NEXUL-122 (branding, blocked by 120 and ticket 01 here).
- Read first: `CONTEXT.md` (Instance, Enterprise feature, License key), ADR 0184, `ee/LICENSE`,
  `practices/architecture.md`, `practices/go.md`, `practices/mcp.md`; for the web,
  `practices/react-guide.md` and `practices/design-language.md`.
- Prototype tickets call the `prototype` skill and design-mode; grilling tickets call `grilling` and
  `domain-modeling`.
- Settled while charting (2026-11-11), recorded in ADR 0184:
  - Only new capabilities become enterprise features. Nothing that has shipped free moves behind the key.
  - One Enterprise plan: a key unlocks every enterprise feature at one price per instance, though the key
    lists its features so one could be sold alone later.
  - One key per instance, bound to its public URL. A partner buys one per client instance. No user count.
  - The key is an offline signed file, checked against a public key in the binary, with no phone-home.
    Expiry stops the features and keeps the data; nothing locks.
  - Keys are signed by hand for now; the private key lives in the owner's password vault.
  - SSO starts with OIDC: one provider per instance, a verified email matches an account, and an
    SSO-only switch that never locks the instance Owner out. GitHub, Google, Discord and passkeys stay free.
  - Branding: product name and logo everywhere the instance shows or sends them, the accent colour (the
    `brand` token's value only), and an instance-wide imported VS Code theme kept readable. The about
    page keeps "Runs on Nexul".
  - Priority support runs on Nexul itself: a workspace per client on the owner's instance, its `#general` as
    the support chat (ticket 02).

## Decisions so far

- NEXUL-120 (license key): `license.Service.Enabled(ctx, feature)` in `ee/license` is the one check 121 and 122
  call. Instance settings they add are fields on the `instance_update` MCP tool and read back on `instance_get`
  (ADR 0185); the web settings page lives in `web/src/ee/`.
- [Can a support workspace give each client only their own conversation](issues/02-a-support-space-on-the-instance.md):
  yes with no code, as one workspace per client (a shared one shows every client's people); setup is NEXUL-123.
- NEXUL-121 (single sign-on, ADR 0186): `ee/sso`, one OIDC provider; the sign-in routes are `/auth/oidc` and its
  callback, the settings `/api/instance/sso`, and `instance_update` (now in `internal/mcp/composite`) takes an `sso`
  object. SSO-only is refused on the server for every method but an instance Owner's passkey, and turns on only while one
  holds a passkey. A branding ticket reads `license.Service.Enabled(ctx, license.FeatureBranding)` the same way.
- Ticket 01 (License and Branding pages): one Branding card of hairline rows under License, an Owner's imported
  theme paints its own mode and an accent alone repaints only the default palette; see
  [the answer](issues/01-branding-and-license-pages.md).

## Not yet specified

- Selling keys through a shop that issues the key after payment, once there is a second buyer.
- A console where a partner sees and upgrades every instance they host (the README's "Many instances in
  one place"), once one partner runs several.
- Audit log: the API already returns 45 days of changes for free. What would be paid is a longer
  retention setting and streaming the log out; sharpen it when a client asks.
- SAML, when a client's identity provider speaks only SAML.
- Cloud hosting: a control plane the owner runs for customers. Coarse until the first two partners show
  what they would hand over.

## Out of scope

- A branded phone app: its name and icon are fixed at build, so it would be a build per client.
- Per-person imported themes: a free core feature, its own effort, not a sales feature.
- A user count in the key, and a partner key covering several instances.
