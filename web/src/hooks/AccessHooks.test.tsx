import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useCanOpen, workspaceAccess } from "@/hooks/AccessHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

describe("workspaceAccess", () => {
  it("opens the Interview template in Configuration to whoever may edit it, with no project's memories:read", () => {
    const editor = workspaceAccess([], ["memories:template"]);
    expect(editor.configurationSections).toContain("interview");
    expect(editor.canOpen("configuration")).toBe(true);
    expect(workspaceAccess([], []).configurationSections).not.toContain("interview");
  });

  it("opens Access for whoever manages access or edits roles, and Configuration for a role editor not at all", () => {
    expect(workspaceAccess([], ["projects:access"]).canOpen("access")).toBe(true);
    expect(workspaceAccess([], ["roles:write"]).canOpen("access")).toBe(true);
    expect(workspaceAccess([], ["roles:write"]).canOpen("configuration")).toBe(false);
    expect(workspaceAccess([], ["tickets:read"]).canOpen("access")).toBe(false);
  });
});

describe("useCanOpen", () => {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider>
  );

  it("opens the project wizard to a creator in no project, even with a project from another workspace still selected", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/me") {
        return { data: { role_name: "Administrator", permissions: ["projects:create"], projects: [], new_project_actions: ["projects:read", "projects:write"] } };
      }
      return { data: { instance_permissions: [] } };
    });
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "p-elsewhere" });

    const { result } = renderHook(() => useCanOpen(), { wrapper });

    await waitFor(() => expect(result.current("wizard")).toBe(true));
  });
});
