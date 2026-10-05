# 02: Where "Deploy this repository" shows on a project

Type: prototype
Status: open
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
