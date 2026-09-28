# 14 — Android release and OTA publishing

**Type:** grilling
**Status:** open
**Blocked by:** 08, 09, 10

## Question

How does the app get built and updated? The app's own version and tag scheme next to the server's, the GitHub Actions workflow that builds the `.apk`, where the release signing key lives (it must never change, or installs cannot upgrade), when an OTA bundle is published (every master push, or on request) and to which channel, and which runtime-version policy decides between an OTA bundle and a new `.apk`: the stack research picked `fingerprint`, but Xprem's docs warn against it (see ticket 07).
