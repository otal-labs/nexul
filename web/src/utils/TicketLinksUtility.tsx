import type { BranchLink, PRLink, TicketLinks } from "@/models/Ticket";

export interface RepoLinks {
  owner: string;
  repo: string;
  branches: BranchLink[];
  prs: PRLink[];
}

// GitHub owner and repo names are case-insensitive, so Acme/App and acme/app share a group.
export const groupLinksByRepo = ({ prs, branches }: TicketLinks): RepoLinks[] => {
  const groups = new Map<string, RepoLinks>();
  const groupOf = (owner: string, repo: string) => {
    const key = `${owner}/${repo}`.toLowerCase();
    const existing = groups.get(key);
    if (existing) return existing;
    const group: RepoLinks = { owner, repo, branches: [], prs: [] };
    groups.set(key, group);
    return group;
  };
  for (const branch of branches) groupOf(branch.owner, branch.repo).branches.push(branch);
  for (const pr of prs) groupOf(pr.owner, pr.repo).prs.push(pr);
  return [...groups.values()];
};
