// Mirrors internal/calendar: a meeting as the calendar routes return it, and the occurrences a range expands it to.
export type Repeat = "none" | "daily" | "weekly" | "monthly";

export type Reply = "pending" | "accepted" | "declined";

export interface Invitee {
  user_id: string;
  reply: Reply;
  replied_at?: string;
}

export interface Meeting {
  id: string;
  workspace_id: string;
  organiser_id: string;
  title: string;
  notes: string;
  starts_at: string;
  ends_at: string;
  timezone: string;
  voice_channel_id: string;
  link: string;
  repeats: Repeat;
  repeat_until?: string;
  remind_minutes: number;
  // Set on one occurrence of a series changed on its own: the series, and the slot it stands in for.
  series_id?: string;
  replaces_at?: string;
  cancelled_at?: string;
  invitees: Invitee[];
  created_at: string;
  updated_at: string;
}

// One time a meeting takes place; slot names it in a change to that occurrence alone.
export interface Occurrence {
  meeting_id: string;
  slot: string;
  starts_at: string;
  ends_at: string;
}

export interface CalendarRange {
  meetings: Meeting[];
  occurrences: Occurrence[];
}

export type CalendarView = "month" | "week";

export const REPEAT_LABELS: Record<Repeat, string> = { none: "Doesn't repeat", daily: "Daily", weekly: "Weekly", monthly: "Monthly" };

// The reminder choices a meeting offers, in minutes before it starts.
export const REMIND_OPTIONS = [
  { minutes: 0, label: "When it starts" },
  { minutes: 5, label: "5 minutes before" },
  { minutes: 10, label: "10 minutes before" },
  { minutes: 15, label: "15 minutes before" },
  { minutes: 30, label: "30 minutes before" },
  { minutes: 60, label: "1 hour before" },
  { minutes: 1440, label: "1 day before" },
] as const;

export const DEFAULT_REMIND_MINUTES = 10;

// A local date as YYYY-MM-DD, the key a day's cell and a ticket's due date share.
export const dayKey = (d: Date): string =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

const startOfDay = (d: Date): Date => new Date(d.getFullYear(), d.getMonth(), d.getDate());

const addDays = (d: Date, n: number): Date => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);

// The Monday on or before d.
export const startOfWeek = (d: Date): Date => addDays(startOfDay(d), -((d.getDay() + 6) % 7));

// The first and last day a view shows around anchor: a month as whole weeks from Monday, a week as its seven days.
const viewBounds = (view: CalendarView, anchor: Date): { first: Date; last: Date } => {
  const first = view === "week" ? startOfWeek(anchor) : startOfWeek(new Date(anchor.getFullYear(), anchor.getMonth(), 1));
  const last = view === "week" ? addDays(first, 6) : addDays(startOfWeek(new Date(anchor.getFullYear(), anchor.getMonth() + 1, 0)), 6);
  return { first, last };
};

// The days a view shows around anchor.
export const viewDays = (view: CalendarView, anchor: Date): Date[] => {
  const { first, last } = viewBounds(view, anchor);
  const days: Date[] = [];
  for (let d = first; d <= last; d = addDays(d, 1)) {
    days.push(d);
  }
  return days;
};

// The range to ask the calendar routes for: from the first shown day's midnight to the day after the last.
export const viewRange = (view: CalendarView, anchor: Date): { from: Date; to: Date } => {
  const { first, last } = viewBounds(view, anchor);
  return { from: first, to: addDays(last, 1) };
};

// The anchor one view earlier or later.
export const stepAnchor = (view: CalendarView, anchor: Date, by: -1 | 1): Date =>
  view === "week" ? addDays(anchor, 7 * by) : new Date(anchor.getFullYear(), anchor.getMonth() + by, 1);

// Occurrences grouped by the local day each starts on, in start order.
export const occurrencesByDay = (occurrences: Occurrence[]): Map<string, Occurrence[]> => {
  const out = new Map<string, Occurrence[]>();
  const sorted = [...occurrences].sort((a, b) => Date.parse(a.starts_at) - Date.parse(b.starts_at));
  for (const o of sorted) {
    const key = dayKey(new Date(o.starts_at));
    out.set(key, [...(out.get(key) ?? []), o]);
  }
  return out;
};

export interface Replies {
  accepted: Invitee[];
  declined: Invitee[];
  pending: Invitee[];
}

export const groupReplies = (meeting: Meeting): Replies => ({
  accepted: meeting.invitees.filter((i) => i.reply === "accepted"),
  declined: meeting.invitees.filter((i) => i.reply === "declined"),
  pending: meeting.invitees.filter((i) => i.reply === "pending"),
});

export const isCancelled = (meeting: Meeting): boolean => Boolean(meeting.cancelled_at);

export const myReply = (meeting: Meeting, userId: string): Reply | undefined =>
  meeting.invitees.find((i) => i.user_id === userId)?.reply;

// The server's rule for who changes a meeting: its organiser with calendar:write, or anyone with calendar:delete.
export const canChangeMeeting = (meeting: Meeting, userId: string, holds: (action: "calendar:write" | "calendar:delete") => boolean): boolean =>
  (meeting.organiser_id === userId && holds("calendar:write")) || holds("calendar:delete");

// A change to one occurrence of a repeating meeting asks which ones it covers; a meeting that doesn't repeat never asks.
export const asksScope = (meeting: Meeting): boolean => meeting.repeats !== "none";

// Whether the app should tell the server the timezone it runs in: only when it knows one the server doesn't have.
export const shouldReportTimezone = (stored: string | undefined, device: string | undefined): device is string =>
  Boolean(device) && device !== stored;
