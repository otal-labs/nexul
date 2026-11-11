# 08 — Should Nexul notice a rotated 1Password value and offer a redeploy?

**Status:** open
**Type:** grilling
**Blocked by:** None — can start immediately

## Question

A rotated value reaches a stack only on its next deploy (ticket 04) and an automation only on its next
connection (ticket 05). Connectors re-resolve on a 401/403 (ticket 06). Should Nexul ever notice that a
referenced value changed (the SDK returns item ids, and items carry versions) and tell someone or offer a
redeploy? If so, who checks (only runners and hosts can read their vaults), how often against the
1,000-a-day budget, and what the stack or automation page shows. Or is "redeploy to pick it up" the
whole answer?
