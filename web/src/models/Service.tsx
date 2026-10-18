export const DeployStrategy = {
  Compose: "compose",
  Run: "run",
} as const;

export type DeployStrategy = (typeof DeployStrategy)[keyof typeof DeployStrategy];

export const DeployStatus = {
  Pending: "pending",
  Running: "running",
  Healthy: "healthy",
  Failed: "failed",
} as const;

export type DeployStatus = (typeof DeployStatus)[keyof typeof DeployStatus];

export const DeployKind = {
  Build: "build",
  Deploy: "deploy",
} as const;

export type DeployKind = (typeof DeployKind)[keyof typeof DeployKind];

export interface HealthCheck {
  url: string;
  expected_status?: number;
  timeout?: number;
  interval?: number;
  max_retries?: number;
}

export interface BuildSource {
  repo_owner?: string;
  repo_name?: string;
  branch?: string;
  dockerfile?: string;
  compose_path?: string;
}

// pattern is an exact branch name or a trailing wildcard; a wildcard rule always derives a clone.
export interface BranchDeployRule {
  pattern: string;
  docker_network: string;
  hostname_template?: string;
  name_suffix?: string;
}

export interface ServiceDef {
  id: string;
  project_id: string;
  name: string;
  target: string;
  // The machine it runs on; target is the older address field and is empty on stacks made since.
  machine?: string;
  strategy: DeployStrategy;
  compose_dir?: string;
  health_check: HealthCheck;
  env?: Record<string, string>;
  docker_network?: string;
  build_source?: BuildSource;
  branch_deploy_rules?: BranchDeployRule[];
  // Set on a branch deployment (a clone of a base service); never present on the base service itself.
  derived_from?: string;
  branch?: string;
  branch_deployments?: ServiceDef[];
  created_at: string;
  updated_at: string;
}

export interface Deploy {
  id: string;
  kind?: DeployKind;
  service_id: string;
  service: string;
  target: string;
  image: string;
  status: DeployStatus;
  strategy: DeployStrategy;
  triggered_by?: string;
  rule_id?: string;
  rule_name?: string;
  ticket_id?: string;
  pr_number?: number;
  created_at: string;
  updated_at: string;
}

// Newest deploy by creation time; the API's order is not part of its contract.
export const latestDeploy = (deploys: Deploy[] | undefined): Deploy | undefined =>
  deploys?.reduce<Deploy | undefined>((best, d) => (!best || d.created_at > best.created_at ? d : best), undefined);
