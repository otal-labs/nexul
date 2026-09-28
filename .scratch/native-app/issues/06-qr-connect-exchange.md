# 06 — Connect a phone: the QR exchange

**Type:** grilling
**Status:** resolved
**Blocked by:** 02

## Question

What is the exact QR flow? The code's format and storage, the payload (JSON with host and code, or a `nexul://connect?…` link so the system camera opens the app), the exchange endpoint and its rate limit, what the phone sends to name itself, what the web shows once the phone has connected, and how the phone checks the server version before finishing sign-in (see ticket 09), and how the web renders the QR code (a new web dependency or a server-rendered SVG; ticket 05 needs one).

## Answer

Agreed with the owner 2026-09-28.

- **Connect code.** 12 characters from Crockford base32 (no O/0, I/1
  look-alikes), displayed `XXXX-XXXX-XXXX`, single use, valid 2 minutes,
  stored hashed in a `connect_codes` table (user, hash, created, expires,
  used). One live code per user: issuing a new one invalidates the last.
- **QR payload.** A link, `nexul://connect?host=<instance URL>&code=<code>`,
  so the system camera opens the app and the in-app scanner parses the same
  thing. Manual fallback in the app: instance address plus code.
- **Issuing.** `POST` behind session authentication only; a personal access
  token is refused, so no agent can sign a phone in. No MCP tool.
- **Exchange.** One public endpoint takes `{code, device: {model, os,
  app_version}}` and returns a phone session (`ses_`, 90-day sliding) plus
  the server version. Every failure is one generic "invalid or expired"
  answer. Five wrong codes from one IP in ten minutes returns 429, the same
  limiter shape as setup-code unlocking.
- **Version first.** The phone reads the server version (ticket 09) before
  it spends a code, and refuses without burning it when the server is older.
- **Web feedback.** The new session publishes `session.created`; the Devices
  page, while showing a code, flips to the connected state on it. No polling.
- **QR rendering.** `uqr` in `web/` (MIT, zero dependencies, SVG output),
  approved by the owner.
