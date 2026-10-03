# 02: The default questions

Type: grilling
Status: resolved
Blocked by: 01

## Question

What questions does the code default of the Interview template ask? Today
it is thirteen headings (stack, architecture, errors and logging, testing,
code style, dependencies, security, performance, CI, branching, docs, UI,
vocabulary). Which become questions, in what order, with which options and
hints, and which are better left for the follow-up run to ask only when
the answers or the code call for it.

## Answer

The template asks what only a person knows; facts the code holds (the CI
gates, performance budgets) are left to the follow-up run, which reads them
and asks only when it cannot tell or the code disagrees. Twelve questions,
in this order:

1. What languages and frameworks does this project use? Free text. Hint:
   name versions only where they are pinned on purpose.
2. How is the code organised? One of: layers (routes, logic, storage);
   feature folders; entities and systems; functional core with a thin
   outer shell.
3. How do errors travel? One of: returned and wrapped; thrown and caught at
   the edge; result types. Hint: and what gets logged, at which level.
4. When are tests written? One of: before the code; with the change; only
   for bugs; no tests yet.
5. Which tests does a change need? Several of: unit; integration against
   real dependencies; end-to-end in this repo; end-to-end in a separate
   repo. Hint: and the coverage floor, if any.
6. Which style rules matter most? Several of: early return, no else; small
   functions; comments only for why; strict types, no any. Hint: skip what a
   linter already enforces.
7. When may a change add a dependency? One of: freely; when it saves real
   code; only after asking; only with a written decision.
8. Where do secrets live? One of: environment variables; a secrets manager;
   an encrypted file in the repo. Hint: and what must never be committed or
   logged.
9. How does a change reach the main branch? One of: pull request, squash
   merge; pull request, merge commit; straight to main. Hint: and how commit
   messages are written.
10. Where are decisions written down? One of: decision records in the repo;
    a docs folder; a wiki outside the repo; nowhere yet.
11. Does this project have a user interface? One of: web; mobile; both;
    none. Hint: if yes, the design system, screen sizes, and accessibility
    rules.
12. Which words mean something specific here? Free text. Hint: one per
    line, the term then what it means.
