# Permission followups

**Status:** needs-triage

## Scope

The permission model in ADR 0148 has shipped. These remaining defects and one
verification gap need separate triage. They do not reopen the shipped effort.
No implementation is approved by this spec. Each ticket is independently scoped.

## Ticket index

| Ticket | Status | Scope |
| --- | --- | --- |
| [01: Per-play Run controls](issues/01-per-play-run-controls.md) | done | A play's own Run allow must reach its button. |
| [02: See and Run explanations](issues/02-see-and-run-explanations.md) | needs-triage | Explain the See prerequisite when Run is allowed. |
| [03: Phone locked mentions](issues/03-phone-locked-mentions.md) | done | Inert doc and ticket references on the phone. |
| [04: Phone acceptance verification](issues/04-phone-acceptance-verification.md) | wontfix | Preserve the explicitly skipped verification gap. |

## Existing efforts

The role-aware UI effort covers action gating, including its tickets
[13](../role-aware-ui/issues/13-sweep-the-rest.md),
[14](../role-aware-ui/issues/14-sweep-the-phone.md) and
[15](../role-aware-ui/issues/15-single-item-answers.md). It does not specify these
exact defects. Coordinate with those tickets and consolidate ownership during
triage if their scope expands. The onboarding effort concerns welcome steps
and template roles, not these permission defects.

## Out of scope

Role-aware action sweeps, onboarding, new permission semantics, and changes to
storage or dependencies. Ticket 02 requires a decision before changing behavior.
Ticket 04 records missing evidence only; it does not require implementation.
