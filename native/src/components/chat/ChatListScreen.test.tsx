import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { ChatListScreen } from "@/components/chat/ChatListScreen";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("expo-secure-store", () => ({ getItem: () => null, setItem: jest.fn(), deleteItemAsync: jest.fn() }));

// The icon package ships untransformed ESM; the rows are asserted by text, not by icon.
jest.mock("lucide-react-native", () => ({ FileText: () => null, Hash: () => null, SquareKanban: () => null, User: () => null }));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ useRouter: () => ({ push: mockPush }) }));

jest.mock("@/api/client", () => ({
  api: { get: jest.fn(), post: jest.fn() },
  errorMessage: (error: Error) => error.message,
}));

const get = jest.mocked(api.get);

const responses: Record<string, unknown> = {
  "/api/auth/me": { user: { id: "me", login: "onik", name: "Onik" } },
  "/api/workspaces": [{ id: "w1", name: "Main" }],
  "/api/workspaces/w1/members": {
    members: [
      { user_id: "me", login: "onik" },
      { user_id: "ana", login: "ana" },
    ],
  },
  "/api/chat/conversations?workspace_id=w1": [
    { id: "c1", workspace_id: "w1", kind: "channel", name: "general", created_by: "me", created_at: "", updated_at: "" },
    { id: "c2", workspace_id: "w1", kind: "dm", participant_ids: ["me", "ana"], created_by: "me", created_at: "", updated_at: "" },
    { id: "c3", workspace_id: "w1", kind: "voice_channel", name: "standup", created_by: "me", created_at: "", updated_at: "" },
  ],
  "/api/chat/unread?workspace_id=w1": { c1: 3 },
};

const renderScreen = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } })}>
      <ChatListScreen />
    </QueryClientProvider>,
  );


beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "w1" });
});

describe("ChatListScreen", () => {
  beforeEach(() => {
    get.mockReset();
    mockPush.mockReset();
    get.mockImplementation(async (path: string) => {
      if (path in responses) return responses[path];
      throw new Error(`unexpected GET ${path}`);
    });
  });

  test("a failed load shows the error, not an empty list", async () => {
    get.mockImplementation(async (path: string) => {
      if (path === "/api/workspaces") return [{ id: "w1", name: "Main" }];
      throw new Error("conversations failed");
    });
    await renderScreen();

    expect(await screen.findByText("conversations failed")).toBeTruthy();
    expect(screen.queryByText("No conversations yet.")).toBeNull();
  });

  test("an empty workspace says there is nothing yet", async () => {
    get.mockImplementation(async (path: string) =>
      path === "/api/workspaces" ? [{ id: "w1", name: "Main" }] : path.startsWith("/api/chat/conversations") ? [] : {},
    );
    await renderScreen();

    expect(await screen.findByText("No conversations yet.")).toBeTruthy();
  });

  test("rows show channels and DMs by name with the unread count, and voice channels are left out", async () => {
    await renderScreen();

    expect(await screen.findByRole("button", { name: "general" })).toBeTruthy();
    expect(await screen.findByRole("button", { name: "ana" })).toBeTruthy();
    expect(screen.getByLabelText("3 unread")).toBeTruthy();
    expect(screen.queryByText("standup")).toBeNull();
  });

  test("tapping a row opens its thread", async () => {
    await renderScreen();

    await userEvent.press(await screen.findByRole("button", { name: "general" }));

    expect(mockPush).toHaveBeenCalledWith({ pathname: "/chat/[id]", params: { id: "c1" } });
  });
});
