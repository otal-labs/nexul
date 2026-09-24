import { describe, expect, it } from "vitest";

import { acceptanceCriteria } from "@/utils/AcceptanceCriteriaUtility";

describe("acceptanceCriteria", () => {
  it("takes the section up to the next heading of the same level", () => {
    const body = "## Why\n\nSlow.\n\n## Acceptance criteria\n\n- [ ] Loads fast\n\n### Detail\n\nUnder 1s\n\n## Out of scope\n\nMobile";
    expect(acceptanceCriteria(body)).toBe("- [ ] Loads fast\n\n### Detail\n\nUnder 1s");
  });

  it("runs to the end when no heading follows, matching the heading case-insensitively", () => {
    expect(acceptanceCriteria("# ACCEPTANCE CRITERIA #\nWorks")).toBe("Works");
  });

  it("is empty without the section", () => {
    expect(acceptanceCriteria("## What needs doing\n\nStuff")).toBe("");
  });

  it("reads a rich-text body", () => {
    const doc = {
      type: "doc",
      content: [
        { type: "heading", attrs: { level: 2 }, content: [{ type: "text", text: "Acceptance criteria" }] },
        { type: "paragraph", content: [{ type: "text", text: "Login works" }] },
      ],
    };
    expect(acceptanceCriteria(JSON.stringify(doc))).toBe("Login works");
  });
});
