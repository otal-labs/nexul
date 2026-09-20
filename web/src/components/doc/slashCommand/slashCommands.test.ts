import { describe, expect, it } from "vitest";

import { filterSlashCommands, slashCommandItems } from "@/components/doc/slashCommand/slashCommands";

describe("filterSlashCommands", () => {
  it("returns everything for an empty query", () => {
    expect(filterSlashCommands("")).toEqual(slashCommandItems);
  });

  it("matches by label", () => {
    expect(filterSlashCommands("heading").map((i) => i.id)).toEqual(["h1", "h2", "h3"]);
  });

  it("matches by keyword even when the label differs", () => {
    expect(filterSlashCommands("ul").map((i) => i.id)).toEqual(["bulletList"]);
  });

  it("offers Image only when the editor can upload", () => {
    expect(filterSlashCommands("image").map((i) => i.id)).toEqual(["image"]);
    expect(filterSlashCommands("image", false)).toEqual([]);
    expect(filterSlashCommands("", false).some((i) => i.id === "image")).toBe(false);
  });

  it("returns nothing for an unmatched query", () => {
    expect(filterSlashCommands("zzz")).toEqual([]);
  });
});
