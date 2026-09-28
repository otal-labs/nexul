# 05 — Your settings and Devices look

**Type:** prototype
**Status:** resolved
**Blocked by:** 02, 03

## Question

How do the footer gear, Your settings and the Devices page look and behave? Prototype the footer (user, gear, where the theme toggle went), the Devices list with its current-device row and sign-out, and the Connect a phone QR panel with its countdown and refresh, through `design-mode`, for the owner to react to.

## Answer

Locked with the owner 2026-09-28 on a live prototype, branch
`proto/your-settings` (mock device data, stubbed bootstrap status; a
reference, never merged).

**Footer.** Avatar and name on the left, a gear icon button on the right
where the theme toggle was; the gear opens Your settings. The account menu
keeps Support and Logout only.

**Your settings** (`/settings`, eyebrow "You", title "Settings"): Profile,
Appearance, Security, T3 pairing. Security has two tabs, Devices and Tokens
(personal access tokens).

**Devices tab**, top to bottom:
- Two cards side by side from 1024px, stacked below:
  - *Connect a phone*: the QR code on a white tile (scanners need dark on
    light in both themes), "Works once, for two minutes", the code as mono
    text, a mono countdown, and a New code button. On expiry the QR fades to
    10% and a New code button appears over it.
  - *Connect the desktop app*: one line of explanation and a "Copy
    connection token" button that mints and copies the token in one click.
- *Signed-in devices* card: the current device boxed alone with a muted mono
  "THIS DEVICE" tag, then "Other devices" as hairline rows (icon tile,
  "Platform · Client" line, mono "IP · relative time" line, ✕ that arms a
  "Sign out" confirm). Footer strip: one explanatory sentence and a
  destructive "Sign out everywhere else" button. No others: an EmptyRow.

**Profile**: a Profile card (picture, display name, picture URL, Save in the
footer) and a Sign-in accounts card listing GitHub, Google, Discord as rows:
linked ones show the account and an unlink icon (hidden for the last one,
which reads "Your only sign-in"), unlinked ones a "Link <provider>" button.

**Motion**, locked:
- Phone connected (the one hero moment): the QR content crossfades to a green
  check tile and "<Phone> is connected / It's in your device list below. The
  code it used no longer works." at 800ms ease-out with a 4px blur and 0.98
  scale; the new row enters the top of Other devices (fade plus 4px rise,
  800ms ease-out) and a background glow on it fades out over 5600ms after an
  800ms hold. The card stays confirmed until the page is left; no "connect
  another" button. This deliberately exceeds the 150–250ms product baseline
  because it happens once per phone; record it as the exception in the
  design-language motion baseline when it is built.
- Rejected: the same without the row glow (the owner wanted the list to
  answer too) and a toast alone (lands far from where the eye is).
- Micro: new QR fades in 150ms; expiry fades opacity 200ms; a signed-out row
  exits with fade plus 4px lift at 150ms; "sign out everywhere else" exits
  all others together, then the EmptyRow fades in at 200ms; copy swaps the
  icon for a green check (fade, scale from 50%, 2px blur, 200ms) and the
  label to "Copied" for two seconds; copy failure shows an error toast.
- Reduced motion: the global block in `index.css` collapses all of it.

**Found while prototyping, to ship with the build:**
- Section navs on settings-style pages (Your settings, Configuration,
  Project settings, Stack) switch to the side column only from 1024px; at
  768px the side column left about 250px for content.
- Configuration's nav drops Appearance, Tokens and T3 pairing.
- A QR renderer is needed: a new dependency (web) or a server-rendered SVG;
  decide in the QR exchange ticket.
