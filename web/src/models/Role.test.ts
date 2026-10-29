import { describe, expect, it } from "vitest";

import { headName } from "@/models/Role";

describe("headName", () => {
  it.each([
    ["keeps a name that fits", "Finance & billing reviewers", "Finance & billing reviewers"],
    ["keeps a short name", "QA", "QA"],
    ["cuts at a whole word that leaves room for its ellipsis, never on a joining word", "External contractors and freelance developers (night shift)", "External…"],
    ["drops a trailing comma before the ellipsis", "Platform, infra, reliability and on-call rotation", "Platform, infra…"],
    ["cuts one unbreakable word by characters", "Supercalifragilisticexpialidociousunbreakable", "Supercalifragilisticexpialidocio…"],
    ["leaves room for the ellipsis on a wider head", "External contractors and freelance developers (night shift)", "External contractors…", 13],
  ])("%s", (_, name, want, width = 11) => {
    expect(headName(name, width)).toBe(want);
  });
});
