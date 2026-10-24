import { z } from "zod";

import { DEFAULT_REMIND_MINUTES, dayKey, occurrenceTimes, type Meeting, type Repeat } from "@nexul/client-core/calendar";

import { dateInput, endOfDayInstant, instantOf, timeInput } from "@/utils/CalendarUtility";

// The calendar's address, and the search params that open a meeting's panel on it.
export const MEETING_PARAM = "meeting";
export const SLOT_PARAM = "slot";

export const meetingPath = (id: string): string => `/calendar?${MEETING_PARAM}=${encodeURIComponent(id)}`;

export type MeetingScope = "this" | "following";

export const MeetingScopeSchema = z.object({ scope: z.enum(["this", "following"]) });

export type MeetingScopeFormData = z.infer<typeof MeetingScopeSchema>;

export const WHERE_OPTIONS = [
  { value: "nowhere", label: "Nowhere" },
  { value: "channel", label: "Voice channel" },
  { value: "link", label: "Link" },
];

export const MeetingFormSchema = z
  .object({
    title: z.string().trim().min(1, "Add a title"),
    date: z.string().min(1, "Pick a day"),
    start: z.string().min(1, "Pick a start"),
    end: z.string().min(1, "Pick an end"),
    repeats: z.enum(["none", "daily", "weekly", "monthly"]),
    repeat_until: z.string(),
    // A meeting has a voice channel or a link, never both, so the form asks which and sends only that one.
    where: z.enum(["nowhere", "channel", "link"]),
    voice_channel_id: z.string(),
    link: z.string().trim(),
    invitee_ids: z.array(z.string()),
    remind_minutes: z.string(),
    notes: z.string(),
  })
  .superRefine((form, ctx) => {
    if (form.date && form.start && form.end && instantOf(form.date, form.end) <= instantOf(form.date, form.start)) {
      ctx.addIssue({ code: "custom", path: ["end"], message: "End after the start" });
    }
    if (form.repeat_until && form.date && form.repeat_until < form.date) {
      ctx.addIssue({ code: "custom", path: ["repeat_until"], message: "Not before the first day" });
    }
    if (form.where === "channel" && !form.voice_channel_id) {
      ctx.addIssue({ code: "custom", path: ["voice_channel_id"], message: "Pick a channel" });
    }
    if (form.where === "link" && !/^https?:\/\/\S+$/i.test(form.link)) {
      ctx.addIssue({ code: "custom", path: ["link"], message: "Use an http or https link" });
    }
  });

export type MeetingFormData = z.infer<typeof MeetingFormSchema>;

const HOUR_MS = 3_600_000;

// A new meeting starts at the next full hour and runs half an hour.
export const newMeetingDefaults = (now: Date = new Date()): MeetingFormData => {
  const start = new Date(Math.floor(now.getTime() / HOUR_MS) * HOUR_MS + HOUR_MS);
  const end = new Date(start.getTime() + HOUR_MS / 2);
  return {
    title: "",
    date: dayKey(start),
    start: timeInput(start.toISOString()),
    end: timeInput(end.toISOString()),
    repeats: "none",
    repeat_until: "",
    where: "nowhere",
    voice_channel_id: "",
    link: "",
    invitee_ids: [],
    remind_minutes: String(DEFAULT_REMIND_MINUTES),
    notes: "",
  };
};

// An existing meeting's fields, the day and times those of the occurrence being changed.
export const meetingFormDefaults = (meeting: Meeting, slot?: string): MeetingFormData => {
  const times = occurrenceTimes(meeting, slot);
  const where = (): MeetingFormData["where"] => {
    if (meeting.voice_channel_id) return "channel";
    return meeting.link ? "link" : "nowhere";
  };
  return {
    title: meeting.title,
    date: dateInput(times.starts_at),
    start: timeInput(times.starts_at),
    end: timeInput(times.ends_at),
    repeats: meeting.repeats,
    repeat_until: meeting.repeat_until ? dateInput(meeting.repeat_until) : "",
    where: where(),
    voice_channel_id: meeting.voice_channel_id,
    link: meeting.link,
    invitee_ids: meeting.invitees.map((i) => i.user_id),
    remind_minutes: String(meeting.remind_minutes),
    notes: meeting.notes,
  };
};

export interface MeetingBody {
  title: string;
  notes: string;
  starts_at: string;
  ends_at: string;
  timezone?: string;
  voice_channel_id: string;
  link: string;
  repeats?: Repeat;
  repeat_until?: string;
  remind_minutes: number;
  invitee_ids: string[];
}

// create sets the zone the meeting repeats in; "one" is a change to a single occurrence, which cannot repeat.
export type MeetingBodyKind = "create" | "change" | "one";

export const browserTimezone = (): string => Intl.DateTimeFormat().resolvedOptions().timeZone;

export const meetingBody = (form: MeetingFormData, kind: MeetingBodyKind): MeetingBody => ({
  title: form.title,
  notes: form.notes,
  starts_at: instantOf(form.date, form.start),
  ends_at: instantOf(form.date, form.end),
  voice_channel_id: form.where === "channel" ? form.voice_channel_id : "",
  link: form.where === "link" ? form.link : "",
  remind_minutes: Number(form.remind_minutes),
  invitee_ids: form.invitee_ids,
  ...(kind === "create" && { timezone: browserTimezone() }),
  ...(kind !== "one" && { repeats: form.repeats }),
  // An empty end clears it on a change; a new meeting has none to clear.
  ...(kind === "change" && { repeat_until: form.repeat_until ? endOfDayInstant(form.repeat_until) : "" }),
  ...(kind === "create" && !!form.repeat_until && { repeat_until: endOfDayInstant(form.repeat_until) }),
});
