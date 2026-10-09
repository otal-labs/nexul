import { QueryClient, QueryClientProvider, focusManager, notifyManager } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react-native";
import { createContext, type ReactNode } from "react";

import { api } from "@/api/client";
import { fetchAbout } from "@/api/connect";
import { BoardScreen } from "@/components/board/BoardScreen";
import { TicketScreen } from "@/components/board/TicketScreen";
import { ChatListScreen } from "@/components/chat/ChatListScreen";
import { ChatThreadScreen } from "@/components/chat/ChatThreadScreen";
import { VersionGate } from "@/components/connect/VersionGate";
import { DeploysScreen } from "@/components/deploys/DeploysScreen";
import { InboxScreen } from "@/components/inbox/InboxScreen";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { dispatch } from "@/hooks/useLiveEvents";
import { useAreaAccess, useEnsureWorkspaceSelected } from "@/hooks/WorkspaceHooks";
import type { Message } from "@/models/Chat";
import { useBoardStore } from "@/stores/boardStore";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("expo-secure-store", () => ({ getItem: () => null, setItem: jest.fn(), deleteItemAsync: jest.fn() }));
jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));
jest.mock("@/api/connect", () => ({ fetchAbout: jest.fn() }));

// Each pushed screen reads its own route params, as it does on its own stack.
const mockParams = createContext<Record<string, string>>({});
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: jest.fn() }),
  useLocalSearchParams: () => jest.requireActual<typeof import("react")>("react").useContext(mockParams),
  Stack: { Screen: () => null },
}));
jest.mock("expo-router/react-navigation", () => ({ useHeaderHeight: () => 0 }));
jest.mock("react-native-enriched-markdown", () => jest.requireActual("react-native-enriched-markdown/jest"));

// Jest has no layout pass, so the list renders the slice of rows a scroll position would mount, and scrollTo moves it.
const mockScroll = {
  start: 0,
  size: Infinity,
  listeners: new Set<() => void>(),
  to(start: number) {
    this.start = start;
    this.listeners.forEach((listener) => listener());
  },
};
jest.mock("@legendapp/list/react-native", () => ({
  LegendList: (props: { data: unknown[]; keyExtractor: (item: unknown) => string; renderItem: (info: { item: unknown }) => unknown }) => {
    const { Fragment, createElement, useSyncExternalStore } = jest.requireActual<typeof import("react")>("react");
    const subscribe = (listener: () => void) => {
      mockScroll.listeners.add(listener);
      return () => void mockScroll.listeners.delete(listener);
    };
    const start = useSyncExternalStore(subscribe, () => mockScroll.start);
    return props.data
      .slice(start, start + mockScroll.size)
      .map((item) => createElement(Fragment, { key: props.keyExtractor(item) }, props.renderItem({ item }) as never));
  },
}));

// Query notifications otherwise land on a zero timeout after act has returned.
notifyManager.setScheduler((callback) => callback());

const get = jest.mocked(api.get);

const message = (n: number): Message => ({
  id: `m${n}`,
  conversation_id: "c1",
  author_id: ["ana", "bo", "me"][n % 3] ?? "ana",
  author_kind: "user",
  body: n % 4 === 0 ? `see [the spec](https://nexul.example.com/docs/d1) and **this** ${n}` : `message ${n}`,
  mentions: null,
  created_at: `2026-09-28T10:${String(n).padStart(2, "0")}:00Z`,
  updated_at: `2026-09-28T10:${String(n).padStart(2, "0")}:00Z`,
});

const ticket = (id: string, number: number) => ({
  id, project_id: "p-1", type_id: "", title: `Ticket ${number}`, body: "", status: "st-todo", position: 0, number, developer: "", tester: "", labels: null,
});

const responses: Record<string, unknown> = {
  "/api/auth/me": { user: { id: "me", login: "onik97", name: "Onik" } },
  "/api/workspaces": [{ id: "ws-1", name: "Acme", slug: "acme" }],
  "/api/workspaces/ws-1/me": { role_name: "Member", permissions: ["tickets:read", "docs:read", "stacks:read", "deploys:read"] },
  "/api/workspaces/ws-1/people": {
    people: [
      { user_id: "ana", login: "ana", display_name: "Ana", avatar_url: "" },
      { user_id: "bo", login: "bo", display_name: "Bo", avatar_url: "" },
      { user_id: "me", login: "onik97", display_name: "Onik", avatar_url: "" },
    ],
  },
  "/api/notifications?workspace_id=ws-1": [
    { id: "n1", user_id: "me", workspace_id: "ws-1", kind: "ticket.assigned", subject_type: "ticket", subject_id: "t-1", subject_title: "Ticket 1", read: false, created_at: "2026-09-28T10:00:00Z" },
  ],
  "/api/notifications/unread-count?workspace_id=ws-1": { count: 1 },
  "/api/chat/conversations?workspace_id=ws-1": [
    { id: "c1", workspace_id: "ws-1", kind: "channel", name: "general", created_by: "me", created_at: "", updated_at: "" },
    { id: "c2", workspace_id: "ws-1", kind: "ticket_thread", ticket_id: "t-2", created_by: "me", created_at: "", updated_at: "" },
    { id: "c3", workspace_id: "ws-1", kind: "ticket_thread", ticket_id: "t-3", created_by: "me", created_at: "", updated_at: "" },
    { id: "c4", workspace_id: "ws-1", kind: "doc_thread", doc_id: "d1", created_by: "me", created_at: "", updated_at: "" },
  ],
  "/api/chat/unread?workspace_id=ws-1": { c1: 0 },
  "/api/chat/conversations/c1/messages?limit=100": Array.from({ length: 40 }, (_, i) => message(i + 1)),
  "/api/attachments?conversation_id=c1": [],
  "/api/projects?workspace_id=ws-1": [{ id: "p-1", name: "Nexul", prefix: "NEX" }],
  "/api/projects/p-1": { id: "p-1", name: "Nexul", prefix: "NEX" },
  "/api/statuses?project_id=p-1": [{ id: "st-todo", name: "Todo", position: 0, kind: "backlog" }],
  "/api/ticket-types?project_id=p-1": [],
  "/api/tickets?project_id=p-1": [ticket("t-1", 1), ticket("t-2", 2), ticket("t-3", 3)],
  "/api/tickets/t-1": ticket("t-1", 1),
  "/api/tickets/t-2": ticket("t-2", 2),
  "/api/tickets/t-3": ticket("t-3", 3),
  "/api/docs/d1": { id: "d1", project_id: "p-1", title: "Spec", body: "", version: 1, archived: false, created_at: "", updated_at: "" },
  "/api/stacks?workspace_id=ws-1": [
    { id: "s-1", project_id: "p-1", name: "api", machine: "box-1", strategy: "run", managed: true },
    { id: "s-2", project_id: "p-1", name: "worker", machine: "box-1", strategy: "run", managed: true },
  ],
  "/api/stacks/s-1/deploys": [],
  "/api/stacks/s-2/deploys": [],
};

const unexpected: string[] = [];

// What the root and tab layouts read around the screens: the viewer, the workspace, the Inbox badge, the tab gates.
const Shell = () => {
  useFetchMe(true);
  useEnsureWorkspaceSelected();
  useFetchUnreadCount();
  useAreaAccess();
  return null;
};

const Pushed = ({ id, children }: { id: string; children: ReactNode }) => <mockParams.Provider value={{ id }}>{children}</mockParams.Provider>;

// A session that has visited every tab: each tab's root stays mounted, Chat has a channel open and Board a ticket.
const Session = () => (
  <VersionGate>
    <Shell />
    <InboxScreen />
    <ChatListScreen />
    <Pushed id="c1">
      <ChatThreadScreen />
    </Pushed>
    <BoardScreen />
    <Pushed id="t-1">
      <TicketScreen />
    </Pushed>
    <DeploysScreen />
  </VersionGate>
);

// The GET paths sent while run happens, once every refetch it started has settled.
const requestsDuring = async (client: QueryClient, run: () => Promise<void> | void): Promise<string[]> => {
  const before = get.mock.calls.length;
  await act(async () => {
    await run();
  });
  await waitFor(() => expect(client.isFetching()).toBe(0));
  return get.mock.calls.slice(before).map(([path]) => path);
};

const renderSession = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <Session />
    </QueryClientProvider>,
  );
  await waitFor(() => expect(client.isFetching()).toBe(0));
  return client;
};

beforeEach(() => {
  unexpected.length = 0;
  mockScroll.start = 0;
  mockScroll.size = Infinity;
  get.mockReset().mockImplementation(async (path: string) => {
    if (path in responses) return responses[path];
    unexpected.push(path);
    throw new Error(`unexpected GET ${path}`);
  });
  jest.mocked(api.post).mockReset().mockResolvedValue(null);
  jest.mocked(fetchAbout).mockReset().mockResolvedValue({ product: "nexul", version: "v9.0.0" });
  useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  useBoardStore.setState({ selectedProjectId: "p-1" });
});

afterEach(() => {
  expect(unexpected).toEqual([]);
  focusManager.setFocused(undefined);
});

const ticketFrame = (topic: string, id: string) => ({ topic, type: "event", payload: { ticket: { id, project_id: "p-1" }, from: "a", to: "b" } });

describe("requests a session sends", () => {
  // Every query any mounted screen holds refetches once: the socket was closed in the background, so it may be behind.
  test("coming back to the foreground refetches each open read once", async () => {
    const client = await renderSession();
    const aboutBefore = jest.mocked(fetchAbout).mock.calls.length;

    const gets = await requestsDuring(client, () => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
    });

    expect(new Set(gets).size).toBe(gets.length);
    expect(gets).toHaveLength(21);
    expect(jest.mocked(fetchAbout).mock.calls.length - aboutBefore).toBe(1);
  });

  // Each row reads the viewer, their role and the workspace's people as it mounts, so these must come from the cache.
  test("scrolling a chat thread mounts rows without a request", async () => {
    mockScroll.size = 10;
    const client = await renderSession();
    expect(screen.getByText("message 6")).toBeTruthy();

    const gets = await requestsDuring(client, () => [10, 20, 30].forEach((start) => mockScroll.to(start)));

    expect(screen.queryByText("message 6")).toBeNull();
    expect(screen.getByText("message 35")).toBeTruthy();
    expect(gets).toEqual([]);
  });

  test("a ticket frame refetches the board's list and that ticket, nothing else", async () => {
    const client = await renderSession();

    const gets = await requestsDuring(client, () => dispatch(client)(ticketFrame("ticket.status_changed", "t-1")));

    expect(gets.sort()).toEqual(["/api/tickets/t-1", "/api/tickets?project_id=p-1"]);
  });

  // The server publishes ticket.assignee_changed beside ticket.developer_changed for older clients; following both refetched twice.
  test("a developer change, published under both of its topics, refetches once", async () => {
    const client = await renderSession();

    const gets = await requestsDuring(client, () => {
      dispatch(client)(ticketFrame("ticket.developer_changed", "t-2"));
      dispatch(client)(ticketFrame("ticket.assignee_changed", "t-2"));
    });

    expect(gets.sort()).toEqual(["/api/tickets/t-2", "/api/tickets?project_id=p-1"]);
  });
});
