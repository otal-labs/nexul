import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocsListPane } from "@/components/doc/DocsListPane";
import type { DocListItem } from "@/models/Doc";
import type { Project } from "@/models/Project";
import { useDocSortStore } from "@/stores/docSortStore";

vi.mock("@/hooks/useCreateDocDialog", () => ({ useCreateDocDialog: () => undefined }));
vi.mock("@/components/doc/DocGroupSection", () => ({
  DocGroupSection: ({ group }: { group: { docs: { id: string }[] } }) => (
    <ul>
      {group.docs.map((doc) => (
        <li key={doc.id}>{doc.id}</li>
      ))}
    </ul>
  ),
}));

const doc = (id: string, created: string, updated: string): DocListItem => ({
  id,
  project_id: "project-1",
  title: id,
  version: 1,
  archived: false,
  locked: false,
  can_open: true,
  created_at: created,
  updated_at: updated,
});

// EP01 is the oldest doc but was edited last.
const docs = [
  doc("EP01", "2026-01-01T09:00:00Z", "2026-01-09T09:00:00Z"),
  doc("EP06", "2026-01-03T09:00:00Z", "2026-01-03T09:00:00Z"),
  doc("EP07", "2026-01-04T09:00:00Z", "2026-01-04T09:00:00Z"),
];

const listed = () => screen.getAllByRole("listitem").map((li) => li.textContent);

describe("DocsListPane sort", () => {
  beforeEach(() => {
    useDocSortStore.setState({ sortBy: "created_at" });
  });

  it("lists by creation by default and by last edit once toggled", async () => {
    render(<DocsListPane docs={docs} project={{ id: "project-1" } as Project} selectedId={undefined} />);
    expect(listed()).toEqual(["EP07", "EP06", "EP01"]);

    await userEvent.click(screen.getByRole("radio", { name: "Sort by last edited" }));
    expect(listed()).toEqual(["EP01", "EP07", "EP06"]);

    await userEvent.click(screen.getByRole("radio", { name: "Sort by created" }));
    expect(listed()).toEqual(["EP07", "EP06", "EP01"]);
  });
});
