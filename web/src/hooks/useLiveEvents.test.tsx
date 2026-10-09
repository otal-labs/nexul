import { act, render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { LiveSocket } from "@nexul/client-core/liveSocket";

import { useLiveEvents } from "@/hooks/useLiveEvents";
import { isStale } from "@/test/followFrame";

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  close = vi.fn();
}

const Live = ({ wsFactory }: { wsFactory: () => LiveSocket }) => {
  useLiveEvents("ws://live/ws/events", { wsFactory });
  return null;
};

// A reconnect also checks the server's version, which needs a server.
vi.mock("@/hooks/VersionHooks", async (importOriginal) => ({ ...(await importOriginal<object>()), notifyIfServerUpdated: vi.fn() }));

// Mounts the socket over a client holding the given reads, and returns a way to push frames into it.
const connect = async (client: QueryClient) => {
  const sockets: FakeSocket[] = [];
  const wsFactory = () => {
    const s = new FakeSocket();
    sockets.push(s);
    return s;
  };
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Live wsFactory={wsFactory} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await vi.waitFor(() => expect(sockets[0]?.onmessage).toBeTruthy());
  return (topic: string, payload: unknown) => act(() => sockets[0]!.onmessage?.({ data: JSON.stringify({ topic, type: "event", payload }) }));
};

describe("the live socket", () => {
  it("hands a frame to every domain that follows its topic", async () => {
    const client = new QueryClient();
    client.setQueryData(["getDocs", "byProject", "p-1"], [{ id: "d-1", project_id: "p-1", folder_id: "f-1", title: "Spec" }]);
    client.setQueryData(["getNotifications", "ws-1"], [{ id: "n-1", subject_type: "doc", subject_id: "d-1", folder_id: "f-1" }]);
    const push = await connect(client);
    push("doc.moved", { doc: { id: "d-1", project_id: "p-1", folder_id: "f-2", title: "Spec" }, from_folder_id: "f-1" });
    expect(client.getQueryData<{ folder_id: string }[]>(["getDocs", "byProject", "p-1"])?.[0]?.folder_id).toBe("f-2");
    expect(isStale(client, ["getNotifications", "ws-1"])).toBe(true);
  });

  it("leaves the cache alone for a topic nobody follows", async () => {
    const client = new QueryClient();
    client.setQueryData(["getTickets"], []);
    const push = await connect(client);
    push("ticket.assignee_changed", { ticket: { id: "t-1" } });
    expect(isStale(client, ["getTickets"])).toBe(false);
  });

  it("refetches every open read once a dropped socket reconnects, since the frames sent meanwhile are lost", async () => {
    vi.useFakeTimers({ toFake: ["setTimeout"] });
    const client = new QueryClient();
    client.setQueryData(["getTickets"], []);
    const sockets: FakeSocket[] = [];
    const wsFactory = () => {
      const s = new FakeSocket();
      sockets.push(s);
      return s;
    };
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <Live wsFactory={wsFactory} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    act(() => sockets[0]!.onopen?.(null));
    expect(isStale(client, ["getTickets"])).toBe(false);

    vi.spyOn(console, "warn").mockImplementation(() => {});
    act(() => sockets[0]!.onclose?.(null));
    act(() => void vi.advanceTimersByTime(1_000));
    act(() => sockets[1]!.onopen?.(null));
    vi.useRealTimers();

    expect(isStale(client, ["getTickets"])).toBe(true);
  });

  describe("a permission change", () => {
    const signedIn = () => {
      const client = new QueryClient();
      client.setQueryData(["getMe"], { user: { id: "u-1" }, instance_permissions: [] });
      client.setQueryData(["getTickets"], []);
      return client;
    };

    it.each(["workspace.member.updated", "workspace.member.removed", "access.grant.changed"])(
      "refetches every open read when %s names the viewer, and nothing unrelated when it names someone else",
      async (topic) => {
        const client = signedIn();
        const push = await connect(client);
        push(topic, { user_id: "u-2", workspace_id: "ws-1", resource_type: "doc", resource_id: "d-1" });
        expect(isStale(client, ["getTickets"])).toBe(false);

        push(topic, { user_id: "u-1", workspace_id: "ws-1", resource_type: "doc", resource_id: "d-1" });
        await vi.waitFor(() => expect(isStale(client, ["getTickets"])).toBe(true));
      },
    );

    it("refetches every open read after role.updated only when the viewer's own permissions moved", async () => {
      const client = new QueryClient();
      let held = ["docs:read"];
      client.setQueryDefaults(["getMyRole"], { queryFn: () => ({ role_name: "Editor", permissions: held }) });
      client.setQueryDefaults(["getMe"], { queryFn: () => ({ user: { id: "u-1" }, instance_permissions: [] }) });
      await client.fetchQuery({ queryKey: ["getMyRole", "ws-1"] });
      await client.fetchQuery({ queryKey: ["getMe"] });
      client.setQueryData(["getTickets"], []);
      const push = await connect(client);
      const roleFetches = () => client.getQueryState(["getMyRole", "ws-1"])?.dataUpdateCount;

      push("role.updated", { role_id: "r-9", workspace_id: "ws-2" });
      push("role.updated", { role_id: "r-other", workspace_id: "ws-1" });
      await vi.waitFor(() => expect(roleFetches()).toBe(2));
      expect(isStale(client, ["getTickets"])).toBe(false);

      held = ["docs:read", "memories:read"];
      push("role.updated", { role_id: "r-mine", workspace_id: "ws-1" });
      await vi.waitFor(() => expect(isStale(client, ["getTickets"])).toBe(true));
    });
  });
});
