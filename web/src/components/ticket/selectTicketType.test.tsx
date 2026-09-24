import { describe, expect, it } from "vitest";

import { bodyForType } from "@/components/ticket/selectTicketType";
import type { TicketType } from "@/models/TicketType";

const type = (id: string, body_template: string): TicketType => ({
  id,
  name: id,
  position: 0,
  color: "",
  body_template,
  created_at: "",
  updated_at: "",
});

const types = [type("task", "## What needs doing\n\n"), type("bug", "## Steps to reproduce\n\n"), type("chore", "")];

describe("bodyForType", () => {
  it.each([
    ["an empty body takes the chosen type's template", "", "bug", "## Steps to reproduce\n\n"],
    ["a blank body takes the chosen type's template", "  \n", "bug", "## Steps to reproduce\n\n"],
    ["an unedited template swaps to the new type's", "## What needs doing\n\n", "bug", "## Steps to reproduce\n\n"],
    ["an unedited template clears for a type without one", "## What needs doing\n\n", "chore", ""],
    ["typed text is never replaced", "## What needs doing\n\nship it", "bug", "## What needs doing\n\nship it"],
    ["an unknown type leaves an empty body", "", "missing", ""],
  ])("%s", (_name, body, typeId, want) => {
    expect(bodyForType(body, types, typeId)).toBe(want);
  });
});
