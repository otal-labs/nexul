# 03 — How a runner gets its 1Password token and tells Nexul it can resolve

**Status:** open
**Type:** grilling
**Blocked by:** 01, 02

## Question

How does a runner's token get onto its machine (a `nexul` install or runner flag, the runner's env file,
rotation without reinstalling) on systemd, launchd and the Windows service host? How does the runner tell
Nexul which managers it can resolve, and what do Nexul's stack page and `stack_deploy` do when a stack
has references and the runner that deploys it cannot resolve them?
