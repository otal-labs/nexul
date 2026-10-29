import type { Project, ProjectSetup } from "@/models/Project";

export const WizardSteps = ["project", "people", "repository", "deploy", "service", "env", "reach", "branches", "done"] as const;
export type WizardStepId = (typeof WizardSteps)[number];

// Deploy and Done exist only in the browser; the server's setup records the rest.
export type ServerStepId = Exclude<WizardStepId, "deploy" | "done">;

// The steps that need a service; a project with none folds them into Deploy on the rail and on Done.
export const DeploySteps: readonly ServerStepId[] = ["service", "env", "reach", "branches"];

export const isDeployStep = (id: WizardStepId): boolean => (DeploySteps as readonly WizardStepId[]).includes(id);

export type WizardDoor = "setup" | "add" | "attach";

const doorSteps: Record<WizardDoor, readonly WizardStepId[]> = {
  setup: WizardSteps,
  add: ["repository", "service", "env", "reach", "branches", "done"],
  attach: ["repository", "service", "env", "branches", "done"],
};

// Environment shows only when the scan or the recorded setup has keys for it.
export const wizardOrder = (door: WizardDoor, showEnv: boolean): WizardStepId[] =>
  doorSteps[door].filter((id) => id !== "env" || showEnv);

const labels: Record<WizardStepId, string> = {
  project: "Name",
  people: "People",
  repository: "Repository",
  deploy: "Deploy",
  service: "Service",
  env: "Environment",
  reach: "Domain",
  branches: "Branches",
  done: "Done",
};

export const wizardStepLabel = (id: WizardStepId, door: WizardDoor): string =>
  id === "service" && door === "attach" ? "Attach" : labels[id];

const titles: Record<Exclude<WizardStepId, "done">, string> = {
  project: "Name your project",
  people: "Who's in it?",
  repository: "Where's the code?",
  deploy: "Deploy it now?",
  service: "Where should it run?",
  env: "Environment values",
  reach: "Give it a domain",
  branches: "Deploy other branches?",
};

export const wizardStepTitle = (id: WizardStepId, door: WizardDoor, projectName: string | null): string => {
  if (id === "done") return projectName ? `${projectName} is ready` : "All set";
  if (id === "service" && door === "attach") return "Attach it";
  return titles[id];
};

const leads: Partial<Record<WizardStepId, string>> = {
  project: "Its prefix starts every ticket key, like BE-12.",
  people: "Owners and you are always in.",
  env: "Found in .env.example. Blank ones stay unset.",
  branches: "Each gets its own copy and URL.",
};

export const wizardStepLead = (id: WizardStepId): string | undefined => leads[id];

// The rail's microheader names the door; a project that was already there is being set up again.
export const wizardDoorWord = (door: WizardDoor, arrivedWithProject: boolean): string => {
  if (door === "add") return "New service";
  if (door === "attach") return "Attach repository";
  return arrivedWithProject ? "Project setup" : "New project";
};

export interface WizardFacts {
  door: WizardDoor;
  // The run has a project, made here or arrived with.
  project: boolean;
  // The service this run works on exists.
  stack: boolean;
  // An app repository is attached or picked and no service builds from it yet.
  appRepo: boolean;
  // In the setup run, app repositories are attached and every one already deploys from a service.
  deployed: boolean;
  // A scan with something to build is loaded.
  candidate: boolean;
  // The scan that fills in for a missing candidate: reading, errored, or finished with nothing to build.
  scan: "idle" | "running" | "failed" | "empty";
}

export type StepView =
  | { kind: "form" }
  | { kind: "summary" }
  | { kind: "scanning" }
  | { kind: "scanFailed" }
  | { kind: "nothingFound" }
  | { kind: "deployed" }
  | { kind: "needs"; needs: NeedsStep };

// The steps whose product the others build on.
export type NeedsStep = "project" | "repository" | "service";

const form: StepView = { kind: "form" };
const needs = (step: NeedsStep): StepView => ({ kind: "needs", needs: step });

const scanView = (scan: WizardFacts["scan"]): StepView => {
  if (scan === "failed") return { kind: "scanFailed" };
  if (scan === "empty") return { kind: "nothingFound" };
  return { kind: "scanning" };
};

const buildView = (f: WizardFacts): StepView => {
  if (f.stack) return { kind: "summary" };
  if (f.candidate) return form;
  if (!f.appRepo) return f.deployed ? { kind: "deployed" } : needs("repository");
  return scanView(f.scan);
};

// What a step shows. Missing groundwork names the one step that supplies it, and that step always has a view of its own.
export const wizardStepView = (step: WizardStepId, f: WizardFacts): StepView => {
  const steps = doorSteps[f.door];
  if (!steps.includes(step)) return needs(f.door === "setup" ? "project" : "repository");
  if (step === "project") return f.project ? { kind: "summary" } : form;
  if (!f.project && f.door === "setup") return needs("project");
  if (!f.project) return step === "repository" ? form : needs("repository");
  if (step === "deploy" || step === "service") return buildView(f);
  if (step === "env" || step === "reach" || step === "branches") return f.stack ? form : needs("service");
  return form;
};

const skippable: Record<WizardDoor, readonly WizardStepId[]> = {
  setup: ["people", "service", "env", "reach", "branches"],
  add: ["env", "reach", "branches"],
  attach: ["env", "branches"],
};

// The add and attach doors exist to make a service, so Service has no way past it there.
export const wizardCanSkip = (step: WizardStepId, door: WizardDoor): boolean => skippable[door].includes(step);

export interface SkipPlan {
  to: WizardStepId | undefined;
  // Steps recorded skipped with it.
  alsoSkipped: ServerStepId[];
}

// With no service, skipping a deploy step is Later: the rest skip with it and setup goes to Done, not through empty screens.
export const wizardSkipPlan = (step: WizardStepId, door: WizardDoor, order: WizardStepId[], stack: boolean): SkipPlan => {
  if (door === "setup" && isDeployStep(step) && !stack) {
    return { to: "done", alsoSkipped: DeploySteps.filter((id) => id !== step && order.includes(id)) };
  }
  return { to: order[order.indexOf(step) + 1], alsoSkipped: [] };
};

export const needsMessage: Record<NeedsStep, string> = {
  project: "Name the project first.",
  repository: "Pick a repository first.",
  service: "Create the service first.",
};

export type RailMark = "current" | "done" | "skipped" | "later" | "left";

interface RailContext {
  door: WizardDoor;
  order: WizardStepId[];
  step: WizardStepId;
  setup: ProjectSetup | undefined;
  stack: boolean;
}

// What the project records for a step, Deploy read from the service; the other doors show only what this run passed.
export const railMark = (id: WizardStepId, { door, order, step, setup, stack }: RailContext): RailMark => {
  if (id === step) return "current";
  if (door !== "setup") return order.indexOf(id) < order.indexOf(step) ? "done" : "left";
  if (id === "done") return setup?.finished ? "done" : "left";
  if (id === "deploy") {
    if (stack || setup?.steps.service === "done") return "done";
    return setup?.steps.service === "skipped" ? "later" : "left";
  }
  return setup?.steps[id] ?? "left";
};

export interface RailRow {
  id: WizardStepId;
  nested: boolean;
}

// In the setup door the steps after Deploy are its sub-steps: shown while one of them is open, or once a service exists.
export const railRows = (door: WizardDoor, order: WizardStepId[], step: WizardStepId, stack: boolean): RailRow[] => {
  if (door !== "setup") return order.map((id) => ({ id, nested: false }));
  const open = stack || step === "deploy" || isDeployStep(step);
  return order.filter((id) => open || !isDeployStep(id)).map((id) => ({ id, nested: isDeployStep(id) }));
};

// Continue setup lands on the first step neither done nor skipped, else on Done; a step that needs a service with none made opens Deploy.
export const setupResumePath = (project: Project): string => {
  const first = WizardSteps.find(
    (id): id is ServerStepId =>
      id !== "deploy" && id !== "done" && (id !== "env" || (project.setup.env_keys?.length ?? 0) > 0) && !project.setup.steps[id],
  );
  const step = first && isDeployStep(first) && !project.setup.stack_id ? "deploy" : (first ?? "done");
  return `/wizard/project/${step}?project=${project.id}`;
};
