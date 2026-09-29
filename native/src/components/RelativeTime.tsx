import { formatRelativeTime } from "@/lib/time";
import { useClockStore } from "@/stores/clockStore";

// The one way a screen shows an age: it re-renders on the app-wide minute tick, so it never freezes at first render.
export const RelativeTime = ({ iso }: { iso: string }) => formatRelativeTime(iso, useClockStore((s) => s.now));
