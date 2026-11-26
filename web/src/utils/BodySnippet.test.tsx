// @vitest-environment node
import { describe, expect, it } from "vitest";

import { bodySnippet, readableSnippet } from "@/utils/BodySnippet";

const item = (text: string) => ({ type: "listItem", content: [{ type: "paragraph", content: [{ type: "text", text }] }] });

describe("bodySnippet", () => {
  it("keeps a list's items apart", () => {
    const body = JSON.stringify({ type: "doc", content: [{ type: "orderedList", content: [item("Roll back."), item("Post it.")] }] });
    expect(bodySnippet(body)).toBe("Roll back. Post it.");
  });
});

describe("readableSnippet", () => {
  it("drops record ids, whole or cut short, and the gaps they leave", () => {
    expect(readableSnippet("Once 01a15dce-bcf0-729d-ac90-7957103c3e91 tickets and 01a15dfe-685c-7c31-98be")).toBe("Once tickets and");
    expect(readableSnippet("See 01a15dce-bcf0-729d-ac90-7957103c3e91, then ship.")).toBe("See, then ship.");
  });
});
