import type { Deploy } from "@/models/Stack";
import { daysAgo, formatRelativeTime as formatRelativeTimeCore } from "@/utils/TimeUtility";

// Extends the shared TimeUtility core with a months/years tail past 30 days.
export const formatRelativeTime = (ts: string): string => {
  const days = daysAgo(ts);
  if (Number.isNaN(days)) return ts;
  if (days < 30) return formatRelativeTimeCore(ts);
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
};

// UTC day boundaries, deliberate: local-calendar grouping would flip "Today"/"Yesterday" per viewer timezone.
const startOfUtcDay = (d: Date): number =>
  Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate());
const MS_PER_DAY = 86_400_000;

// Renders the deploy timeline's group heading: "Today"/"Yesterday"/a date.
export const formatDayLabel = (ts: string): string => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return "Unknown date";
  const now = new Date();
  const diffDays = Math.round((startOfUtcDay(now) - startOfUtcDay(date)) / MS_PER_DAY);
  if (diffDays === 0) return "Today";
  if (diffDays === 1) return "Yesterday";
  return date.toLocaleDateString("en-US", {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
    year: date.getUTCFullYear() === now.getUTCFullYear() ? undefined : "numeric",
  });
};

export interface DeployDayGroup {
  label: string;
  deploys: Deploy[];
}

// Assumes deploys already arrive newest-first; this is presentation grouping only, not a re-sort.
export const groupDeploysByDay = (deploys: Deploy[]): DeployDayGroup[] => {
  const groups: DeployDayGroup[] = [];
  for (const deploy of deploys) {
    const label = formatDayLabel(deploy.created_at);
    const current = groups.at(-1);
    if (current && current.label === label) {
      current.deploys.push(deploy);
      continue;
    }
    groups.push({ label, deploys: [deploy] });
  }
  return groups;
};
