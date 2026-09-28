# 33 — APK release and OTA update workflows

**Type:** implementation
**Status:** done
**Blocked by:** 24
**Decided in:** ticket 14

## What to build

Two hand-run workflows. `native-release.yml` (workflow_dispatch): bun
install in `native/`, `expo prebuild --platform android`, decode the keystore
from `ANDROID_KEYSTORE_BASE64` with `ANDROID_KEYSTORE_PASSWORD`,
`ANDROID_KEY_ALIAS` and `ANDROID_KEY_PASSWORD`, `./gradlew assembleRelease`,
and create a GitHub release `android-v<app.json version>` with the `.apk`
attached (fail if the tag exists). `native-update.yml` (workflow_dispatch):
`npx eoas@<pinned> publish --branch production --platform android
--nonInteractive` with `EOO_TOKEN`, against the update server URL in app
config. Document both, the secrets they need, and the rule that native
changes bump the version, in `practices/native.md` and the website's CI and
releases page.

## Acceptance criteria

- [ ] `native-release.yml` builds a signed release APK in a dry run on a fork or with a test keystore
- [ ] `native-update.yml` fails clearly when `EOO_TOKEN` is missing
- [ ] Neither runs on push or schedule

## Surfaces

- CI: two workflows
- Docs: `practices/native.md`, website CI and releases page

## Read first

`practices/native.md`, the website page `contributing/ci-and-releases`, `.github/workflows/release.yml` for house style, tickets 07 and 14.

## Verification

`actionlint` on both workflows if available; a local `./gradlew assembleRelease` with a throwaway keystore.

## Files likely touched

- .github/workflows/native-release.yml, native-update.yml (new)
- `native/app.config.ts`
- `practices/native.md`, `website/src/content/docs/docs/contributing/`

**Size:** M
