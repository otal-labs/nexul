# 13 — Cloudflare is optional in setup and onboarding

**Status:** ready-for-agent

**Blocked by:** 06

Read first: `practices/react-guide.md`, `practices/design-language.md`, the spec (What this unlocks).

## What to build

- Setup wizard (`web/src/components/setup/`, `web/src/models/DNS.tsx:93`): the three ways to reach Nexul stay
  as equals; nothing says or implies agents need Cloudflare.
- Owner wizard (`web/src/pages/OwnerWizardPage.tsx:17`): "Connect your tools" describes Cloudflare as
  optional, for the domain, exposures and DNS; step 4's copy loses the tunnel.
- `TunnelPrerequisiteAlert` is reachable only from an old computer's re-pair (deleted in ticket 14).
- Connectors' Cloudflare description names what it is for now.
- Docs: `setup-wizard.md` step 4 drops "Pairing through a tunnel needs Cloudflare connected with Zero
  Trust"; `paired-computers.md` drops the prerequisite; `topology-and-dns.md` keeps Zero Trust only where
  exposures still need it, if they do.
- The project memory note that set Cloudflare toward required is marked superseded by the owner.

## Acceptance criteria

- [ ] `rg -i 'zero trust'` over `web/src` and the guide finds only DNS and exposure contexts.
- [ ] Screenshots of the changed wizard steps at 768, 1024 and 1440px.
