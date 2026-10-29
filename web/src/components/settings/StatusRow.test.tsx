import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StatusRow } from "@/components/settings/StatusRow";
import { StatusKind, type BoardStatus } from "@/models/Status";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", async () => {
  const { asOwner } = await import("@/test/viewer");
  const { ticketCounts } = await import("@/test/ticketCounts");
  const counts = ticketCounts({ tickets_by_status: { open: 2 } });
  return { api: { get: vi.fn(asOwner((url) => (url.endsWith("/ticket-counts") ? counts : []))), put: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() }, errorMessage: vi.fn() };
});

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const status = (overrides: Partial<BoardStatus> = {}): BoardStatus => ({
  id: "open",
  name: "Open",
  position: 0,
  kind: StatusKind.Progress,
  icon: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const renderRow = (overrides: Partial<BoardStatus> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <StatusRow projectId="p-1" status={status(overrides)} index={0} total={1} onMove={vi.fn()} />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("StatusRow", () => {
  // The kind is stated once by the group header (BoardSettingsSection), so
  // the row carries it as a colored icon/dot only — never as a repeated
  // text badge.
  it("shows a progress stage in the info hue without repeating the kind as text", () => {
    const { container } = renderRow({ kind: StatusKind.Progress });
    expect(screen.queryByText("progress")).not.toBeInTheDocument();
    expect(container.querySelector(".bg-info")).toBeInTheDocument();
  });

  it("shows a done stage in the success hue without repeating the kind as text", () => {
    const { container } = renderRow({ kind: StatusKind.Done });
    expect(screen.queryByText("done")).not.toBeInTheDocument();
    expect(container.querySelector(".bg-success")).toBeInTheDocument();
  });

  it("collapses row actions into a single kebab menu", async () => {
    const user = userEvent.setup();
    renderRow();
    expect(screen.queryByRole("button", { name: /^Move status/ })).not.toBeInTheDocument();

    await user.click(await screen.findByRole("button", { name: "Actions for Open" }));
    expect(screen.getByRole("button", { name: "Rename" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Delete/ })).toBeInTheDocument();
  });

  it("counts the column's tickets and keeps Delete off while it holds any, saying why", async () => {
    const user = userEvent.setup();
    renderRow();

    expect(await screen.findByText("2 tickets")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Actions for Open" }));
    expect(screen.getByRole("button", { name: /^Delete\s*Used by 2 tickets$/ })).toBeDisabled();
  });
});
