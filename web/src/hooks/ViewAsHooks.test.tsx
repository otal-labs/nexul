import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { InternalAxiosRequestConfig } from "axios";
import type { ReactNode } from "react";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { api } from "@/api/client";
import { useLogout } from "@/hooks/AuthHooks";
import { useStartViewAs } from "@/hooks/ViewAsHooks";
import { useSessionStore } from "@/stores/sessionStore";
import { useViewAsStore } from "@/stores/viewAsStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const alice = { workspaceId: "ws-1", workspaceName: "Acme", userId: "u-alice", name: "Alice", avatarUrl: "" };

const sent: InternalAxiosRequestConfig[] = [];
const realAdapter = api.defaults.adapter ?? [];

const answer = (config: InternalAxiosRequestConfig) => {
  sent.push(config);
  const data = config.url === "/api/workspaces" ? [{ id: "ws-1", slug: "acme", name: "Acme" }] : { workspace_id: "ws-1", user_id: "u-alice", name: "Alice", avatar_url: "" };
  return Promise.resolve({ data, status: 200, statusText: "OK", headers: {}, config });
};

const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={client}>
    <MemoryRouter initialEntries={["/settings/team"]}>{children}</MemoryRouter>
  </QueryClientProvider>
);

describe("view as hooks", () => {
  beforeEach(() => {
    sent.length = 0;
    client.clear();
    api.defaults.adapter = answer;
    useSessionStore.getState().login("tok-1");
    useViewAsStore.getState().exit();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-9", selectedWorkspaceSlug: "other" });
  });

  afterEach(() => {
    api.defaults.adapter = realAdapter;
  });

  it("starts a view: records the target, selects its workspace, drops the viewer's cached reads and opens the workspace home", async () => {
    client.setQueryData(["getTeam"], { people: [] });
    const { result } = renderHook(() => ({ start: useStartViewAs(), at: useLocation().pathname }), { wrapper });
    await waitFor(() => expect(sent.some((c) => c.url === "/api/workspaces")).toBe(true));

    await act(() => result.current.start.mutateAsync({ workspaceId: "ws-1", workspaceName: "Acme", userId: "u-alice" }));

    expect(sent.find((c) => c.method === "post")?.url).toBe("/api/workspaces/ws-1/members/u-alice/view-as");
    expect(useViewAsStore.getState().target).toEqual(alice);
    expect(useWorkspaceStore.getState()).toMatchObject({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
    expect(client.getQueryData(["getTeam"])).toBeUndefined();
    expect(result.current.at).toBe("/acme");
  });

  it("signs out as the viewer, not as the person being viewed", async () => {
    useViewAsStore.getState().start(alice);
    const { result } = renderHook(() => useLogout(), { wrapper });

    await act(() => result.current.mutateAsync());

    const request = sent.find((c) => c.method === "delete");
    expect(request?.url).toBe("/api/auth/sessions/current");
    expect(request?.headers.has("X-View-As-User")).toBe(false);
    expect(useViewAsStore.getState().target).toBeNull();
    expect(useSessionStore.getState().isLoggedIn).toBe(false);
  });
});
