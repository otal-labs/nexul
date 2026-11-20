import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import type { ScheduledMessage } from "@nexul/client-core/chat";

import { api } from "@/api/client";
import { ScheduleMessageSheet } from "@/components/chat/ScheduleMessageSheet";
import { getScheduledMessagesKey } from "@/hooks/ScheduledMessageHooks";
import { useScheduleStore } from "@/stores/scheduleStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn(), patch: jest.fn() } }));
jest.mock("@react-native-community/datetimepicker", () => ({ __esModule: true, default: () => null, DateTimePickerAndroid: { open: jest.fn() } }));

const mockBack = jest.fn();
const mockParams: { current: Record<string, string> } = { current: {} };
jest.mock("expo-router", () => ({
  useRouter: () => ({ back: mockBack }),
  useLocalSearchParams: () => mockParams.current,
}));

const weekly: ScheduledMessage = {
  id: "sm-1",
  conversation_id: "c1",
  author_id: "me",
  body: "Standup notes",
  starts_at: "2026-11-02T09:00:00.000Z",
  send_at: "2099-11-23T09:00:00.000Z",
  repeats: "weekly",
  timezone: "Europe/London",
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

const renderSheet = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData([getScheduledMessagesKey, "c1"], [weekly]);
  await render(
    <QueryClientProvider client={client}>
      <ScheduleMessageSheet />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mockBack.mockReset();
  jest.mocked(api.get).mockReset().mockResolvedValue([weekly]);
  jest.mocked(api.post).mockReset();
  jest.mocked(api.patch).mockReset();
  useScheduleStore.setState({ handedOff: {} });
});

describe("ScheduleMessageSheet", () => {
  test("a refused schedule shows why and keeps the composer's text", async () => {
    jest.mocked(api.post).mockRejectedValue(new Error("channels:post required"));
    mockParams.current = { conversationId: "c1", body: "Heads up for tomorrow" };
    await renderSheet();

    expect(screen.getByDisplayValue("Heads up for tomorrow")).toBeTruthy();
    await userEvent.setup().press(screen.getByRole("button", { name: "Schedule" }));

    expect(await screen.findByText("channels:post required")).toBeTruthy();
    expect(mockBack).not.toHaveBeenCalled();
    expect(useScheduleStore.getState().handedOff.c1).toBeUndefined();
  });

  test("scheduling sends the text once, at the next full hour, and hands the composer's text off", async () => {
    jest.mocked(api.post).mockResolvedValue({ ...weekly, repeats: "none" });
    mockParams.current = { conversationId: "c1", body: "Heads up for tomorrow" };
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Schedule" }));

    const [path, input] = jest.mocked(api.post).mock.calls[0] as [string, Record<string, string>];
    expect(path).toBe("/api/chat/conversations/c1/scheduled-messages");
    expect(input).toMatchObject({ body: "Heads up for tomorrow", repeats: "none" });
    expect(new Date(input.send_at ?? "").getUTCMinutes()).toBe(0);
    expect(mockBack).toHaveBeenCalled();
    expect(useScheduleStore.getState().handedOff.c1).toBe(1);
  });

  test("editing only the text leaves the send time out, so the repeat keeps its day", async () => {
    jest.mocked(api.patch).mockResolvedValue(weekly);
    mockParams.current = { conversationId: "c1", id: "sm-1" };
    await renderSheet();

    const field = screen.getByDisplayValue("Standup notes");
    await userEvent.setup().clear(field);
    await userEvent.setup().type(field, "Standup notes, moved");
    await userEvent.setup().press(screen.getByRole("button", { name: "Save" }));

    expect(api.patch).toHaveBeenCalledWith("/api/chat/scheduled-messages/sm-1", { body: "Standup notes, moved", repeats: "weekly", timezone: expect.any(String) });
  });
});
