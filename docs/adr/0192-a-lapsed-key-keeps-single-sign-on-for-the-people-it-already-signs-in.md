# A lapsed key keeps single sign-on for the people it already signs in

Amends ADR 0186, which turned the sign-in button off with the key. Single sign-on makes accounts whose only way in is
the provider, so a key that expired, was removed or stopped fitting the address locked those people out, and ADR 0184
says nothing locks when a key lapses. Without a key that unlocks `sso`, a saved provider now still signs in whoever it
is already linked to, and the hourly re-check keeps ending the sessions of people removed at the provider, since they
can still sign in. Everything else stops as before: no account is made or linked by email (that sign-in is refused with
`sso_lapsed`), the SSO-only switch stops hiding the other methods, and the settings cannot be changed, only removed.

Why not a grace period instead: it only moves the lockout a few days later.

Decided: 2026-11-15.
