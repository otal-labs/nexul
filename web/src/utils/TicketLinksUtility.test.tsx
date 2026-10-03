import { describe, expect, it } from "vitest";

import type { PRLink } from "@/models/Ticket";
import { groupLinksByRepo } from "@/utils/TicketLinksUtility";

const pr = (owner: string, repo: string, number: number): PRLink => ({
  owner,
  repo,
  number,
  title: "t",
  sha: "s",
  state: "open",
});

describe("groupLinksByRepo", () => {
  it("groups branches and prs under one entry per repo, ignoring case", () => {
    const groups = groupLinksByRepo({
      branches: [{ owner: "acme", repo: "app", branch: "feat" }],
      prs: [pr("Acme", "App", 1), pr("acme", "api", 2), pr("acme", "app", 3)],
    });
    expect(groups.map((g) => [g.owner, g.repo, g.branches.length, g.prs.map((p) => p.number)])).toEqual([
      ["acme", "app", 1, [1, 3]],
      ["acme", "api", 0, [2]],
    ]);
  });
});
