import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { AppRouter } from "@/Router";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/hooks/useLiveEvents", () => ({ useLiveEvents: vi.fn() }));

vi.mock("@/api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api/client")>();
  return { ...actual, api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() } };
});

const user = {
  id: "u1",
  provider: "github",
  provider_user_id: "1",
  login: "onik97",
  name: "Onik",
  avatar_url: "",
  first_login_done: true,
  created_at: "",
};

const forbidden = Object.assign(new Error("forbidden"), { response: { status: 403, data: { message: "forbidden" } } });

const mockApi = (anywhere: string[], permissions: string[]) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/bootstrap-status") return { data: { configured: true } };
    if (url === "/api/auth/me") {
      return { data: { user, needs_owner_wizard: false, needs_first_login_wizard: false, instance_permissions: anywhere } };
    }
    if (url === "/api/workspaces") return { data: [{ id: "ws-1", name: "Acme", created_at: "", updated_at: "" }] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/team") throw forbidden;
    if (url === "/api/automations/a-1") throw forbidden;
    return { data: [] };
  });

const renderAt = (path: string) => {
  window.history.pushState({}, "", path);
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <AppRouter />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  localStorage.clear();
  vi.mocked(api.get).mockReset();
  useSessionStore.setState({ token: "t", isLoggedIn: true });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("route permissions", () => {
  it.each([
    { path: "/runners", anywhere: [], permissions: [], page: null },
    { path: "/runners", anywhere: [], permissions: ["runners:read"], page: "Runners" },
    { path: "/topology", anywhere: [], permissions: ["runners:read"], page: null },
    { path: "/automations", anywhere: [], permissions: ["runners:read"], page: null },
    { path: "/board", anywhere: [], permissions: ["chat:read"], page: null },
    { path: "/configuration", anywhere: [], permissions: ["chat:read"], page: null },
    { path: "/configuration", anywhere: [], permissions: ["roles:write"], page: "Configuration" },
    { path: "/configuration", anywhere: ["instance:read"], permissions: [], page: null },
    { path: "/configuration", anywhere: ["members:write"], permissions: [], page: "Configuration" },
    { path: "/settings", anywhere: ["connectors:read"], permissions: [], page: "Settings" },
    { path: "/inbox", anywhere: [], permissions: [], page: "Inbox" },
  ])("$path with $permissions (anywhere: $anywhere) opens $page", async ({ path, anywhere, permissions, page }) => {
    mockApi(anywhere, permissions);
    renderAt(path);

    if (page === null) {
      expect(await screen.findByRole("heading", { name: "Page not found" })).toBeInTheDocument();
      return;
    }
    expect(await screen.findByRole("heading", { level: 1, name: page })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Page not found" })).not.toBeInTheDocument();
  });

  it("a deep link to an item the server refuses lands on not-found", async () => {
    mockApi([], ["automations:read"]);
    renderAt("/automations/a-1");

    expect(await screen.findByRole("heading", { name: "Page not found" })).toBeInTheDocument();
  });
});
