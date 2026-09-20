import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MembersPage } from "@/pages/MembersPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), open: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: vi.fn(), patch: vi.fn(), delete: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/useFormDialog", () => ({ useFormDialog: () => ({ open: mocks.open }) }));

describe("MembersPage", () => {
  beforeEach(() => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: [{ id: "r-1", workspace_id: "ws-1", name: "Editor", permissions: [], is_owner_role: false, created_at: "", updated_at: "" }] });
      if (url === "/api/workspaces/ws-1/members") return Promise.resolve({ data: { members: [] } });
      if (url === "/api/invitations") return Promise.resolve({ data: [] });
      if (url === "/api/workspaces") return Promise.resolve({ data: [{ id: "ws-1", name: "Engineering", created_at: "", updated_at: "" }] });
      if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: [] } });
      return Promise.reject(new Error(`unexpected GET ${url}`));
    });
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.open.mockResolvedValue({ success: false, data: null });
  });

  it("offers link invitations and no identifier input", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={client}><MemoryRouter><MembersPage /></MemoryRouter></QueryClientProvider>);
    expect(await screen.findByRole("button", { name: "Invite" })).toBeInTheDocument();
    expect(screen.queryByLabelText(/github username/i)).not.toBeInTheDocument();
  });

  it("preselects the current workspace when opening an invitation", async () => {
    const user = (await import("@testing-library/user-event")).default.setup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={client}><MemoryRouter><MembersPage /></MemoryRouter></QueryClientProvider>);
    await user.click(await screen.findByRole("button", { name: "Invite" }));
    expect(mocks.open).toHaveBeenCalledWith(expect.objectContaining({ formOptions: expect.objectContaining({ defaultValues: expect.objectContaining({ grants: [expect.objectContaining({ workspace_id: "ws-1" })] }) }) }));
  });
});
