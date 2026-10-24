import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { NewMeetingButton } from "@/components/calendar/NewMeetingButton";
import { api } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { calendarReads, meetingOf } from "@/test/calendarApi";
import { settle } from "@/test/viewer";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), post: vi.fn() }, errorMessage: (e: Error) => e.message }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const Where = () => <output aria-label="where">{useLocation().search}</output>;

const renderButton = (holds = ["calendar:write"]) => {
  vi.mocked(api.get).mockImplementation(calendarReads(holds, { range: { meetings: [], occurrences: [] }, due: [], meetings: [] }) as never);
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={["/acme/calendar"]}>
        <NewMeetingButton />
        <Where />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const submitted = () => vi.mocked(api.post).mock.calls.at(-1) as [string, Record<string, unknown>];

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(new Date(2026, 10, 4, 8, 20));
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.post).mockReset();
});

afterEach(() => vi.useRealTimers());

describe("the New meeting form", () => {
  it("asks for a title before it saves anything", async () => {
    const user = userEvent.setup();
    renderButton();
    await user.click(await screen.findByRole("button", { name: "New meeting" }));
    await user.click(await screen.findByRole("button", { name: "Create" }));
    expect(await screen.findByText("Add a title")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("starts at the next full hour, and refuses an end before the start", async () => {
    const user = userEvent.setup();
    renderButton();
    await user.click(await screen.findByRole("button", { name: "New meeting" }));
    expect(await screen.findByLabelText("Day")).toHaveValue("2026-11-04");
    expect(screen.getByLabelText("Starts")).toHaveValue("09:00");
    expect(screen.getByLabelText("Ends")).toHaveValue("09:30");

    await user.type(screen.getByLabelText("Title"), "Planning");
    await user.clear(screen.getByLabelText("Ends"));
    await user.type(screen.getByLabelText("Ends"), "08:30");
    await user.click(screen.getByRole("button", { name: "Create" }));
    expect(await screen.findByText("End after the start")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("takes a voice channel or a link, never both", async () => {
    const user = userEvent.setup();
    renderButton();
    await user.click(await screen.findByRole("button", { name: "New meeting" }));
    await user.click(await screen.findByRole("combobox", { name: "Where" }));
    await user.click(await screen.findByRole("option", { name: "Link" }));
    expect(screen.getByLabelText("Link")).toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "Channel" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("combobox", { name: "Where" }));
    await user.click(await screen.findByRole("option", { name: "Voice channel" }));
    expect(screen.queryByLabelText("Link")).not.toBeInTheDocument();
    expect(await screen.findByRole("combobox", { name: "Channel" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("sends the body for what was filled in, then opens the meeting it made", async () => {
    const user = userEvent.setup();
    renderButton();
    vi.mocked(api.post).mockResolvedValue({ data: meetingOf({ id: "m-new", repeats: "weekly" }) });
    await user.click(await screen.findByRole("button", { name: "New meeting" }));
    await user.type(await screen.findByLabelText("Title"), "Planning");
    await user.click(screen.getByRole("combobox", { name: "Where" }));
    await user.click(await screen.findByRole("option", { name: "Voice channel" }));
    await user.click(await screen.findByRole("combobox", { name: "Channel" }));
    await user.click(await screen.findByRole("option", { name: "huddle" }));
    await user.click(screen.getByRole("combobox", { name: "Repeats" }));
    await user.click(await screen.findByRole("option", { name: "Weekly" }));
    await user.click(await screen.findByRole("checkbox", { name: "Bob Baker" }));
    await user.click(screen.getByRole("combobox", { name: "Reminder" }));
    await user.click(await screen.findByRole("option", { name: "30 minutes before" }));
    await user.type(screen.getByLabelText("Notes"), "Agenda");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => expect(api.post).toHaveBeenCalled());
    const [url, body] = submitted();
    expect(url).toBe("/api/workspaces/ws-1/meetings");
    expect(body).toEqual({
      title: "Planning",
      notes: "Agenda",
      starts_at: new Date(2026, 10, 4, 9).toISOString(),
      ends_at: new Date(2026, 10, 4, 9, 30).toISOString(),
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      voice_channel_id: "vc-1",
      link: "",
      repeats: "weekly",
      remind_minutes: 30,
      invitee_ids: ["u-bob"],
    });
    await waitFor(() => expect(screen.getByLabelText("where")).toHaveTextContent(/^\?meeting=m-new$/));
  });

  it("shows the server's refusal and keeps what was typed", async () => {
    const user = userEvent.setup();
    renderButton();
    vi.mocked(api.post).mockRejectedValue(new Error("a meeting has a voice channel or a link, not both"));
    await user.click(await screen.findByRole("button", { name: "New meeting" }));
    await user.type(await screen.findByLabelText("Title"), "Planning");
    await user.click(screen.getByRole("button", { name: "Create" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("a meeting has a voice channel or a link, not both");
    expect(screen.getByLabelText("Title")).toHaveValue("Planning");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });
});

describe("the New meeting button for someone who may not create meetings", () => {
  it("is not drawn", async () => {
    renderButton([]);
    await settle();
    expect(screen.queryByRole("button", { name: "New meeting" })).not.toBeInTheDocument();
  });
});
