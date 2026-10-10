import type { Project } from "@/models/Project";

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

// Continue setup lands on the first step neither done nor skipped, else on Done to finish.
export const setupResumePath = (project: Project): string => {
  const step = WizardSteps.find((id) =>
    id !== "done" && (id !== "env" || (project.setup.env_keys?.length ?? 0) > 0) && !project.setup.steps[id],
  ) ?? "done";
  return `/wizard/project/${step}?project=${project.id}`;
};
