import { WELCOME_STEP_NAMES, type WelcomeProgress, type WelcomeStep } from "@nexul/client-core/welcome";

// Profile always comes first and is no role's tick, so it lives only in the wizard.
export type WizardStep = "profile" | WelcomeStep;

// The URL's ?step: one step on screen, or the finish view.
export type WizardView = WizardStep | "finish";

export type StepMark = "done" | "skipped" | "left";

export const STEP_NAMES: Record<WizardStep, string> = { profile: "Profile", ...WELCOME_STEP_NAMES };

export const MARK_WORD: Record<StepMark, string> = { done: "Done", skipped: "Skipped", left: "To do" };

export const STEP_COPY: Record<WizardStep, { title: string; lead: string }> = {
  profile: {
    title: "Introduce yourself",
    lead: "The name and picture your team sees. Your sign-in filled them in; keep them or change them.",
  },
  github: {
    title: "Connect GitHub",
    lead: "Nexul lists repositories and links pull requests with your own GitHub account, so it only ever sees what you can open.",
  },
  computer: {
    title: "Add your computer",
    lead: "Plays run on your own computer through T3 Code. One command installs the Nexul app, finds T3 Code and pairs it.",
  },
  projects: {
    title: "Your projects",
    lead: "Where you've been added, and what you can do in each.",
  },
  deploy_server: {
    title: "Add a deploy server",
    lead: "A deploy server builds and runs your stacks. Run one command on it and Nexul waits for it to connect.",
  },
  phone: {
    title: "Get the phone app",
    lead: "Tickets, chat and plays in your pocket. Install the app, then scan the code with it to sign in.",
  },
};

export const wizardSteps = (progress: WelcomeProgress): WizardStep[] => ["profile", ...progress.steps.map((step) => step.key)];

// Profile counts as done once the first run happened, or once the person moved past it, which saved it.
export const stepMarks = (progress: WelcomeProgress, view: WizardView): Record<WizardStep, StepMark> => {
  const marks = { profile: progress.first_run_done || view !== "profile" ? "done" : "left" } as Record<WizardStep, StepMark>;
  for (const step of progress.steps) marks[step.key] = step.state;
  return marks;
};

const isView = (steps: WizardStep[], value: string | null): value is WizardView =>
  value === "finish" || steps.includes(value as WizardStep);

// A named step opens as asked; otherwise the first run starts at Profile and every later visit at the first step not done.
export const openingView = (progress: WelcomeProgress, requested: string | null): WizardView => {
  if (isView(wizardSteps(progress), requested)) return requested;
  if (!progress.first_run_done) return "profile";
  return progress.steps.find((step) => step.state !== "done")?.key ?? "finish";
};

export const viewIndex = (steps: WizardStep[], view: WizardView): number => (view === "finish" ? steps.length : steps.indexOf(view));

// The view one step on or back; past the last step is the finish view.
export const neighbourView = (steps: WizardStep[], view: WizardView, by: 1 | -1): WizardView =>
  steps[viewIndex(steps, view) + by] ?? (by > 0 ? "finish" : "profile");

export const laterTitle = (count: number): string => {
  if (count === 0) return "You're set up";
  if (count === 1) return "One thing left for later";
  return `${count} things left for later`;
};
