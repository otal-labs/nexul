// Mirrors internal/platform/welcomestep: the Welcome wizard steps a role can give, in the order the wizard shows them.
export const WELCOME_STEPS = ["github", "computer", "projects", "deploy_server", "phone"] as const;

export type WelcomeStep = (typeof WELCOME_STEPS)[number];

export const WELCOME_STEP_NAMES: Record<WelcomeStep, string> = {
  github: "Connect GitHub",
  computer: "Your computer",
  projects: "Your projects",
  deploy_server: "Add a deploy server",
  phone: "Get the phone app",
};

// Mirrors internal/welcome's Step: one of a person's steps as GET /api/me/welcome returns it.
export interface WelcomeStepProgress {
  key: WelcomeStep;
  state: "done" | "skipped" | "left";
  // When it was skipped, or for the phone when one first signed in.
  at?: string;
  // Given again after the first run; only projects reads it.
  given: boolean;
}

export interface WelcomeProgress {
  first_run_done: boolean;
  steps: WelcomeStepProgress[];
  dot: boolean;
}

// The dot rule, as the server's LightsDot: a step left lights it, but projects only once someone gave it, since
// a person can't add themselves to a project.
export const lightsDot = (step: WelcomeStepProgress): boolean => step.state === "left" && (step.key !== "projects" || step.given);

export const showsDot = (progress: WelcomeProgress): boolean => progress.steps.some(lightsDot);
