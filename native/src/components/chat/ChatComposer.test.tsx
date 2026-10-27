import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen } from "@testing-library/react-native";

import { api } from "@/api/client";
import { ChatComposer } from "@/components/chat/ChatComposer";

jest.mock("@/api/client", () => ({ api: { post: jest.fn() } }));

const post = jest.mocked(api.post);

const renderComposer = async () => {
  // A sent message's mutation would otherwise hold a five-minute cache timer open after the test.
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false, gcTime: 0 } } });
  await render(
    <QueryClientProvider client={client}>
      <ChatComposer conversationId="c-1" />
    </QueryClientProvider>,
  );
  await act(async () => fireEvent.changeText(screen.getByLabelText("Message"), "@Agent after that"));
};

describe("ChatComposer", () => {
  beforeEach(() => post.mockReset().mockResolvedValue({ id: "m-1", conversation_id: "c-1" }));

  test("Send steers the message into the Agent's turn under way", async () => {
    await renderComposer();
    await act(async () => fireEvent.press(screen.getByRole("button", { name: "Send" })));
    expect(post).toHaveBeenCalledWith("/api/chat/conversations/c-1/messages", { body: "@Agent after that" });
  });

  test.each([
    ["holding Send", () => fireEvent(screen.getByRole("button", { name: "Send" }), "longPress")],
    ["its accessibility action", () => fireEvent(screen.getByRole("button", { name: "Send" }), "accessibilityAction", { nativeEvent: { actionName: "longpress" } })],
  ])("%s queues it behind the Agent's turn", async (_name, act_) => {
    await renderComposer();
    await act(async () => act_());
    expect(post).toHaveBeenCalledWith("/api/chat/conversations/c-1/messages", { body: "@Agent after that", queue: true });
  });
});
