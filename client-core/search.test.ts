// @vitest-environment node
import { describe, expect, it } from "vitest";

import { groupSearchHits, searchCut, type SearchHit } from "@nexul/client-core/search";

const hit = (type: SearchHit["type"], id: string, in_body = false): SearchHit => ({ type, id, title: id, in_body, updated_at: "" });

describe("groupSearchHits", () => {
  it("keeps the kind order, puts text-only hits last and drops empty groups", () => {
    const groups = groupSearchHits([hit("channel", "c-1"), hit("ticket", "t-2", true), hit("ticket", "t-1"), hit("doc", "d-1", true)]);
    expect(groups.map((g) => [g.heading, g.hits.map((h) => h.id)])).toEqual([
      ["Tickets", ["t-1"]],
      ["Channels", ["c-1"]],
      ["In text", ["t-2", "d-1"]],
    ]);
  });

  it("leaves out the kinds an app cannot open", () => {
    const groups = groupSearchHits([hit("memory", "m-1"), hit("stack", "s-1")], ["stack"]);
    expect(groups.map((g) => g.heading)).toEqual(["Stacks"]);
  });
});

describe("searchCut", () => {
  it("is true only when a shown kind had more", () => {
    const found = { hits: [hit("ticket", "t-1")], more: ["memory" as const] };
    expect(searchCut(found)).toBe(true);
    expect(searchCut(found, ["ticket"])).toBe(false);
    expect(searchCut({ hits: [], more: ["ticket"] })).toBe(false);
    expect(searchCut(undefined)).toBe(false);
  });
});
