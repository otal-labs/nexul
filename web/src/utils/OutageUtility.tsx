import type { ServiceOutage } from "@/models/Stack";
import { formatClockTime } from "@/utils/TimeUtility";

// "Down since 10:42 · exit 137 · out of memory"; a kill with no exit seen yet says nothing about one, and the crashes after the first count last.
export const outageText = (outage: ServiceOutage): string => {
  const parts = [`Down since ${formatClockTime(outage.started_at)}`];
  if (outage.exit_code !== 0) parts.push(`exit ${outage.exit_code}`);
  if (outage.oom) parts.push("out of memory");
  if (outage.restart_count > 0) parts.push(`crashed ${outage.restart_count} more ${outage.restart_count === 1 ? "time" : "times"}`);
  return parts.join(" · ");
};
