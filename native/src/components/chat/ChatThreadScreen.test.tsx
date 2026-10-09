import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
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

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useLocalSearchParams: () => ({ id: "c1" }),
  useRouter: () => ({ push: mockPush }),
  Stack: { Screen: () => null },
}));
jest.mock("expo-router/react-navigation", () => ({ useHeaderHeight: () => 0 }));
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

const botPost = (id: string, overrides: Partial<Message>): Message => ({
  ...message(id, "", 1, "bw_7f3a"),
  author_kind: "bot",
  author_name: "GitHub",
  ...overrides,
});

const fields = (n: number) => Array.from({ length: n }, (_, i) => ({ name: `Field ${i + 1}`, value: `value ${i + 1}` }));

let thread: Message[] = [];

const respond = async (path: string): Promise<unknown> => {
  if (path === "/api/auth/me") return { user: { id: "me", login: "onik", name: "Onik" } };
  if (path === "/api/workspaces") return [{ id: "w1", name: "Main" }];
  if (path === "/api/workspaces/w1/people") return { people: [{ user_id: "ana", login: "ana97", display_name: "Ana Lima", avatar_url: "" }] };
  if (path === "/api/chat/conversations?workspace_id=w1") return [];
  if (path.startsWith("/api/chat/conversations/c1/messages")) return thread;
  if (path === "/api/attachments?conversation_id=c1") {
    return [{ id: "f1", conversation_id: "c1", name: "findings.md", content_type: "text/markdown", size: 2048, created_at: "2026-09-28T10:00:00Z" }];
  }
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
    mockPush.mockReset();
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

  test("a note shows its summary and file pill, and the pill opens the note", async () => {
    thread = [{ ...message("n1", "Found the cause", 1), author_kind: "agent", attachment_id: "f1" }];
    await renderThread();

    expect(await screen.findByText("Found the cause")).toBeTruthy();
    expect(screen.getByText("2 KB")).toBeTruthy();
    await userEvent.press(await screen.findByRole("button", { name: "Open findings.md" }));

    expect(mockPush).toHaveBeenCalledWith({ pathname: "/chat/note/[id]", params: { id: "f1", name: "findings.md" } });
  });

  test("an Agent reply that handed off work shows a pill per hand-off, and a pill opens that hand-off", async () => {
    const handoff = (id: string, title: string, state: "running" | "done") => ({
      id, driver: "codex", model: "gpt-5.5", title, prompt: "p", state, reply: "", steps: [],
    });
    thread = [{ ...message("r1", "Both checks are in", 1), author_kind: "agent", handoffs: [handoff("h1", "Run the tests", "done"), handoff("h2", "Check the docs", "running")] }];
    await renderThread();

    expect(await screen.findByRole("button", { name: "Open Run the tests, Done" })).toBeTruthy();
    await userEvent.press(screen.getByRole("button", { name: "Open Check the docs, Running" }));

    expect(mockPush).toHaveBeenCalledWith({ pathname: "/chat/handoff/[id]", params: { id: "h2", conversationId: "c1", messageId: "r1" } });
  });

  test("a reacted message shows each emoji with its count, and a reaction push updates it", async () => {
    thread = [{ ...message("m1", "shipped", 1), reactions: [{ emoji: "👍", user_ids: ["ana", "me"] }] }];
    const client = await renderThread();

    expect(await screen.findByLabelText("👍 2")).toBeTruthy();
    const unreact = { conversation_id: "c1", message_id: "m1", user_id: "me", emoji: "👍", reacted: false };
    await act(async () => dispatch(client)({ topic: "chat.message.reactions_changed", type: "event", payload: unreact }));
    expect(await screen.findByLabelText("👍 1")).toBeTruthy();
  });

  test("a message written in T3 says so beside its time", async () => {
    thread = [{ ...message("t1", "Use two threads", 1), via: "T3" }];
    await renderThread();

    expect(await screen.findByText("via T3")).toBeTruthy();
  });

  test("a bot's post shows the name it posted with, the BOT tag, and via, never the bot's id", async () => {
    thread = [botPost("b1", { body: "Deploy **finished**", via: "Release bot" })];
    await renderThread();

    expect(await screen.findByText("GitHub")).toBeTruthy();
    expect(screen.getByText("Bot")).toBeTruthy();
    expect(screen.getByText("via Release bot")).toBeTruthy();
    expect(screen.getByText("finished")).toBeTruthy();
    expect(screen.queryByText(/bw_7f3a/)).toBeNull();
  });

  test("a bot's served avatar loads with the bearer token", async () => {
    thread = [botPost("b1", { body: "hi", author_avatar_url: "/api/botwebhooks/bw_7f3a/avatar?v=2" })];
    await renderThread();

    expect((await screen.findByLabelText("Avatar of GitHub")).props.source).toEqual([
      { uri: `${host}/api/botwebhooks/bw_7f3a/avatar?v=2`, headers: { Authorization: "Bearer ses_abc" } },
    ]);
  });

  test("a bot's external pictures load through the media proxy, so the sender never sees the reader", async () => {
    const proxied = (url: string) => [
      { uri: `${host}/api/botwebhooks/media?message=b1&url=${encodeURIComponent(url)}`, headers: { Authorization: "Bearer ses_abc" } },
    ];
    thread = [
      botPost("b1", {
        author_avatar_url: "https://cdn.example.com/a.png",
        embeds: [{ title: "Run", image: { url: "https://cdn.example.com/chart.png" }, thumbnail: { url: "https://cdn.example.com/t.png" } }],
      }),
    ];
    await renderThread();

    expect((await screen.findByLabelText("Avatar of GitHub")).props.source).toEqual(proxied("https://cdn.example.com/a.png"));
    expect(screen.getByLabelText("Embed image").props.source).toEqual(proxied("https://cdn.example.com/chart.png"));
    expect(screen.getByLabelText("Embed thumbnail").props.source).toEqual(proxied("https://cdn.example.com/t.png"));
  });

  test("a bot post folds the fields past six and the embeds past two, and each bar opens and closes them", async () => {
    thread = [botPost("b1", { embeds: [{ title: "Checks", fields: fields(7) }, { title: "Second" }, { title: "Third" }] })];
    await renderThread();

    expect(await screen.findByText("Field 6")).toBeTruthy();
    expect(screen.queryByText("Field 7")).toBeNull();
    expect(screen.queryByText("Third")).toBeNull();

    await userEvent.press(screen.getByRole("button", { name: "Show 1 more field" }));
    expect(screen.getByText("Field 7")).toBeTruthy();
    await userEvent.press(screen.getByRole("button", { name: "Show less" }));
    expect(screen.queryByText("Field 7")).toBeNull();

    await userEvent.press(screen.getByRole("button", { name: "Show 1 more embed" }));
    expect(screen.getByText("Third")).toBeTruthy();
  });

  test("an agent message without a file shows no pill", async () => {
    thread = [{ ...message("n1", "Just a reply", 1), author_kind: "agent" }];
    await renderThread();

    expect(await screen.findByText("Just a reply")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /^Open / })).toBeNull();
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
    expect(mockListProps.current).toMatchObject({ initialScrollAtEnd: true, alignItemsAtEnd: true, maintainScrollAtEnd: true });
  });

  test("a run of one person's messages shows one name, and a pause of more than five minutes starts a new one", async () => {
    thread = [message("m1", "first", 1), message("m2", "second", 2), message("m3", "third", 3), message("m4", "much later", 14)];
    await renderThread();

    expect(await screen.findByText("much later")).toBeTruthy();
    expect(screen.getAllByText(/^(first|second|third|much later)$/)).toHaveLength(4);
    expect(screen.getAllByText("Ana Lima")).toHaveLength(2);
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

  test("a live message event shows in the open thread", async () => {
    const client = await renderThread();
    await screen.findByText("newest");

    await act(async () => dispatch(client)({ topic: "chat.message.created", type: "event", payload: { message: message("m4", "sent from the web", 4) } }));

    expect(await screen.findByText("sent from the web")).toBeTruthy();
  });

  test("losing a private channel while it is open turns the thread into not found, with no stale messages", async () => {
    const client = await renderThread();
    await screen.findByText("newest");

    get.mockImplementation(async (path: string) => {
      if (path.startsWith("/api/chat/conversations/c1/messages")) throw new ApiError(404, { message: "not found" }, "GET failed: 404");
      return respond(path);
    });
    const invalidate = jest.spyOn(client, "invalidateQueries");
    await act(async () => {
      dispatch(client)({ topic: "chat.conversation.members_changed", type: "event", payload: { conversation_id: "c1", removed_user_ids: ["me"] } });
      await Promise.all(invalidate.mock.results.map((result) => result.value));
    });

    expect(await screen.findByText("This conversation doesn't exist or was deleted.")).toBeTruthy();
    expect(screen.queryByText("newest")).toBeNull();
    expect(screen.queryByLabelText("Message")).toBeNull();
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
