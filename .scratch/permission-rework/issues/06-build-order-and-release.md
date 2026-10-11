# 06 — Build order and the release that wipes

**Type:** grilling
**Status:** open
**Blocked by:** 02, 03, 04, 05, 08

## Question

How is the build sliced, given that the wipe happens once, at upgrade? The release that wipes has to carry
opt-in projects, the setup grid and several roles together, or users land in a broken state. What ships
before it (for example the evaluator that names sources), and what after it (folders, stacks, channels,
computers, instance areas)? Which of the original 14 tickets survive, merge or go?
