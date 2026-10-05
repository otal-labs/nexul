# Research: attaching without deploying and the Add-service shortcut

Findings for ticket 03, read from the code on 2026-10-05. Paths are repo-relative.

## 1. Where the wizard attaches the repository

Never through `POST /api/projects/{id}/repos`. The Service step sends
`build_source` with `link_repository: true` in one `POST /api/stacks`
(`web/src/components/wizard/WizardServiceStep.tsx:155-158`), and
`CreateStackWithOptions` links before creating (`internal/deploy/usecase.go:527-531`;
MCP `stack_create` has the same flag, `internal/deploy/mcp.go:322`).

Attaching first and creating the stack later works. A re-link hits the
`(owner, name)` primary key, and `LinkRepo` swallows the conflict when the
repository is already in this project (`server/cmd/wire_deploy.go:59-70`). It
fails only for another project's repository, as ADR 0039 requires. Covered by
`server/cmd/wire_deploy_test.go:29-32`. `POST /repos` defaults the role to `app`
(`internal/workspace/usecase.go:339-362`).

## 2. Scanning before attaching

A scan doesn't need the repository attached (`server/cmd/wire_repository.go:55-57`).
"GitHub App not installed" only ever comes from the scan
(`wire_repository.go:143-152`), and `AddRepo` never checks the installation.
Repositories picked from the search are already from the installation list, so a
"not installed" result is rare.

## 3. The tests-location answer on early exit

Lost today. It is saved only in `advance()`
(`web/src/components/wizard/WizardRepositoryStep.tsx:47-52`), which only the
deploy path reaches. `WizardSkipButton` navigates without saving, and the store
resets on unmount (`web/src/pages/ProjectWizardPage.tsx:97`). The question still
means something with no app repository: `tests_location` is a project field, and
a separate tests repository needs no app repository.

## 4. "Undeployed repository"

No field or query gives it on HTTP or MCP. Derive it from the project's repos
(`GET /api/projects/{id}/repos`, `ProjectHooks.tsx:95-99`) and its stacks
(`GET /api/stacks?project_id=`, `StackHooks.tsx:16-21`). It is an `app` repo that
no stack's `build_source` points at. `ListStacks` returns base stacks only, which
is fine. Stacks pass through `readableStacks`, so a viewer without stack read
access would see every repo as undeployed.

## 5. The Add-service shortcut

`AddServiceLink` goes to `/wizard/project/repository?project=<id>`
(`web/src/components/wizard/AddServiceLink.tsx:36`). `useSeedPreselectedProject`
seeds the project asynchronously (`ProjectWizardPage.tsx:18-28`).
`WizardServiceStep` renders only with `candidate` and `projectId` set
(`WizardServiceStep.tsx:78`). `furthestStep` (`ProjectWizardPage.tsx:59-65`)
allows the Service step once `candidate` is set, and it is re-read on each route
change. So a seed hook that sets `repository`, `scanResult` and `candidate` and
then calls `goTo("service")` needs no change to the guard.

## 6. The Done step without a stack

Today it doesn't render: `ProjectWizardStepContent` checks
`step === "done" && stackId`. `furthestStep` returns `done` only with `stackId`
(`ProjectWizardPage.tsx:61`), and `useCanRevisitStep` is keyed on `stackId`
(`web/src/hooks/useWizardNavigation.tsx:21-24`). The headline "{name} is
deploying on {machine}" is false without a stack. "View stack" is already guarded
on `stackId`. The interview offer and "View on the canvas" need only the project.
