import { formatDateParts } from "@/utils/TimeUtility";

const sameDay = (a: Date, b: Date) => a.toDateString() === b.toDateString();

// A play's last run on one scale: "Today", "Yesterday", else the short date ("26 Nov"), with the year once it is not this one.
export const lastRunDate = (iso: string, now = new Date()): string => {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  if (sameDay(date, now)) return "Today";
  if (sameDay(date, new Date(now.getTime() - 24 * 60 * 60 * 1000))) return "Yesterday";
  if (date.getFullYear() !== now.getFullYear()) return formatDateParts(date, { day: "numeric", month: "short", year: "numeric" });
  return formatDateParts(date, { day: "numeric", month: "short" });
};
