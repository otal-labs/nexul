import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Meeting } from "@nexul/client-core/calendar";

import { MeetingSheet } from "@/components/calendar/MeetingSheet";
import { api } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { at, calendarReads, meetingOf, VIEWER } from "@/test/calendarApi";
import { settle } from "@/test/viewer";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), patch: vi.fn(), put: vi.fn() }, errorMessage: (e: Error) => e.message }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const SLOT = at(11, 10);

const weekly = meetingOf({ id: "m-w", title: "Standup", repeats: "weekly", voice_channel_id: "vc-1", notes: "Bring **updates**", invitees: [{ user_id: "u-bob", reply: "accepted" }, { user_id: "u-cy", reply: "declined" }, { user_id: VIEWER, reply: "pending" }] });
// Someone else's meeting the viewer is invited to.
const theirs = meetingOf({ id: "m-t", organiser_id: "u-bob", title: "Review", link: "https://meet.example.com/r", invitees: [{ user_id: VIEWER, reply: "accepted" }] });
const cancelled = meetingOf({ id: "m-c", title: "Retro", cancelled_at: at(2, 9) });
const mine = meetingOf({ id: "m-1", title: "Planning" });

const Where = () => {
  const { search } = useLocation();
  return <output aria-label="where">{search}</output>;
};

const renderSheet = (holds: string[], open: Meeting, slot = "") => {
  vi.mocked(api.get).mockImplementation(
    calendarReads(holds, { range: { meetings: [], occurrences: [] }, due: [], meetings: [weekly, theirs, cancelled, mine] }) as never,
  );
  const query = slot ? `&slot=${encodeURIComponent(slot)}` : "";
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={[`/acme/calendar?meeting=${open.id}${query}`]}>
        <MeetingSheet />
        <Where />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const lastPatch = () => vi.mocked(api.patch).mock.calls.at(-1);

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  for (const fn of [api.get, api.patch, api.put]) vi.mocked(fn).mockReset();
});

describe("the meeting panel", () => {
  it("shows what, when, where, the notes and who is coming, declined and silent", async () => {
    renderSheet([], weekly, SLOT);
    expect(await screen.findByRole("heading", { name: "Standup" })).toBeInTheDocument();
    expect(screen.getByText(/10:00–10:30|10:00 AM–10:30 AM/)).toBeInTheDocument();
    expect(screen.getByText("Weekly")).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: /Join huddle/ })).toHaveAttribute("href", "/acme/chat/vc-1");
    expect(screen.getByText("updates").tagName).toBe("STRONG");
    expect(await within(screen.getByRole("region", { name: "Organiser" })).findByText("Alice Archer")).toBeInTheDocument();
    expect(await within(screen.getByRole("region", { name: "Coming" })).findByText("Bob Baker")).toBeInTheDocument();
    expect(await within(screen.getByRole("region", { name: "Declined" })).findByText("Cy Cole")).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "No reply" })).getByText("Alice Archer")).toBeInTheDocument();
  });

  it("opens an external link in a new tab", async () => {
    renderSheet([], theirs);
    expect(await screen.findByRole("link", { name: "Open link" })).toHaveAttribute("target", "_blank");
  });

  it("closes into the calendar's own address", async () => {
    const user = userEvent.setup();
    renderSheet([], mine);
    await screen.findByRole("heading", { name: "Planning" });
    await user.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.getByLabelText("where")).toHaveTextContent(/^$/));
  });
});

describe("answering an invitation", () => {
  it("shows the answer given and sends a new one", async () => {
    const user = userEvent.setup();
    renderSheet([], theirs);
    expect(await screen.findByRole("radio", { name: "Accept" })).toBeChecked();
    expect(screen.getByRole("radio", { name: "Decline" })).not.toBeChecked();

    vi.mocked(api.put).mockResolvedValue({ data: { ...theirs, invitees: [{ user_id: VIEWER, reply: "declined" }] } });
    await user.click(screen.getByRole("radio", { name: "Decline" }));
    expect(api.put).toHaveBeenCalledWith("/api/meetings/m-t/reply", { reply: "declined" });
    await waitFor(() => expect(screen.getByRole("radio", { name: "Decline" })).toBeChecked());
  });

  it("offers no answer to the organiser", async () => {
    renderSheet(["calendar:write"], mine);
    await screen.findByRole("heading", { name: "Planning" });
    expect(screen.queryByRole("radio", { name: "Accept" })).not.toBeInTheDocument();
  });
});

describe("who may change a meeting", () => {
  it.each([
    ["the organiser with calendar:write", ["calendar:write"], mine, true],
    ["the organiser without calendar:write", [], mine, false],
    ["an invitee with calendar:write only", ["calendar:write"], theirs, false],
    ["an invitee with calendar:delete", ["calendar:delete"], theirs, true],
  ])("%s", async (_who, holds, meeting, may) => {
    renderSheet(holds, meeting);
    await screen.findByRole("heading", { name: meeting.title });
    await settle();
    expect(!!screen.queryByRole("button", { name: "Edit" })).toBe(may);
    expect(!!screen.queryByRole("button", { name: "Cancel meeting" })).toBe(may);
  });

  it("offers Restore, not Cancel, on a cancelled meeting, and restores it", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], cancelled);
    vi.mocked(api.patch).mockResolvedValue({ data: { ...cancelled, cancelled_at: undefined } });
    const restore = await screen.findByRole("button", { name: "Restore" });
    expect(screen.queryByRole("button", { name: "Cancel meeting" })).not.toBeInTheDocument();
    await user.click(restore);
    await waitFor(() => expect(lastPatch()).toEqual(["/api/meetings/m-c", { cancelled: false }]));
  });
});

describe("cancelling", () => {
  it("confirms a meeting that doesn't repeat, then cancels it whole", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], mine);
    vi.mocked(api.patch).mockResolvedValue({ data: { ...mine, cancelled_at: at(3, 9) } });
    await user.click(await screen.findByRole("button", { name: "Cancel meeting" }));
    expect(await screen.findByText(/Cancel this meeting\? Everyone invited is told/)).toBeInTheDocument();
    expect(api.patch).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Cancel meeting" }));
    await waitFor(() => expect(lastPatch()).toEqual(["/api/meetings/m-1", { cancelled: true }]));
  });

  it("keeps the meeting when the confirmation is declined", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], mine);
    await user.click(await screen.findByRole("button", { name: "Cancel meeting" }));
    await user.click(await screen.findByRole("button", { name: "Keep" }));
    expect(api.patch).not.toHaveBeenCalled();
  });

  it("asks which meetings of a series, then sends the slot and the scope, and follows the change", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], weekly, SLOT);
    const rest = { ...weekly, id: "m-rest", starts_at: SLOT, cancelled_at: at(3, 9) };
    vi.mocked(api.patch).mockResolvedValue({ data: rest });
    await user.click(await screen.findByRole("button", { name: "Cancel meeting" }));
    expect(await screen.findByRole("radio", { name: "This meeting" })).toBeChecked();
    await user.click(screen.getByRole("radio", { name: "This and following meetings" }));
    await user.click(screen.getByRole("button", { name: "Cancel meeting" }));

    await waitFor(() => expect(lastPatch()).toEqual(["/api/meetings/m-w", { cancelled: true, slot: SLOT, scope: "following" }]));
    await waitFor(() => expect(screen.getByLabelText("where")).toHaveTextContent("meeting=m-rest"));
    expect(screen.getByLabelText("where")).toHaveTextContent(`slot=${encodeURIComponent(SLOT)}`);
  });

  it("cancels one occurrence on its own and moves to the copy that stands in for it", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], weekly, SLOT);
    vi.mocked(api.patch).mockResolvedValue({ data: { ...weekly, id: "m-one", repeats: "none", series_id: "m-w", replaces_at: SLOT, cancelled_at: at(3, 9) } });
    await user.click(await screen.findByRole("button", { name: "Cancel meeting" }));
    await user.click(await screen.findByRole("button", { name: "Cancel meeting" }));
    await waitFor(() => expect(lastPatch()).toEqual(["/api/meetings/m-w", { cancelled: true, slot: SLOT, scope: "this" }]));
    await waitFor(() => expect(screen.getByLabelText("where")).toHaveTextContent(/^\?meeting=m-one$/));
  });

  it("goes back to the panel when the scope question is dismissed", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], weekly, SLOT);
    await user.click(await screen.findByRole("button", { name: "Cancel meeting" }));
    await user.click(await screen.findByRole("button", { name: "Keep" }));
    await waitFor(() => expect(screen.queryByRole("radio", { name: "This meeting" })).not.toBeInTheDocument());
    expect(screen.getByRole("heading", { name: "Standup" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel meeting" })).toBeInTheDocument();
    expect(api.patch).not.toHaveBeenCalled();
  });
});

describe("editing", () => {
  it("asks which meetings of a series, then sends only what the occurrence can take", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], weekly, SLOT);
    vi.mocked(api.patch).mockResolvedValue({ data: { ...weekly, id: "m-one", repeats: "none", title: "Standup (moved)" } });
    await user.click(await screen.findByRole("button", { name: "Edit" }));
    await user.click(await screen.findByRole("button", { name: "Continue" }));

    const title = await screen.findByLabelText("Title");
    expect(title).toHaveValue("Standup");
    expect(screen.queryByRole("combobox", { name: "Repeats" })).not.toBeInTheDocument();
    await user.clear(title);
    await user.type(title, "Standup (moved)");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(api.patch).toHaveBeenCalled());
    const [url, body] = lastPatch() as [string, Record<string, unknown>];
    expect(url).toBe("/api/meetings/m-w");
    expect(body).toMatchObject({ title: "Standup (moved)", slot: SLOT, scope: "this", starts_at: SLOT, ends_at: at(11, 10, 30), voice_channel_id: "vc-1", link: "", remind_minutes: 10, invitee_ids: ["u-bob", "u-cy", VIEWER] });
    expect(body).not.toHaveProperty("repeats");
    expect(body).not.toHaveProperty("timezone");
  });

  it("keeps the repeat fields for this and the following meetings", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], weekly, SLOT);
    await user.click(await screen.findByRole("button", { name: "Edit" }));
    await user.click(await screen.findByRole("radio", { name: "This and following meetings" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("combobox", { name: "Repeats" })).toHaveTextContent("Weekly");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("edits a meeting that doesn't repeat with no slot", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], mine);
    vi.mocked(api.patch).mockResolvedValue({ data: mine });
    await user.click(await screen.findByRole("button", { name: "Edit" }));
    await user.click(await screen.findByRole("button", { name: "Save" }));
    await waitFor(() => expect(api.patch).toHaveBeenCalled());
    const [, body] = lastPatch() as [string, Record<string, unknown>];
    expect(body).not.toHaveProperty("slot");
    expect(body).not.toHaveProperty("scope");
    expect(body).toMatchObject({ repeats: "none", repeat_until: "" });
  });

  it("shows the server's refusal in the dialog", async () => {
    const user = userEvent.setup();
    renderSheet(["calendar:write"], mine);
    vi.mocked(api.patch).mockRejectedValue(new Error("a private channel needs everyone invited"));
    await user.click(await screen.findByRole("button", { name: "Edit" }));
    await user.click(await screen.findByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("a private channel needs everyone invited");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });
});
