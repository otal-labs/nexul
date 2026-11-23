import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ScheduledMessage } from "@nexul/client-core/chat";

import { TooltipProvider } from "@/components/ui/tooltip";
import { ScheduledMessageRow } from "@/components/chat/ScheduledMessageRow";
import { api } from "@/api/client";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), post: vi.fn() }, errorMessage: (e: Error) => e.message }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const scheduled: ScheduledMessage = {
  id: "sm-1",
  conversation_id: "c-1",
  author_id: "u-1",
  body: "Standup in five",
  starts_at: "2030-01-01T09:00:00Z",
  send_at: "2030-01-01T09:00:00Z",
  repeats: "none",
  timezone: "UTC",
  created_at: "2029-12-31T09:00:00Z",
  updated_at: "2029-12-31T09:00:00Z",
};

const renderRow = () =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TooltipProvider delayDuration={0}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ul>
          <ScheduledMessageRow scheduled={scheduled} />
        </ul>
      </TooltipProvider>
    </QueryClientProvider>,
  );

beforeEach(() => {
  vi.mocked(api.post).mockReset();
  vi.mocked(api.post).mockResolvedValue({ data: { id: "m-1", conversation_id: "c-1" } });
});

describe("Send now on a scheduled message", () => {
  it("asks first and posts nothing when the person keeps it scheduled", async () => {
    const user = userEvent.setup();
    renderRow();
    await user.click(screen.getByRole("button", { name: "Send now" }));

    expect(await screen.findByRole("dialog", { name: "Post it now?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Keep it scheduled" }));

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(api.post).not.toHaveBeenCalled();
  });

  it("posts once the person confirms", async () => {
    const user = userEvent.setup();
    renderRow();
    await user.click(screen.getByRole("button", { name: "Send now" }));
    await user.click(await screen.findByRole("button", { name: "Post now" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/chat/scheduled-messages/sm-1/send"));
  });
});

it("names each icon-only action in a tooltip on focus", async () => {
  const user = userEvent.setup();
  renderRow();
  for (const name of ["Edit scheduled message", "Send now", "Cancel scheduled message"]) {
    await user.tab();
    expect(await screen.findByRole("tooltip", { name })).toBeInTheDocument();
  }
});
