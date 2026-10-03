import type { QuestionItem } from "@/models/Question";

// Throwaway prototype data: the default template's twelve questions, two follow-ups, and a generated memory.
export interface ProtoQuestion extends QuestionItem {
  why?: string;
}

const opts = (...labels: string[]) => labels.map((label) => ({ label }));

export const TEMPLATE: ProtoQuestion[] = [
  { id: "q1", text: "What languages and frameworks does this project use?", header: "Name versions only where they are pinned on purpose.", options: [] },
  {
    id: "q2",
    text: "How is the code organised?",
    options: opts("Layers (routes, logic, storage)", "Feature folders", "Entities and systems", "Functional core with a thin outer shell"),
  },
  {
    id: "q3",
    text: "How do errors travel?",
    header: "And what gets logged, at which level.",
    options: opts("Returned and wrapped", "Thrown and caught at the edge", "Result types"),
  },
  { id: "q4", text: "When are tests written?", options: opts("Before the code", "With the change", "Only for bugs", "No tests yet") },
  {
    id: "q5",
    text: "Which tests does a change need?",
    header: "And the coverage floor, if any.",
    multi_select: true,
    options: opts("Unit", "Integration against real dependencies", "End-to-end in this repo", "End-to-end in a separate repo"),
  },
  {
    id: "q6",
    text: "Which style rules matter most?",
    header: "Skip what a linter already enforces.",
    multi_select: true,
    options: opts("Early return, no else", "Small functions", "Comments only for why", "Strict types, no any"),
  },
  {
    id: "q7",
    text: "When may a change add a dependency?",
    options: opts("Freely", "When it saves real code", "Only after asking", "Only with a written decision"),
  },
  {
    id: "q8",
    text: "Where do secrets live?",
    header: "And what must never be committed or logged.",
    options: opts("Environment variables", "A secrets manager", "An encrypted file in the repo"),
  },
  {
    id: "q9",
    text: "How does a change reach the main branch?",
    header: "And how commit messages are written.",
    options: opts("Pull request, squash merge", "Pull request, merge commit", "Straight to main"),
  },
  {
    id: "q10",
    text: "Where are decisions written down?",
    options: opts("Decision records in the repo", "A docs folder", "A wiki outside the repo", "Nowhere yet"),
  },
  {
    id: "q11",
    text: "Does this project have a user interface?",
    header: "If yes, the design system, screen sizes, and accessibility rules.",
    options: opts("Web", "Mobile", "Both", "None"),
  },
  { id: "q12", text: "Which words mean something specific here?", header: "One per line, the term then what it means.", options: [] },
];

export const FOLLOW_UPS: ProtoQuestion[] = [
  {
    id: "f1",
    text: "When are tests written?",
    why: "You skipped this; most commits in the last month add a test file beside the code they change.",
    options: opts("With the change (Recommended)", "Before the code", "Only for bugs"),
  },
  {
    id: "f2",
    text: "Where do the end-to-end tests live?",
    why: "You picked end-to-end tests in this repo, but there is no e2e folder or Playwright config in the checkout.",
    options: opts("Add them here as the project grows (Recommended)", "They live in a separate repo", "Drop end-to-end from the rules"),
  },
];

export const LONG_STACK_ANSWER =
  "TypeScript 5.6 with React 19 and Vite on the web side, Go 1.23 for the API server and the background workers, SQLite through sqlc for storage, Tailwind 4 for styling, and Bun as the package manager and test runner";

export const MEMORY_BODY = `## Stack
- TypeScript 5.6, React 19 and Vite for the web app; Go 1.23 for the API server and workers.
- SQLite through sqlc. Never hand-write a query the generator can produce.
- Bun is the package manager and test runner; never add npm or yarn lockfiles.

## Architecture
- Layers: routes call use-cases, use-cases call storage. A route never touches the database.
- One folder per domain; domains talk only through the event bus.

## Errors and logging
- Return errors and wrap them with context (\`fmt.Errorf("load cart: %w", err)\`); never panic in a request path.
- Log once, at the edge, with \`slog\`: warn for a refused request, error for a broken one.

## Testing
- Tests come with the change, in the same pull request, beside the code they cover.
- A change needs unit tests and integration tests against a real SQLite file. No mocks of the database.
- End-to-end tests are planned for this repo; until the folder exists, describe the manual check in the PR.
- Coverage floor is 80%. Test the error paths first.

## Style
- Early return, no else.
- Comments only for why; the code says what.
- Strict types: no \`any\`, no non-null assertions.

## Dependencies
- Add a dependency only when it saves real code, and say what it replaces in the PR.

## Secrets
- Environment variables only. Never commit a \`.env\` file, never log a token or a request body.

## Shipping
- Every change is a pull request, squash merged.
- Commit messages say what changed and why, in the present tense.

## Decisions
- Decision records live in \`docs/adr/\`, one per decision that was hard to reverse.

## Interface
- Web only, built at 768px first and checked at 1024 and 1440.
- Every control is reachable by keyboard and has a visible focus ring.

## Vocabulary
- **Basket**: the items a shopper has picked but not paid for. Never "cart" in the UI.
- **Drop**: a limited release with a start time. Not a sale.
`;
