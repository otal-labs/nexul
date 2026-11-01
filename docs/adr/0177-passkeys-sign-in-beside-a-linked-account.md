# Passkeys sign in beside a linked account, never instead of one

Every sign-in went through GitHub, Google or Discord. A passkey signs a person in with their device's own unlock
and never leaves the instance.

Decision: a person adds passkeys from Profile and signs in with one from the sign-in page, verified by
`go-webauthn/webauthn`. A passkey is extra: the rule that a person keeps at least one linked provider account is
unchanged, and passkeys do not count toward it.

- **Why not passkeys alone.** A passkey belongs to the domain it was made for (its RP ID is the instance's host), so
  moving the instance to a new domain makes every passkey useless. With a linked account always kept, a domain move
  never locks anyone out; the old passkeys stay listed as made for another domain until removed.
- **Only https or localhost.** Browsers refuse WebAuthn on plain http elsewhere and an IP address can't be an RP ID,
  so the setup address (`IP:5123`) never offers passkeys.
- **Ceremonies live in memory.** Start hands back a random id; finish takes it once, within five minutes. A restart
  drops ceremonies in flight, and the person tries again. A cap bounds what the public sign-in start can make the
  server hold. Rejected: a table, which buys surviving a restart mid-prompt at the cost of a write per attempt and a
  sweep.
- **User verification required.** A passkey is the whole sign-in, not a second factor, so a security key without a
  PIN doesn't qualify.
- **The browser's own JSON helpers** (`parseCreationOptionsFromJSON`, `toJSON`) do the encoding on the web, so the
  web app adds no dependency; a browser without them shows no passkey controls.
- **Not on the phone app for now.** Android passkeys need a Digital Asset Links file on the instance's domain and a
  native module; the phone signs in by QR code from the web app.
