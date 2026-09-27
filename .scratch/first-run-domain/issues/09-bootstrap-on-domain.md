# 09 — GitHub bootstrap on the domain

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 03, 04

## What to build

On the domain, `/setup#code=…` unlocks the pass and shows the GitHub App step with the instance URL fixed (server installs no longer type it). Sign-in starts from the domain. The owner wizard drops its DNS step and the `/wizard/onboarding/dns` hop; the DNS page stays reachable from the app for later changes.

## Acceptance criteria

- [ ] Fresh instance, end to end: IP:port → domain → GitHub sign-in → owner wizard, with no "oauth state mismatch"
- [ ] Opening the domain without the code shows the code screen, not an error
- [ ] Owner wizard no longer sends to DNS onboarding

## Read first

`practices/react-guide.md`, `internal/auth/handler.go` (`bootstrap`, `callbackGET`, `spaOrigin`), `web/src/pages/OwnerWizardPage.tsx`, `web/src/pages/DnsOnboardingPage.tsx`.
