# There is no environment object — branch deploy rules on a stack replace it

Every comparable platform models `dev` / `staging` / `prod` as first-class
environments, which then need their own creation flow, their own config
inheritance, and a second mechanism bolted alongside for preview deployments.
A stack instead carries a list of rules mapping a branch pattern — an exact
name like `main`, or a single trailing wildcard like `feature/*` — to a docker
network, an optional hostname template, and an optional name suffix. An exact
rule with no suffix redeploys the base stack in place (the prod case); any
other rule derives a clone, and a wildcard rule derives one per matching
branch and tears it down when the branch is deleted (the preview case). One
mechanism, evaluated on every push, covers both.

Configuring a rule is the whole of the owner's consent: nothing auto-deploys
out of the box, there is no per-branch approval step on top, and the
"ticket finished → deploy" automation ships as a template that is never
enabled by default. A platform that deploys a push nobody asked it to deploy
is a platform people turn off, so the opt-in sits at the one place the owner
already has to visit — the stack's own rules — rather than in a global
setting that is easy to forget is on.

Deploy owns this consumer directly instead of expressing branch rules as
automation rules: a rule is stack configuration, and routing it through the
automations engine would make a core deploy path depend on a user-editable
rule that can be deleted. Automations keeps its own separate ability to
trigger a deploy.

Consequence: there is nowhere to hang per-environment configuration. A branch
deployment copies the base stack's env verbatim, so a preview shares the base's
database until per-branch overrides are built.

Decided: 2026-08-27
