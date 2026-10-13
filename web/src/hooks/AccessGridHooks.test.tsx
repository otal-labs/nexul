import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { accessGridFollower, useAccessSetupState } from "@/hooks/AccessGridHooks";
import { chatFollower } from "@/hooks/ChatFollower";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "" }));

const setupState = async (me: object) => {
  mocks.get.mockResolvedValue({ data: me });
  const client = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  const { result } = renderHook(() => useAccessSetupState(), { wrapper });
  await waitFor(() => expect(mocks.get).toHaveBeenCalled());
  await waitFor(() => expect(client.isFetching()).toBe(0));
  return result.current;
};

const owner = { id: "r-o", name: "Owner", is_owner: true };
const dev = { id: "r-d", name: "Developer", is_owner: false };

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  mocks.get.mockReset();
});

describe("the access setup state", () => {
  it("asks the Owner to set it up, and a member no project lists to wait", async () => {
    expect(await setupState({ roles: [owner], permissions: [], projects: [], access_setup_finished: false })).toBe("owner");
    expect(await setupState({ roles: [dev], permissions: [], projects: [], access_setup_finished: false })).toBe("member");
  });

  it("asks nothing once it is done, of a member already in a project, or of a server that sends no marker", async () => {
    expect(await setupState({ roles: [owner], permissions: [], projects: [], access_setup_finished: true })).toBeUndefined();
    expect(await setupState({ roles: [dev], permissions: [], projects: [{ project_id: "p-1", actions: [] }], access_setup_finished: false })).toBeUndefined();
    expect(await setupState({ roles: [dev], permissions: [], projects: [] })).toBeUndefined();
  });
});

const grids = () =>
  seeded([
    [["getAccessGrid", "ws-1"], {}],
    [["getAccessGrid", "ws-2"], {}],
    [["getChatConversations", "ws-1"], []],
    [["getChatConversations", "ws-2"], []],
  ]);

describe("the access grid follower", () => {
  it("rereads the saved workspace's grid alone, and every grid for a rule on one entity", async () => {
    const client = grids();
    await followFrame(accessGridFollower, "access.grant.changed", { resource_type: "workspace", resource_id: "ws-2" }, client);
    expect([isStale(client, ["getAccessGrid", "ws-1"]), isStale(client, ["getAccessGrid", "ws-2"])]).toEqual([false, true]);
    await followFrame(accessGridFollower, "access.grant.changed", { resource_type: "project", resource_id: "p-1" }, client);
    expect(isStale(client, ["getAccessGrid", "ws-1"])).toBe(true);
  });

  it("rereads a workspace's grid when its roles or members move", async () => {
    const client = grids();
    await followFrame(accessGridFollower, "role.updated", { workspace_id: "ws-1" }, client);
    expect([isStale(client, ["getAccessGrid", "ws-1"]), isStale(client, ["getAccessGrid", "ws-2"])]).toEqual([true, false]);
  });
});

describe("the chat follower on access frames", () => {
  it("rereads a saved workspace's channel list, every list for a channel rule, and none for a doc rule", async () => {
    const client = grids();
    await followFrame(chatFollower, "access.grant.changed", { resource_type: "doc", resource_id: "d-1" }, client);
    expect(isStale(client, ["getChatConversations", "ws-1"])).toBe(false);
    await followFrame(chatFollower, "access.grant.changed", { resource_type: "workspace", resource_id: "ws-1" }, client);
    expect([isStale(client, ["getChatConversations", "ws-1"]), isStale(client, ["getChatConversations", "ws-2"])]).toEqual([true, false]);
    await followFrame(chatFollower, "access.grant.changed", { resource_type: "channel", resource_id: "c-1" }, client);
    expect(isStale(client, ["getChatConversations", "ws-2"])).toBe(true);
  });
});
