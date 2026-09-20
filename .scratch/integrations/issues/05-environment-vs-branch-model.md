# 05 — Environment vs branch: the deployment-target model

**Type:** grilling
**Status:** resolved
**Blocked by:** 01

## Question

the owner (charting round 1): "merge into 'dev' → deploy on QA, merge into
'main' → deploy on Prod … not sure if this should be env based or not,
technically it's branch." Plus feature previews: merge into `feature/x` →
deployed at `feature-x.example.com`.

`deployDomains.md` DP1 deliberately deferred environments ("the target
server is the environment; staging is a separate service if needed") —
this ticket reopens that on purpose.

Model the concept:

- Is there a first-class **environment**, or is the model purely a
  **branch → deployment mapping** on a service (branch pattern → network +
  hostname pattern + target)? What does Discord-style simplicity look like
  here vs Vercel-style preview machinery?
- Fixed branches (`dev`, `main`) and wildcard branches (`feature/*`) — one
  mechanism or two?
- Where the mapping lives: on the service definition? the project? the
  network (01's model may make the network the natural home)?
- Sharpen the vocabulary and record it in `CONTEXT.md` (environment,
  preview deployment, branch deployment — pick canonical terms).

Close by updating `deployDomains.md` (DP1's deferral paragraph) with the
decided model.

## Answer

Resolved by owner delegation (2026-08-27), flagged for review. Follows
the owner's own instinct ("technically it's branch").

- **No first-class environment object.** The model is **branch deploy
  rules** on a service definition: a list of
  `{branch pattern, docker network, hostname template, name suffix}`.
  "Environment" never enters the vocabulary; `dev`→QA and `main`→prod are
  just rules whose pattern is an exact branch name.
- **One mechanism for exact and wildcard patterns.** An exact rule
  (`main`, `dev`) maintains one stable instance; a wildcard rule
  (`feature/*`) spawns one instance per matching branch. Same rule shape,
  same matcher.
- **Empty name suffix = deploy the base service itself in place** (the
  `main` → prod case). Non-empty (default: the branch slug) = a derived
  clone (ticket 06's lifecycle).
- **Rules live on the service definition** (deploy domain), not the
  project or network — the service is what gets deployed, and DP1 already
  hangs network/build-source config there. The network in a rule must have
  a gateway (ticket 01) for the hostname template to mean anything;
  a rule may omit the hostname (deploy without exposing).
- **Vocabulary for `CONTEXT.md`:** *branch deploy rule*, *branch
  deployment* (an instance a rule maintains for a branch), *preview
  deployment* (a branch deployment from a wildcard rule), *base service*.
