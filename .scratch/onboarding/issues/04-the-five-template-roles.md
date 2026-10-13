# 04: The five Template roles: their permissions and their steps

Type: grilling
Status: resolved
Blocked by: 01, 03

## Question

What does each Template role grant, and which steps does it tick? Client, Junior
Developer, Senior Developer, DevOps, Administrator.

Charting proposed these ticks:
- Client: Profile, Your projects
- Junior Developer: Profile, Your projects, Your computer
- Senior Developer: Profile, Connect GitHub, Your computer
- DevOps: Profile, Add a deploy server
- Administrator: Profile, Connect GitHub
- Get the phone app: on for nobody by default, or on for everyone?

The permissions come after the rework settles its vocabulary. Prod's hand-built
roles are the starting point (`.scratch/permission-rework/research/what-the-rework-touches.md`).
Also decide how the instance edits Template roles in Settings → Templates, and
whether a fresh install's first workspace gets them (it should).

## Answer

Owner's calls, 2026-10-13, against the permission list on the `permission-switch` branch. Projects
are opt-in (ADR 0148), so a template role reaches no project until a project lists it.

**Permissions** (R read, W write, D delete):

| Area | Client | Junior Developer | Senior Developer | DevOps | Administrator |
|---|---|---|---|---|---|
| In projects: tickets | R W | R W | R W D | R W | all |
| docs | R W, thread | R W, thread | R W D, thread, clone, lock | R W | all |
| attachments | R W | R W | R W D | R W | all |
| memories and interview | R | R | R W D, template | R | all |
| pull requests, code reviews | — | R | R | R | all |
| repositories | — | R | R W | R W | all |
| stacks and deploys | — | R, logs | R W, logs | R W D, logs | all |
| projects and board settings | — | — | R W, create | R | all, including access |
| Workspace: chat, channels, voice, mentions, notifications | yes | yes | yes | yes | all |
| plays | — | R, run | R W D, run | R | all |
| auto plays | — | — | R W | — | all |
| members and invites, roles, workspaces | — | — | — | — | all |
| Instance: runners, machines, DNS and gateways, topology | — | — | R | R W D | R W D |
| accounts, audit, instance settings, templates, connectors, integrations, automations, events | — | — | — | — | — |

Administrator holds every project and workspace permission and the deploy infrastructure, but
none of the instance's own administration. That stays with the Owner role (the owner may later
add a broader template role for it).

**Welcome steps**: as charting proposed.
- Client: Your projects
- Junior Developer: Your projects, Your computer
- Senior Developer: Connect GitHub, Your computer
- DevOps: Add a deploy server
- Administrator: Connect GitHub

"Get the phone app" is ticked on none; an admin who wants it ticks it.

**Editing**: Settings → Templates gets a Roles card listing the template roles. Each one opens
the same role editor as a workspace role (Instance, Workspace, In projects, Welcome).
- The five that ship have **Reset to default** and can't be deleted, because their code
  default would come back.
- **The instance can add its own template roles** and delete them. A new workspace copies every
  template role, shipped or added, beside Owner, and never rewrites the copy.

**First workspace**: a fresh install's first workspace gets the template roles like every
workspace after it.
