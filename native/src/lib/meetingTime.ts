import { formatMessageTime } from "@/lib/time";

// "09:30–10:00" in the phone's own zone.
export const timeRange = (startsAt: string, endsAt: string): string => `${formatMessageTime(startsAt)}–${formatMessageTime(endsAt)}`;

// When the occurrence starting at startsAt ends: the meeting keeps one length for every occurrence of its row.
export const occurrenceEnd = (meeting: { starts_at: string; ends_at: string }, startsAt: string): string =>
  new Date(Date.parse(startsAt) + Date.parse(meeting.ends_at) - Date.parse(meeting.starts_at)).toISOString();
