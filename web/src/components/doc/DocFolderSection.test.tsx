import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocFolderSection } from "@/components/doc/DocFolderSection";
import type { DocFolderGroup } from "@/components/doc/docGroups";
import type { DocListItem } from "@/models/Doc";
import { useDocFolderStore } from "@/stores/docFolderStore";

vi.mock("@/hooks/useDocFolderActions", () => ({ useDocFolderActions: () => ({}) }));
vi.mock("@/hooks/useCanManageAccess", () => ({ useCanManageAccess: () => false }));
vi.mock("@/components/doc/DocListRow", () => ({ DocListRow: ({ doc }: { doc: DocListItem }) => <li>{doc.title}</li> }));

const group = (projectId: string): DocFolderGroup => ({
  folder: { id: "f-gs", project_id: projectId, parent_id: "", name: "GetSource", is_default: false, created_at: "", updated_at: "" },
  docs: [{ id: "d-1", title: "EP01", folder_id: "f-gs" } as DocListItem],
  total: 1,
  folders: [],
});

const savedCollapsed = () => JSON.parse(localStorage.getItem("doc-folders-collapsed") ?? "{}").state?.collapsed;

describe("DocFolderSection", () => {
  beforeEach(() => {
    localStorage.clear();
    useDocFolderStore.setState({ collapsed: {} });
  });

  it("collapses on its label, kept per project in the browser, and a search shows its docs anyway", async () => {
    const { rerender } = render(<DocFolderSection group={group("p-1")} projectToken="p" selectedId={undefined} forceOpen={false} />);
    const toggle = screen.getByRole("button", { name: /GetSource/ });
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    await userEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("EP01")).not.toBeInTheDocument();
    expect(savedCollapsed()).toEqual({ "p-1": ["f-gs"] });

    rerender(<DocFolderSection group={group("p-2")} projectToken="p" selectedId={undefined} forceOpen={false} />);
    expect(screen.getByText("EP01")).toBeInTheDocument();

    rerender(<DocFolderSection group={group("p-1")} projectToken="p" selectedId={undefined} forceOpen />);
    expect(screen.getByText("EP01")).toBeInTheDocument();
  });
});

describe("DocFolderSection inside another", () => {
  beforeEach(() => {
    localStorage.clear();
    useDocFolderStore.setState({ collapsed: {} });
  });

  it("shows the folders inside before its docs, and collapsing it hides both", async () => {
    const inner: DocFolderGroup = {
      folder: { id: "f-acme", project_id: "p-1", parent_id: "f-gs", name: "Acme", is_default: false, created_at: "", updated_at: "" },
      docs: [{ id: "d-2", title: "Brief", folder_id: "f-acme" } as DocListItem],
      total: 1,
      folders: [],
    };
    render(<DocFolderSection group={{ ...group("p-1"), total: 2, folders: [inner] }} projectToken="p" selectedId={undefined} forceOpen={false} />);
    const acme = screen.getByRole("region", { name: "Acme" });
    expect(acme.compareDocumentPosition(screen.getByText("EP01")) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByText("Brief")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /GetSource/ }));
    expect(screen.queryByRole("region", { name: "Acme" })).not.toBeInTheDocument();
    expect(screen.queryByText("Brief")).not.toBeInTheDocument();
  });
});
