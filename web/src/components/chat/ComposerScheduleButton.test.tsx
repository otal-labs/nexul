import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ChatComposer } from "@/components/chat/ChatComposer";
import { useChatDraftStore } from "@/stores/chatDraftStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), post: vi.fn() }, errorMessage: (e: Error) => e.message }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderComposer = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ChatComposer workspaceId="ws-1" conversationId="c-1" onSend={vi.fn(async () => {})} />
    </QueryClientProvider>,
  );

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(new Date(2026, 10, 20, 12, 50));
  useChatDraftStore.setState({ drafts: {} });
  vi.mocked(api.get).mockResolvedValue({ data: { people: [] } });
  vi.mocked(api.post).mockReset();
  vi.mocked(api.post).mockResolvedValue({ data: { id: "sm-1", conversation_id: "c-1" } });
});

afterEach(() => vi.useRealTimers());

describe("scheduling from the composer", () => {
  it("schedules what is typed for the next full hour and empties the field", async () => {
    const user = userEvent.setup();
    renderComposer();
    await user.type(screen.getByLabelText("Message"), "Heads up for tomorrow");
    await user.click(screen.getByRole("button", { name: "Schedule message" }));

    const dialog = await screen.findByRole("dialog");
    expect(screen.getByLabelText("Time")).toHaveValue("14:00");
    await user.click(screen.getByRole("button", { name: "Schedule" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledOnce());
    expect(api.post).toHaveBeenCalledWith("/api/chat/conversations/c-1/scheduled-messages", {
      body: "Heads up for tomorrow",
      send_at: new Date(2026, 10, 20, 14, 0).toISOString(),
      repeats: "none",
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    });
    await waitFor(() => expect(dialog).not.toBeInTheDocument());
    expect(screen.getByRole("textbox", { name: "Message" })).toHaveValue("");
  });

  it("refuses a time already past and keeps the text in the field", async () => {
    const user = userEvent.setup();
    renderComposer();
    await user.type(screen.getByLabelText("Message"), "Too late");
    await user.click(screen.getByRole("button", { name: "Schedule message" }));
    const time = await screen.findByLabelText("Time");
    await user.clear(time);
    await user.type(time, "12:00");
    await user.click(screen.getByRole("button", { name: "Schedule" }));

    expect(await screen.findByText("Pick a time ahead")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getAllByLabelText("Message")[0]).toHaveValue("Too late");
  });
});
