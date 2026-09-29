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

export const ContainerStatus = {
  Pending: "pending",
  Running: "running",
  Healthy: "healthy",
  Exited: "exited",
  Stopped: "stopped",
} as const;

export type ContainerStatus = (typeof ContainerStatus)[keyof typeof ContainerStatus];

const DEPLOY_STATUS_DOT: Record<DeployStatus, string> = {
  [DeployStatus.Pending]: "bg-warning",
  [DeployStatus.Running]: "bg-info",
  [DeployStatus.Healthy]: "bg-success",
  [DeployStatus.Failed]: "bg-destructive",
};

// A stack with no deploys yet has no status to show; the dot stays neutral rather than picking a default state.
export const deployStatusDot = (status: DeployStatus | undefined): string =>
  status ? DEPLOY_STATUS_DOT[status] : "bg-muted-foreground";

const CONTAINER_STATUS_DOT: Record<ContainerStatus, string> = {
  [ContainerStatus.Pending]: "bg-warning",
  [ContainerStatus.Running]: "bg-info",
  [ContainerStatus.Healthy]: "bg-success",
  [ContainerStatus.Exited]: "bg-destructive",
  [ContainerStatus.Stopped]: "bg-muted-foreground",
};

export const containerStatusDot = (status: ContainerStatus): string => CONTAINER_STATUS_DOT[status];

// Stack is a deploy stack definition: a workload on one machine. Native reads it read-only, so this carries
// only the fields the phone screens render, not the full editable shape web's wizard needs.
export interface Stack {
  id: string;
  project_id: string;
  name: string;
  machine: string;
  strategy: DeployStrategy;
  managed: boolean;
  created_at: string;
  updated_at: string;
}

export interface Deploy {
  id: string;
  stack_id: string;
  image: string;
  status: DeployStatus;
  created_at: string;
  updated_at: string;
}

// Newest deploy by creation time; the API's order is not part of its contract.
export const latestDeploy = (deploys: Deploy[] | undefined): Deploy | undefined =>
  deploys?.reduce<Deploy | undefined>((best, d) => (!best || d.created_at > best.created_at ? d : best), undefined);

// The deploy's short image tag ("abc123" from "ghcr.io/org/app:abc123"), or a short id for a build with no image.
export const deployTitle = (deploy: Deploy): string => {
  if (!deploy.image) return `Build ${deploy.id.slice(0, 7)}`;
  const ref = deploy.image.split("/").at(-1) ?? deploy.image;
  const [named = ref, digest] = ref.split("@");
  if (digest) return digest.replace(/^sha256:/, "").slice(0, 7);
  return named.split(":")[1] ?? named;
};

// Declared is the compose/run parse's view of a container, before anything has been observed running.
export interface Declared {
  image?: string;
  build?: string;
}

// Container is one observed container belonging to a stack; read-only on the phone (the compose file is the
// only source of truth for what it declares).
export interface Container {
  id: string;
  stack_id: string;
  name: string;
  declared: Declared;
  image?: string;
  status: ContainerStatus;
}

// A container's declared image, or the image observed running once the runner reports one.
export const containerImage = (container: Container): string =>
  container.image || container.declared.image || container.declared.build || "—";

// "" is a line outside any phase (the final "deploy failed: …"); it belongs to whichever phase was last active.
export type DeployPhase = "checkout" | "build" | "deploy" | "";

// One line of GET /api/deploys/{id}/log, oldest first; ts is unix milliseconds.
export interface DeployLogLine {
  seq: number;
  ts: number;
  phase: DeployPhase;
  text: string;
}
