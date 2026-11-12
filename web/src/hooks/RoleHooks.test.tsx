import { QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { roleFollower, useCreateWorkspaceRole } from "@/hooks/RoleHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("@/api/client", () => ({ api: { post: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

describe("the role follower", () => {
  it("refetches the roles of the workspace whose role changed, and no other", async () => {
    const client = seeded([
      [["getWorkspaceRoles", "ws-1"], []],
      [["getWorkspaceRoles", "ws-2"], []],
    ]);
    await followFrame(roleFollower, "role.updated", { role_id: "r-1", workspace_id: "ws-2" }, client);
    expect([isStale(client, ["getWorkspaceRoles", "ws-1"]), isStale(client, ["getWorkspaceRoles", "ws-2"])]).toEqual([false, true]);
  });
});

describe("useCreateWorkspaceRole", () => {
  it("refetches the workspace's access grid, whose columns are its roles", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { id: "r-2", name: "Sales" } });
    const client = seeded([
      [["getAccessGrid", "ws-1"], {}],
      [["getAccessGrid", "ws-2"], {}],
    ]);
    const { result } = renderHook(() => useCreateWorkspaceRole("ws-1"), {
      wrapper: ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ name: "Sales", description: "", actions: [], welcome_steps: [], show_locked_actions: false });
    expect([isStale(client, ["getAccessGrid", "ws-1"]), isStale(client, ["getAccessGrid", "ws-2"])]).toEqual([true, false]);
  });
});
