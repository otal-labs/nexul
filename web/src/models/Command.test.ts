// @vitest-environment node
import { describe, expect, it } from "vitest";

import { arrangeCommandGroups, IN_TEXT, type CommandGroup, type CommandItem } from "@/models/Command";

const row = (label: string, extra: Partial<CommandItem> = {}): CommandItem => ({ id: label, label, icon: () => null, run: () => {}, ...extra });
const labels = (groups: CommandGroup[]) => groups.map((group) => [group.heading, group.items.map((item) => item.label)]);

describe("arrangeCommandGroups", () => {
  const found: CommandGroup[] = [
    { heading: "Tickets", items: [row("Theme picker crashes")] },
    { heading: IN_TEXT, items: [row("Release notes")] },
  ];

  it("leads with at most three local rows whose label starts with a typed word, and puts every other match before In text", () => {
    const local: CommandGroup[] = [
      { heading: "Theme", items: [row("Nexul", { keywords: "theme palette" }), row("Ember", { keywords: "theme palette" })] },
      { heading: "Go to", items: [row("Themes gallery"), row("Theme settings"), row("Theme history"), row("Theme export")] },
    ];

    expect(labels(arrangeCommandGroups(local, found, "theme"))).toEqual([
      ["Top matches", ["Themes gallery", "Theme settings", "Theme history"]],
      ["Tickets", ["Theme picker crashes"]],
      ["Other matches", ["Theme export", "Nexul", "Ember"]],
      [IN_TEXT, ["Release notes"]],
    ]);
  });

  it("keeps browsing groups in their order and drops empty ones", () => {
    const local: CommandGroup[] = [{ heading: "Create", items: [] }, { heading: "Go to", items: [row("Inbox")] }];

    expect(labels(arrangeCommandGroups(local, [{ heading: "Recently updated", items: [row("Doc")] }], " "))).toEqual([
      ["Recently updated", ["Doc"]],
      ["Go to", ["Inbox"]],
    ]);
  });
});
