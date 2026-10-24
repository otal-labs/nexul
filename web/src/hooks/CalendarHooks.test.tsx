import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Meeting } from "@nexul/client-core/calendar";

import { api } from "@/api/client";
import {
  calendarFollower,
  useCreateMeeting,
  useFetchCalendarRange,
  useFetchDueTickets,
  useFetchMeeting,
  useReplyToMeeting,
  useUpdateMeeting,
} from "@/hooks/CalendarHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), put: vi.fn() },
  errorMessage: (error: Error) => error.message,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const meeting: Meeting = {
  id: "m-1", workspace_id: "ws-1", organiser_id: "u-1", title: "Standup", notes: "", starts_at: "2026-11-02T10:00:00Z", ends_at: "2026-11-02T10:30:00Z",
  timezone: "UTC", voice_channel_id: "", link: "", repeats: "none", remind_minutes: 10, invitees: [], created_at: "", updated_at: "",
};

const newClient = () => new QueryClient({ defaultOptions: { queries: { retry: false } } });

const wrapperFor = (client: QueryClient) => ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => {
  for (const fn of [api.get, api.post, api.patch, api.put, toast.success, toast.error]) vi.mocked(fn).mockReset();
});

describe("the calendar reads", () => {
  it("asks for the range, the due days and the meeting", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { meetings: [], occurrences: [], items: [] } });
    const wrapper = wrapperFor(newClient());
    const range = renderHook(() => useFetchCalendarRange("ws-1", "2026-10-26T00:00:00.000Z", "2026-12-07T00:00:00.000Z"), { wrapper });
    const due = renderHook(() => useFetchDueTickets("ws-1", "2026-10-26", "2026-12-06"), { wrapper });
    const one = renderHook(() => useFetchMeeting("m-1"), { wrapper });
    await waitFor(() => expect([range, due, one].every((r) => r.result.current.isSuccess)).toBe(true));
    expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/meetings", { params: { from: "2026-10-26T00:00:00.000Z", to: "2026-12-07T00:00:00.000Z" } });
    expect(api.get).toHaveBeenCalledWith("/api/tickets/due", { params: { workspace_id: "ws-1", from: "2026-10-26", to: "2026-12-06" } });
    expect(api.get).toHaveBeenCalledWith("/api/meetings/m-1");
    expect(due.result.current.data).toEqual([]);
  });

  it("waits for a workspace and a meeting id", () => {
    renderHook(() => [useFetchCalendarRange("", "a", "b"), useFetchDueTickets("", "a", "b"), useFetchMeeting("")], { wrapper: wrapperFor(newClient()) });
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("the meeting writes", () => {
  it("create posts the body, keeps the meeting and refreshes the calendar", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: meeting });
    const client = seeded([[["getCalendar", "ws-1", "a", "b"], { meetings: [], occurrences: [] }]]);
    const { result } = renderHook(() => useCreateMeeting("ws-1"), { wrapper: wrapperFor(client) });
    await result.current.mutateAsync({ title: "Standup" } as never);
    expect(api.post).toHaveBeenCalledWith("/api/workspaces/ws-1/meetings", { title: "Standup" });
    expect(client.getQueryData(["getMeeting", "m-1"])).toEqual(meeting);
    expect(isStale(client, ["getCalendar", "ws-1", "a", "b"])).toBe(true);
    expect(toast.success).toHaveBeenCalledWith("Meeting created");
  });

  it.each([
    [{ cancelled: true }, "Meeting cancelled"],
    [{ cancelled: false }, "Meeting restored"],
    [{ title: "New" }, "Meeting saved"],
  ])("update patches %j and says %s", async (change, said) => {
    vi.mocked(api.patch).mockResolvedValue({ data: meeting });
    const { result } = renderHook(() => useUpdateMeeting(), { wrapper: wrapperFor(newClient()) });
    await result.current.mutateAsync({ id: "m-1", change });
    expect(api.patch).toHaveBeenCalledWith("/api/meetings/m-1", change);
    expect(toast.success).toHaveBeenCalledWith(said);
  });

  it("update marks the meeting it was asked about stale when the change landed on another", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...meeting, id: "m-2", series_id: "m-1" } });
    const client = seeded([[["getMeeting", "m-1"], meeting]]);
    const { result } = renderHook(() => useUpdateMeeting(), { wrapper: wrapperFor(client) });
    await result.current.mutateAsync({ id: "m-1", change: { slot: "2026-11-09T10:00:00Z", scope: "this" } });
    expect(client.getQueryData(["getMeeting", "m-2"])).toMatchObject({ id: "m-2" });
    expect(isStale(client, ["getMeeting", "m-1"])).toBe(true);
  });

  it("reply puts the answer in and toasts a refusal", async () => {
    vi.mocked(api.put).mockResolvedValue({ data: { ...meeting, invitees: [{ user_id: "u-2", reply: "accepted" }] } });
    const client = newClient();
    const { result } = renderHook(() => useReplyToMeeting(), { wrapper: wrapperFor(client) });
    await result.current.mutateAsync({ id: "m-1", reply: "accepted" });
    expect(api.put).toHaveBeenCalledWith("/api/meetings/m-1/reply", { reply: "accepted" });
    expect(client.getQueryData<Meeting>(["getMeeting", "m-1"])?.invitees[0]?.reply).toBe("accepted");

    vi.mocked(api.put).mockRejectedValue(new Error("not invited"));
    await result.current.mutateAsync({ id: "m-1", reply: "declined" }).catch(() => {});
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("not invited"));
  });
});

describe("the calendar follower", () => {
  const keys = {
    range: ["getCalendar", "ws-1", "a", "b"],
    otherRange: ["getCalendar", "ws-2", "a", "b"],
    open: ["getMeeting", "m-1"],
    other: ["getMeeting", "m-9"],
    ticket: ["getTicket", "t-1"],
    tickets: ["getTicketsByProject", "p-1"],
    people: ["getWorkspacePeople", "ws-1"],
  };
  const seed = () => seeded(Object.values(keys).map((key) => [key, []]));
  const staleOnes = (client: QueryClient) => Object.entries(keys).filter(([, key]) => isStale(client, key)).map(([name]) => name);

  it.each(["meeting.created", "meeting.updated", "meeting.cancelled", "meeting.restored"])("%s refreshes the workspace's calendar and the meeting, and nothing else", async (topic) => {
    const client = seed();
    await followFrame(calendarFollower, topic, { meeting: { id: "m-1", workspace_id: "ws-1" }, user_ids: ["u-1"] }, client);
    expect(staleOnes(client)).toEqual(["range", "open"]);
  });

  it("meeting.replied does the same", async () => {
    const client = seed();
    await followFrame(calendarFollower, "meeting.replied", { meeting_id: "m-1", workspace_id: "ws-1", user_ids: ["u-1"] }, client);
    expect(staleOnes(client)).toEqual(["range", "open"]);
  });

  it("a due date change refreshes only the due days that reach the old or the new day", async () => {
    const reaching = ["getDueTickets", "ws-1", "2026-10-26", "2026-11-01"];
    const later = ["getDueTickets", "ws-1", "2026-11-02", "2026-11-08"];
    const earlier = ["getDueTickets", "ws-1", "2026-10-19", "2026-10-25"];
    const client = seeded([[reaching, []], [later, []], [earlier, []], [keys.tickets, []]]);
    await followFrame(calendarFollower, "ticket.due_date_changed", { ticket: { due_on: "2026-10-30" }, previous_due_on: "" }, client);
    expect([reaching, later, earlier, keys.tickets].map((key) => isStale(client, key))).toEqual([true, false, false, false]);

    await followFrame(calendarFollower, "ticket.due_date_changed", { ticket: { due_on: "" }, previous_due_on: "2026-11-03" }, client);
    expect(isStale(client, later)).toBe(true);
    expect(isStale(client, earlier)).toBe(false);
  });
});
