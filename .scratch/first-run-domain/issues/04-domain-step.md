# 04 — Domain step and the handoff to the domain

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 03

## What to build

After unlocking on `IP:port`, the setup page shows the domain step: pick tunnel, reverse proxy, or "I already have HTTPS" (paths land in 05, 06 and 08). No skip. The shared finish is `PUT /api/setup/instance-url {url}` (setup pass only, no user may exist), which runs the existing `VerifyInstanceURL` check and stores the URL. Then: "Nexul is live at https://<domain>" with a Continue link to `https://<domain>/setup#code=<code>`.

## Acceptance criteria

- [ ] The URL is only stored after the domain answers Nexul over HTTPS
- [ ] The handoff link opens the GitHub step on the domain with the code prefilled
- [ ] Reloading IP:port after the URL is stored goes straight to the handoff screen
- [ ] 320/375/414/768px

## Read first

`practices/react-guide.md`, `practices/design-language.md`, `practices/go.md`, `web/src/Router.tsx` (`AppRouter`), `web/src/pages/InstanceBootstrapPage.tsx`, `web/src/components/dns/DnsSetupStepper.tsx`.
