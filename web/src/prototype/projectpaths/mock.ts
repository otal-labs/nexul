import type { Repo } from "@/models/Repository";
import type { WizardProgressState } from "@/components/wizard/WizardProgressItem";
import type { WizardStepId } from "@/models/ProjectWizard";

export const PROJECT_NAME = "Northwind Portal";

const repo = (id: number, owner: string, name: string, branch = "main"): Repo => ({
  id,
  owner,
  name,
  full_name: `${owner}/${name}`,
  default_branch: branch,
  html_url: `https://example.com/${owner}/${name}`,
  provider: "github",
});

export const SEARCH_TEXT = "northwind";

export const REPOS: Repo[] = [
  repo(1, "acme", "northwind-portal"),
  repo(2, "acme", "northwind-portal-marketing-site-and-customer-help-centre", "release/2026-10"),
  repo(3, "acme", "northwind-e2e"),
  repo(4, "globex", "northwind-integrations", "develop"),
];

export const PICKED = REPOS[0]!;

// What a scan of the picked repository found, as one mono line.
export const SCAN_LINE = "compose.yaml · web, worker, postgres";

export type Scene = "search" | "picked" | "done-none" | "done-attached";

export const STEPS: WizardStepId[] = ["project", "people", "repository", "service", "reach", "branches", "done"];

export const DEPLOY_STEPS: WizardStepId[] = ["service", "reach", "branches"];

// The row on the Repository step: Info and Who's in it done, the rest untouched.
export const repoStepStates = (): Record<WizardStepId, WizardProgressState> => ({
  project: "done",
  people: "done",
  repository: "current",
  service: "unvisited",
  env: "unvisited",
  reach: "unvisited",
  branches: "unvisited",
  done: "unvisited",
});
