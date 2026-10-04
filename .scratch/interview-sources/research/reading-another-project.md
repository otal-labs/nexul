# How a run reads another project and the sources it is given

Research for ticket 02. Every claim cites the code as it stands on
`wayfinder/interview-sources` (base `875dd4ff`).

## Where a run works on the paired computer, and reading a second checkout

**One run, one T3 project.** A play run resolves its target through the
starter's pairing for the run's own project only.
`Runner.launch` calls `r.harness.ResolveTarget(ctx, trail.StarterID,
trail.ProjectID, pick)` (`internal/plays/run.go:306`). Pairing answers from
the starter's project link for that project, then their defaults
(`internal/pairing/usecase.go:697-708`):

> `link, err := s.repo.GetProjectLink(ctx, userID, projectID)` ...
> `return link.ComputerID, link.HarnessProjectID, modelPick{...}, nil`

A project link holds one `HarnessProjectID` per person per Nexul project
(`internal/pairing/model.go:295-306`). CONTEXT.md, Project link: "One
person's choice of paired computer, T3 project, provider, model, model
options, and start-in for one project they can open". The agent pipeline
puts that one id into `harness.Target.ProjectID`
(`internal/agent/pipeline.go:378-386`). `harness.Target` has a single
`ProjectID` and no field for a second project
(`internal/harness/harness.go:155-169`).

**Folder or worktree.** `Target.Worktree` comes from the start-in setting:
the project link's `start_in`, or else the defaults'
(`internal/pairing/usecase.go:678-694`). On protocol 2 a worktree thread is
launched with `WorkspaceStrategy: workspaceStrategy{Type: "worktree",
BaseRef: base}` (`internal/t3clientv2/turn.go:340-350`). Its base is the
branch the T3 project's folder is on, read from the T3 project's `Path`
(`turn.go:359-372`). Protocol 1 cannot launch a worktree, so it starts in
the folder and adds a note (`internal/t3client/harness.go:205-206, 230-231`).
Both protocols create the thread with `RuntimeMode: fullAccess`
(`t3clientv2/turn.go:25, 332, 348`; `t3client/harness.go:223`). Nexul puts
no file-system limit on what the agent can read once it is running.

**What Nexul knows about a second checkout.** It knows only what the
starter's own project link for that project says, plus the computer's T3
project registry. `harness.Project` carries `ID`, `Title` and `Path`
(`internal/harness/harness.go:67-71`), and `pairing.Service.ListProjects`
reads it for the settings picker (`internal/pairing/usecase.go:137-151`).
So for project B on the starter's computer, a path could be looked up from
B's project link (`HarnessProjectID`) matched against `ListProjects`. No
code makes that lookup today. The link may also name a different computer
from the one the run is on.

**Who clones or updates it: nobody.** The `harness.Client` interface
(`internal/harness/harness.go:383-407`) has `ListProjects` but no way to add
a project, clone, or pull. A checkout reaches a computer only because the
person opened that folder in T3 Code themselves. Setup says so: "open any
project in T3 Code on %s first" (`internal/pairing/usecase.go:586-587`). The
only cloning Nexul does is a runner's build: `GitTokens` hands the
connector token to a runner for a build request
(`internal/runner/handler.go:47-48, 387-388`). That happens on a deploy
machine, not a paired computer.

**When it does not exist there.** No code path covers it. A run would only
find out when the agent's file reads failed. The nearest behaviour is
protocol 2's unknown project: "An unknown project is left to
thread.create, which refuses it in T3's own words"
(`t3clientv2/turn.go:364-367`). That covers only the run's own project.

**Nearest seam.** A second checkout would hang off pairing, as a lookup
beside `resolveTargetSource` that turns another Nexul project id into a
path on the run's computer. The result would reach the prompt as a play
block (see the ADR 0111 section below).

## Reading another project's docs, memories, and interview answers over MCP

**Whose identity the run has.** A run's tools are authenticated by the
computer's own personal access token, not by a token minted for the run.
"MCPToken is a computer's own personal access token, the one its providers
connect to Nexul's MCP server with" (`internal/pairing/mcp_token.go:11`).
Setup writes it into the provider's user-level MCP config
(`internal/pairing/setup_prompt.go:47`). `/mcp` is mounted behind
`RequireAuth(withIdentity(...))` (`server/cmd/routes.go:221`).
`withIdentity` sets the actor to the token's user
(`server/cmd/main.go:200-207`). A run only resolves on a computer its
starter owns: `fetchTargetComputer` reads `GetComputer(ctx, userID,
computerID)` (`internal/pairing/usecase.go:814-829`). So the MCP calls act
as the starter. That holds for every MCP call from that computer, run or
not.

**Nothing in the tools pins a run to its own project.** Each tool takes the
project or id as an argument:

- `memory_list` takes any `project_id` (`internal/memories/mcp.go:19-20`).
  `ListForProject` checks only `MemoriesRead` on that project
  (`internal/memories/usecase.go:150-163`).
- `memory_get` takes any memory id. `getChecked` checks the action on that
  memory's project (`usecase.go:111-127`).
- A project's interview answers come back on `memory_get` of its interview
  memory. `memoryOut` adds `Questions` and `ListAnswers(m.ProjectID)`
  (`internal/memories/mcp.go:288-308`). `ListAnswers` needs `MemoriesRead`
  on the project (`internal/memories/answers.go:24-36`).
- `doc_list` takes `project_id` or none ("Omit for every project",
  `internal/docs/mcp.go:40`). `doc_get` checks `DocsRead` per doc
  (`internal/docs/usecase.go:203`).

So the agent can already read project B's docs, memories, and interview
answers with the existing tools, if the starter can.

**Two gaps:**

- **Answers without a memory are unreachable.** Answers are only reachable
  through B's interview memory. If B's memory was deleted (the answers
  survive, per ADR 0115), the only way to get it back is
  `memory_create kind interview`. That needs `MemoriesWrite`:
  `projectForWrite(ctx, projectID, permissions.MemoriesWrite)`
  (`internal/memories/interview.go:108`). Read-only access to B therefore
  cannot reach B's answers once B's memory is gone.
- **The run's named memories are refused outside its project.** A run's
  memory selection rejects anything outside the run's project:
  `"memory %s is not in this project"` (`internal/plays/run.go:574`).
  Another project's memories cannot be named through the run dialog's
  memory list.

**Restricted member.** Every check above goes through
`access.RequireProject` (`internal/access/gate.go:49-60`), which calls
`require`:

> `if !ws.owner && ws.hidden() { return apperrs.ErrNotFound }`
> (`gate.go:33-35`)

ADR 0097: for a Restricted member "a project-area action answers from that
project's access alone and is refused with no project", and "a direct link
reads as not found". If the starter is restricted and holds no Project
access on B, every read of B returns not found. The agent cannot tell that
apart from a missing project; the MCP server instructions already warn "not
found can mean no access". If they hold access on B but not the
memories-read level, the reads are refused as forbidden (`gate.go:43-46`).
A doc's own sharing applies on top (ADR 0097). Project links follow the same
rule: one can only be set for a project the person can open,
`s.projects.RequireProject(ctx, projectID, permissions.Member)`
(`internal/pairing/usecase.go:499`). So a restricted starter cannot have a
link, and so no path, for a project hidden from them.

## How a run is handed things per ADR 0111, and where a source list fits

**Today's shape.** ADR 0111: "a turn's prompt names each piece of context
and the tool that reads it, and carries none of it." A play's prompt has
"the play and its instructions, the ticket or doc line, its link blocks, the
memories to read, and the starter's instructions" (ADR 0111, lines 17-18).
The fixed lines each name a tool: `docLine = "Doc: %q (id %s). Read it with
doc_get before you start."`, `memoriesLine = "Read these memories with
memory_get before you start..."` (`internal/agent/prompt.go:44-52`).

**The block seam.** Free-form lines reach the prompt as
`PlayContext.Blocks` (`internal/agent/prompt.go:148`). The interview
already uses it: `links = append(links, interviewBlock(tgt))`
(`internal/plays/run.go:329`). `interviewBlock` names the project id and
the wizard's tests answer (`run.go:1011-1020`). The Interview play's
instructions start with `memory_create` `kind` `interview` to read
questions and answers, then "read the checkout you are running in"
(`internal/plays/usecase.go:34-36`).

**Resume loses blocks.** When an answer resumes a run whose session the
harness lost, `answerContext` rebuilds only label, instructions, and
memories:

> `pc := &agent.PlayContext{Label: play.Label, Instructions: play.Instructions}`
> (`internal/plays/run.go:587`)

`Blocks` and `Custom` are not rebuilt, and nothing on the trail records
them (`run.go:581-597`, used at `run.go:422`). This matches ADR 0111's list
for a resumed session: "the play, the ticket or doc line, the trail's
recorded memories, and the answer". A source list carried only as a block,
or as the starter's custom instructions, would therefore be missing from a
resumed turn. The interview block is already missing in that case.

**Both options exist.** A source list could be named in the prompt (a
block, recorded on the trail the way `SelectedMemoryIDs` is). It could also
be read through a tool: `memory_get` on the interview memory, which already
returns `questions` and `answers` (`internal/memories/mcp.go:144-150`), so
sources stored per project could come back the same way. The second needs
no resume handling, because the agent re-reads the list. Named sources of
the doc and memory kinds fit ADR 0111 as is (an id plus "read it with
`doc_get`" or `memory_get`). Paths in another checkout have no reading tool,
only a file path the agent opens itself.

## GitHub issues through the connector

**No.** Nothing in Nexul reads GitHub issues today:

- The git seam has no issue method. `GitProvider` offers `GetRepo`,
  `ListPRs`, `GetPR`, `PRsForCommit`, webhooks, installations, `GetTree`,
  and `GetFile` (`internal/gitprovider/provider.go:5-23`). The GitHub client
  implements exactly those (`internal/gitprovider/github/client.go:56-309`).
- No MCP tool reads issues. The git-side tools are `pull_request_list`,
  `pull_request_get` (`internal/gitprovider/mcp.go:39, 75`),
  `repository_list`, and `repository_scan`
  (`internal/repository/mcp.go:31, 64`).
- The webhook passes on an `issue_comment` only when it is on a pull
  request: "maps only a PR comment, not a plain issue"
  (`internal/gitprovider/webhook.go:186, 332-333`). Subscribed events are
  `pull_request`, `pull_request_review`, `pull_request_review_comment`,
  `issue_comment`, and `push` (`github/client.go:107`).
- The documented GitHub App permissions are Metadata, Contents: Read, Pull
  requests and Webhooks: Read and write. Issues is not among them
  (`website/src/content/docs/docs/guide/github-app.md:30-37`;
  `web/src/components/setup/GitHubAppForm.tsx:104-106`). The guide also
  says a permission added later "doesn't apply to installations that
  already exist" until the owner accepts a review request
  (`github-app.md:55`).

**Nearest seam.** The token is already there: `connectors.AccessToken`
returns a live token for the `github` connector
(`internal/connectors/usecase.go:369-397`). `gitProviderRouter.resolve`
builds a provider per repo from it, keyed by the repo's linked connector
(`server/cmd/wire_gitprovider.go:42-79`). An issue read would be a new
`GitProvider` method, plus an MCP tool gated like pull requests through
`RequireRepo(..., permissions.ReposRead)`
(`internal/gitprovider/usecase.go:11-22`), plus the Issues: Read
permission on the App. The connector token never reaches a paired computer.
An agent there can only read issues with credentials the person set up
outside Nexul.

## What this means for the drafting run

- Docs, memories, and interview answers of another project can be read with
  today's tools, as the starter, if the starter can see that project. The
  one exception is answers whose interview memory was deleted, which need
  write access to reach.
- A restricted starter without Project access on the source project gets
  not found on every read. The page could check this when a source is
  added, so the run does not hit it.
- Another project's checkout is only readable if the person already has it
  open in T3 Code on the run's computer. Nexul does not clone it, cannot
  name its path today, and fails late (at the agent's first read) when the
  checkout is missing.
- A source list carried in the prompt must be recorded on the trail, or it
  is lost when an answer resumes a lost session. A list read through the
  interview memory's tool output avoids that.
- GitHub issues as a source kind need new code (a provider method and a
  tool) and a new App permission that existing installations must accept.
