# 10: The Welcome wizard

Type: task
Status: open
Blocked by: 09

Read first: `AGENTS.md`, `CONTEXT.md` (Welcome wizard, Template role), `practices/react-guide.md` (F1 to F7 and the self-review checklist), `practices/design-language.md`,
`practices/testing.md`, ticket 05's answer, the branch `proto/welcome-wizard` (read it; never merge it).

## What to build

- `/wizard/welcome` in the Rail frame with every step screen from ticket 05. Reuse the existing pieces it
  names: Add a computer's checks, `RunnerInstallCommands`, `QrFrame`, `ConnectGitHubPrompt`, `EmptyState`.
- `OnboardingGate` sends a person whose first run hasn't happened to the wizard. The wizard replaces
  `FirstLoginWizardPage` and `/wizard/onboarding/profile`, which 404 afterwards (no redirect).
- Skip for now, Finish later, the end screen with ticket 05's copy fix, and opening the wizard on a named step.

## Acceptance criteria

- [ ] Web tests: the steps shown follow the progress, a step already done shows what it found, Skip calls the
      route, and Finish later sets the first run.
- [ ] Each step's live check settles from `welcome.progress_changed` without a refresh.
- [ ] Screenshots at 768, 1024, 1440 match ticket 05.
