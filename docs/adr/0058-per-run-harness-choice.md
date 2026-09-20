# 0058. A play run may pick its own computer, provider, and model

A run may pin the computer, provider, and model in the run dialog instead of
always inheriting the project link or the starter's pairing defaults. The
dialog preselects the resolved target, or the starter's own last choice for
that play and project if they have one, as one line the starter can change
before confirming. The trail records whichever computer, provider, and
model actually ran, whether picked or resolved, so a run's harness choice is
never a guess read back from settings that may have since changed.

The runner still resolves the harness before starting the turn, exactly as a
chat mention does, so pairing keeps owning the one place a computer's
ownership, presence, and project mapping are checked: an unowned or offline
computer refuses the run the same way an unconfigured harness always has.

This reverses this effort's own earlier "nothing else is asked per click, no
per-click provider or model override" decision. The reason for the reversal
is the first live runs after shipping plays: cost and fit vary per play (a
quick doc split does not need the same model as a large refactor), and only
the owner using the feature surfaced that. Amends `docs/adr/0055` in spirit,
not in the decision it recorded — a play still runs on the clicking user's
own harness and permissions; only which of their computers, and which
provider and model on it, moved from fixed to a per-run choice.

The trade-off is one more thing in the run dialog. It is hidden behind a
pill in the footer, not surfaced as a form field the starter has to fill in
every time, so the common case — accept the resolved default — stays a
single click.

Decided 2026-09-18.
