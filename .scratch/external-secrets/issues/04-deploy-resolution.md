# 04 — When a runner resolves references, and what a failed resolution does to a deploy

**Status:** open
**Type:** grilling
**Blocked by:** 01, 02, 07

## Question

When does the runner resolve: every deploy with no cache, or cached for a time? If a reference cannot be
resolved (missing item, no access, 1Password down), does the deploy fail before build, naming the
variable but never a value, and how does that show in the deploy history and the trail? Resolved values
land in the compose `.env` (mode 0600) and in `-e` arguments today (`internal/runner/executor.go`,
`writeComposeEnv`): is that acceptable for referenced values, or do they need another route into the
containers?
