import { z } from "zod";

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
  log: string;
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

export const ServiceFormSchema = z
  .object({
    name: z.string().trim().min(1, "Service name is required"),
    target: z.string().trim().min(1, "Runner is required"),
    strategy: z.enum([DeployStrategy.Compose, DeployStrategy.Run]),
    compose_dir: z.string().trim(),
    docker_network: z.string().trim(),
    health_url: z.string().trim().min(1, "Health check URL is required"),
    env: z.string(),
    // Optional first-deploy inputs (image or ref to build).
    image: z.string().trim(),
    ref: z.string().trim(),
    // The repo the runner clones and builds from; required before a ref deploy can trigger.
    build_repo_owner: z.string().trim().optional(),
    build_repo_name: z.string().trim().optional(),
    build_branch: z.string().trim().optional(),
    build_dockerfile: z.string().trim().optional(),
    build_compose_path: z.string().trim().optional(),
  })
  .superRefine((val, ctx) => {
    if (val.strategy === DeployStrategy.Compose && !val.compose_dir) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["compose_dir"], message: "Compose directory is required for the compose strategy" });
    }
    if (val.strategy === DeployStrategy.Run && !val.docker_network) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["docker_network"], message: "Docker network is required for the run strategy" });
    }
  });

export type ServiceFormData = z.infer<typeof ServiceFormSchema>;
