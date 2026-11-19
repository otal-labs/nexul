import type { Checkout, Computer } from "@/models/Pairing";
import type { RepoRef } from "@/models/Project";

type FactProject = NonNullable<Computer["facts"]>["projects"][number];

// Where a project's runs in one of its repositories would land on one computer, as the server decides it (ADR 0204):
// the checkout picked there for that repository, else the one whose git remote is it; several are a pick, none is missing.
export type ProjectCheckout =
  | { state: "has"; project: FactProject; picked: boolean }
  | { state: "ambiguous"; projects: FactProject[] }
  | { state: "missing" }
  | { state: "unknown" };

// A project's GitHub repository as T3 names a checkout's remote, and as owner/name, the key its checkout is kept under.
export interface ProjectRemote {
  remote: string;
  repository: string;
  app: boolean;
}

export const projectRemotes = (repos: RepoRef[]): ProjectRemote[] =>
  repos
    .filter((r) => r.connector_id === "github")
    .map((r) => ({ remote: `github.com/${r.full_name.toLowerCase()}`, repository: r.full_name.toLowerCase(), app: r.role === "app" }));

// The repository a run that names none works on: the app's, else the first.
export const defaultRemote = (remotes: ProjectRemote[]): ProjectRemote | undefined => remotes.find((r) => r.app) ?? remotes[0];

const remoteOf = (projects: FactProject[], id: string) => projects.find((p) => p.id === id)?.repository?.toLowerCase() ?? "";

// The pick for repo on computerId; one from before picks were per repository counts for the repository it clones, or
// for the default one when it clones none of the project's.
const pickedId = (computerId: string, checkouts: Checkout[], remotes: ProjectRemote[], repo: ProjectRemote | undefined, projects: FactProject[]) => {
  const key = repo?.repository ?? "";
  const own = checkouts.find((c) => c.computer_id === computerId && c.repository === key);
  if (own) return own.harness_project_id;
  const legacy = checkouts.find((c) => c.computer_id === computerId && c.repository === "");
  if (!legacy || !repo) return undefined;
  const remote = remoteOf(projects, legacy.harness_project_id);
  if (remote === repo.remote) return legacy.harness_project_id;
  if (remotes.some((r) => r.remote === remote)) return undefined;
  if (defaultRemote(remotes)?.repository === repo.repository) return legacy.harness_project_id;
  return undefined;
};

export const projectCheckout = (computer: Computer, checkouts: Checkout[], remotes: ProjectRemote[], repo: ProjectRemote | undefined): ProjectCheckout => {
  const projects = computer.facts?.projects;
  if (!projects) return { state: "unknown" };
  const pick = pickedId(computer.id, checkouts, remotes, repo, projects);
  const picked = projects.find((p) => p.id === pick);
  if (picked) return { state: "has", project: picked, picked: true };
  const matches = projects.filter((p) => !!repo && !!p.repository && p.repository.toLowerCase() === repo.remote);
  if (matches.length === 1) return { state: "has", project: matches[0]!, picked: false };
  if (matches.length > 1) return { state: "ambiguous", projects: matches };
  return { state: "missing" };
};

// One repository's checkout on a computer; repository is "" for a project with none.
export interface RepoCheckout {
  repository: string;
  checkout: ProjectCheckout;
}

// The checkout a run in each of the project's repositories would use on computer; one row for a project with one or none.
export const repoCheckouts = (computer: Computer, checkouts: Checkout[], remotes: ProjectRemote[]): RepoCheckout[] => {
  if (remotes.length === 0) return [{ repository: "", checkout: projectCheckout(computer, checkouts, remotes, undefined) }];
  return remotes.map((r) => ({ repository: r.repository, checkout: projectCheckout(computer, checkouts, remotes, r) }));
};

// The checkouts with the one picked for repository on computerId set to harnessProjectId, the others kept.
export const withCheckout = (checkouts: Checkout[], computerId: string, repository: string, harnessProjectId: string): Checkout[] => [
  ...checkouts.filter((c) => c.computer_id !== computerId || c.repository !== repository),
  { computer_id: computerId, repository, harness_project_id: harnessProjectId },
];
