import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { RowActions } from "@/components/listpane/RowActions";
import { useDocRowActions } from "@/hooks/useDocRowActions";
import type { DocListItem } from "@/models/Doc";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const doc = (archived: boolean): DocListItem => ({
  id: "d-1",
  project_id: "p-1",
  folder_id: "f-1",
  title: "Runbook",
  version: 1,
  archived,
  locked: false,
  can_open: true,
  created_at: "",
  updated_at: "",
});

const Row = ({ item }: { item: DocListItem }) => <RowActions itemLabel={item.title} {...useDocRowActions(item, false)} />;

const renderRow = (item: DocListItem, permissions: string[]) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    return { data: [] };
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Row item={item} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("a doc row's archive actions", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    vi.mocked(api.post).mockReset().mockResolvedValue({ data: doc(false) });
  });

  it("restores an archived doc from its menu", async () => {
    const user = userEvent.setup();
    renderRow(doc(true), ["docs:write"]);

    await user.click(screen.getByRole("button", { name: "More actions for Runbook" }));
    await user.click(await screen.findByRole("menuitem", { name: "Restore" }));

    expect(api.post).toHaveBeenCalledWith("/api/docs/d-1/restore");
    expect(screen.queryByRole("menuitem", { name: "Archive" })).not.toBeInTheDocument();
  });

  it("archives a live doc from its menu", async () => {
    const user = userEvent.setup();
    renderRow(doc(false), ["docs:write"]);

    await user.click(screen.getByRole("button", { name: "More actions for Runbook" }));
    await user.click(await screen.findByRole("menuitem", { name: "Archive" }));

    expect(api.post).toHaveBeenCalledWith("/api/docs/d-1/archive");
  });

  it("offers neither to someone who can't write docs", async () => {
    const user = userEvent.setup();
    renderRow(doc(true), ["docs:read"]);

    await user.click(screen.getByRole("button", { name: "More actions for Runbook" }));
    await screen.findByRole("menuitem", { name: "Pin" });

    expect(screen.queryByRole("menuitem", { name: "Restore" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Archive" })).not.toBeInTheDocument();
  });
});
