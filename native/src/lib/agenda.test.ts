import type { Meeting, Occurrence } from "@nexul/client-core/calendar";

import { agendaDays, agendaWindow } from "@/lib/agenda";
import type { Ticket } from "@/models/Ticket";

const now = new Date(2026, 9, 24, 15, 30);
const at = (day: number, hour: number) => new Date(2026, 9, day, hour, 0).toISOString();

const meeting = (id: string, extra: Partial<Meeting> = {}): Meeting => ({
  id, workspace_id: "ws-1", organiser_id: "u1", title: id, notes: "", starts_at: at(24, 9), ends_at: at(24, 10), timezone: "UTC",
  voice_channel_id: "", link: "", repeats: "none", remind_minutes: 10, invitees: [], created_at: "", updated_at: "", ...extra,
});

const occurrence = (meetingId: string, day: number, hour: number): Occurrence => ({
  meeting_id: meetingId, slot: at(day, hour), starts_at: at(day, hour), ends_at: at(day, hour + 1),
});

const due = (id: string, dueOn: string): Ticket => ({ id, due_on: dueOn }) as Ticket;

describe("agendaWindow", () => {
  test("runs from this morning's midnight for 30 days, the due range ending on the 30th day itself", () => {
    const window = agendaWindow(now);

    expect(window.from).toBe(new Date(2026, 9, 24).toISOString());
    expect(window.to).toBe(new Date(2026, 10, 23).toISOString());
    expect([window.fromDay, window.toDay]).toEqual(["2026-10-24", "2026-11-22"]);
  });

  test("is the same for any moment of the day, so a query keyed by it does not refetch", () => {
    expect(agendaWindow(new Date(2026, 9, 24, 0, 1))).toEqual(agendaWindow(new Date(2026, 9, 24, 23, 59)));
  });
});

describe("agendaDays", () => {
  test("groups meetings by the local day they start on, in start order, and labels the days from today", () => {
    const range = {
      meetings: [meeting("standup"), meeting("review"), meeting("retro")],
      occurrences: [occurrence("review", 25, 14), occurrence("retro", 28, 9), occurrence("standup", 24, 16), occurrence("standup", 25, 9)],
    };

    const days = agendaDays(range, [], now);

    expect(days.map((d) => [d.label, d.items.map((i) => i.kind === "meeting" && i.meeting.title)])).toEqual([
      ["Today", ["standup"]],
      ["Tomorrow", ["standup", "review"]],
      ["Wed 28 Oct", ["retro"]],
    ]);
  });

  test("puts a day's due tickets first and gives a day with only a due ticket its own entry", () => {
    const range = { meetings: [meeting("standup")], occurrences: [occurrence("standup", 25, 9)] };

    const days = agendaDays(range, [due("t-1", "2026-10-25"), due("t-2", "2026-10-27")], now);

    expect(days.map((d) => [d.key, d.items.map((i) => i.kind)])).toEqual([
      ["2026-10-25", ["due", "meeting"]],
      ["2026-10-27", ["due"]],
    ]);
  });

  test("keeps a cancelled meeting and skips an occurrence whose meeting the range lacks and a ticket with no due day", () => {
    const range = {
      meetings: [meeting("old", { cancelled_at: at(20, 9) })],
      occurrences: [occurrence("old", 25, 9), occurrence("missing", 25, 10)],
    };

    const days = agendaDays(range, [{ id: "t-3" } as Ticket], now);

    expect(days).toHaveLength(1);
    expect(days[0]?.items).toHaveLength(1);
  });

  test("is empty when nothing falls in the window", () => {
    expect(agendaDays({ meetings: [], occurrences: [] }, [], now)).toEqual([]);
  });
});
