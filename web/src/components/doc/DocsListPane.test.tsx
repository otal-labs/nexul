import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocsListPane } from "@/components/doc/DocsListPane";
import type { DocFolderGroup } from "@/components/doc/docGroups";
import type { DocListItem } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";
import type { Project } from "@/models/Project";
import { useDocSortStore } from "@/stores/docSortStore";

const main: DocFolder = { id: "f-main", project_id: "project-1", name: "Main", is_default: true, created_at: "", updated_at: "" };

vi.mock("@/hooks/useCreateDocDialog", () => ({ useCreateDocDialog: () => undefined }));
vi.mock("@/components/doc/NewDocFolderButton", () => ({ NewDocFolderButton: () => null }));
vi.mock("@/hooks/DocFolderHooks", () => ({ useFetchDocFolders: () => ({ data: [main], error: null, isPending: false }) }));
vi.mock("@/components/doc/DocFolderSection", () => ({
  DocFolderSection: ({ group, forceOpen }: { group: DocFolderGroup; forceOpen: boolean }) => (
    <ul aria-label={`${group.folder.name}${forceOpen ? " (open)" : ""}`}>
      {group.docs.map((doc) => (
        <li key={doc.id}>{doc.id}</li>
      ))}
    </ul>
  ),
}));

const doc = (id: string, created: string, updated: string): DocListItem => ({
  id,
  project_id: "project-1",
  folder_id: "f-main",
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

describe("DocsListPane", () => {
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

  it("opens every matching folder while a search runs, and says when nothing matches", async () => {
    render(<DocsListPane docs={docs} project={{ id: "project-1" } as Project} selectedId={undefined} />);
    await userEvent.type(screen.getByRole("textbox", { name: "Search docs" }), "ep06");
    expect(screen.getByRole("list", { name: "Main (open)" })).toBeInTheDocument();
    expect(listed()).toEqual(["EP06"]);

    await userEvent.type(screen.getByRole("textbox", { name: "Search docs" }), "x");
    expect(screen.getByText("Nothing matches")).toBeInTheDocument();
  });
});
