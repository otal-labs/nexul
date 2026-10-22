# Nexul

An open-source, self-hostable platform that unifies project management,
documentation, CI/CD, and deployment to your own servers — with a first-class
MCP server so LLM agents can drive the whole software development lifecycle.

This file is the project's **ubiquitous language**: the words we use and what
they mean. It is a glossary and nothing else — no specs, no implementation
detail. Decisions live in `docs/adr/`; specs and open work live in `.scratch/`.

## Language

### The product

**The Loop**:
The core product cycle: docs → tickets → code and PRs → deploy → index → repeat.
The thing that makes Nexul one product rather than three bolted together.

**Workspace**:
The top-level content and configuration boundary inside an instance. An
instance hosts multiple workspaces, each an isolation boundary for its
projects, docs, and configuration; members switch between them in the normal
UI via a picker. Sign-in itself stays instance-wide.
_Avoid_: Organization, tenant, team (the Team is the instance's people)

**Workspace slug**:
The workspace's name in every URL of its pages (`/otal/board`): lowercase
letters and digits joined by dashes, unique on the instance, derived from the
name at creation and kept through a rename unless changed on purpose, in
Configuration → General or by `workspace_update`; old links then stop working
and are not redirected. The URL decides which workspace is on screen.
_Avoid_: Workspace key, handle, subdomain

**Archived workspace**:
A workspace taken out of use without deleting anything. It leaves every list, search, inbox and live frame for the
people in it, and its address no longer resolves; its projects, tickets, docs, people and roles stay stored. An Owner
archives it (`workspaces:delete`); the instance Owner restores it (`instance:write`), which brings back all of it. The
instance's own workspace can't be archived. Nothing automatic runs in it: its automations get no events, its auto plays
queue and start nothing, and branch deploys on pushes stop; a restore resumes them without replaying what came
meanwhile. A permanent delete does not exist yet.
_Avoid_: Deleted workspace, disabled workspace

**Project**:
The grouping inside a workspace that tickets, docs, project memories,
repositories, and stacks belong to. Only the project wizard makes one
(`project_create` over MCP runs the same use-case), and it arrives with its
status columns, ticket types, and a starter memory. Nothing seeds one: a new
workspace, the owner's first included, has none until the wizard runs.
_Avoid_: Board, app, default project

**Project setup**:
The project wizard's record of a project: whether Finish was pressed, and each
wizard step done or skipped. A project the wizard makes stays in setup until
Finish, and its sidebar offers Continue setup instead of its pages; one made any
other way starts set up. A signal, never a lock: every page still opens. It also
names the service the wizard made and the environment keys it detected, shown only
to someone who may read the project's stacks. Adding a service from the project's
page is not part of setup and never changes it.
_Avoid_: Onboarding, draft project, unfinished project

**Prefix**:
A project's 2-5 character tag, a letter then letters or digits, set once and
unique within its workspace; it starts the project's ticket keys.
_Avoid_: Project code, short name

**Ticket key**:
A ticket's human name, its project's prefix and its number (`WEB-12`). Unique
only within a workspace, so a key lookup names the workspace, and a key in two
of someone's workspaces is refused rather than guessed. The ticket's id stays
the one reference that needs nothing else.
_Avoid_: Ticket id (that is the UUID), ticket number (only the second half)

**Instance**:
One self-hosted install of Nexul, owned by one person or group.
Single-tenant by design.

**Runner**:
A named service on a machine that executes builds and deploys, connected to
the server over a WebSocket with its own host credential. A machine can run
several; the one `nexul install` puts on the instance's own server is named
`instance`. A runner that is not a personal runner builds and deploys, on a
machine.
_Avoid_: Agent, worker, executor

**Personal runner**:
The runner on a person's own computer, enrolled by that person for that
computer. It reaches the computer's harness for Nexul and reports the
computer's facts; it never builds or deploys. Removing the computer, or
disabling or removing its person's account, revokes it, and it uninstalls
itself.
_Avoid_: agent, relay, daemon, helper, connector

**Foreground supervisor**:
`nexul run computer`: on a Linux host with no systemd, such as a container, the process that runs a computer's personal
runner and T3 Code as its children, in the foreground. It starts either one again 5 seconds after it exits, passes
SIGINT and SIGTERM to both, and removes the install once the runner was removed. It starts nothing at boot.
_Avoid_: daemon, service, init

**Agent**:
The LLM participant in chat, mentioned as `@Agent`. Runs on the mentioning
user's own paired Harness and acts with only that user's permissions. Never a
Runner — a Runner executes builds; the Agent converses and drives the product
through MCP.

**Harness**:
The agent tool running on a user's paired computer that executes an Agent
turn (T3 Code today, others later). The server talks to every harness through
one interface, `harness.Client`, with one implementation per kind. T3 Code is
two kinds, `t3code` for its original orchestrator and `t3code-v2` for
orchestrator V2; a computer moves from the first to the second once, on its
first call after T3 Code updates, and never back. A turn's
prompt names its ticket, doc, and memories for the agent to read through MCP
rather than carrying them; an image embedded in the body of the ticket or doc
a turn is about still travels to the harness as an attachment, capped at
10 MiB per image and 25 MiB per turn, and an oversized or non-image one is
left out. A Note's file reaches the turn as text. Once a ticket is done, or a
doc's clarification finds no gaps left or is closed, and no turn runs on its
thread, the harness session is settled, which moves it out of the harness's
active list until new work wakes it.
_Avoid_: Backend (that is the Go server), Runtime, Driver

**Model options**:
The per-model settings a turn runs with, such as reasoning level, context
window, and fast mode: whatever the harness lists for that model, with its
defaults, read live from the harness and never kept as a list in Nexul. They
are picked beside a model and stored with it (the defaults on Your computers, a project link,
a play's trail, a computer's setup choices); an option left unset runs on the
harness default, and picking another model starts it on its own defaults. A
run's pick also holds on a thread that already exists: on T3 Code's
orchestrator V2 a follow-up switches the thread to the run's model and options
when they differ. A run on the same provider that names no model keeps the
thread's model, one on the same model that names no options keeps its options,
and another model starts on its own defaults.
_Avoid_: Traits, knobs, model settings

**Paired computer**:
A user's own machine running a harness, attached to their Nexul account.
It is reached through its personal runner, which also pairs it, and pairs it
again before its session ends, with no one asking; one paired before personal
runners, through a tunnel or by URL, runs nothing until its owner moves it onto
one (ADR 0146). Owned by one user; nobody
else can see it or run on it, a workspace Owner included, unless its owner shares
it with them by name (Computer sharing).
_Avoid_: Device, host, runner (a runner builds and deploys)

**Computer sharing**:
A paired computer's owner letting named people use it, through a rule on the
computer naming each person (ADR 0148): See this computer, Run agents, Run
commands and Read logs, each of the other three bringing See along. Only the
owner writes these rules, for people who share a workspace with them, never for a role, and none denies. No
role, Owner or permission bit reaches a computer, so to everyone no rule names it
is not found. A rule ends for good when the two share no workspace or either
account is disabled or removed. The person it is shared with sees it as
"<owner>'s <computer> (shared)", by its name and whether it is online only,
and with Read logs also its Docker containers and the logs of the runner and
the containers (Runner log).
_Avoid_: lend, delegate, borrow

**Runner log**:
The personal runner's own log, kept in `runner.log` beside its credential on every
OS (JSON lines, owner-only, 5 MiB and one previous file) and read live through
the runner like container logs, never stored in Nexul. Its owner reads it, and a
person its share gives Read logs; each record is one line, `time level message
key=value…`, with tokens hidden.
_Avoid_: Service log, journal

**Computer facts**:
What a paired computer's personal runner and harness last reported about it:
OS, T3 Code's state, port and version, Docker, provider versions, models and
sign-in, T3 projects, git identity, GitHub sign-in, free disk. A snapshot, stored only when it changes,
and its owner's alone: no workspace Owner and no one the computer is shared
with reads it.
_Avoid_: inventory, telemetry, specs

**Your computers**:
One person's own paired computers in their order, set by dragging in Your
settings → Computers. A run goes to the first ready one: online, set up for
its provider, offering its model, holding a project checkout, and with room
for a queued run; every computer it passed over is named with why (ADR 0165).
Placement is fill in order (the first ready) or spread (the least busy for its
size). A computer shared with the person is never in it.
_Avoid_: Default computer, pool, failover group

**Runs at once**:
How many play runs one paired computer takes at a time, 1 to 5, set by its
owner; pressed or automatic and whoever started them, never `@Agent` chats.
A queued run waits when no ready computer has room; a pressed run never waits
(ADR 0166).
_Avoid_: Slots per person, concurrency limit

**Project checkout**:
The T3 project on one computer that a project's runs use there: the one the
person picked for it, else the one whose git remote matches a repository of
the project. A computer with none is passed over, unless the person's Clone
automatically is on, which clones the repository there first. Outside a
project a computer uses its chat folder instead.
_Avoid_: Checkout (that is a stack's on a machine), fallback T3 project

**Project link**:
One person's provider, model, model options, and start-in for one project
they can open, set in Your settings → Computers → Projects, used on whichever
computer the run lands on. It follows Your computers or names its own list in
its own order (ADR 0165). Each person has their own; nobody else's turns use
it. A play pressed in a project no computer of theirs has asks where to run,
and remembers the answer as that computer's project checkout.
_Avoid_: Project pairing, shared link

**Start-in**:
Where a T3 thread Nexul creates for a person begins: the T3 project's folder,
or a new git worktree off the branch that folder is on, so play runs and
chats running side by side never edit the same files. Set in the defaults on Your
computers; a project link can override it or follow the defaults. A reused
thread stays where it began, and a T3 Code that cannot launch a worktree
starts the thread in the folder and says so.
_Avoid_: Workspace (that is the tenant boundary), env mode

**Setup confirmation**:
An agent's assessment, recorded through MCP and nowhere else, that a paired
computer is ready for agent work: once overall, and once per provider on
that computer. Unconfirmed means the provider cannot run agent work there.
Nothing ever withdraws it except an agent un-confirming it through MCP.
Each provider's confirmation also records the version of Nexul's skills it
found; an older one shows as skills out of date, a signal, never a block. A
skills update, one agent turn that rewrites the skills on the computer, records
the new version for every confirmed provider there and confirms nothing.
A provider its owner switches off in the Set up step counts as unconfirmed while
it is off: pickers leave it out and a run aimed at it is refused, and its
confirmation comes back when it is switched on again.
_Avoid_: Onboarded, verified, setup flag

**Triage skill**:
`nexul-triage`, the second skill Nexul ships beside `nexul-memory`, installed by
computer setup and served by `skill_get`. A person asks their own agent to run
it for a bug in Nexul itself; the agent gathers the instance's facts, diagnoses
against the source at the installed version, and, once the person approves the
draft, files a Triage report on `otal-labs/nexul` through the person's own
GitHub. A report never carries the instance's address or anyone's names. A
problem in a team's own app is a crash ticket on their own board instead.
_Avoid_: Bug reporter, support bot

**Memory**:
Knowledge written for agents, not people: a title, a one-line when-to-use
phrase, and a rich-text body, belonging to exactly one project. It reaches
only that project's turns, by name and never by body: every Agent turn there
names the always-included ones for the agent to read first, a play run adds
the ones the user picked, the agent finds the rest itself, and a plain chat
with no ticket or doc names none. Memories sit in two folders, Main and
Footer; a play run names its footer memories last, for the agent to read
once the work is done and conclude the run, such as deciding which column
the ticket now belongs in.
Agents may write memories too. Every save appends a version with its
author, and any version can be reverted to. Cloned, never shared, to
another project. Plural in the UI: "memories".
_Avoid_: Doc (docs are for clients and requirements), skill, note

**Interview**:
How a project's rules for agents are established: its stack, paradigm,
testing strategy, principles, and vocabulary. A person answers the
workspace's Interview template, a list of questions, on the project's
Interview page, with no agent involved; the template is the instance
template until the workspace edits its own, which takes `memories:template`. Then the Interview play reads
those answers, the codebase, and the question sources, asks follow-ups about
skipped questions, gaps, anything the code contradicts, and what a question
source did that the answers do not settle, a round at a time on the same
page, and writes the interview memory. The page holds the questions, their stored
answers, and the memory. Running it again changes only what the changed
answers change. The answers are stored per project, one per question per
round (round 0 the template's, then one round per batch of follow-ups),
apart from the interview memory: deleting the memory keeps them, deleting the
project removes them. With follow sources, a person can first run the Draft
interview play, which reads them and drafts answers that wait on the page
until confirmed; drafts are kept apart from answers and never count as one.
The project's interview thread is plumbing that holds the run's harness
session, which no page shows, not where the interview happens.
_Avoid_: Onboarding, questionnaire, setup

**Interview source**:
Material a project's interview is pointed at instead of starting blank: a
path in its checkout, a doc, a memory, another project, or text someone
pastes. Each has a stance. Follow is what the team stands behind, and an
agent drafts answers from it; question is how something was done, not how
it should be, such as a predecessor's code, and is only ever asked about,
by the Interview play, which reads follow sources only as context.
The Draft interview play drafts from follow sources only: a template
question with no answer or a skip gets a draft where a source speaks to it,
and an answered one gets a suggested change only where the sources now
disagree with the answer. A draft is not an answer until a person confirms
it. A source is pointed at, never copied in; another project is read
through its memories and answers, plus its checkout when the person has a
project checkout of it on the run's computer.
_Avoid_: Import, context, reference material

**Interview memory**:
The project memory an interview produces, written as rules and kept short,
and named first, for the agent to read before anything else, in every Agent
turn in that project.
_Avoid_: Practices doc, guidelines, rules file

**Audit**:
A doc the Audit via AI play writes from a project's Interview page once the
interview memory exists, measuring code against that memory: a
predecessor's, the project sources under question, or with none the
project's own. Titled "Audit of <what>, <date>" in the project's Main
folder, one line per finding under the memory's headings: a verdict (Carry
over, Rebuild, or Avoid for a predecessor; Keeps or Fix for own code),
where it is, and the rule. The answers about question sources are checked
first, as the known breaks. Each run writes a new one, and To tickets via
AI turns it into work.
_Avoid_: Review, report, scan

**Decisions log**:
A project memory recording only the tickets that changed how the project
works, three lines at most per entry, with reversed decisions marked
superseded so it reads as what is true now. Found among the project's
memories when relevant, never named every turn.
_Avoid_: Changelog, history, release notes

**Decisions check**:
A seeded ticket play, shown on done tickets, with one seeded auto play: a
ticket entering a done stage runs it on whoever moved the card, or the
ticket's developer when an automation moved it. It adds a decisions-log
entry, marks an older one superseded, or leaves the log alone. A check that
cannot start stays on the ticket as "Decisions check didn't run" with a way
to run it again. A workspace switches it on or off on the play's Auto plays
tab; it starts off.
_Avoid_: Closing summary, retrospective, done hook

**Play**:
A pre-configured Agent turn a user fires from a ticket page, a doc page, or
a project's Interview page, or that runs on a stack, with one button ("Fix with AI", "To tickets via
AI", "Run the interview"), or that starts by itself through an auto play.
Defined per workspace with a label (unique in the workspace, ignoring case,
since an automation names a play by it), a type (ticket, doc, interview, or stack),
a one-line description, base instructions, an enabled switch, and an
excluded-projects list; a ticket
play also names the one stage it shows in. No default memories live on the
definition — the run dialog picks those per run, footer memories among
them. Runs on the clicking user's own paired harness and posts into the
target's thread, which for an interview is hidden plumbing and for a stack is the stack's own thread; a play never moves its ticket itself, the agent does, as
its footer memories say. Every workspace,
new or existing, is seeded with the same ten, "Fix with AI" (ticket,
progress stage), "To tickets via AI" (doc), "Interview" (interview),
"Test with AI" (ticket, testing stage), "Draft interview" (interview),
"Clarify via AI" (doc), "Update doc via AI" (doc), "Audit via AI" (interview), "Decisions check"
(ticket, done stage, with its auto play), and "Investigate a crash" (stack,
with its auto play), as ordinary plays
a member
may edit or delete; each keeps a built-in key through renames, and a new
workspace's start from the instance template of their instructions. Seen with `plays:read`, fired with `plays:run`,
managed with `plays:write` and `plays:delete`. Each play may carry Permission
overwrites for a role or a person, its own See (`plays:read`) among them: a
play someone can't see is gone from their lists and menus, and its link is not
found, while its runs stay readable to whoever started them.
_Avoid_: Agent action, button, automation (that is event-driven code)

**Auto play**:
A play starting by itself when a moment matches, set up on the Auto plays tab
of the play's page under Plays in the sidebar; a play can have several. Each names one moment (a ticket
becomes unblocked, enters a stage, is created, gets a developer or tester,
fails a test; a doc is created, or changed once edits have settled and not
by an agent; a stack's service crashes, or recovers), conditions on the
ticket's, doc's or stack's existing fields in all/any groups (a stack's are
its project, the stack and the service by name), a priority, limits, and whom
it runs on: the developer, the tester, whoever caused the moment (a doc has
no developer, so it offers only that), or a chosen computer, which every
play type offers and a stack play offers alone (see **Run on a chosen
computer**), and memories each run reads beside the always-included ones,
only those in its target's project. The play's show-when stage does not apply; a
"Stage is" condition does that. A match waits in that person's queue, highest
priority then oldest first, until a ready computer of theirs has room
(Runs at once), and is checked again before it starts; a waiting run names
each of their computers and why it could not take the run. A ticket past the daily
cap on automatic runs shows "Auto plays paused" until resumed. Managed with
`autoplays:read`, `autoplays:write`, `autoplays:delete`; the person it runs
on needs to see the play and hold `plays:run`. An automation starts one in code with `runPlay`.
_Avoid_: Rule, trigger, workflow, automation (that is event-driven code)

**Run on a chosen computer**:
A choice of whose computers, in order, an auto play's run lands on, offered on every play
type and the only one a stack play has, since a stack has no developer, tester
or causer. The auto play stores the computers and the person who picked them,
who must own each or hold `computers:run_agents` on its share, and
the run acts as that person: the trail is theirs and the run waits in their
queue. The right is checked again when the moment matches and at the front of
the queue; a person who has lost it leaves the run as "didn't run" with the
reason. An auto play that runs on a computer stays off until one is picked.
_Avoid_: Shared computer run, service account

**Investigate a crash**:
A seeded stack play with one seeded auto play, on a service crashing, off
until someone picks the computer it runs on. It reads the stack with
`stack_get` and the crashed service's logs, finds the cause against the
project's repository, and writes the findings on the crash's ticket, or a new
one. It never deploys, restarts or edits the stack, and opens a fix pull
request only if its instructions are edited to say so.
_Avoid_: Auto-fix, self-healing, incident response

**Trail**:
What one press of a play leaves behind: who started it, on which ticket,
doc, or project interview, with which memories and instructions, every step the Agent took,
and how it ended. A step is structured (tool call, tool result, question,
or assistant text, with the tool name, an argument preview, the raw detail,
and its time), never a bare line, so the trail reads as a transcript. The
transcript is the Agent's turn as a conversation: the starter's "Started
<play>" message, one fold per turn (the live step while it runs, then
"Worked for" and what it did) holding what the Agent said between actions,
neighbouring file reads and edits as one line of file names, each command in
a well, and any other tool by its name, the question card and the answer
where they happened, a message the starter typed in T3 Code itself as their
own bubble marked "via T3" (ADR 0126), the final reply as prose, and the lines
about the run itself (a stop, a reconnect) as muted lines.
Persisted, never ephemeral; the "Trail" section on a
ticket, doc, or Interview page lists them, and the target's thread shows the same turn
groups above the Agent's reply, question, or closing line, so the run reads
the same way in the conversation it landed in. States: `starting` at the press, `running` once
the harness accepts, `waiting` while the Agent's question to the starter is
unanswered (the ticket stays put, the silence clock pauses, answering runs
on again in the same trail and session), then one of `done`, `failed`, or
`interrupted`. Stop (the starter or a `plays:write` holder) interrupts it,
from `waiting` too; fifteen minutes of harness silence fails it and stops its turn in T3 Code. A dropped
harness connection is redialed and the turn resumed where it left off; it
fails once that keeps missing. Whenever the thread of an ended run gets new
activity in T3 Code (the run carried on, or the starter continued it there)
while its owner has Nexul open, the trail reopens and catches up
(ADR 0127). An ended trail can be continued: a message sent to its own
harness thread as its next turn, without the play's instructions again, or
the play started again as a new run when that thread is gone (ADR 0128). After a server
restart every `running` trail follows its harness thread again and ends with
the harness's own outcome (ADR 0119); a `starting` one, which no thread
accepted yet, ends `interrupted`, and a `waiting` one stays for its answer.
Also records the computer, T3 project, start-in, provider, and model the run
used, whether the starter picked them in the run dialog or they came from the
starter's own project link and Your computers (ADR 0058, ADR 0165), and the
computers it passed over and why; its answers and continues go there,
whatever the link says later.
_Avoid_: Run history, log, execution

**Doc thread**:
A doc's one conversation, the counterpart of a ticket thread: a real
conversation people reply in, and where a doc play's run lands. Gated by
`docs:thread`, so a reader of the doc need not see the work behind it.

**Clarification**:
A doc's rounds of questions and answers, filling the gaps in what someone
wrote. Each press of "Clarify via AI" is one round: the agent reads the
doc and the earlier rounds and asks a batch of questions, which anyone who
can edit the doc answers, skips, or adds their own question to, whenever
they like. The body keeps its author's words until a round finds no gaps
left and writes the answers into it, or "Update doc via AI" writes them in
without asking more. Whoever runs plays closes it, and
another round reopens it. The people answering see only the questions,
never the agent behind them.
_Avoid_: Wayfinder, questionnaire, review, interview (that is a project's rules for agents)

**Note**:
What an agent leaves on a ticket instead of growing its body: one Agent
message in the ticket's thread together with the markdown file it carries,
which anyone who may edit the ticket opens and edits live, the way a doc is
edited. Message and file are one thing: deleting the message deletes the
file. A ticket's notes also list on its own Notes tab, newest first. The body
stays the ticket's spec and changes only when a person asks.
_Avoid_: Comment, addendum, memory (a memory is written for agents)

**Attachment**:
A file uploaded onto exactly one doc, ticket, conversation, or memory, and
embedded in a body or message by its `/api/attachments/<id>` link: a raster
image renders inline, any other file is a download link. Reading one takes
read on its owner, uploading one takes change; people and agents (through
`attachment_create` and `attachment_get`) share the same 10 MiB cap.
_Avoid_: Upload, asset, blob

**Reaction**:
One person's emoji on a chat message, shown as a chip with the emoji and how
many reacted with it. Anyone who reads the conversation may react; reacting
again with the same emoji takes it back. A reaction is not an edit: it never
marks the message edited and publishes its own event.
_Avoid_: Like, vote, emoji reply

**Locked doc**:
A doc set read-only for everyone as a guard against accidental edits: its
title and body refuse every change, from people and agents alike, until it
is unlocked. Locking and unlocking take `docs:lock`, a permission of its own
apart from editing. Starting a doc play locks its doc, whatever the starter
holds, and the doc stays locked after the run ends, except the lock of a
Clarify or Update doc round, which comes off when its run ends. Archiving, cloning, and
deleting still work, and a clone starts unlocked.
_Avoid_: Frozen, protected, read-only doc

**Watcher**:
A person who gets a doc's change notifications. A doc's creator becomes one by
creating it, and anyone who saves an edit to its title or body by saving it,
through the browser, MCP, or a play acting for them. Anyone who can open the
doc may start or stop watching it, for themselves only. Stopping sticks: their
own later edits do not make them a watcher again until they choose to watch.
An edit tells every watcher but its editor; a mention tells the person
mentioned whether or not they watch, and does not make them a watcher.
_Avoid_: Follower, subscriber

**Folder**:
A group of a project's docs, one level deep: every doc lives in exactly one
folder of its project. Each project has a default folder, Main, where a new
doc lands unless it was started in another folder; it can be renamed but
never deleted, and deleting any other folder moves its docs there, never
deleting one. A doc copied into another project lands in that project's
default folder. Made, renamed, and deleted, and docs moved between them, with
`docs:write`; a reader sees only the folders holding a doc they can open. Its
Permission overwrites reach every doc in it, between the project's and the
doc's own, so moving a doc changes which reach it; they are written with
`permissions:write` on the folder and deleted with it.
_Avoid_: Category (that groups tickets on the board), section (a heading
inside a doc), collection (the retired workspace-wide doc grouping)

**Channel**:
A workspace conversation: a text channel, named with a leading `#`, or a voice
channel. Every member reads a public channel unless a Permission overwrite on
it denies a role they hold or them; a Private channel only its members read.
Posting in one takes `channels:post` from a role, which its Permission
overwrites may deny or allow a role or a person; a reader who may not post
still reads it. Its overwrites are written with `channels:write`. Created and
renamed with `channels:write`, deleted together with every message in it with
`channels:delete`. Each workspace has one `#general`, made with the workspace,
which can be renamed but never deleted.
_Avoid_: Room, group, group chat

**Voice channel**:
Chat's fifth conversation kind — a Discord-style voice room with screen
share and camera, backed by your own LiveKit server. Carries its own text
chat like any channel.

**Direct message** (DM):
A conversation between named people, read by them alone. It starts in a
workspace, between members of it, and shows in that workspace and in every
other one all its people belong to. Starting one takes `chat:write`, except
with only the workspace's Owners, which takes membership alone. Never renamed
or deleted.
_Avoid_: Private message, PM

**Bot**:
A named poster inside one conversation that outside systems drive through
a webhook URL: the URL is the credential and the binding, the payload is
Discord's webhook JSON, and the bot is the message's author with its own
name and avatar. Not a login, not an Integration, and never the Agent.
Gated by `botwebhook:read`, `botwebhook:write`, `botwebhook:delete`;
posting through the URL needs no permission of the sender, and in a channel
posts only while the bot's creator may post there.
_Avoid_: Integration (a separate service with a scoped token), Agent, app

**Occupancy**:
Who's currently in a voice channel, shown live in the channel list without
joining.

**Topology**:
The map of your infrastructure, drawn as a canvas. The canvas JSON *is* the
infra model — not a picture of it. Each workspace has its own: its projects'
services, the gateways routing to them, and what its people drew.
_Avoid_: Diagram, graph, architecture view

**Base service**:
A deploy service definition that owns branch deploy rules. The thing being
deployed; a branch deployment is always a clone *of* a base service, never a
base service itself.

**Branch deploy rule**:
A base service's mapping from a branch pattern (`main`, or a single trailing
wildcard like `feature/*`) to a docker network, optional hostname template,
derived-service name suffix, and optional overrides of the base's settings
for that branch. Evaluated on every push; there is no
separate "environment" concept — `dev` → QA and `main` → prod are just rules.

**Branch deployment**:
The running instance a branch deploy rule maintains for one branch: either
the base service redeployed in place (an exact rule with no name suffix), or
a derived clone service record.

**Preview deployment**:
A branch deployment spawned by a *wildcard* branch deploy rule — one clone
per matching branch, torn down when the branch is deleted.

**Test target**:
Where a tester checks a ticket in a testing-stage column: the preview
deployment of a branch linked to it, else a shared test environment other
work also lands on. Never production, meaning the default branch's own
deployment or a branch deployment on its network with nothing overridden.
_Avoid_: Test URL, staging

**Stack**:
One repository's worth of deployable containers: a compose file, or a
Dockerfile as a stack of one. The stack is what gets deployed, rolled back,
and torn down; the services inside it are observed, not deployed on their
own. Its Permission overwrites set the stacks and deploys actions between its
project's and the stack, reach its deploys and branch deployments, and are
written with `stacks:write` on it and deleted with it; one denying
`stacks:read` hides it, deploys included. Decided 2026-09-08 for the project
wizard.
_Avoid_: Compose project, resource, application

**Instance stack**:
A stack that belongs to the instance rather than to a project: the backing
stack of a gateway, the tunnel or reverse proxy first run deploys among
them. It never builds from a repository, shows on Topology and in
Settings → DNS, and is checked like the topology: holding the stack
action in any of the caller's workspaces is enough.
_Avoid_: System stack, infrastructure project

**Stack slug**:
The DNS- and Docker-safe token derived once from a stack's name at creation,
unique per machine. It is the compose project name, a run stack's container
name, and the checkout directory under the stack root. Never generated or
suffixed: a duplicate is rejected, because every routing mechanism addresses
the container by this name.

**Stack environment**:
The variables a stack's next deploy starts its containers with, a name and a value each, and the overrides on
its branch deploy rules. Saving one redeploys nothing; Deploy now ships it. Every value is treated as secret:
whoever holds `stacks:write` on the stack is sent them, and anyone else gets the names with empty values and
`env_masked` set. An event and an MCP result never carry a value.
_Avoid_: Secrets (no value is marked, all are held back), config, settings

**Observation report**:
The list of containers a runner sends back with a terminal deploy result,
one entry per container the stack started: name, image, status, networks with
addresses, and ports (published ones with their host port, the rest as reachable
on the container's networks only). The only source of a service's observed facts;
the compose file remains the only source of its declared ones.
_Avoid_: Status report, inspect result

**Retry**:
Starting a failed deploy again as a new deploy of the same commit it built,
with the stack's settings as they are now, so fixing a missing environment
variable ships on the same code even when the branch has moved on. A deploy
records its ref (what it was asked to build) and its commit (what the runner
checked out, reported with the terminal result); a deploy that failed before
the checkout retries its ref, and one that recorded neither cannot be retried.
The failed deploy stays in the history and `deploy.retried` names the new one.
_Avoid_: Redeploy (that pulls an image again), re-run

**Tests repository**:
A second repository a project may attach to hold its tests. Never deployed:
no stack builds from it, so a project still has one repository that ships.
Its presence, or the answer that tests live in the deployed repository, is
what the interview starts from on testing.
_Avoid_: E2E repo, QA repo, test project

**Service** (deploy):
One running container Nexul keeps a record of: name, image, networks,
address, status. A stack deploy creates or updates one service per
container it started; an import adopts one per container found running.
_Avoid_: Container (the Docker thing a service is a record of)

**Outage**:
One stretch of a service being down: its container died with a non-zero exit code or was OOM-killed, until it has
stayed running for 60 seconds. A container that keeps crashing and restarting is one outage whose restart count
grows. Opening one publishes `service.container_stopped` and closing one `service.container_recovered`; neither carries
output lines, because event payloads are stored and container output never is. Nothing is opened while a deploy of the
stack runs, for a container that exits 0, or from a personal runner.
_Avoid_: Incident, downtime, crash (one exit is not an outage)

**Error-rate episode**:
One stretch of a stack answering too many requests with 5xx: it opens when the stack's error-rate rule trips (the share of
5xx above its limit over the last few minutes, with at least its floor of requests) and ends once every minute of such a
window has stayed at or under the limit. Defaults: 5% for 5 minutes with 20 requests, off until switched on, one rule per
stack. A window with a minute nobody could count neither opens nor ends one. It keeps the worst window it saw, publishes
`stack.error_rate_crossed` and `stack.error_rate_recovered`, tells the stack's watchers, opens a crash ticket when that switch
is on, and starts stack plays waiting for `stack.error_rate_crossed`. Switching the rule off ends an open one with no
recovered event. At most one is open per stack.
_Avoid_: Incident, alert, outage (that is a service being down)

**Request counts**:
How many requests a stack's hostnames served each minute, and how many answered 5xx, kept for a day and never as
requests. A reverse proxy gateway's Traefik counts them per route and the machine's runner reads them; for a Cloudflare
tunnel gateway the server asks Cloudflare's analytics. Only a stack whose error-rate rule is on is counted. A minute nobody could
count is stored as missing, which is not zero.
_Avoid_: Traffic, metrics, access log

**Crash ticket**:
The ticket a stack opens in its project when a service's outage starts, if the stack's "Open a ticket when a service
crashes" switch is on (off by default; only a stack with a project has it). It takes the project's default ticket type
and first stage, has no developer or tester, and is created by the instance. A service has one while that ticket is
open: a repeat crash or a recovery adds a comment to its thread and never closes it, and once the ticket reaches a done
stage the next crash opens a new one. A stack's error-rate episode has its own ticket the same way, titled by the 5xx share.
_Avoid_: Incident ticket, alert ticket

**Watching a stack**:
Getting a notification when one of its services goes down and another when it
is back. Everyone who can read a stack in a project watches it by default; the
reverse of a doc's Watcher, which is opt-in. Turning Watching off is an opt-out
kept per person and per stack, so a member who joins later starts watching, and
turning it back on removes the opt-out. Losing the right to read the stack ends
the notices, whatever the setting. A stack outside every project has no one to
tell.
_Avoid_: Subscribing, following

**Container logs**:
What a service's container prints on stdout and stderr, each line with its
timestamp and stream, read live from Docker through the machine's runner and
never stored in Nexul: a snapshot of the last lines, or a live tail that
follows new ones. The stack's own env values are masked before a line leaves
the server. Gated by `stacks:logs`. A paired computer with Docker offers the
same for its own containers through its personal runner, to its owner unmasked
and to a person its share gives Read logs with the container's own env values
masked; those containers are not services.
_Avoid_: Deploy log (the build and start output Nexul stores per deploy),
service logs (Nexul's own: Instance logs)

**Instance logs**:
Nexul's own log lines, kept in the instance's log store (OpenObserve): the
server's, the browser's errors and warnings, and those of the runners that
serve a machine and of automations hosts, which they send over their own
connection and the server stamps with where each line came from. A personal
runner's never ship. Read through Nexul, never through the log
store's own login: newest first, filtered by source, level, time range, trace,
name or text, with secret-named attributes dropped and tokens redacted. Gated
by `logs:read`, held by the instance Owner by right and handed out like any
instance area. Never a container's output (Container logs) and never stored
in Nexul's database.
_Avoid_: Service logs, server logs (only one of its sources), audit log
(that records changes)

**Stack root**:
The directory on a machine under which every stack's persistent checkout
lives, at `stacks/<slug>/repo` beneath it. Set by the runner that creates the
machine when it enrolls, then configurable per machine; a stack's location is
fixed once it has deployed.

**Unmanaged stack**:
A stack adopted from what was already running on a machine (a manual
`docker run`, a stack some other tool deployed) with no repository attached. Its
containers can be seen, wired, and exposed, but it cannot be deployed until
a repository is attached, which makes it managed. Deleting it leaves its
containers running: Nexul did not start them (ADR 0158).

**Instance upgrade**:
Moving every Nexul service on the instance's server to the newest release of
its channel, or to a chosen one, with `nexul upgrade`: on the host, or from
the UI and the `instance_upgrade` MCP tool, which ask the `instance` runner to
start it detached from its own service. The server records a UI or MCP
upgrade and resolves the record when it boots on the target version.
_Avoid_: Update (that word is the runner's own binary swap), deploy

**Install directory**:
Where `nexul install` keeps an instance's data: `/data/nexul` on a Linux
server and `~/nexul` on a Mac or Windows PC, holding the generated `.env`,
`data/`, `logs/` and `stacks/`. The binaries and service definitions live
outside it. Uninstall keeps it unless purged, and installing into it again
brings the same instance back.
_Avoid_: Checkout (there is no git checkout of Nexul on a server)

**Machine**:
A server runners and automations hosts run on. Runners belong to a machine;
a service targets a machine and any of its runners may take the job, so the
number of runners per machine is the scaling knob. Replaces "target is the runner" from
2026-09-07 once built. Deleting one takes Delete on it and is refused while a stack or
gateway targets it or one of its runners is connected, naming each (ADR 0153); a free
machine goes with its offline runners and its rules.
_Avoid_: Host, node, server (in the UI)

**Machine rules**:
The rules a machine carries, each naming a role of any workspace or a person and
holding four permissions in Allow, Fallback or Deny: **See**, **Deploy to**,
**Manage** and **Delete**. Machines are opt-in, as projects are: See is held only
through a machine's rules, and a machine someone can't See is not found, and so is
every other action on it. On a machine they See, the other three come from the
union of their roles' bits across their workspaces, then the machine's rules for
those roles, of which Allow wins, then a rule naming them. The instance Owner
holds all four on every machine and is never denied; an Owner of another
workspace holds nothing here by right. The instance Owner and anyone holding
Manage on the machine write its rules, and nobody allows, or lifts a Deny on,
what they don't hold there. **Add machines** is not one of them: it is an
instance-area permission that lets a person bring in a new machine. A machine's
runners follow it: the machine list, the runner list, the queue and the live
frames of its runners leave out a machine someone can't See, and adding a runner
to a machine that exists, discovering what runs on it or removing a runner takes
Manage on it. A runner enrolled under a machine name nobody has makes the
machine, and its adder gets a rule with all four permissions in the same write;
the maker of an enrollment code is checked again when the code is used. An
automations host follows its machine the same way.
_Avoid_: Machine access

**Wizard**:
A guided multi-step flow that ends in something working, reached under
`/wizard/<context>/<step-owner>` (for example `/wizard/onboarding/owner`,
`/wizard/project/service`). Dialogs are not wizards.

**Welcome wizard**:
The wizard where a person sets up what is theirs to set up, such as their
GitHub link and their computer. The steps it shows are the ones ticked on any
Role they hold, in any workspace. It is one wizard per person, so a step done
once counts everywhere. A step counts as done when the thing it sets up exists,
not when someone clicked through it. Getting the phone app is the exception:
it stays done once a phone has connected. A step can be skipped, and the skip
lasts until the person is given that step again, either because a role they
hold gains it or because they are given a role that has it. It opens by
itself only at first sign-in. After that it stays at the top of the account
menu, which shows a dot while a step is neither done nor skipped.
_Avoid_: Onboarding, getting started, first-login wizard

**Template role**:
A Role kept as an Instance template. A new workspace starts with a copy of
each one beside its Owner role, including its one-line description and the
Welcome wizard steps it ticks.
The copy is never rewritten afterwards, so editing a template role changes
only workspaces created after the edit. Five ship with Nexul (Client, Junior
Developer, Senior Developer, DevOps and Administrator). They can be reset but
not deleted. The instance can add its own template roles and delete them.
Show locked actions is on in every shipped template role but Client.
_Avoid_: Default role, preset, starter role

**Locked action**:
An action shown dimmed with the reason it is refused, such as "Your role
can't deploy" or "Denied on stack api for bob", instead of being left out.
Only people holding a Role with **Show locked actions** on, a switch each
role carries that is off for a new role, see it; for everyone else an action
they can't take is not there. A thing they can't read
is not found either way.
_Avoid_: Disabled button, greyed out, read-only mode

**View as**:
A read-only look at Nexul as another member of one workspace, to check that
their roles give them neither too much nor too little. It takes the
permission **View Nexul as another member**, and the member must not be an
Owner and must hold nothing the viewer could not give. Everything shown is
what that member is shown in that workspace: their projects, their chat, their
answer to every action. Changes are off, their personal things (tokens,
sessions, computers, inbox) are not found, and a DM or private channel shows
only if the viewer could read it too. The record of who viewed as whom is the
audit row of the call that starts it.
_Avoid_: Impersonate, log in as, sudo, preview mode

**Automation**:
First-party event-driven code: when an event happens, a function runs.
Written in TS/JS against the SDK — **Default automations** ship with the
instance, **Custom automations** are owner-written. An automation belongs to
one workspace: it hears that workspace's events (and instance-level ones),
reads that workspace's secrets, and is switched and configured there; every
workspace has its own copy of each default. An automation connects out to the
instance and acts back through the API with its own scoped token; Nexul never
calls in to it.
_Avoid_: Rule, workflow (the v1 rule engine is gone)

**Automations host**:
A named service that runs the automations placed on it, each automation on
exactly one host. The one `nexul install` puts on the instance's own server is
named `instance` and is where new automations go; more can be installed on
other machines. A host follows its machine's rules (Machine rules): seeing it
takes See on the machine, adding or removing one takes Manage, and placing an
automation on it takes Deploy to there, beside `automations:write` in the
automation's workspace. A host on a machine Nexul doesn't know yet is the
instance Owner's alone.
_Avoid_: Automations container, runner (a runner builds and deploys)

**Enrollment code**:
A one-time code, valid for an hour, that lets one named runner or automations
host, or a personal runner bound to its person and computer, enroll with the
instance and receive its host credential. A personal runner's code enrolls
nothing else, and no other code enrolls a personal runner. It travels
inside the install command the instance renders and is useless once used.
Until then the person who made it, someone who manages its machine, or the
instance Owner can revoke it; a computer's code is revoked by its owner alone.
_Avoid_: Join token, registration token, runner secret

**Host credential**:
The credential one runner, personal runner or automations host authenticates
with, its own and no other host's, from enrollment until the host is removed. Removal
revokes it, and a host refused with a revoked credential uninstalls itself.
_Avoid_: Runner secret, shared secret, token (a token acts for a person, an
integration, or an automation)

**Connector**:
A third-party tool the instance holds a credential for and calls out to:
GitHub, Cloudflare, LiveKit. One credential per tool per instance, encrypted
at rest. Nexul acting as itself against someone else's API; an
Integration is the opposite direction.
_Avoid_: Integration (the other direction), OAuth app, provider

**Installation**:
One account or organisation Nexul's GitHub App is installed on, granting all
of its repositories or a selection. A person sees the installations their own
GitHub link sees, never anyone else's. Attaching one of its repositories to a
project links the account to that project's workspace, by the account's
GitHub id, which survives a rename; that link is what lets background work
there (clones, webhooks, pull request reads) read the attached repositories as
the App once its private key is set, and detaching it is the way back. Nobody
links an installation by hand. One the App no longer lists is gone:
uninstalled, and its links grant nothing. Without a key, background work on
attached repositories reads as the GitHub connector's user.
_Avoid_: Install (that is the `nexul install` command), connection, grant

**Pending version**:
An automation code version that has been pushed or seeded but is not active.
Merging it activates it and respawns the worker. Shipped upgrades land here
too, so an upgrade never silently changes a default's behaviour.
_Avoid_: Draft, staged

**Self-read**:
An automation token reading its own automation record. Permitted regardless
of scopes while the automation is switched on; that is how the host loads its
state and active code.
Reading yourself is identity, not a grant.

**Integration**:
An external service that listens to signed events and calls the scoped API. A
separate service, never an in-process plugin. Third-party by positioning:
first-party event-driven code is an **Automation**.

**Bot**:
A named poster bound to one conversation and driven from outside through a
webhook URL that accepts Discord's execute-webhook JSON. The URL is the
credential; there is no login and no user behind it. A bot's messages carry
author kind `bot`. Not an **Integration**, which listens to signed events and
calls the scoped API, and not the **Agent**.
_Avoid_: Webhook (alone; integrations already send signed webhooks), app, integration

**Trust tier**:
How much an integration is vouched for — `verified` or `community`.

**Dial-in**:
An automation's outbound connection to the instance: it connects out,
announces the subscriptions its code declares, and receives events down that
connection. The opposite of the signed-webhook path integrations use.

**Stage**:
One of the five fixed, ordered buckets every status column declares:
`backlog`, `progress`, `review`, `testing`, `done`. Column names are the
owner's; stages are the product's, so rules read the stage and never the
name. Only `done` is terminal.
_Avoid_: Kind, phase, state

**Finished** (ticket):
A ticket with at least one linked PR, none still open, and at least one
merged. PRs closed unmerged are ignored. Publishes `ticket.finished`.
_Avoid_: Done (a status column's name), closed, completed

**Developer / Tester / Reporter**:
The three people on a ticket. The developer builds it and the tester checks
it, one person each and both optional. The reporter filed it, is set once,
and shows as Nexul on behalf of a person when an agent or automation filed
it.
_Avoid_: Assignee, owner, creator

**Found in**:
A bug's link to the ticket it was found in, carried by every bug unless its
reporter marked the origin unknown. A done ticket is never reopened; a bug
found after done is a new ticket found in it. A bug is a ticket whose type is
named `bug`; a type renamed away from it is an ordinary type.
_Avoid_: Regression of, caused by, parent

**Source** (ticket):
The doc a ticket was derived from, at most one and optional. Set when the
ticket is filed from a doc, and changed or cleared later from the ticket.
Only a doc can be a source.
_Avoid_: Origin (that is Found in), parent doc, spec link

**Blocked by**:
A ticket's link to another ticket that must be finished or reach done first,
in any project of the same workspace. It shows on the card and warns before a
play runs, but never stops a card moving. A ticket not yet done becomes
unblocked when its last open blocker finishes, enters a done stage, loses its
link, or is deleted. A finished blocker stays cleared wherever it moves; an
unfinished one leaving done and entering it again unblocks it again.
_Avoid_: Depends on, dependency, blocker stage

**Body template**:
The markdown sections a ticket type pre-fills into a new ticket's body, edited
on the type in project settings. A new project's task, bug, and feature types
start from the instance template of the same name. Guidance only: never
validated, and editing it never rewrites a ticket already born from it.
_Avoid_: Ticket template, form, checklist

**Instance template**:
The instance's own version of a template that every workspace or project
starts from: the Interview template, the mention chip template, each built-in
play's instructions, each default ticket type's body template, and each
Template role. The Intro
and Footer every full Agent prompt opens and closes with are instance
templates too, but live only at the instance, with nothing below. A template
resolves code default, then instance template, then workspace or project; one
nobody edited at the instance is the code default. A workspace's Interview and
mention chip templates follow the instance's until the workspace edits its
own, while play instructions and body templates are copied in when the
workspace or project is created and never rewritten. Resetting a workspace's
or project's template gives it the instance's; resetting the instance's gives
the code default. Any template can be cloned over another of the same kind,
plays matched by their built-in key and ticket types by name. Read by every
member, edited with `templates:write`.
_Avoid_: Global template, default template (that is the code's), master copy

### Identity and access

**Permission**:
One capability, written `<domain>:<action>` where the action is `read`,
`write`, or `delete` (`docs:write`, `members:delete`), or a verb the domain
declares for an act that is neither (`plays:run`, `memories:clone`,
`roles:clone`, `docs:thread`, `docs:clone`, `docs:lock`, `stacks:logs`, `projects:create`, `projects:access`,
`memories:template`). One vocabulary for every actor: a role, a
scoped token, and the agent are checked against the same values. Checked in
the workspace the entity belongs to (a workspace's topology canvas in that workspace); DNS,
connectors, the instance's own stacks, and the instance itself (its settings,
upgrades, accounts, workspace creation), which belong to none, against every
workspace the caller is in, and then the rules on the Instance area it acts on, and a machine against the same roles and then the rules it carries (Machine rules). A member holds a set of roles, or none, and holds what any of them allows. The Owner of
any workspace therefore holds every permission outside the instance areas, and only the instance Owner holds those by right; nobody grants a permission they don't hold. What
every member reads (public channels, their own DMs and inbox, People) takes membership, not a permission. Each domain
is a project area, a workspace area, or an instance area; a role's project areas count only inside the projects the
person is In, and Create projects (`projects:create`) and Edit the workspace Interview template (`memories:template`)
are workspace permissions. Something a person cannot read is
not found for them, never forbidden (ADR 0148).
_Avoid_: Right, privilege, capability, ACL entry, instance admin (holding the
instance's permissions is what that meant)

**Permission overwrite**:
The rule on one entity (a project, doc folder, doc, stack, play, channel,
computer, or instance area) for a role or a person, holding each permission as
Allow, Fallback, or Deny. Fallback says nothing, so the answer comes from
further out. The nearer entity beats the farther one; on one entity a rule
naming the person beats the role rules, and between role rules Allow wins. Every
answer names its source, the rule that decided it, or the roles. The workspace
Owner bypasses all of it, except on a personal computer. There is no rule for
everyone and none on the workspace itself.
_Avoid_: Grant, share, ACL, override (Team's per-person override is gone),
Everyone rule

**Source**:
What decided a permission answer: the Owner role, membership, the person's
roles, or the Permission overwrite that last set Allow or Deny on the way in.
Every check carries it as ids; explaining someone's access and a forbidden
answer turn it into a sentence, such as "Denied on doc Pricing for bob" or
"Not allowed: role Client doesn't grant it". A not-found answer carries none,
so it never confirms a rule exists.
_Avoid_: Reason, cause, origin

**In this project**:
Being in a project: what the project's Permission overwrites allow, for a role
the person holds or for them by name. A project someone is not In, and
everything in it, is not found for them, its name included. Nothing else lets
anyone in, no role holds it, and the Owner is In every project. Written as
`projects:open` on a project's rules, by whoever holds Manage this project's
access (`projects:access`) there. A new project has its creator and the
creator's roles In it, and the creator manages its access. Listed on the
project's "People in this project"; its row on the project's panel reads "See
this project".
_Avoid_: Project member, project access, Every project, restricted, private
project

**Kept out**:
A person whose rule on a project denies In this project, so they are not In it
whatever roles they hold. Listed under its own heading on "People in this
project".
_Avoid_: Banned, blocked, excluded, Restricted member

**Access page**:
A workspace's page of who is In each project and which roles see each public
channel, as tick grids of projects and channels by roles. An Owner's first
visit after the upgrade that rebuilt permissions opens it as the one-time setup
"Who's in each project"; a workspace with no projects at that upgrade, and
every one made later, starts set up. Shown to anyone who may manage a project's
access or edit roles; only an Owner sees the channels grid. Its Roles tab is
where a workspace's roles are defined, and a role editor who manages no project
sees that tab alone.
_Avoid_: Permissions page, project access, setup wizard (that is the instance's)

**Private channel**:
A Channel only its members see and read, rather than every member of the
workspace; to anyone else it reads as not found. Switched either way with
`channels:write`; a member adds people, `channels:write` removes them, and
anyone but the last member may leave. The workspace Owner sees every one.
Its events, a DM's too, reach no Integration or Automation.
`#general` is never private.
_Avoid_: Group, locked channel, hidden channel

**Invitation**:
A single-use bearer link that admits one person to the instance and gives
them at least one Role in each of one or more Workspaces, and nothing per
person; which projects they are In follows from those roles. The
link expires after one or seven days and is not bound to a provider identity.
Revoking it, or its roles or its maker's right to give them going away, ends it
early; an accepted or ended link stays on record.
It is only for someone with no account yet. An existing account that opens it
is turned away, and the link stays unused for the person it was meant for.
Adding an existing person to a workspace happens on Team.
_Avoid_: Allowlist entry, invite code, join link

**Account status**:
Whether a registered User may authenticate to the Instance: active, disabled,
or removed. Workspace membership and Roles remain separate.
_Avoid_: Allowlist status, membership status

**Team**:
Everyone registered on the instance and what each can reach: every account
with its Account status, whether it is online or when it was last seen, and,
per workspace, the Roles it holds, or none. A holder of
`accounts:read` sees all of it, under Instance settings; someone who manages
members in a workspace finds it in Configuration and sees only the workspaces
they manage. A change
inside a workspace always needs `members:write` there, and the Owner role is
never given or taken through it. Nobody gives a role holding more than they
hold, or takes away one they couldn't give. Deleting a role takes it from
everyone holding it, who may be left with none.
_Avoid_: Members (one workspace's roster), registered accounts, users

**Instance settings**:
The group on the Settings page for what belongs to the whole instance rather
than one workspace: Instance, Team, Sign-in providers, Connectors, DNS, and
Templates.
Each entry shows only to a viewer holding its permission in any workspace, so
a role holding one bit reaches that entry and nothing else. Workspace-level
sections stay in Configuration.
_Avoid_: Whole instance, admin settings

**Instance area**:
One part of the instance a Permission overwrite may sit on, outside every
workspace: the instance itself (its URL, upgrades, failed events), Sign-in
providers, each Connector, DNS and gateways, Instance templates, accounts,
adding machines, the topology registry, integrations, the audit log, and the
instance's logs.
Each names the permissions its rules may set. A person holds
one of them through a role in any workspace they are in, then the area's rules
for their roles, of which Allow wins, then a rule naming them. Only the instance
Owner, the Owner of the default workspace, holds every area and is never denied
one; an Owner of another workspace holds one only like anyone else. Only the
instance Owner writes these rules, so one area is handed out without a
custom role: "bob manages DNS but not connectors". They set them from the Access
card on each instance Settings entry.
_Avoid_: Admin area, instance permission (that is the bit, not the place)

**Display name**:
What a person is called wherever they appear: the name they set in their
profile, else their sign-in account's name, else their login. The login
stays where a handle is meant: a chat @mention, a Mention written as
markdown, and the line under a name. A person's Mention chip shows the
display name.
_Avoid_: Username, nickname, name (alone, that is the sign-in account's)

**People**:
A workspace's members as any member sees them: login, display name, and
picture, and nothing else. Readable by every member of that workspace,
whatever their role; managing members is the Team's.
_Avoid_: Directory, roster, members (the managed list with roles)

**Mention**:
An @-reference in a doc, ticket, or memory body to a ticket, a doc, or a
person, stored as a node holding the target's id and shown as a live chip.
A person's holds their user id and its chip shows their picture and
display name from People; as markdown it is `[@login](/people/<user id>)`.
A link to a ticket or doc page in the same workspace becomes a Mention
when it is pasted or loaded (ADR 0120); one made from a ticket URL holds
the ticket's key. Saving a doc or ticket that newly mentions someone tells
them in their inbox. A chip of a ticket or doc its reader can't open still
shows it, locked: a doc's title, a ticket's title, status and developer,
even from a project they are not In. Chat's `@login` and `@Agent` are plain
text in the message instead.
_Avoid_: Tag, ping

**Personal access token**:
A long-lived, revocable credential (`dep_`) carrying exactly one user's own
permissions. What an agent authenticates its MCP connection with. A paired
computer has at most one of its own, "Nexul MCP on <computer>", revoked when
its setup is un-confirmed or it is removed; a saved transcript shows any
personal access token as `[redacted token]`.

**Sign-in identity**:
One provider account attached to a user: GitHub, Google or Discord, keyed by
the provider's own id, with the login it yields. A user holds one per
provider at most and signs in through any of them; the first one created is
the one their login, name and picture follow. Linked from Profile while
signed in, unlinked from there too, never down to none.
_Avoid_: Account (that is the user), login method, connected account (a
Connector is the instance's credential, not a person's)

**GitHub link**:
A person's own GitHub App user token, kept encrypted from their GitHub sign-in
or from Connect GitHub (which links the GitHub sign-in to their existing
account), and refreshed as it lapses. Every repository list, search and scan
reads with it, so it shows exactly what that person can open where the App is
installed. Without one, or after GitHub refuses a refresh, nothing lists until
they connect again; Disconnect forgets it.
_Avoid_: GitHub connector (the instance's credential), personal access token
(a Nexul token)

**Session**:
One signed-in device (`ses_`): a browser, the desktop app, or a phone. Stored
as a row with its platform and label, listed on the user's Devices, signed
out one at a time or everywhere else at once, and expiring 30 days after
last use (90 for a phone). A user's own; no permission bit and no MCP tool,
so an agent can never sign a person out.
_Avoid_: Login token, JWT, cookie, refresh token

**Device**:
What a session belongs to, as the sign-in saw it: the client kind (browser,
desktop, phone), the platform ("Linux", "Android"), and a label (the browser
name, "Nexul desktop", or the phone's model). Shown as "platform · label".
_Avoid_: Paired computer (that runs a harness), machine, host

**Connect code**:
The code a phone trades for its session: twelve Crockford base32 characters
shown as `XXXX-XXXX-XXXX` and inside the QR link `nexul://connect?host=…&code=…`,
valid two minutes, single use, one live code per user with a newer one
replacing the last. Issued only from a signed-in device, never by a personal
access token, so no agent can sign a phone in. Only its hash is stored.
_Avoid_: Pairing code (that pairs a computer), QR token, login code

**Push token**:
The Expo token a phone registers on its own session so the instance can
reach it. One push per inbox notification per phone, titled "Nexul" with an
id-only payload the app fetches the content for; posted from a consumer of
the notification event, never from the request that made it. Signing the
session out drops the token with the row, and a device Expo reports as no
longer registered loses it.
_Avoid_: Device token, FCM token (that is Expo's concern), notification key

**Scoped token**:
The credential an integration (`int_`) or an automation (`dat_`) acts with,
minted with a chosen subset of permissions and revocable on its own. A scoped
token granted `X:write` also receives `X:read` at mint.

**Audit log**:
The instance's record of changes: one row per change made over the HTTP
gateway or by an MCP tool that is not read-only, naming the user who made
it, kept 45 days. Reads are not recorded. Read with `audit:read`.
_Avoid_: Audit (that is the doc a play writes), activity log, access log

**Setup code**:
The code (`nxs_`) that proves someone at the instance's own server is the one
setting it up. The server writes a fresh one on every start until the first
user exists, valid a day, and the installer prints it. Unlocking does not use
it up, so the owner can carry it from the server's address to the domain.
Gone once the first user exists.
_Avoid_: Enrollment code (that enrolls a host), invite code, admin password

**Setup pass**:
What a correct setup code unlocks: a bearer (`nxsp_`) good for an hour that
lets one browser set up the domain and the GitHub App before any user exists.
It reaches only the first-run routes and acts as the identity `setup`. Every
pass stops working the moment the first user exists.
_Avoid_: Session (that belongs to a user), setup token, bootstrap key

**Connection token**:
A signed JWT carrying server information only (instance URL, MCP endpoint,
basic settings) so a standalone client can be pointed at an instance. No
identity, no credentials, therefore not secret.

### The domains

A **domain** is a bounded context under `internal/<domain>/`. It owns its model,
storage, use-cases, events, and adapters, and never imports another domain's
internals.

**Access** — what an actor may do. Permissions and access control. A
permission is `<domain>:<read|write|delete>` (`docs:write`,
`members:delete`), or a verb a domain declares beside those three
(`plays:run`, `memories:clone`, `roles:clone`, `docs:thread`, `docs:clone`, `docs:lock`, `stacks:logs`,
`projects:create`, `projects:access`, `memories:template`), one vocabulary shared by roles, token scopes, and the
agent; `projects:open`
is held only through a project's rules. Distinct from auth.

**Auth** — who a user is. Identity and device sessions, via an owner-configured OAuth provider.
Distinct from access.

**Automations** — event-driven code: Default and Custom automations run
functions when events happen, acting through the scoped API, on the
automations hosts they are placed on.

**Bot webhooks** (`botwebhook`) — bots: their URL and token, the execute
route, limits, lifecycle, and the media proxy their messages' images load
through. Posts reach chat through one use-case chat exposes; chat never
learns what a webhook is.

**Chat** — conversations inside a workspace: channels, direct messages
(shown in every workspace their people share), threads, and ticket threads,
with the mentionable Agent and its memories.

**Code review** — a mirror of the provider's review state, one record per PR.
Nexul does not host review threads.

**Deploy** — stacks, the containers they run, and the act of shipping them to
a machine through a runner.

**DNS** — DNS records and domain management. Cloudflare is the first provider
and is special: it gives both records and tunnels. A **gateway** is a
Nexul-deployed service (tunnel or reverse proxy) giving one docker
network internet reachability via hostnames — the canonical term for what's
elsewhere been called an "exit node" or "entry path"; an **exposure** routes
one hostname through a gateway to a container.

**Docs** — documentation as the source of truth, grouped into each project's
folders, with `@` cross-references to tickets and other docs.

**Git provider** — the abstraction over GitHub and future providers: branches,
PRs, webhooks.

**Integrations** — the platform for external services: scoped tokens, signed
webhooks, the store.

**MCP** — the Model Context Protocol adapter, tools shaped per task over the
use-case layer, so LLM agents can drive the product.

**Memories** — the Memory entity: its table, page, permission verbs, and MCP
tools. Never shares a list, a page, or search with Docs.

**Runner** — runner enrollment and removal, the WebSocket protocol, queueing
and dispatch.

**Search** — FTS5 full-text search over docs and tickets. Not a package: each
domain registers its own index and query.

**Tickets** — work items on a Kanban board, linked to docs, branches, and PRs.

**Topology** — the canvas model: nodes, edges, live status.

**Voice** — voice channel LiveKit interaction: join tokens and live
occupancy, over a bring-your-own LiveKit server.

**Workspace** — the workspace boundary itself, its members, its settings, and
notifications. Notifications are a capability here rather than a domain of
their own: every domain publishes events, and the workspace turns the ones a
member cares about into an inbox. A read notification is deleted 90 days after it was
read, and any notification 180 days after it was sent.

### How we talk about the code

**Use-case layer**:
The single source of business behaviour. Both adapters call it; neither
duplicates it.

**Adapter**:
A thin entry point over the use-case layer. There are two: the HTTP/JSON
gateway for the browser, and the MCP server for LLMs.
_Avoid_: Controller, handler (a handler is one function *inside* an adapter)

**Seam**:
A boundary designed for future splitting. The EventBus is *the* seam — the one
place a domain could later become its own process.

**Event catalog**:
The list of every event topic with the payload it carries, assembled from each
domain's own `Topics()` by `internal/eventcatalog`, which generates each
topic's published schema from its payload type. Additive-only.

**Outbox**:
Events written in the same transaction as the domain change, then relayed to
the bus — so an event can never be lost or published for a change that rolled
back.

**Access memo**:
The answers access checks have already read, kept for one request (or, in
the live hub, across frames) and dropped at the next commit, so a revoked
grant takes effect on the very next check.
_Avoid_: Permission cache, access cache

**List window**:
The page a list call reads: an offset and a limit (50 by default, 100 at
most), read in SQL after access filters, beside an honest total, the count
of exactly what all the pages hold.
_Avoid_: Cursor, page token, scan cap

**Composition root**:
A `cmd/main.go`. The only place concrete implementations are wired to
interfaces.

**MCP**:
Model Context Protocol — the standard LLM agents use to call tools. The
"MCP-first" thesis: the MCP server is a first-class adapter, not an afterthought.

### How we talk about the frontend

**Design language**:
The visual spec of record for the web app, the public site and the phone app:
raised surfaces over a still light field, frosted panels in the web app, dark
first, one ember accent (`brand`) for action and selection, focus in ink,
status in its own hues. Tokens in `web/src/index.css`,
`website/src/styles/tokens.css` and `native/src/global.css`, spec in
`practices/design-language.md` (ADR 0133).
_Avoid_: the Mono Console (the monochrome direction it replaced).

**Page → Feed → Section → Card**:
The required component hierarchy. A Page composes Feeds; a Feed lists entities;
a Section is a titled group; a Card/Row/Item is one entity.

**Defensive ordering**:
Negative checks first — loading, error, empty — each in its own `&&` block,
before the happy path.

**Live follower**:
A web domain's answer to the live frames it follows, beside its queries: per
topic, it patches the views holding the entity a frame carries or refetches
only what the frame's ids name. Always "live follower", never a bare
"follower": a person who gets a doc's notifications is a Watcher.
_Avoid_: Frame handler, topic map, live table

**Query declaration**:
A phone query defined once with the topics that refresh it and how a frame
finds its entry; the phone's live dispatch and stale times are derived from
the declarations.
_Avoid_: Topic table, live table

**Client core**:
The modules the web and phone apps both run, kept once in `client-core/`:
rules over the wire shapes, never a component, a hook, or a platform API.
_Avoid_: Shared package, common, SDK (that is what automations are written
against)

**The Frontend Commandments**:
F1–F7 in the [coding standards](https://nexul.io/docs/contributing/coding-standards/#react-the-frontend-commandments). Hard rules for anything in `web/`, not
suggestions.

**Ticker**:
A form that hands credentials or config to a third party verifies them first
as a list of named checks. Each row names one thing the provider must
accept, spins while its request runs, then ticks or crosses with the
provider's own reason, so a failure says exactly which permission or value is
wrong. Continue/Confirm/Set up only unlocks once every row is green.
_Avoid_: Checklist, health check, validation list
