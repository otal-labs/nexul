import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { MessageActionsSheet } from "@/components/chat/MessageActionsSheet";
import { getChatMessagesKey } from "@/hooks/ChatHooks";
import type { Message } from "@/models/Chat";
import { useMessageEditStore } from "@/stores/messageEditStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), delete: jest.fn() } }));

const mockBack = jest.fn();
const mockParams: { current: Record<string, string> } = { current: {} };
jest.mock("expo-router", () => ({
  useRouter: () => ({ back: mockBack }),
  useLocalSearchParams: () => mockParams.current,
}));

const mine: Message = {
  id: "m2",
  conversation_id: "c1",
  author_id: "me",
  author_kind: "user",
  body: "ship it frday",
  mentions: null,
  created_at: "2026-09-28T10:02:00Z",
  updated_at: "2026-09-28T10:02:00Z",
};

const renderSheet = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData([getChatMessagesKey, "c1"], [mine]);
  await render(
    <QueryClientProvider client={client}>
      <MessageActionsSheet />
    </QueryClientProvider>,
  );
  return client;
};

beforeEach(() => {
  mockBack.mockReset();
  jest.mocked(api.delete).mockReset();
  jest.mocked(api.get).mockReset().mockResolvedValue([mine]);
  mockParams.current = { conversationId: "c1", messageId: "m2" };
  useMessageEditStore.getState().stop();
});

describe("MessageActionsSheet", () => {
  test("Delete asks first, naming the message, and a refused delete shows why and keeps the message", async () => {
    jest.mocked(api.delete).mockRejectedValue(new Error("only the author may delete this message"));
    const client = await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByText("Delete this message?")).toBeTruthy();
    expect(screen.getByText("ship it frday")).toBeTruthy();
    expect(screen.getByText(/cannot be brought back/)).toBeTruthy();
    await userEvent.setup().press(screen.getByRole("button", { name: "Delete message" }));

    expect(await screen.findByText("only the author may delete this message")).toBeTruthy();
    expect(mockBack).not.toHaveBeenCalled();
    expect(client.getQueryData<Message[]>([getChatMessagesKey, "c1"])?.[0]?.deleted_at).toBeUndefined();
  });

  test("confirming deletes the message, takes it out of the thread and dismisses", async () => {
    jest.mocked(api.delete).mockResolvedValue(null);
    mockParams.current = { conversationId: "c1", messageId: "m2", mode: "delete" };
    const client = await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Delete message" }));

    expect(api.delete).toHaveBeenCalledWith("/api/chat/messages/m2");
    expect(client.getQueryData<Message[]>([getChatMessagesKey, "c1"])?.[0]?.deleted_at).toBeTruthy();
    expect(mockBack).toHaveBeenCalled();
  });

  test("Edit hands the message to the composer and dismisses without deleting", async () => {
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Edit" }));

    expect(useMessageEditStore.getState().editing).toEqual({ conversationId: "c1", messageId: "m2", body: "ship it frday" });
    expect(api.delete).not.toHaveBeenCalled();
    expect(mockBack).toHaveBeenCalled();
  });

  test("a message deleted from elsewhere while the sheet is open says it is gone and offers only Close", async () => {
    const client = await renderSheet();
    client.setQueryData([getChatMessagesKey, "c1"], [{ ...mine, deleted_at: "2026-09-28T10:03:00Z" }]);

    expect(await screen.findByText("This message is gone.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Edit" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Delete" })).toBeNull();
  });
});
