# 31 — Harness picker in the run dialog

**What to build:** The run dialog shows which harness the run will use, preselected from the project link or the user's pairing defaults, as one line "computer · provider · model" the user can change per run: computer from their paired list, provider and model from that computer's live list. The choice travels on the run request and the trail records it; the dialog remembers the last choice per user, play, and project like memories and move-to. Amends the spec's "no per-click provider or model override" decision, recorded in an ADR.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] The dialog preselects the resolved computer, provider, and model and shows them before the confirm button
- [ ] Changing any of the three is honoured by the run and recorded on the trail
- [ ] A computer that is offline or unpaired cannot be chosen; the readiness reason shows next to it
- [ ] The last choice is pre-selected next time for that user, play, and project
