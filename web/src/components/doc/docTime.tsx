import { daysAgo, formatRelativeTime } from "@/utils/TimeUtility";

// Stays relative under a week; older docs fall back to an absolute date so the rail never guesses.
export function formatUpdatedAgo(iso: string): string {
  const days = daysAgo(iso);
  if (Number.isNaN(days)) return "";
  if (days < 7) return formatRelativeTime(iso);
  return new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}
