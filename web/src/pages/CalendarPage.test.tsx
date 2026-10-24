import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { CalendarPage } from "@/pages/CalendarPage";
import { useCalendarStore } from "@/stores/calendarStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { at, calendarReads, dueTicket, meetingOf, occurrenceOf, type CalendarData } from "@/test/calendarApi";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

const standup = meetingOf();
const cancelled = meetingOf({ id: "m-2", title: "Retro", starts_at: at(5, 15), ends_at: at(5, 16), cancelled_at: "2026-11-02T00:00:00Z" });

const data: CalendarData = {
  range: { meetings: [standup, cancelled], occurrences: [occurrenceOf(standup), occurrenceOf(cancelled)] },
  due: [dueTicket()],
  meetings: [standup, cancelled],
};

const Where = () => <output aria-label="where">{useLocation().search}</output>;

const renderPage = (holds: string[] = ["calendar:write"]) => {
  vi.mocked(api.get).mockImplementation(calendarReads(holds, data) as never);
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/acme/calendar"]}>
        <CalendarPage />
        <Where />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const rangeAsked = () => {
  const asked = vi.mocked(api.get).mock.calls.filter(([url]) => url === "/api/workspaces/ws-1/meetings");
  return asked.map(([, config]) => (config as { params: { from: string; to: string } }).params);
};

beforeEach(() => {
  // Only the clock is held, so the query client's own timers keep running.
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(new Date(2026, 10, 4, 12));
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  useCalendarStore.setState({ view: "month", anchor: new Date(2026, 10, 4) });
});

afterEach(() => vi.useRealTimers());

describe("CalendarPage", () => {
  it("shows the month's meetings by time and title, its due tickets by key, and marks today", async () => {
    renderPage();
    const day = await screen.findByRole("region", { name: /Wednesday 4 November/ });
    expect(await within(day).findByRole("button", { name: /Standup/ })).toBeInTheDocument();
    expect(within(day).getByText("4")).toHaveAttribute("aria-current", "date");

    const due = await screen.findByRole("link", { name: /ATL-7.*Ship it/ });
    expect(due).toHaveAttribute("href", "/acme/tickets/ATL-7");
    expect(within(screen.getByRole("region", { name: /Thursday 5 November/ })).getByRole("link", { name: /ATL-7/ })).toBe(due);
    expect(screen.getByRole("heading", { name: /November 2026/ })).toBeInTheDocument();
  });

  it("keeps a cancelled meeting on the calendar, named as cancelled, and still opens it", async () => {
    const user = userEvent.setup();
    renderPage();
    const retro = await screen.findByRole("button", { name: /Retro.*Cancelled|Cancelled.*Retro/ });
    await user.click(retro);
    expect(screen.getByLabelText("where")).toHaveTextContent("meeting=m-2");
  });

  it("switches between the month and the week, which the store keeps", async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByRole("button", { name: /Standup/ });
    await user.click(screen.getByRole("radio", { name: "Week" }));

    expect(await screen.findByRole("heading", { name: /2 Nov – 8 Nov|8 Nov/ })).toBeInTheDocument();
    expect(useCalendarStore.getState().view).toBe("week");
    const [first, last] = [rangeAsked()[0], rangeAsked().at(-1)];
    expect(first).not.toEqual(last);
    expect(new Date(last?.to ?? "").getTime() - new Date(last?.from ?? "").getTime()).toBe(7 * 24 * 3_600_000);

    await user.click(screen.getByRole("radio", { name: "Month" }));
    expect(await screen.findByRole("heading", { name: /November 2026/ })).toBeInTheDocument();
    expect(useCalendarStore.getState().view).toBe("month");
  });

  it("asks for the next period's range, and its due days, when stepped", async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByRole("button", { name: /Standup/ });
    const before = rangeAsked().length;
    await user.click(screen.getByRole("button", { name: "Next month" }));

    expect(await screen.findByRole("heading", { name: /December 2026/ })).toBeInTheDocument();
    const asked = rangeAsked();
    expect(asked.length).toBeGreaterThan(before);
    expect(new Date(asked.at(-1)?.from ?? "").getTime()).toBe(new Date(2026, 10, 30).getTime());
    expect(vi.mocked(api.get)).toHaveBeenCalledWith("/api/tickets/due", { params: { workspace_id: "ws-1", from: "2026-11-30", to: "2027-01-03" } });

    await user.click(screen.getByRole("button", { name: "Today" }));
    expect(await screen.findByRole("heading", { name: /November 2026/ })).toBeInTheDocument();
  });

  it("opens a busy day's week from its more row", async () => {
    const many = Array.from({ length: 5 }, (_, i) => meetingOf({ id: `m-${i + 10}`, title: `Sync ${i}`, starts_at: at(4, i + 1), ends_at: at(4, i + 1, 30) }));
    data.range = { meetings: many, occurrences: many.map((m) => occurrenceOf(m)) };
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: "2 more" }));
    expect(useCalendarStore.getState().view).toBe("week");
    expect(await screen.findAllByRole("button", { name: /Sync/ })).toHaveLength(5);
  });
});

describe("the New meeting button", () => {
  it("is there for someone who may create meetings", async () => {
    renderPage();
    expect(await screen.findByRole("button", { name: "New meeting" })).toBeEnabled();
  });

  it("is absent for a role that doesn't show locked actions", async () => {
    renderPage([]);
    await screen.findByRole("heading", { name: "Calendar" });
    await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    expect(screen.queryByRole("button", { name: "New meeting" })).not.toBeInTheDocument();
  });
});
