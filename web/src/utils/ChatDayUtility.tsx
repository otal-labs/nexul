const DAY_MS = 86_400_000;
const startOfDay = (date: Date): number => new Date(date).setHours(0, 0, 0, 0);

// "Today", "Yesterday", then the weekday and date, with the year only once it is not this one.
export const chatDayLabel = (ts: string, now = new Date()): string => {
  const date = new Date(ts);
  const days = Math.round((startOfDay(now) - startOfDay(date)) / DAY_MS);
  if (days === 0) return "Today";
  if (days === 1) return "Yesterday";
  return date.toLocaleDateString([], {
    weekday: "short",
    day: "numeric",
    month: "short",
    ...(date.getFullYear() === now.getFullYear() ? {} : { year: "numeric" }),
  });
};

export const startsDay = (prev: { created_at: string } | undefined, curr: { created_at: string }): boolean =>
  prev === undefined || new Date(prev.created_at).toDateString() !== new Date(curr.created_at).toDateString();
