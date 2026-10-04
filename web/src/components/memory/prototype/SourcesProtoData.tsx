import type { AnswerValue, QuestionItem } from "@/models/Question";

export type SourceKind = "path" | "doc" | "memory" | "project" | "text";
export type Stance = "follow" | "question";

export interface ProtoSource {
  id: string;
  kind: SourceKind;
  name: string;
  stance: Stance;
  gone?: true;
  hidden?: true;
  fresh?: true;
}

export interface ProtoDraft {
  value: AnswerValue;
  from: string;
  quote: string;
}

const q = (id: string, text: string, header: string, options: string[], multi = false): QuestionItem => ({
  id,
  text,
  header,
  multi_select: multi,
  options: options.map((label) => ({ label })),
});

// The twelve questions of the Interview template's code default (internal/memories DefaultInterviewTemplate).
export const QUESTIONS: QuestionItem[] = [
  q("q1", "What languages and frameworks does this project use?", "Name versions only where they are pinned on purpose.", []),
  q("q2", "How is the code organised?", "", ["Layers (routes, logic, storage)", "Feature folders", "Entities and systems", "Functional core with a thin outer shell"]),
  q("q3", "How do errors travel?", "And what gets logged, at which level.", ["Returned and wrapped", "Thrown and caught at the edge", "Result types"]),
  q("q4", "When are tests written?", "", ["Before the code", "With the change", "Only for bugs", "No tests yet"]),
  q("q5", "Which tests does a change need?", "And the coverage floor, if any.", ["Unit", "Integration against real dependencies", "End-to-end in this repo", "End-to-end in a separate repo"], true),
  q("q6", "Which style rules matter most?", "Skip what a linter already enforces.", ["Early return, no else", "Small functions", "Comments only for why", "Strict types, no any"], true),
  q("q7", "When may a change add a dependency?", "", ["Freely", "When it saves real code", "Only after asking", "Only with a written decision"]),
  q("q8", "Where do secrets live?", "And what must never be committed or logged.", ["Environment variables", "A secrets manager", "An encrypted file in the repo"]),
  q("q9", "How does a change reach the main branch?", "And how commit messages are written.", ["Pull request, squash merge", "Pull request, merge commit", "Straight to main"]),
  q("q10", "Where are decisions written down?", "", ["Decision records in the repo", "A docs folder", "A wiki outside the repo", "Nowhere yet"]),
  q("q11", "Does this project have a user interface?", "If yes, the design system, screen sizes, and accessibility rules.", ["Web", "Mobile", "Both", "None"]),
  q("q12", "Which words mean something specific here?", "One per line, the term then what it means.", []),
];

export const FOLLOW_UPS: QuestionItem[] = [
  q("f1", "Phase 1 kept booking state in the mobile app. Should phase 2 keep it on the server?", "", ["On the server (Recommended)", "In the app, as phase 1 did"]),
  q("f2", "Phase 1 has no tests around payments. Should the payment flow get end-to-end tests first?", "", ["Yes, before any change to it (Recommended)", "Only for new code"]),
];

const pick = (...selected: string[]): AnswerValue => ({ selected });

const LONG_STACK =
  "Go 1.24 on the server with the standard library router and sqlc for queries; React 19 with TypeScript, Vite, TanStack Query and Zustand in the web app; React Native with Expo SDK 54 for the phone app, pinned because the store build breaks on a minor bump; SQLite in WAL mode as the only database.";

export const DRAFTS: Record<string, ProtoDraft> = {
  q1: { value: { text: LONG_STACK }, from: "practices/stack.md", quote: "Go 1.24 and Expo SDK 54 are pinned on purpose" },
  q2: { value: pick("Layers (routes, logic, storage)"), from: "Engineering standards", quote: "Routes call use-cases; use-cases call repositories" },
  q3: { value: { selected: ["Returned and wrapped"], text: "Logged once at the edge with slog" }, from: "practices/errors.md", quote: "Wrap with context, log once where it is handled" },
  q4: { value: pick("With the change"), from: "practices/testing.md", quote: "A change lands with its tests" },
  q5: { value: pick("Unit", "Integration against real dependencies"), from: "practices/testing.md", quote: "Test error paths first" },
  q6: { value: pick("Early return, no else", "Comments only for why", "Strict types, no any"), from: "Engineering standards", quote: "No else after a return" },
  q7: { value: pick("Only after asking"), from: "Engineering standards", quote: "Ask in the channel before adding a package" },
  q9: { value: pick("Pull request, squash merge"), from: "practices/git.md", quote: "Squash on merge, the PR title is the commit" },
  q11: { value: pick("Both"), from: "Engineering standards", quote: "The web app and the phone app share one design system" },
};

export const DRAFT_ORDER = ["q1", "q2", "q3", "q4", "q5", "q6", "q7", "q9", "q11"];

export const ANSWERS: Record<string, AnswerValue> = {
  ...Object.fromEntries(Object.entries(DRAFTS).map(([id, d]) => [id, d.value])),
  q8: pick("A secrets manager"),
  q10: pick("Decision records in the repo"),
  q12: { text: "Booking: a confirmed slot with a payment attached\nHold: a slot kept for ten minutes while the customer pays" },
  f1: pick("On the server (Recommended)"),
  f2: pick("Yes, before any change to it (Recommended)"),
};

export const SUGGESTIONS: Record<string, ProtoDraft> = {
  q4: { value: pick("Before the code"), from: "Phase 2 requirements", quote: "Every payment change starts from a failing test" },
  q5: {
    value: pick("Unit", "Integration against real dependencies", "End-to-end in this repo"),
    from: "Phase 2 requirements",
    quote: "The booking and payment flows each have an end-to-end test",
  },
};

const PHASE_ONE: ProtoSource = { id: "s3", kind: "project", name: "Clutch Hub phase 1", stance: "question" };

export const SOURCES_THREE: ProtoSource[] = [
  { id: "s1", kind: "path", name: "practices/", stance: "follow" },
  { id: "s2", kind: "doc", name: "Engineering standards", stance: "follow" },
  PHASE_ONE,
];

export const SOURCES_LATER: ProtoSource[] = [
  ...SOURCES_THREE,
  { id: "s4", kind: "path", name: "docs/engineering/standards/backend-error-handling-and-logging.md", stance: "follow" },
  { id: "s5", kind: "doc", name: "Clutch Hub platform conventions for services, data storage, payments, and the release process", stance: "follow" },
  { id: "s6", kind: "doc", name: "Phase 2 requirements", stance: "follow", fresh: true },
  { id: "s7", kind: "memory", name: "", stance: "follow", gone: true },
  { id: "s8", kind: "doc", name: "", stance: "follow", hidden: true },
];

export const SOURCES_SUPERSEDING: ProtoSource[] = [PHASE_ONE];

export const PICKABLE: Record<"doc" | "memory" | "project", string[]> = {
  doc: ["Engineering standards", "Phase 2 requirements", "Clutch Hub platform conventions for services, data storage, payments, and the release process", "Onboarding"],
  memory: ["Interview (Clutch Hub phase 1)", "Deploying to the Hetzner box", "Payment provider quirks"],
  project: ["Clutch Hub phase 1", "MgClutch website", "Booking widget"],
};

export const PASTED = {
  label: "Phase 1 handover notes",
  body: "Bookings are created in the app and synced on reconnect, which lost two bookings last spring.\nPayments go through the provider's hosted page; refunds are done by hand in its dashboard.\nThere are no tests around the payment webhook.",
};

export const MEMORY_BODY = `## Stack
- Go 1.24 server, standard library router, sqlc. SQLite in WAL mode is the only database.
- React 19 web app with TanStack Query and Zustand; Expo SDK 54 phone app, pinned.

## Code
- Layers: routes call use-cases, use-cases call repositories. No layer skips one.
- Errors are returned and wrapped with context, logged once where they are handled.

## Tests
- A change lands with its tests: unit plus integration against a real SQLite file.
- Payment changes get an end-to-end test before anything else changes in that flow.

## Changes
- Pull request, squash merge; the PR title is the commit message.
- Ask before adding a dependency. Secrets live in the secrets manager, never in the repo.`;
