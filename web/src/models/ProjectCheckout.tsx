import type { Checkout, Computer } from "@/models/Pairing";
import type { RepoRef } from "@/models/Project";

type FactProject = NonNullable<Computer["facts"]>["projects"][number];

// Where a project's runs would land on one computer, as the server decides it (ADR 0165): the checkout picked there,
// else the one whose git remote is a repository of the project; several are a pick, none is missing.
export type ProjectCheckout =
  | { state: "has"; project: FactProject; picked: boolean }
  | { state: "ambiguous"; projects: FactProject[] }
  | { state: "missing" }
  | { state: "unknown" };

// A project's GitHub repositories as T3 names a checkout's remote.
export const projectRemotes = (repos: RepoRef[]): string[] =>
  repos.filter((r) => r.connector_id === "github").map((r) => `github.com/${r.full_name.toLowerCase()}`);

export const projectCheckout = (computer: Computer, checkouts: Checkout[], remotes: string[]): ProjectCheckout => {
  const projects = computer.facts?.projects;
  if (!projects) return { state: "unknown" };
  const pickedId = checkouts.find((c) => c.computer_id === computer.id)?.harness_project_id;
  const picked = projects.find((p) => p.id === pickedId);
  if (picked) return { state: "has", project: picked, picked: true };
  const matches = projects.filter((p) => !!p.repository && remotes.includes(p.repository.toLowerCase()));
  if (matches.length === 1) return { state: "has", project: matches[0]!, picked: false };
  if (matches.length > 1) return { state: "ambiguous", projects: matches };
  return { state: "missing" };
};

// The checkouts with computerId's set to harnessProjectId, replacing what was picked there before.
export const withCheckout = (checkouts: Checkout[], computerId: string, harnessProjectId: string): Checkout[] => [
  ...checkouts.filter((c) => c.computer_id !== computerId),
  { computer_id: computerId, harness_project_id: harnessProjectId },
];
