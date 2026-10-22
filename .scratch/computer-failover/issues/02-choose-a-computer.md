# 02 — Choose a computer: pressed plays and chats

**Status:** ready-for-agent
**Blocked by:** 01

Spec: `../spec.md`, Choosing a computer and Callers. ADR 0165.

One function in `internal/pairing` returns the computer and project checkout for a person, project, provider and model,
or a refusal naming every computer passed over and why, in the spec's check order, with fill in order and spread.
`ResolveTarget`, `ResolvePersonRun` and the `@Agent` pipeline go through it; `needs_location` fires only when no
computer has the project, and a pick is remembered as a checkout. A trail records `passed_over`. The run dialog's
**Where to run** shows where it lands and **Change** picks for one run.

Done when: table tests cover each pass-over reason, both placements, the ambiguous match, a pressed run with every
computer full, a chat outside a project on the chat folder; and a pressed play in an unlinked project with a matching
checkout runs without asking.
