# 01: What a source is and where it is kept

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

A project's interview gets a list of sources, each with a follow or
question stance. Decide:

- The kinds and how each is named: a path (file or folder) in the
  project's checkout, a doc, a memory, another project in the workspace,
  pasted text, an uploaded file. Whether "another project" is the whole
  project or picks parts of it (checkout, answers, memories).
- GitHub issues of the project's repository: not reachable today (see
  the research for 02). A source kind worth the App's Issues permission,
  or pasted text for now.
- Where the list lives: a new memories-domain table beside the stored
  answers, keyed per project; what deleting a source, its doc, or the
  other project does to the list and to drafts already made from it.
- Pasted text and uploads: an attachment needs an owner (doc, ticket,
  conversation, or memory). Whether pasted text is stored on the source
  row, and what an upload is attached to.
- Permission: adding a source takes `memories:write` on this project, and
  pointing at another project or a doc takes read on it. What a person
  with less access sees in a list someone else built.
- Events, live push, and the MCP surface, extending the interview memory
  tools before adding one.

## Answer

- **Kinds**, each a kind and a ref:
  - `path`: a file or folder in this project's checkout, relative. Absolute
    paths and `..` are refused. The server cannot see the checkout, so a
    missing path shows up when the run reads it, and the run reports it.
  - `doc`: a doc in any project of the workspace.
  - `memory`: a memory in any project of the workspace.
  - `project`: another project as a whole: its checkout (found through the
    starter's project link, per 02), its memories, and its interview
    answers. There is no picking parts of it; to take only part of
    another project, add its docs or memories one by one.
  - `text`: pasted text with a label the person gives it, stored on the
    source row, up to 32,000 characters like the template.
- **Uploads** are not a kind. The page reads a text or markdown file in
  the browser and adds it as a `text` source labelled with the file name.
  Any other file is refused with "paste its text instead". That avoids
  giving a source an attachment owner.
- **GitHub issues** are not a kind for now; paste them. Whether they earn
  the App's Issues permission waits on the walkthrough (fog).
- **Stance** is stored per source, `follow` or `question`, and the server
  requires it. The page suggests the default when adding: follow for a
  doc, a memory, text, and a markdown path or folder; question for a
  project and any other path.
- **Storage**: a new numbered migration adds `interview_sources` in the
  memories domain: id, workspace, project (cascade on project delete),
  kind, ref, label (`text` only), body (`text` only), stance, who added it
  and when, and when it last changed. It is unique on project, kind, and
  ref for everything but `text`, with at most 50 sources per project.
  Deleting the interview memory keeps the sources, like the answers.
- **A ref that goes away** (a doc, memory, or project deleted) leaves the
  source in place: refs are resolved when they are read, the page shows
  "no longer there" with Remove, and the run skips it. Drafts made from it
  stay drafts and confirmed answers are untouched. There is no
  cross-domain cleanup.
- **Permission**: listing takes `memories:read` on the project, and adding,
  changing the stance, or removing takes `memories:write`. Adding a doc,
  memory, or project also takes read on what it points at
  (`docs:read`, `memories:read`, `memories:read` on that project), through
  the access gate and a new doc-to-project lookup port. This stops someone
  who cannot read a doc from getting it drafted into answers they can
  see. Labels for `doc`, `memory`, and `project` are resolved with the
  reader's access when read, so a person without access sees the kind and
  "not visible to you", never the name.
- **Events**: source added, changed, and removed, each an outbox event with
  a catalog row (project, kind, stance, author, never a ref's content or a
  text body), pushed live to the project's audience like the answers.
- **HTTP**: list, add, update stance or label, and remove under the
  project's interview routes.
- **MCP**: no new tool. `memory_get` on the interview memory also returns
  the sources, with `text` bodies included, which is how the run reads
  them and survives a resume (02). `memory_update` on it also takes
  sources to add, remove, or change the stance of, so an agent can add material as a
  peer of the page.
