import { daysAgo, formatRelativeTime } from "@/utils/TimeUtility";

// Stays relative under a week; older docs fall back to an absolute date so the rail never guesses.
export function formatUpdatedAgo(iso: string): string {
  const days = daysAgo(iso);
  if (Number.isNaN(days)) return "";
  if (days < 7) return formatRelativeTime(iso);
  return new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

// A list's trailing time: the unit without "ago", so a narrow column keeps its room for the title.
export function formatShortAgo(iso: string): string {
  const days = daysAgo(iso);
  if (Number.isNaN(days)) return "";
  if (days >= 7) return new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric" });
  return formatRelativeTime(iso).replace(" ago", "").replace("just now", "now");
}
