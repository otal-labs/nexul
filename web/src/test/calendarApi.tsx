import type { CalendarRange, Meeting, Occurrence } from "@nexul/client-core/calendar";

import type { Ticket } from "@/models/Ticket";
import { asViewer, memberWho } from "@/test/viewer";

// The viewer in test/viewer.ts is "u-viewer"; a meeting they organise names that id.
export const VIEWER = "u-viewer";

// A moment on a November 2026 day in the reader's own zone, which is where the calendar places it.
export const at = (day: number, hour: number, minute = 0): string => new Date(2026, 10, day, hour, minute).toISOString();

export const meetingOf = (over: Partial<Meeting> = {}): Meeting => ({
  id: "m-1",
  workspace_id: "ws-1",
  organiser_id: VIEWER,
  title: "Standup",
  notes: "",
  starts_at: at(4, 10),
  ends_at: at(4, 10, 30),
  timezone: "UTC",
  voice_channel_id: "",
  link: "",
  repeats: "none",
  remind_minutes: 10,
  invitees: [],
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
  ...over,
});

export const occurrenceOf = (meeting: Meeting, startsAt = meeting.starts_at): Occurrence => ({
  meeting_id: meeting.id,
  slot: startsAt,
  starts_at: startsAt,
  ends_at: new Date(Date.parse(startsAt) + Date.parse(meeting.ends_at) - Date.parse(meeting.starts_at)).toISOString(),
});

export const dueTicket = (over: Partial<Ticket> = {}): Ticket =>
  ({ id: "t-1", project_id: "p-1", number: 7, title: "Ship it", due_on: "2026-11-05", ...over }) as Ticket;

export interface CalendarData {
  range: CalendarRange;
  due: Ticket[];
  // Meetings the panel opens by id; the range's are included.
  meetings: Meeting[];
}

const person = (id: string, name: string) => ({ user_id: id, login: id, display_name: name, avatar_url: "" });

export const people = [person(VIEWER, "Alice Archer"), person("u-bob", "Bob Baker"), person("u-cy", "Cy Cole")];

// Answers the reads the calendar page and its panel make: the viewer's own permissions first, then the calendar's.
export const calendarReads = (permissions: string[], data: CalendarData) =>
  asViewer(memberWho(permissions, false), (url) => {
    if (url === "/api/workspaces/ws-1/meetings") return data.range;
    if (url === "/api/tickets/due") return { items: data.due };
    if (url.startsWith("/api/meetings/")) return data.meetings.find((m) => url === `/api/meetings/${m.id}`);
    if (url === "/api/projects") return [{ id: "p-1", prefix: "ATL", name: "Atlas" }];
    if (url === "/api/chat/conversations") return [{ id: "vc-1", workspace_id: "ws-1", kind: "voice_channel", name: "huddle" }];
    if (url === "/api/workspaces/ws-1/people") return { people };
    return [];
  });
