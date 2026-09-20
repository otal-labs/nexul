import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { DocsFeed } from "@/components/doc/DocsFeed";
import type { DocListItem } from "@/models/Doc";

const docs: DocListItem[] = [
  { id: "doc-1", project_id: "p-1", title: "Storage Spine", version: 1, archived: false, can_open: true, updated_at: "2026-08-02T12:00:00Z" },
  { id: "doc-2", project_id: "p-1", title: "Event Bus", version: 3, archived: false, can_open: true, updated_at: "2026-08-02T12:00:00Z" },
  { id: "doc-3", project_id: "p-1", title: "Restricted", version: 1, archived: false, can_open: false, updated_at: "2026-08-02T12:00:00Z" },
];

const baseProps = {
  docs,
  selected: [] as string[],
  onToggleSelect: () => {},
  onSelect: () => {},
  onCreate: () => {},
  onPermissions: () => {},
};

describe("DocsFeed", () => {
  it("lists docs with their versions", () => {
    render(<DocsFeed {...baseProps} />);
    expect(screen.getByText("Storage Spine")).toBeInTheDocument();
    expect(screen.getByText("Event Bus")).toBeInTheDocument();
    expect(screen.getByText(/^v3 · updated/)).toBeInTheDocument();
  });

  it("shows an archived chip on archived docs", () => {
    const archivedDocs: DocListItem[] = [{ ...docs[0]!, archived: true }];
    render(<DocsFeed {...baseProps} docs={archivedDocs} />);
    expect(screen.getByText("archived")).toBeInTheDocument();
  });

  it("shows an empty state", () => {
    render(<DocsFeed {...baseProps} docs={[]} />);
    expect(screen.getByText("No docs yet.")).toBeInTheDocument();
  });

  it("selects an openable doc and opens the create dialog", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const onCreate = vi.fn();
    render(<DocsFeed {...baseProps} onSelect={onSelect} onCreate={onCreate} />);

    await user.click(screen.getByText("Storage Spine"));
    expect(onSelect).toHaveBeenCalledWith("doc-1");

    await user.click(screen.getByRole("button", { name: "New doc" }));
    expect(onCreate).toHaveBeenCalled();
  });

  it("does not open a restricted doc", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<DocsFeed {...baseProps} onSelect={onSelect} />);

    await user.click(screen.getByText("Restricted"));
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("shows the permissions button only when docs are selected", async () => {
    const user = userEvent.setup();
    const onToggleSelect = vi.fn();
    render(<DocsFeed {...baseProps} onToggleSelect={onToggleSelect} />);
    expect(screen.queryByRole("button", { name: /Permissions/ })).not.toBeInTheDocument();

    await user.click(screen.getByLabelText("Select Storage Spine"));
    expect(onToggleSelect).toHaveBeenCalledWith("doc-1");
  });

  it("renders a permissions button when docs are selected", async () => {
    const user = userEvent.setup();
    const onPermissions = vi.fn();
    render(<DocsFeed {...baseProps} selected={["doc-1"]} onPermissions={onPermissions} />);

    const button = screen.getByRole("button", { name: "Permissions (1)" });
    await user.click(button);
    expect(onPermissions).toHaveBeenCalled();
  });

  it("filters rows by title via the search field", async () => {
    const user = userEvent.setup();
    render(<DocsFeed {...baseProps} />);

    expect(screen.getByText("Storage Spine")).toBeInTheDocument();
    expect(screen.getByText("Event Bus")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Search docs"), "event");

    expect(screen.queryByText("Storage Spine")).not.toBeInTheDocument();
    expect(screen.getByText("Event Bus")).toBeInTheDocument();
  });

  it("shows a search-specific empty state when no doc matches the query", async () => {
    const user = userEvent.setup();
    render(<DocsFeed {...baseProps} />);

    await user.type(screen.getByLabelText("Search docs"), "nonexistent-doc-title");

    expect(await screen.findByText(/No docs match/)).toBeInTheDocument();
  });
});
