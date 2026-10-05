import { act, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { LiveSocket } from "@/api/ws";
import { useLiveEvents } from "@/hooks/useLiveEvents";
import { getMeKey } from "@/hooks/AuthHooks";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";
import { getMyRoleKey, getWorkspacesKey } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";
import { useFlowStore } from "@/stores/flowStore";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

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
  const { pathname, search } = useLocation();
  return <p data-testid="location">{pathname + search}</p>;
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
    useDeviceArrivalStore.setState({ arrivals: [] });
  });

  const setup = (path = "/") => {
    client = new QueryClient();
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={[path]}>
          <Harness
            wsFactory={() => {
              const s = new FakeSocket();
              sockets.push(s);
              return s;
            }}
          />
        </MemoryRouter>
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

  it("records the viewer's own phone on session.created and refetches the devices list", async () => {
    setup();
    client.setQueryData([getMeKey], { user: { id: "u1" } });
    const socket = await connectedSocket();
    const spy = invalidate();
    const push = (payload: Record<string, string>) =>
      act(() => socket.message(JSON.stringify({ topic: "session.created", type: "event", payload })));

    push({ session_id: "s-browser", user_id: "u1", client: "browser", platform: "Linux", label: "Chrome" });
    push({ session_id: "s-other", user_id: "u2", client: "phone", platform: "Android", label: "Pixel 8" });
    push({ session_id: "s-phone", user_id: "u1", client: "phone", platform: "Android", label: "Pixel 8" });

    expect(spy).toHaveBeenCalledTimes(3);
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getSessions"] });
    const arrivals = useDeviceArrivalStore.getState().arrivals;
    expect(arrivals).toHaveLength(1);
    expect(arrivals[0]).toMatchObject({ id: "s-phone", platform: "Android", label: "Pixel 8" });
  });

  it.each(["doc.moved", "doc.folder.updated", "doc.folder.deleted"])("refetches the inbox on %s, whose rows group by a doc's folder", async (topic) => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() => socket.message(JSON.stringify({ topic, type: "event", payload: { id: "d-1" } })));
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getNotifications"] });
  });

  it("refetches every template view on instance_template.updated, so unedited workspaces show the new text", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    act(() => socket.message(JSON.stringify({ topic: "instance_template.updated", type: "event", payload: { kind: "interview", key: "" } })));
    for (const key of ["getTemplates", "getWorkspaces", "getInterviewTemplate", "getWorkspacePlays", "getProjectTicketTypes"]) {
      expect(spy).toHaveBeenCalledWith({ queryKey: [key] });
    }
  });

  describe("a permission change", () => {
    const refetchedEverything = (spy: ReturnType<typeof invalidate>) => spy.mock.calls.some((args) => args[0] === undefined);
    const push = (socket: FakeSocket, topic: string, payload: Record<string, string>) =>
      act(() => socket.message(JSON.stringify({ topic, type: "event", payload })));

    it.each(["workspace.member.updated", "workspace.member.removed", "access.grant.changed"])(
      "refetches every open read when %s names the viewer, and nothing extra when it names someone else",
      async (topic) => {
        setup();
        client.setQueryData([getMeKey], { user: { id: "u1" }, instance_permissions: [] });
        const socket = await connectedSocket();
        const spy = invalidate();

        push(socket, topic, { user_id: "u2", workspace_id: "ws-1", resource_type: "doc", resource_id: "d-1" });
        expect(refetchedEverything(spy)).toBe(false);

        push(socket, topic, { user_id: "u1", workspace_id: "ws-1", resource_type: "doc", resource_id: "d-1" });
        await vi.waitFor(() => expect(refetchedEverything(spy)).toBe(true));
      },
    );

    it.each(["access.grant.changed", "workspace.member.updated"])(
      "refreshes Team, People with access, and the pickers when %s moves someone else's Project access",
      async (topic) => {
        setup();
        client.setQueryData([getMeKey], { user: { id: "u1" }, instance_permissions: [] });
        const socket = await connectedSocket();
        const spy = invalidate();

        push(socket, topic, { user_id: "u2", workspace_id: "ws-1", resource_type: "project", resource_id: "p-1" });

        expect(spy).toHaveBeenCalledWith({ queryKey: ["getTeam"] });
        expect(spy).toHaveBeenCalledWith({ queryKey: ["getProjectAccess"] });
        expect(spy).toHaveBeenCalledWith({ queryKey: ["getProjectPeople"] });
      },
    );

    it("refetches every open read after role.updated only when the viewer's own permissions moved", async () => {
      setup();
      let held = ["docs:read"];
      client.setQueryDefaults([getMyRoleKey], { queryFn: () => ({ role_name: "Editor", permissions: held }) });
      client.setQueryDefaults([getMeKey], { queryFn: () => ({ user: { id: "u1" }, instance_permissions: [] }) });
      await client.fetchQuery({ queryKey: [getMyRoleKey, "ws-1"] });
      await client.fetchQuery({ queryKey: [getMeKey] });
      const socket = await connectedSocket();
      const spy = invalidate();
      const roleFetches = () => client.getQueryState([getMyRoleKey, "ws-1"])?.dataUpdateCount;

      push(socket, "role.updated", { role_id: "r-other", workspace_id: "ws-1" });
      await vi.waitFor(() => expect(roleFetches()).toBe(2));
      expect(refetchedEverything(spy)).toBe(false);

      held = ["docs:read", "memories:read"];
      push(socket, "role.updated", { role_id: "r-mine", workspace_id: "ws-1" });
      await vi.waitFor(() => expect(refetchedEverything(spy)).toBe(true));
    });
  });

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

  it("refetches a doc's clarification on every doc.clarification topic, the panel and the run dialog alike", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const topics = ["round_started", "round_posted", "round_ended", "round_answered", "answer_saved", "answer_cleared", "anything_else_saved", "closed"];
    for (const topic of topics) {
      act(() => socket.message(JSON.stringify({ topic: `doc.clarification.${topic}`, type: "event", payload: {} })));
    }
    expect(spy).toHaveBeenCalledTimes(topics.length);
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getDocClarification"] });
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
    spy.mockClear();
    act(() =>
      socket.message(JSON.stringify({ topic: "chat.conversation.members_changed", type: "event", payload: {} })),
    );
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getChatConversations"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getChatUnread"] });
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

  it("merges chat.agent.stream hand-offs by id, keeping earlier ones across frames that carry none", async () => {
    setup();
    const socket = await connectedSocket();
    const helper = (id: string, state: string) => ({ id, driver: "codex", model: "gpt-5-codex", title: id, prompt: "Review it", state, reply: "", steps: [] });
    const frame = (payload: object) =>
      act(() =>
        socket.message(
          JSON.stringify({ topic: "chat.agent.stream", type: "event", payload: { conversation_id: "c1", message_id: "", text: "", streaming: true, ...payload } }),
        ),
      );

    frame({ handoff: helper("sa-1", "running") });
    frame({ handoff: helper("sa-2", "running") });
    frame({ text: "Waiting on the helpers" });
    frame({ handoff: helper("sa-1", "done") });

    expect(useAgentStreamStore.getState().streams.c1?.handoffs.map((h) => [h.id, h.state])).toEqual([
      ["sa-1", "done"],
      ["sa-2", "running"],
    ]);
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

  it("keeps the live hand-offs when a question lands mid-turn, and clears them with the reply", async () => {
    setup();
    const socket = await connectedSocket();
    const helper = { id: "sa-1", driver: "codex", model: "", title: "sa-1", prompt: "Review it", state: "running", reply: "", steps: [] };
    const question = "```nexul-question\n" + JSON.stringify({ request_id: "req-1", questions: [] }) + "\n```";
    const created = (body: string) =>
      act(() => socket.message(JSON.stringify({ topic: "chat.message.created", type: "event", payload: { message: { id: body, conversation_id: "c1", author_kind: "agent", body } } })));
    act(() =>
      socket.message(
        JSON.stringify({ topic: "chat.agent.stream", type: "event", payload: { conversation_id: "c1", message_id: "m-1", text: "Asking first", streaming: true, handoff: helper } }),
      ),
    );

    created(question);
    expect(useAgentStreamStore.getState().streams.c1).toMatchObject({ text: "", streaming: true, handoffs: [helper] });

    created("All done.");
    expect(useAgentStreamStore.getState().streams.c1).toBeUndefined();
  });

  it("leaves the agent stream store alone for a note left mid-turn", async () => {
    setup();
    const socket = await connectedSocket();
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "still going", streaming: true }));
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "chat.message.created",
          type: "event",
          payload: { message: { conversation_id: "c1", author_kind: "agent", attachment_id: "f1" } },
        }),
      ),
    );
    expect(useAgentStreamStore.getState().streams.c1).toBeDefined();
  });

  it("refetches a note's file and its thread's files when the note's message updates", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const note = { id: "m1", conversation_id: "c1", author_kind: "agent", attachment_id: "f1", body: "plan" };
    act(() => socket.message(JSON.stringify({ topic: "chat.message.updated", type: "event", payload: { message: note } })));
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getNoteText", "f1"] });
    expect(spy).toHaveBeenCalledWith({ queryKey: ["getAttachments", { conversation_id: "c1" }] });
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

  it("appends a setup turn's steps to the activity store without refetching, one row per tool call, a message whole", async () => {
    useSetupActivityStore.setState({ steps: {} });
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const activity = (extra: Record<string, string>) =>
      JSON.stringify({ topic: "computer.setup_turn_activity", type: "event", payload: { computer_id: "c1", turn_id: "t1", provider: "codex", ...extra } });
    act(() => socket.message(activity({ status: "nexul mcp add", kind: "tool_call", call_id: "call-1", tool: "Shell", at: "2026-10-01T10:00:00Z" })));
    act(() => socket.message(activity({ status: "nexul mcp add", kind: "tool_result", call_id: "call-1", tool: "Shell", at: "2026-10-01T10:00:02Z" })));
    act(() => socket.message(activity({ status: "All set. Codex…", kind: "text", text: "All set. Codex is confirmed.", at: "2026-10-01T10:00:05Z" })));
    expect(useSetupActivityStore.getState().steps.t1).toEqual([
      { kind: "tool_result", call_id: "call-1", tool: "Shell", summary: "nexul mcp add", detail: "", at: "2026-10-01T10:00:02Z" },
      { kind: "text", call_id: "", tool: "", summary: "All set. Codex…", detail: "All set. Codex is confirmed.", at: "2026-10-01T10:00:05Z" },
    ]);
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

  it("refreshes the computer MCP token and token list when a token is minted or revoked", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const payload = { token_id: "t1", user_id: "u1", name: "Nexul MCP on Laptop", computer_id: "c1" };
    for (const topic of ["personal_access_token.minted", "personal_access_token.revoked"]) {
      spy.mockClear();
      act(() => socket.message(JSON.stringify({ topic, type: "event", payload })));
      expect(spy).toHaveBeenCalledWith({ queryKey: ["getMCPToken"] });
      expect(spy).toHaveBeenCalledWith({ queryKey: ["getPATs"] });
    }
  });

  it("refreshes the people directory when someone joins or changes their name or picture", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    for (const [topic, payload] of [
      ["account.profile_updated", { account_id: "u-lewis" }],
      ["workspace.member.added", { user_id: "u-lewis", workspace_id: "ws-1" }],
    ] as const) {
      spy.mockClear();
      act(() => socket.message(JSON.stringify({ topic, type: "event", payload })));
      expect(spy).toHaveBeenCalledWith({ queryKey: ["getWorkspacePeople"] });
    }
  });

  it("refreshes the sessions list when a device signs in or is signed out", async () => {
    setup();
    const socket = await connectedSocket();
    const spy = invalidate();
    const payload = { session_id: "s1", user_id: "u1", client: "phone", platform: "Android", label: "Pixel 8" };
    for (const topic of ["session.created", "session.revoked"]) {
      spy.mockClear();
      act(() => socket.message(JSON.stringify({ topic, type: "event", payload })));
      expect(spy).toHaveBeenCalledWith({ queryKey: ["getSessions"] });
    }
  });

  it("applies a topology canvas patch to the flow store on a topology push", async () => {
    setup();
    useFlowStore.setState({ workspaceId: "ws-1", nodes: [] });
    const socket = await connectedSocket();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "topology",
          type: "event",
          payload: {
            workspace_id: "ws-1",
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

  it("leaves the canvas alone when a topology push is for another workspace", async () => {
    setup();
    useFlowStore.setState({ workspaceId: "ws-1", nodes: [] });
    const socket = await connectedSocket();
    act(() =>
      socket.message(
        JSON.stringify({
          topic: "topology",
          type: "event",
          payload: {
            workspace_id: "ws-2",
            schema_version: 2,
            nodes: [{ id: "svc-db", type: "service", position: { x: 0, y: 0 }, data: { service_id: "svc-db", name: "db", status: "running" } }],
            edges: [],
          },
        }),
      ),
    );
    expect(useFlowStore.getState().nodes).toHaveLength(0);
  });

  describe("chat.conversation.deleted", () => {
    const push = (socket: FakeSocket) =>
      act(() =>
        socket.message(
          JSON.stringify({
            topic: "chat.conversation.deleted",
            type: "event",
            payload: { conversation_id: "c-9", workspace_id: "ws-1", kind: "voice_channel", name: "huddle" },
          }),
        ),
      );

    it("sends a viewer of the deleted channel to the chat home, out of its call, and drops it from the list", async () => {
      setup("/acme/chat/c-9");
      useVoiceCallStore.setState({ activeConversationId: "c-9", status: "connected", room: null });
      const socket = await connectedSocket();
      const spy = invalidate();

      push(socket);

      expect(screen.getByTestId("location")).toHaveTextContent(/^\/acme\/chat$/);
      expect(useVoiceCallStore.getState().activeConversationId).toBeNull();
      expect(spy).toHaveBeenCalledWith({ queryKey: ["getChatConversations"] });
    });

    it("leaves someone on another conversation where they are", async () => {
      setup("/acme/chat/c-1");
      const socket = await connectedSocket();

      push(socket);

      expect(screen.getByTestId("location")).toHaveTextContent(/^\/acme\/chat\/c-1$/);
    });
  });

  describe("workspace.updated", () => {
    const workspace = (id: string, slug: string, name: string): Workspace => ({
      id,
      slug,
      name,
      mention_chip_template: "",
      mention_chip_template_edited: false,
      created_at: "",
      updated_at: "",
    });
    const push = (socket: FakeSocket, payload: Record<string, string>) =>
      act(() => socket.message(JSON.stringify({ topic: "workspace.updated", type: "event", payload })));

    beforeEach(() => {
      useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
    });

    it("moves the open page onto the workspace's new slug, keeping the rest of the address, and renames it in the list", async () => {
      setup("/acme/configuration/general?tab=x#top");
      client.setQueryData([getWorkspacesKey], [workspace("ws-1", "acme", "Acme"), workspace("ws-2", "other", "Other")]);
      const socket = await connectedSocket();

      push(socket, { workspace_id: "ws-1", name: "Acme Labs", slug: "acme-labs" });

      expect(screen.getByTestId("location")).toHaveTextContent("/acme-labs/configuration/general?tab=x");
      expect(useWorkspaceStore.getState().selectedWorkspaceSlug).toBe("acme-labs");
      expect(client.getQueryData<Workspace[]>([getWorkspacesKey])?.map((w) => [w.slug, w.name])).toEqual([
        ["acme-labs", "Acme Labs"],
        ["other", "Other"],
      ]);
    });

    it("leaves the address alone when the renamed workspace is not the one on screen, and still refreshes the list", async () => {
      setup("/acme/board");
      client.setQueryData([getWorkspacesKey], [workspace("ws-1", "acme", "Acme"), workspace("ws-2", "other", "Other")]);
      const socket = await connectedSocket();

      push(socket, { workspace_id: "ws-2", name: "Other Co", slug: "other-co" });

      expect(screen.getByTestId("location")).toHaveTextContent("/acme/board");
      expect(useWorkspaceStore.getState().selectedWorkspaceSlug).toBe("acme");
      expect(client.getQueryData<Workspace[]>([getWorkspacesKey])?.[1]?.slug).toBe("other-co");
    });
  });
});
