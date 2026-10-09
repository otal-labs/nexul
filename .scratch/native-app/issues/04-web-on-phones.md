# 04 — Web on phones after the 768px rule

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Once the web app targets 768px and up, what does someone opening it on a phone see: nothing different, a dismissible "get the Android app" banner, or something else? Which lines of `AGENTS.md`, `practices/react-guide.md`, `practices/design-language.md` and the contributing docs change, and what does the ADR say?

## Answer

Agreed with the owner 2026-09-28, and landed with this map update as
ADR 0080 plus the rule changes in `AGENTS.md`,
`practices/react-guide.md` and the contributing coding standards.

- Below 768px the web app shows a dismissible banner, never a block: on
  Android "Nexul is built for tablet and desktop. Get the Android app"
  linking to the latest app release, elsewhere "Built for tablet and
  desktop". Dismissal persists per device through a zustand persist store.
  The banner itself is built with the settings split.
- Existing small-screen classes stay; no clean-up pass.
- `web/` builds at 768px first and verifies at 768, 1024 and 1440px.
  `website/` keeps mobile first at 320, 375, 414 and 768px.
- The owner's global instructions now read "mobile first unless the project
  says otherwise".
