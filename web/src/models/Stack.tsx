import { z } from "zod";

// A stack owns one or more observed Container records. This is the canonical model for /api/stacks;
// models/Service.tsx keeps the older ServiceDef shape alive for the project-page "add service" flow until that
// flow moves onto the wizard.

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

export const ContainerStatus = {
  Pending: "pending",
  Running: "running",
  Healthy: "healthy",
  Exited: "exited",
  Stopped: "stopped",
} as const;

export type ContainerStatus = (typeof ContainerStatus)[keyof typeof ContainerStatus];

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
  // Port the hostname template exposes; required whenever hostname_template is set (spec §3).
  port?: number;
}

// Stack is a deploy stack definition (spec §2): a workload owned by one project, on one machine, with a strategy.
export interface Stack {
  id: string;
  project_id: string;
  name: string;
  // Derived once from name at creation; the compose project name and default network prefix.
  slug: string;
  // The machine (not a specific runner) this stack targets; dispatch picks any idle runner on it.
  machine: string;
  strategy: DeployStrategy;
  compose_path?: string;
  env?: Record<string, string>;
  docker_network?: string;
  ports?: string[];
  mounts?: string[];
  command?: string[];
  build_source?: BuildSource;
  branch_deploy_rules?: BranchDeployRule[];
  // Set on a branch deployment (a clone of a base stack); never present on the base stack itself.
  derived_from?: string;
  branch?: string;
  // A stack adopted from machine discovery starts unmanaged until a repository is attached.
  managed: boolean;
  created_at: string;
  updated_at: string;
}

// GET /api/stacks/{id} attaches the base stack's live branch deployments (empty for a derived stack).
export interface StackWithBranches extends Stack {
  branch_deployments?: Stack[];
}

export interface Deploy {
  id: string;
  kind?: DeployKind;
  stack_id: string;
  // service_id is a deprecated alias for stack_id, kept on the wire for one release.
  service_id?: string;
  service: string;
  target: string;
  image: string;
  status: DeployStatus;
  strategy: DeployStrategy;
  log: string;
  // The container's address on its docker network, reported by the runner at start.
  address?: string;
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

// Declared is the compose/run parse's view of a container, before anything has been observed running.
export interface Declared {
  image?: string;
  build?: string;
  ports?: string[];
  env_keys?: string[];
}

// One docker network a container joined, with its address on it (reported by the runner).
export interface ContainerNetwork {
  name: string;
  address?: string;
}

// Container is one observed container belonging to a stack: a compose service, or a run stack's single
// container. Created pending at stack creation from the parse (declared); updated from every observation report.
export interface Container {
  id: string;
  stack_id: string;
  // The compose service name; a run stack's single service is named after the stack's slug.
  name: string;
  declared: Declared;
  container_name?: string;
  image?: string;
  status: ContainerStatus;
  networks?: ContainerNetwork[];
  ports?: string[];
  observed_at?: string;
}

export const RuleFormSchema = z.object({
  pattern: z.string().trim().min(1, "Branch pattern is required"),
  docker_network: z.string().trim().min(1, "Docker network is required"),
  hostname_template: z.string().trim(),
  name_suffix: z.string().trim(),
  port: z.string().trim(),
});

export type RuleFormData = z.infer<typeof RuleFormSchema>;

export const DeployRefFormSchema = z.object({
  ref: z.string().trim().min(1, "Ref is required"),
});
export type DeployRefFormData = z.infer<typeof DeployRefFormSchema>;

// Wizard-facing shapes.
export const StackStrategies = ["compose", "run"] as const;
export type StackStrategy = (typeof StackStrategies)[number];

export type CreateStackInput = Partial<Stack> & {
  name: string;
  project_id: string;
  machine: string;
  strategy: StackStrategy;
  declared?: Record<string, Declared>;
  // Attaches build_source's repo to the project first, a no-op if it's already linked.
  link_repository?: boolean;
  // Enqueues the first deploy; ref defaults server-side to build_source.branch when omitted.
  deploy?: boolean;
  ref?: string;
};

// The one extra field POST /api/stacks' 201 body carries over a plain Stack when `deploy: true` was sent.

export interface StackDeployRef {
  id: string;
  stack_id: string;
  status: string;
  kind: string;
}

export type CreateStackResponse = Stack & { deploy?: StackDeployRef };

export const ContainerStatuses = ["pending", "running", "healthy", "exited", "stopped"] as const;
