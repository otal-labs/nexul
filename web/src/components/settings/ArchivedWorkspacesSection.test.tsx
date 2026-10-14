import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ArchivedWorkspacesSection } from "@/components/settings/ArchivedWorkspacesSection";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const archived = [
  { id: "ws-globex", slug: "globex", name: "Globex", archived_at: "2026-10-01T09:00:00Z" },
  { id: "ws-northwind", slug: "northwind", name: "Northwind", archived_at: "2026-10-02T09:00:00Z" },
];

const renderSection = (instancePermissions: string[], list: unknown[] = archived) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/me") return Promise.resolve({ data: { user: { id: "u-1" }, instance_permissions: instancePermissions } });
    if (url === "/api/workspaces/archived") return Promise.resolve({ data: list });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
  mocks.post.mockResolvedValue({ data: archived[0] });
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ArchivedWorkspacesSection />
    </QueryClientProvider>,
  );
};

describe("ArchivedWorkspacesSection", () => {
  beforeEach(() => vi.clearAllMocks());

  it("lists what the instance Owner can restore and restores the one picked", async () => {
    const user = userEvent.setup();
    renderSection(["instance:write"]);

    expect(await screen.findByText("Globex")).toBeInTheDocument();
    expect(screen.getByText("Northwind")).toBeInTheDocument();
    await user.click(screen.getAllByRole("button", { name: "Restore" })[0]!);
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-globex/restore"));
  });

  it("says so when nothing is archived", async () => {
    renderSection(["instance:write"], []);
    expect(await screen.findByText("No workspace is archived.")).toBeInTheDocument();
  });

  it("shows no card and asks for no list to someone who can't restore", async () => {
    renderSection(["instance:read"]);

    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/auth/me"));
    expect(screen.queryByRole("heading", { name: "Archived workspaces" })).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/workspaces/archived");
  });
});
