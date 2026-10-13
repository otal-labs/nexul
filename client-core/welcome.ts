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
