# Each machine carries rules, and only the instance Owner holds the instance

Status: accepted.

Amends ADR 0148 (an Owner of any workspace is never denied an instance area; runners and machines as one instance
area) and ADR 0087 (machines checked against every workspace).

A machine is root on a server, yet every machine was one switch. A role bit in any workspace listed every machine,
and a stack's machine was a free string that `stacks:write` alone could set, so anyone who could edit stacks in one
project could deploy to any server on the instance, the instance's own host included. An instance-area rule could
hand out all machines or none. And an Owner of any workspace held every instance area, so creating a workspace made
someone the instance's administrator.

Decision: a machine is an entity with rules, opt-in like a project, and only the instance Owner holds the instance by
right.

- **Machine rules.** A machine's rules name a role (of any workspace) or a person and set See (`machines:see`), Deploy
  to (`machines:deploy`), Manage (`machines:write`) and Delete (`machines:delete`), in ADR 0148's three states and
  order. See is held only through the machine's rules, as `projects:open` is; a machine someone can't See is not
  found. On a machine, a person's roles decide the other three, then the machine's rules for those roles, Allow
  winning, then the rule naming the person.
- **Per machine, not per runner.** Every runner on a machine shares its Docker, so runners follow their machine and
  `runners:*` goes. Adding a machine is the instance-area permission Add machines (`machines:create`); whoever adds
  one gets a rule naming them with every action and no other standing.
- **Deploy to is checked twice:** when anything is aimed at a machine, and on every deploy a person starts, so taking
  someone off a machine stops their next deploy. Deploys the server starts pass as the server.
- **Writers.** The instance Owner and the machine's Manage holders, under ADR 0088's hold-what-you-give.
- **Only the instance Owner.** The Owner of the default workspace holds every instance area and every machine and is
  never denied, except on a personal computer (ADR 0146). An Owner of another workspace is full Owner inside it and
  reaches an instance area or a machine only through a rule.
- **Upgrade.** Every machine starts with only the instance Owner on it; the removed bits leave every role, token scope
  and area rule.

The trade-offs: on upgrade everyone but the instance Owner loses every machine at once, and an install with several
workspace Owners loses their instance powers too, until the instance Owner writes rules. A machine check is one more
statement on every stack write and person-started deploy, and machine lists filter in SQL. A second rule set to learn
beside the project's.

Rejected: machines belonging to workspaces, which can't keep one person in a workspace off one machine; permissions
per runner, which the shared Docker makes meaningless; keeping role bits as "every machine" with rules only denying,
which fails open on a machine nobody wrote a rule for; checking Deploy to only when a stack is created, which lets
someone taken off a machine keep redeploying there; and keeping every workspace Owner undeniable on instance areas,
which makes creating a workspace an instance takeover.
