export const WizardSteps = ["project", "repository", "service", "env", "reach", "branches", "done"] as const;
export type WizardStepId = (typeof WizardSteps)[number];

const labels: Record<WizardStepId, string> = {
  project: "Info",
  repository: "Repository",
  service: "Service",
  env: "Environment",
  reach: "Reach",
  branches: "Deploy branches",
  done: "Done",
};

const descriptions: Partial<Record<WizardStepId, string>> = {
  repository: "Pick the repository to deploy. Nexul reads its Dockerfile or compose file.",
  reach: "Optional. Give this service a hostname now, or later from the stack page.",
  branches: "Optional. Deploy other branches as their own copies, each at its own URL.",
};

// Attach mode adopts an existing stack, so its service rung is named for what it does.
export const wizardStepLabel = (id: WizardStepId, isAttach: boolean): string =>
  id === "service" && isAttach ? "Attach" : labels[id];

export const wizardStepDescription = (id: WizardStepId): string | undefined => descriptions[id];
