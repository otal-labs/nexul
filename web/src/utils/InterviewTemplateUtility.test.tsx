// @vitest-environment node
import { describe, expect, it } from "vitest";

import { joinTemplate, readQuestion, splitTemplate } from "@/utils/InterviewTemplateUtility";

const body = "## What is it?\nOne or two sentences.\n\n## Which tests?\nSkip it when none.\n- [ ] Unit\n- [ ] End-to-end: in a browser\n\n## How does it merge?\n- Squash\n- Rebase\n";

describe("splitTemplate", () => {
  it("gives the body back byte for byte when joined", () => {
    expect(joinTemplate(splitTemplate(body))).toBe(body);
    expect(splitTemplate(body)).toHaveLength(3);
  });

  it("keeps text before the first heading as its own part", () => {
    expect(splitTemplate("stray\n## Q")).toEqual(["stray", "## Q"]);
  });
});

describe("readQuestion", () => {
  it("reads the title, the hint up to the first bullet, and the option labels", () => {
    const [, tests, merge] = splitTemplate(body).map(readQuestion);
    expect(tests).toEqual({ title: "Which tests?", hint: "Skip it when none.", multiSelect: true, options: ["Unit", "End-to-end"] });
    expect(merge).toEqual({ title: "How does it merge?", hint: "", multiSelect: false, options: ["Squash", "Rebase"] });
  });

  it("reads nothing from a part without a heading", () => {
    expect(readQuestion("stray")).toBeNull();
  });
});
