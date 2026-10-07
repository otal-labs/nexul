import { describe, expect, it } from "vitest";

import { embedFold, embedFoldLabel, moreEmbedsLabel, type Embed } from "@/models/Embed";

const fields = (n: number) => Array.from({ length: n }, (_, i) => ({ name: `f${i}`, value: "v" }));

describe("embedFold", () => {
  it.each<[string, Embed, string | null]>([
    ["six fields fold nothing", { fields: fields(6) }, null],
    ["a seventh field folds one", { fields: fields(7) }, "Show 1 more field"],
    ["25 fields fold 19", { fields: fields(25) }, "Show 19 more fields"],
    ["a 420-character description stays open", { description: "x".repeat(420) }, null],
    ["a 421-character description clamps", { description: "x".repeat(421) }, "Show the rest"],
    ["eight short lines stay open", { description: Array(8).fill("a").join("\n") }, null],
    ["nine short lines clamp", { description: Array(9).fill("a").join("\n") }, "Show the rest"],
    ["hidden fields name the count even with a long description", { fields: fields(8), description: "x".repeat(500) }, "Show 2 more fields"],
  ])("%s", (_name, embed, label) => {
    expect(embedFoldLabel(embedFold(embed))).toBe(label);
  });

  it("names the embeds past two", () => {
    expect(moreEmbedsLabel(1)).toBe("Show 1 more embed");
    expect(moreEmbedsLabel(8)).toBe("Show 8 more embeds");
  });
});
