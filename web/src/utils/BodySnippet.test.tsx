// @vitest-environment node
import { describe, expect, it } from "vitest";

import { bodySnippet } from "@/utils/BodySnippet";

const item = (text: string) => ({ type: "listItem", content: [{ type: "paragraph", content: [{ type: "text", text }] }] });

describe("bodySnippet", () => {
  it("keeps a list's items apart", () => {
    const body = JSON.stringify({ type: "doc", content: [{ type: "orderedList", content: [item("Roll back."), item("Post it.")] }] });
    expect(bodySnippet(body)).toBe("Roll back. Post it.");
  });
});
