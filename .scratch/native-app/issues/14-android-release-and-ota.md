# 14 — Android release and OTA publishing

**Type:** grilling
**Status:** resolved
**Blocked by:** 08, 09, 10

## Question

How does the app get built and updated? The app's own version and tag scheme next to the server's, the GitHub Actions workflow that builds the `.apk`, where the release signing key lives (it must never change, or installs cannot upgrade), when an OTA bundle is published (every master push, or on request) and to which channel, and which runtime-version policy decides between an OTA bundle and a new `.apk`: the stack research picked `fingerprint`, but Xprem's docs warn against it (see ticket 07).

## Answer

Decided 2026-09-28.

- **Versions:** the app has its own semantic version in `native/app.json`, and
  its releases are tagged `android-v<version>`, separate from the server's
  `v…` tags.
- **Runtime version policy: `appVersion`**, not `fingerprint`, because the
  update server warns against fingerprint. Any change to native dependencies
  or config bumps the app version and ships a new `.apk`. OTA carries only
  JavaScript and assets within one app version.
- **`.apk` workflow:** `native-release.yml`, run by hand only (the owner's rule
  is no releases unless asked). It prebuilds Android, runs `assembleRelease`
  with the signing key from repository secrets, and attaches the `.apk` to a
  GitHub release tagged `android-v<version>`.
- **OTA workflow:** `native-update.yml`, also by hand. It runs `eoas publish
  --branch production --platform android` against the update server with the
  `EOO_TOKEN` secret.
- **Signing key:** generated once with `keytool`. Its base64 and passwords
  become repository secrets, and the owner keeps an offline copy. If the key
  is lost, installed copies can never upgrade.
