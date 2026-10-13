# 02: Where "Deploy this repository" shows on a project

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

A project with a repository no stack builds from shows a prompt to deploy
it. Where, and how loud?

- Candidates: the Services section in project settings (its empty line
  "No services in this project yet"), the Repositories row of the waiting
  repository, the board header.
- One quiet prompt per waiting repository, or one per project.
- Whether it can be dismissed. Dismissing it would need stored state,
  which the map has ruled out, so it should be quiet enough to stay.

Prototype with `design-mode`; the owner picks by looking.

## Answer

Decided in auto design mode, 2026-10-13, on the real pages of `proto/project-paths` (`?p2=a|b|c`).

- **Where**: the project settings Services card, one row per waiting repository, in a bordered box. It replaces the
  "No services yet. Create one to set up its first deploy." line while a repository waits. The card footer keeps
  "New service". This scored 14/15.
- **Look**:
  - The repository name in mono `text-xs`, which truncates.
  - Under it, "Attached, not deployed yet" in muted `text-xs`, with a hollow 6px muted dot.
  - On the right, an outline `sm` "Deploy this repository" button with a rocket icon. No brand colour, and no banner.
- **Who sees it**: someone who can read stacks and add services, as ticket 03 decided. The tests repository is never
  shown as waiting.
- **Dismissing**: it can't be dismissed. It goes once a stack builds from that repository.
- **Several waiting repositories**: each row's button carries its repository (for example `&repo=owner/name`), so
  the Add-service shortcut takes a named repository as well as the single-repository case.
- **Runner-up**: the Repositories row (12/15, `?p2=b`), with a "not deployed" line and an outline Deploy button.
  The board header (11/15) was rejected: it puts the prompt in front of everyone with stack access every day.
- **Not verified**: the card beside existing services, which needs a real stack.
