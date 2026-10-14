# 06: The guide, and a walkthrough on a real install

Type: task
Status: open
Blocked by: 01, 03, 04, 05

## What to build

- **Docs:** check `CONTEXT.md`, ADR 0150, `runners.md`, `people-and-access.md`, `stacks-and-deploys.md` and
  `topology-and-dns.md` against what shipped.
- **Roadmap:** move the entry from Planned to Shipped in `ROADMAP.md` and `website/src/pages/roadmap.astro`.
- **Walkthrough:** on `nexul-box`, upgrade from the release before ticket 03, with two machines, a stack on each, and
  two people: the instance Owner (alice) and the Owner of a second workspace (bob).
  - after the upgrade bob sees no machine, no instance Settings entry, and can't create a stack on either machine,
    by the picker or by typing the name into `stack_create`
  - alice adds bob's role to one machine with See and Deploy to; bob sees and deploys there and nowhere else
  - alice denies bob Deploy to by name; his next deploy is refused with the rule's sentence
  - a push to the stack's branch still deploys
  - bob, given Add machines, adds a third machine and holds every action on it; alice takes him off

## Acceptance criteria

- [ ] The guide pages build, and the walkthrough's findings are fixed or ticketed.
