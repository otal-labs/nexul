import { act, render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { LiveSocket } from "@/api/ws";
import { useLiveEvents } from "@/hooks/useLiveEvents";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { useFlowStore } from "@/stores/flowStore";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  close = vi.fn();
  open() {
    this.onopen?.(null);
  }
  message(data: string) {
    this.onmessage?.({ data });
  }
}

const Harness = ({ wsFactory }: { wsFactory: () => LiveSocket }) => {
  useLiveEvents("ws://live/ws/events", { wsFactory });
  return null;
};

describe("useLiveEvents dispatch", () => {
  let sockets: FakeSocket[];
  let client: QueryClient;

  beforeEach(() => {
    sockets = [];
    useFlowStore.setState({ nodes: [], edges: [], selectedNodeId: null });
    useAgentStreamStore.setState({ streams: {} });
    usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
    useVoiceOccupancyStore.setState({ occupancy: {} });
  });

  const setup = () => {
    client = new QueryClient();
    render(
      <QueryClientProvider client={client}>
        <Harness
          wsFactory={() => {
            const s = new FakeSocket();
            sockets.push(s);
            return s;
          }}
        />
      </QueryClientProvider>,
    );
  };

  const connectedSocket = async () => {
    await vi.waitFor(() => expect(sockets[0]).toBeTruthy());
    const socket = sockets[0]!;
    await vi.waitFor(() => expect(socket.onopen).toBeTruthy());
    return socket;
  };

  const invalidate = () => vi.spyOn(client, "invalidateQueries");

  it("invalidates the runners query on a runner.connected push", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() => socket.message(JSON.stringify({ topic: "runner.connected", type: "event", payload: { runner_id: "r-1" } })));
    expect(spy).toHaveBeenCalledWith({ queryKey: ["runners"] });
  });

  it("invalidates only runners and queue on a deploy.status_changed push, never the deploy reads", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(JSON.stringify({ topic: "deploy.status_changed", type: "event", payload: { id: "d-1", status: "failed" } })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["runners"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["runnerQueue"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getStackDeploys"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getServiceDeploys"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getDeploy"] });
  });

  it("refetches the deploy, its log, and the deploy histories on a deploy.updated push", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(JSON.stringify({ topic: "deploy.updated", type: "event", payload: { id: "d-1", status: "failed" } })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getDeploy"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getDeployLog"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getStackDeploys"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getServiceDeploys"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["runners"] });
  });

  it("leaves the deploy reads alone on the runner's progress pushes", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    for (const topic of ["deploy.build_started", "deploy.build_progress", "deploy.build_completed", "deploy.deploy_progress"]) {
      act(() => socket.message(JSON.stringify({ topic, type: "event", payload: { id: "d-1" } })));
    }
    expect(spy).toHaveBeenCalledWith({ queryKey: ["runners"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getDeploy"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getDeployLog"] });
  });

  it("invalidates the tickets queries on ticket lifecycle topics", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    for (const topic of ["ticket.created", "ticket.finished"]) {
      act(() => socket.message(JSON.stringify({ topic, type: "event", payload: {} })));
    }
    expect(spy).toHaveBeenCalledTimes(2);
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
  });

  it("invalidates the ticket, its links, and the trails query on ticket.updated and ticket.status_changed", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(JSON.stringify({ topic: "ticket.updated", type: "event", payload: { ticket: { id: "t-1" } } })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTicket"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTicketLinks"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTrails"] });

    spy.mockClear();
    act(() =>
      socket.message(JSON.stringify({ topic: "ticket.status_changed", type: "event", payload: { ticket: { id: "t-1" } } })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTicket"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTicketLinks"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTrails"] });
  });

  it("refreshes link sets and the board's blockers on link changes and blocker moves", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    for (const topic of ["ticket.link_created", "ticket.link_deleted", "ticket.status_changed"]) {
      spy.mockClear();
      act(() => socket.message(JSON.stringify({ topic, type: "event", payload: {} })));
      expect(spy).toHaveBeenCalledWith({ queryKey: ["getTicketLinkSet"] });
      expect(spy).toHaveBeenCalledWith({ queryKey: ["getBlockers"] });
    }
  });

  it("invalidates by topic for topics without a mapping", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() => socket.message(JSON.stringify({ topic: "ticket.mentioned", type: "event", payload: {} })));
    expect(spy).toHaveBeenCalledWith({ queryKey: ["ticket.mentioned"] });
  });

  it("invalidates notification queries on notification.created", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(JSON.stringify({ topic: "notification.created", type: "event", payload: { count: 1 } })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getNotifications"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getUnreadCount"] });
  });

  it("invalidates the instance upgrade and server version queries on instance.upgrade_changed", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(JSON.stringify({ topic: "instance.upgrade_changed", type: "event", payload: {} })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["instanceUpgrade"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["serverVersion"] });
  });

  it("invalidates conversation and unread queries on chat lifecycle topics, never the message list", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(JSON.stringify({ topic: "chat.message.created", type: "event", payload: {} })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getChatConversations"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getChatUnread"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getChatMessages"] });
    spy.mockClear();
    act(() =>
      socket.message(JSON.stringify({ topic: "chat.message.deleted", type: "event", payload: {} })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getChatUnread"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getChatMessages"] });
  });

  it("patches the cached message list from chat message frames instead of refetching", async () => {
    setup();
    const socket = await connectedSocket();
    const m1 = { id: "m1", conversation_id: "c1", author_id: "u1", author_kind: "user", body: "hi", mentions: null };
    client.setQueryData(["getChatMessages", "c1", undefined], [m1]);
    const m2 = { ...m1, id: "m2", body: "there" };
    act(() => socket.message(JSON.stringify({ topic: "chat.message.created", type: "event", payload: { message: m2 } })));
    expect(client.getQueryData(["getChatMessages", "c1", undefined])).toEqual([m1, m2]);

    act(() =>
      socket.message(JSON.stringify({ topic: "chat.message.updated", type: "event", payload: { message: { ...m2, body: "edited" } } })),
    );
    expect(client.getQueryData(["getChatMessages", "c1", undefined])).toEqual([m1, { ...m2, body: "edited" }]);

    act(() =>
      socket.message(
        JSON.stringify({
          topic: "chat.message.deleted",
          type: "event",
          payload: { conversation_id: "c1", message_id: "m1", deleted_at: "2026-09-15T00:00:00Z" },
        }),
      ),
    );
    expect(client.getQueryData(["getChatMessages", "c1", undefined])).toEqual([
      { ...m1, deleted_at: "2026-09-15T00:00:00Z" },
      { ...m2, body: "edited" },
    ]);
  });

  it("writes play.run frames into the run store and refetches trails only on a state change", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const base = { trail_id: "tr-1", play_id: "play-1", target_type: "ticket", target_id: "t-1", ended_at: null, last_error: "" };
    act(() => socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "running", activity: null } })));
    expect(usePlayRunStore.getState().activeByTarget["ticket:t-1"]).toBe("tr-1");
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTrails", "ticket", "t-1"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTrail", "tr-1"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getActiveTrails"] });

    spy.mockClear();
    act(() =>
      socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "running", activity: { kind: "tool_call", call_id: "c-1", tool: "Read", summary: "main.go", at: "2026-09-18T10:00:00Z" } } })),
    );
    expect(usePlayRunStore.getState().frames["tr-1"]?.activity).toMatchObject({ tool: "Read", summary: "main.go" });
    expect(spy).not.toHaveBeenCalled();

    act(() => socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "done", activity: null } })));
    expect(usePlayRunStore.getState().activeByTarget["ticket:t-1"]).toBeUndefined();
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTrails", "ticket", "t-1"] });
  });

  it("invalidates the ticket thread-exists query when a ticket run starts", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const base = { trail_id: "tr-2", play_id: "play-1", target_type: "ticket", target_id: "t-9", ended_at: null, last_error: "" };
    act(() => socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "starting", activity: "" } })));
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getChatTicketThreadStatus", "t-9"] });
  });

  it("invalidates the ticket, its links, and the board once a ticket run reaches a terminal state", async () => {
    setup();
    const socket = await connectedSocket();
    const base = { trail_id: "tr-3", play_id: "play-1", target_type: "ticket", target_id: "t-9", ended_at: null, last_error: "" };
    act(() => socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "starting", activity: "" } })));
    const spy = invalidate();
    act(() => socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "done", activity: "" } })));
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTicket", "t-9"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getTicketLinks", "t-9"] });
  });

  it("never invalidates ticket-specific queries for a doc target's play.run frames", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const base = { trail_id: "tr-4", play_id: "play-2", target_type: "doc", target_id: "d-1", ended_at: null, last_error: "" };
    act(() => socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "starting", activity: "" } })));
    act(() => socket.message(JSON.stringify({ topic: "play.run", type: "event", payload: { ...base, state: "done", activity: "" } })));
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getChatTicketThreadStatus", "d-1"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getTicket", "d-1"] });
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ["getTicketLinks", "d-1"] });
  });

  it("keeps the tool activity on chat.agent.stream frames", async () => {
    setup();
    const socket = await connectedSocket();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "chat.agent.stream",
          type: "event",
          payload: { conversation_id: "c1", message_id: "", text: "", streaming: true, activity: "Read main.go started", activity_kind: "tool_call", activity_tool: "Read" },
        }),
      ),
    );
    expect(useAgentStreamStore.getState().streams.c1).toMatchObject({ activity: "Read main.go started", activityKind: "tool_call", activityTool: "Read", streaming: true });
  });

  it("writes chat.agent.stream frames straight into the agent stream store, keyed by conversation", async () => {
    setup();
    const socket = await connectedSocket();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "chat.agent.stream",
          type: "event",
          payload: { conversation_id: "c1", message_id: "stream-1", text: "Looking into it", streaming: true },
        }),
      ),
    );
    expect(useAgentStreamStore.getState().streams.c1).toMatchObject({ messageId: "stream-1", text: "Looking into it", streaming: true });
  });

  it("clears a conversation's agent stream once its final agent message is persisted", async () => {
    setup();
    const socket = await connectedSocket();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "chat.agent.stream",
          type: "event",
          payload: { conversation_id: "c1", message_id: "stream-1", text: "final text", streaming: false },
        }),
      ),
    );
    expect(useAgentStreamStore.getState().streams.c1).toBeDefined();

    act(() =>
      socket.message(
        JSON.stringify({
          topic: "chat.message.created",
          type: "event",
          payload: { message: { conversation_id: "c1", author_kind: "agent" } },
        }),
      ),
    );
    expect(useAgentStreamStore.getState().streams.c1).toBeUndefined();
  });

  it("leaves the agent stream store alone for a plain user message", async () => {
    setup();
    const socket = await connectedSocket();
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "still going", streaming: true }));
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "chat.message.created",
          type: "event",
          payload: { message: { conversation_id: "c1", author_kind: "user" } },
        }),
      ),
    );
    expect(useAgentStreamStore.getState().streams.c1).toBeDefined();
  });

  it("applies a voice.occupancy.changed frame straight into the occupancy store, keyed by conversation", async () => {
    setup();
    const socket = await connectedSocket();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "voice.occupancy.changed",
          type: "event",
          payload: { conversation_id: "c1", occupants: [{ identity: "u2", name: "Dana" }] },
        }),
      ),
    );
    expect(useVoiceOccupancyStore.getState().occupancy.c1).toEqual([{ identity: "u2", name: "Dana" }]);

    act(() =>
      socket.message(
        JSON.stringify({ topic: "voice.occupancy.changed", type: "event", payload: { conversation_id: "c1", occupants: [] } }),
      ),
    );
    expect(useVoiceOccupancyStore.getState().occupancy.c1).toBeUndefined();
  });

  it("replaces a computer's cached tunnel checks on a computer.tunnel_status_changed push", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "computer.tunnel_status_changed",
          type: "event",
          payload: { computer_id: "c1", user_id: "u1", tunnel: "healthy", harness_reachable: true, harness_version: "0.0.40" },
        }),
      ),
    );
    expect(client.getQueryData(["getTunnelStatus", "c1"])).toEqual({ tunnel: "healthy", harness_reachable: true, harness_version: "0.0.40" });
    expect(spy).not.toHaveBeenCalled();

    act(() =>
      socket.message(
        JSON.stringify({ topic: "computer.tunnel_status_changed", type: "event", payload: { computer_id: "c1", user_id: "u1", tunnel: "down", harness_reachable: false } }),
      ),
    );
    expect(client.getQueryData(["getTunnelStatus", "c1"])).toEqual({ tunnel: "down", harness_reachable: false });
  });

  it("refreshes every computer's setup read as a setup turn changes or a run finishes", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(
        JSON.stringify({ topic: "computer.setup_turn_changed", type: "event", payload: { computer_id: "c1", provider: "codex", state: "running" } }),
      ),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getComputerSetup"] });

    spy.mockClear();
    act(() => socket.message(JSON.stringify({ topic: "computer.setup_finished", type: "event", payload: { computer_id: "c1", confirmed: true } })));
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getComputerSetup"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getHarnessProviders"] });
  });

  it("appends a setup turn's commentary line to the activity store without refetching", async () => {
    useSetupActivityStore.setState({ lines: {} });
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const activity = (status: string) =>
      JSON.stringify({ topic: "computer.setup_turn_activity", type: "event", payload: { computer_id: "c1", turn_id: "t1", provider: "codex", status } });
    act(() => socket.message(activity("Installing skills")));
    act(() => socket.message(activity("Checking files")));
    expect(useSetupActivityStore.getState().lines.t1).toEqual(["Installing skills", "Checking files"]);
    expect(spy).not.toHaveBeenCalled();
  });

  it("refreshes the computer rows and readiness when a computer finishes pairing", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() =>
      socket.message(
        JSON.stringify({ topic: "computer.paired", type: "event", payload: { computer_id: "c1", user_id: "u1", server_url: "https://h", token_expires_at: "2026-10-24T00:00:00Z" } }),
      ),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getComputers"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getHarnessResolve"] });
  });

  it("applies a topology canvas patch to the flow store on a topology push", async () => {
    setup();
    const socket = await connectedSocket();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "topology",
          type: "event",
          payload: {
            schema_version: 2,
            nodes: [
              { id: "svc-api", type: "service", position: { x: 0, y: 0 }, data: { service_id: "svc-api", name: "api", status: "running" } },
            ],
            edges: [],
          },
        }),
      ),
    );
    await vi.waitFor(() => {
      expect(useFlowStore.getState().nodes).toHaveLength(1);
    });
    const node = useFlowStore.getState().nodes[0];
    expect(node?.type === "service" ? node.data.status : undefined).toBe("running");
  });
});
