import { describe, expect, it } from "vitest";

import { bodyForType, offeredTicketTypes } from "@/components/ticket/selectTicketType";
import type { TicketType } from "@/models/TicketType";
import { emptyDocJson } from "@/utils/RichtextUtility";

const type = (id: string, body_template: string): TicketType => ({
  id,
  name: id,
  position: 0,
  color: "",
  body_template,
  created_at: "",
  updated_at: "",
});

// What the dialog's editor emits after loading the task template: key order and empty content differ from a parse.
const taskTemplateJson = `{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"What needs doing"}]},{"type":"paragraph"}]}`;
const imageOnlyJson = `{"type":"doc","content":[{"type":"image","attrs":{"src":"blob:staged-1","alt":"shot.png"}}]}`;

const types = [type("task", "## What needs doing\n\n"), type("bug", "## Steps to reproduce\n\n"), type("chore", "")];

describe("bodyForType", () => {
  it.each([
    ["an empty body takes the chosen type's template", "", "bug", "## Steps to reproduce\n\n"],
    ["a blank body takes the chosen type's template", "  \n", "bug", "## Steps to reproduce\n\n"],
    ["an unedited template swaps to the new type's", "## What needs doing\n\n", "bug", "## Steps to reproduce\n\n"],
    ["an unedited template clears for a type without one", "## What needs doing\n\n", "chore", ""],
    ["typed text is never replaced", "## What needs doing\n\nship it", "bug", "## What needs doing\n\nship it"],
    ["an unknown type leaves an empty body", "", "missing", ""],
    ["the editor's JSON of an unedited template swaps", taskTemplateJson, "bug", "## Steps to reproduce\n\n"],
    ["a body cleared in the editor takes the template", emptyDocJson, "bug", "## Steps to reproduce\n\n"],
    ["a body holding only a pasted image is kept", imageOnlyJson, "bug", imageOnlyJson],
  ])("%s", (_name, body, typeId, want) => {
    expect(bodyForType(body, types, typeId)).toBe(want);
  });
});

describe("offeredTicketTypes", () => {
  const withBug = [type("task", ""), type("Bug", ""), type("feature", "")];

  it("leaves the bug type out of the normal dialog", () => {
    expect(offeredTicketTypes(withBug, false).map((t) => t.id)).toEqual(["task", "feature"]);
  });

  it("offers only the bug type when reporting a bug", () => {
    expect(offeredTicketTypes(withBug, true).map((t) => t.id)).toEqual(["Bug"]);
  });

  it("offers every type to a bug report in a project with no bug type", () => {
    const renamed = [type("task", ""), type("defect", "")];
    expect(offeredTicketTypes(renamed, true)).toEqual(renamed);
    expect(offeredTicketTypes(renamed, false)).toEqual(renamed);
  });
});
