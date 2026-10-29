import { useState, type ComponentProps } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocsListPane } from "@/components/doc/DocsListPane";
import type { DocFolderGroup } from "@/components/doc/docGroups";
import type { DocListItem } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";
import type { Project } from "@/models/Project";
import { useDocFolderStore } from "@/stores/docFolderStore";
import { useDocArchiveStore } from "@/stores/docArchiveStore";
import { useDocSortStore } from "@/stores/docSortStore";

const main: DocFolder = { id: "f-main", project_id: "project-1", name: "Main", is_default: true, created_at: "", updated_at: "" };

vi.mock("@/hooks/useWorkspacePath", () => ({ useWorkspacePath: () => (path: string) => path }));
vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("@/hooks/useCreateDocDialog", () => ({ useOpenCreateDoc: () => vi.fn() }));
vi.mock("@/hooks/TrailHooks", () => ({ useDocRunStates: () => [] }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);
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

const doc = (id: string, created: string, updated: string, archived = false): DocListItem => ({
  id,
  project_id: "project-1",
  folder_id: "f-main",
  title: id,
  version: 1,
  archived,
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

// The page owns the search, which it shares with the overview.
const Pane = (props: Omit<ComponentProps<typeof DocsListPane>, "search" | "onSearch">) => {
  const [search, setSearch] = useState("");
  return <DocsListPane {...props} search={search} onSearch={setSearch} />;
};

const pick = async (item: "Last edited" | "Created" | "Show archived") => {
  await userEvent.click(screen.getByRole("button", { name: /^Sort and view/ }));
  await userEvent.click(screen.getByRole(item === "Show archived" ? "menuitemcheckbox" : "menuitemradio", { name: item }));
};

const listed = () => screen.getAllByRole("listitem").map((li) => li.textContent);

describe("DocsListPane", () => {
  beforeEach(() => {
    useDocSortStore.setState({ sortBy: "created_at" });
    useDocArchiveStore.setState({ showArchived: false });
  });

  it("lists by creation by default and by last edit once toggled", async () => {
    render(<Pane docs={docs} project={{ id: "project-1" } as Project} selectedId={undefined} />);
    expect(listed()).toEqual(["EP07", "EP06", "EP01"]);

    await pick("Last edited");
    expect(listed()).toEqual(["EP01", "EP07", "EP06"]);

    await pick("Created");
    expect(listed()).toEqual(["EP07", "EP06", "EP01"]);
  });

  it("opens every matching folder while a search runs, and says when nothing matches", async () => {
    render(<Pane docs={docs} project={{ id: "project-1" } as Project} selectedId={undefined} />);
    await userEvent.type(screen.getByRole("textbox", { name: "Search docs" }), "ep06");
    expect(screen.getByRole("list", { name: "Main (open)" })).toBeInTheDocument();
    expect(listed()).toEqual(["EP06"]);

    await userEvent.type(screen.getByRole("textbox", { name: "Search docs" }), "x");
    expect(screen.getByText("Nothing matches")).toBeInTheDocument();
  });

  it("expands the folder of the doc being opened, and lets it collapse again", () => {
    useDocFolderStore.setState({ collapsed: { "project-1": ["f-main"] } });
    const { rerender } = render(<Pane docs={docs} project={{ id: "project-1" } as Project} selectedId="EP06" />);
    expect(useDocFolderStore.getState().collapsed["project-1"]).toEqual([]);

    useDocFolderStore.getState().toggleCollapsed("project-1", "f-main");
    rerender(<Pane docs={docs} project={{ id: "project-1" } as Project} selectedId="EP06" />);
    expect(useDocFolderStore.getState().collapsed["project-1"]).toEqual(["f-main"]);
  });

  it("hides archived docs until the Archived filter is on, then lists only them", async () => {
    const withArchived = [...docs, doc("EP02", "2026-01-02T09:00:00Z", "2026-01-02T09:00:00Z", true)];
    render(<Pane docs={withArchived} project={{ id: "project-1" } as Project} selectedId={undefined} />);
    expect(listed()).toEqual(["EP07", "EP06", "EP01"]);

    await pick("Show archived");
    expect(listed()).toEqual(["EP02"]);

    await pick("Show archived");
    expect(listed()).toEqual(["EP07", "EP06", "EP01"]);
  });

  it("says when nothing is archived instead of listing empty folders", async () => {
    render(<Pane docs={docs} project={{ id: "project-1" } as Project} selectedId={undefined} />);

    await pick("Show archived");

    expect(screen.getByText("No archived docs")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Main" })).not.toBeInTheDocument();
  });
});
