# 03: How attaching without deploying and the Add-service shortcut work

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

Answer from the code, with file and line evidence:

1. Where the wizard attaches the repository to the project today: does
   `POST /api/projects/{id}/repos` happen on its own, or is the
   repository attached as a side effect of creating the stack with a
   `build_source`? If the repository is attached first ("Attach without
   deploying") and a stack is created later, does stack creation succeed
   or fail with "already belongs to a project"?
2. Whether "Attach without deploying" should still scan. The scan catches
   "GitHub App not installed on this repository" early, which an attach
   without a scan would only find at deploy time.
3. How the Repository step's tests-location question behaves on an early
   exit: is the answer still saved, and does "No repository yet" make the
   question meaningless?
4. How "undeployed repository" is computed: project repositories with role
   `app` that no stack in the project builds from. Is there an existing
   query or field that gives this (HTTP and MCP), or does the web derive
   it from the repos list and the stacks list?
5. How the Add-service shortcut lands: with one undeployed repository,
   door 2 (`/wizard/project/repository?project=<id>`) scans that
   repository and goes to the Service step. Which store seeds and which
   redirect guard (`furthestStep` in `ProjectWizardPage.tsx`) have to know
   about it?
6. What the Done step needs to render without a `stackId` (today it renders
   only when `stackId` is set).

## Answer

Findings with evidence: [research/03-attach-and-shortcut-mechanics.md](../research/03-attach-and-shortcut-mechanics.md).

- No server change. "Attach without deploying" scans as today (early
  "not installed" failure, free), then attaches with the existing
  `POST /api/projects/{id}/repos`. A later stack create with
  `link_repository: true` is a no-op re-link in the same project.
- Every early exit saves the tests-location answer the way `advance()`
  does. Today's skip drops it, a bug the build fixes along the way.
- "Undeployed repository" is derived in the web from the project's repos
  and stacks: an `app` repo no stack builds from. The prompt shows only to
  someone who can read stacks and add services, because a viewer without
  stack access would see every repo as undeployed.
- The Add-service shortcut is a seed hook beside `useSeedPreselectedProject`:
  scan the single undeployed repo, set repository, scan result, and
  candidate, then go to Service. The redirect guard needs no change.
- The Done step renders on the project, not the stack. The store gains a
  "finished without a stack" signal so the redirect guard and back
  navigation treat the early Done the way they treat a created stack, and
  the headline branches on whether a stack exists.
