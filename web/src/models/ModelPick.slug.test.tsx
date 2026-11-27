import { describe, expect, it } from "vitest";

import { slugName } from "@/models/ModelPick";

describe("slugName", () => {
  it.each([
    ["claude-opus-5-5", "Claude Opus 5.5"],
    ["gpt-5", "GPT 5"],
    ["grok-4-7-fast", "Grok 4.7 Fast"],
  ])("reads %s as %s, so a slug the computer can't name never shows as an id", (slug, name) => {
    expect(slugName(slug)).toBe(name);
  });
});
