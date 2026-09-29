import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { ChatThreadScreen } from "@/components/chat/ChatThreadScreen";
import { dispatch } from "@/hooks/useLiveEvents";
import type { Message } from "@/models/Chat";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("expo-secure-store", () => {
  const mockSecrets = new Map<string, string>();
  return {
    getItem: (key: string) => mockSecrets.get(key) ?? null,
    setItem: (key: string, value: string) => void mockSecrets.set(key, value),
    deleteItemAsync: async (key: string) => void mockSecrets.delete(key),
  };
});

jest.mock("expo-network", () => ({
  addNetworkStateListener: () => ({ remove: () => undefined }),
  getNetworkStateAsync: () => Promise.resolve({ isConnected: true }),
}));

jest.mock("expo-router", () => ({ useLocalSearchParams: () => ({ id: "c1" }), Stack: { Screen: () => null } }));
jest.mock("expo-router/react-navigation", () => ({ useHeaderHeight: () => 0 }));
jest.mock("lucide-react-native", () => ({ SendHorizontal: () => null }));
jest.mock("react-native-enriched-markdown", () => jest.requireActual("react-native-enriched-markdown/jest"));

// Jest has no layout pass, so this stand-in renders every row in data order and keeps the props for the anchoring checks.
const mockListProps: { current: Record<string, unknown> } = { current: {} };
jest.mock("@legendapp/list/react-native", () => ({
  LegendList: (props: {
    data: unknown[];
    keyExtractor: (item: unknown) => string;
    renderItem: (info: { item: unknown }) => unknown;
  }) => {
    const { Fragment, createElement } = jest.requireActual<typeof import("react")>("react");
    mockListProps.current = props;
    return props.data.map((item) => createElement(Fragment, { key: props.keyExtractor(item) }, props.renderItem({ item }) as never));
  },
}));

jest.mock("@/api/client", () => ({
  api: { get: jest.fn(), post: jest.fn() },
  errorMessage: (error: Error) => error.message,
  isNotFound: () => false,
}));

// Query notifications otherwise land on a zero timeout after act has returned.
notifyManager.setScheduler((callback) => callback());

const get = jest.mocked(api.get);
const post = jest.mocked(api.post);
const host = "https://nexul.example.com";

const message = (id: string, body: string, minute: number, author = "ana"): Message => ({
  id,
  conversation_id: "c1",
  author_id: author,
  author_kind: "user",
  body,
  mentions: null,
  created_at: `2026-09-28T10:${String(minute).padStart(2, "0")}:00Z`,
  updated_at: `2026-09-28T10:${String(minute).padStart(2, "0")}:00Z`,
});

let thread: Message[] = [];

const respond = async (path: string): Promise<unknown> => {
  if (path === "/api/auth/me") return { user: { id: "me", login: "onik", name: "Onik" } };
  if (path === "/api/workspaces") return [{ id: "w1", name: "Main" }];
  if (path === "/api/workspaces/w1/people") return { people: [{ user_id: "ana", login: "ana97", display_name: "Ana Lima", avatar_url: "" }] };
  if (path === "/api/chat/conversations?workspace_id=w1") return [];
  if (path.startsWith("/api/chat/conversations/c1/messages")) return thread;
  throw new Error(`unexpected GET ${path}`);
};

const renderThread = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { gcTime: Infinity } } });
  await render(
    <QueryClientProvider client={client}>
      <ChatThreadScreen />
    </QueryClientProvider>,
  );
  return client;
};

describe("ChatThreadScreen", () => {
  beforeEach(() => {
    get.mockReset();
    post.mockReset();
    post.mockResolvedValue(null);
    get.mockImplementation(respond);
    useSessionStore.getState().signIn(host, "ses_abc");
    useWorkspaceStore.setState({ selectedWorkspaceId: "w1" });
    thread = [message("m1", "first", 1), message("m2", "second", 2), message("m3", "newest", 3)];
  });

  test("a failed load shows the error and keeps the composer", async () => {
    get.mockImplementation(async (path: string) => {
      if (path.startsWith("/api/chat/conversations/c1/messages")) throw new Error("thread failed");
      return respond(path);
    });
    await renderThread();

    expect(await screen.findByText("thread failed")).toBeTruthy();
    expect(screen.getByLabelText("Message")).toBeTruthy();
  });

  test("a failed send takes the optimistic row back and restores the draft", async () => {
    await renderThread();
    await screen.findByText("newest");
    post.mockImplementation(async (path: string) => {
      if (path.endsWith("/read")) return null;
      throw new Error("send failed");
    });

    await userEvent.type(screen.getByLabelText("Message"), "lost?");
    await userEvent.press(screen.getByRole("button", { name: "Send" }));

    expect(await screen.findByText("send failed")).toBeTruthy();
    expect(screen.getByLabelText("Message").props.value).toBe("lost?");
  });

  test("messages render oldest to newest with the newest one shown", async () => {
    // IDs deliberately don't sort the same way as arrival order, so a future implementation that orders by
    // id instead of trusting the fetch order would fail this, instead of coincidentally passing anyway.
    thread = [message("m3", "first", 1), message("m1", "second", 2), message("m2", "newest", 3)];
    await renderThread();

    expect(await screen.findByText("newest")).toBeTruthy();
    const bodies = screen.getAllByText(/^(first|second|newest)$/).map((node) => node.props.children);
    expect(bodies).toEqual(["first", "second", "newest"]);
    expect((await screen.findAllByText("Ana Lima")).length).toBe(3);
    expect(mockListProps.current).toMatchObject({ initialScrollAtEnd: true, alignItemsAtEnd: true, maintainScrollAtEnd: true });
  });

  test("sending posts the message and shows it before the server answers", async () => {
    let confirm: (m: Message) => void = () => {};
    post.mockImplementation((path: string) => {
      if (path.endsWith("/read")) return Promise.resolve(null);
      return new Promise((resolve) => (confirm = resolve as (m: Message) => void));
    });
    await renderThread();
    await screen.findByText("newest");

    await userEvent.type(screen.getByLabelText("Message"), "hello from the phone");
    await userEvent.press(screen.getByRole("button", { name: "Send" }));

    expect(await screen.findByText("hello from the phone")).toBeTruthy();
    expect(post).toHaveBeenCalledWith("/api/chat/conversations/c1/messages", { body: "hello from the phone" });
    expect(screen.getByLabelText("Message").props.value).toBe("");

    await act(async () => confirm(message("m4", "hello from the phone", 4, "me")));
    expect(screen.getAllByText("hello from the phone")).toHaveLength(1);
  });

  test("a live message event refreshes the open thread", async () => {
    const client = await renderThread();
    await screen.findByText("newest");

    thread = [...thread, message("m4", "sent from the web", 4)];
    const invalidate = jest.spyOn(client, "invalidateQueries");
    await act(async () => {
      dispatch(client)({ topic: "chat.message.created", type: "event", payload: {} });
      await Promise.all(invalidate.mock.results.map((result) => result.value));
    });

    expect(await screen.findByText("sent from the web")).toBeTruthy();
  });

  // Android's Image drops the headers of a single source object, so only an array source reaches the private file route.
  test("markdown goes to the renderer and an attachment loads with the bearer token", async () => {
    thread = [message("m1", "**bold** words\n![shot.png](/api/attachments/a1)", 1)];
    await renderThread();

    expect(await screen.findByText("**bold** words")).toBeTruthy();
    expect(screen.getByLabelText("shot.png").props.source).toEqual([
      { uri: `${host}/api/attachments/a1`, headers: { Authorization: "Bearer ses_abc" } },
    ]);
  });
});
