// Matches internal/repository's model.go JSON exactly (spec §5).

export interface Repo {
  id: number;
  owner: string;
  name: string;
  full_name: string;
  default_branch: string;
  html_url: string;
}

export const CandidateKinds = ["compose", "dockerfile"] as const;
export type CandidateKind = (typeof CandidateKinds)[number];

export interface DeclaredServiceBuild {
  context: string;
  dockerfile: string;
}

export interface DeclaredService {
  name: string;
  image?: string;
  build?: DeclaredServiceBuild;
  ports: number[];
  expose: number[];
  env_keys: string[];
}

export interface Reachable {
  service: string;
  port: number;
}

export interface Candidate {
  kind: CandidateKind;
  path: string;
  name: string;
  services: DeclaredService[];
  reachable?: Reachable;
}

export interface ScanResult {
  default_branch: string;
  candidates: Candidate[];
  env_keys: string[];
}

// A manual candidate the wizard builds itself when a scan finds nothing to deploy (spec §5/§7 issue answer):
// a single Dockerfile-kind stack with no declared services yet.
export const manualCandidate = (repoName: string, dockerfilePath: string): Candidate => ({
  kind: "dockerfile",
  path: dockerfilePath,
  name: repoName,
  services: [],
});

// mapNotInstalled (server/cmd/wire_repository.go) folds "not installed" into the ErrNotFound message as free
// text ending in an install URL; this pulls it back out so the wizard can render a real link instead of prose.
export const parseInstallUrl = (message: string): string | undefined => message.match(/https:\/\/\S+/)?.[0];
