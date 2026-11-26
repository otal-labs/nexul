import type { StackRequests } from "@/models/Stack";
import { formatClockTime } from "@/utils/TimeUtility";
import { minuteAt } from "@/utils/RequestsUtility";

interface RequestBarsProps {
  requests: StackRequests;
}

const SLOT = 10;
const BAR = 7;
const HEIGHT = 64;
const FLOOR = 2;

// One bar a minute on a hairline floor: the answered part in a neutral tone, the 5xx part on top in `warning`; a quiet minute leaves the floor bare, and one nobody counted is a dash on the floor, never a bar.
export const RequestBars = ({ requests }: RequestBarsProps) => {
  const peak = Math.max(1, ...requests.requests.map((r) => r ?? 0));
  const width = requests.requests.length * SLOT;
  return (
    <svg
      role="img"
      aria-label="Requests per minute over the last hour"
      viewBox={`0 0 ${width} ${HEIGHT}`}
      preserveAspectRatio="none"
      className="h-16 w-full"
    >
      <rect x={0} y={HEIGHT - 1} width={width} height={1} className="fill-border" />
      {requests.requests.map((count, i) => {
        const x = i * SLOT + (SLOT - BAR) / 2;
        const time = formatClockTime(minuteAt(requests, i));
        if (count === null) {
          return (
            <rect key={i} x={x} y={HEIGHT - 1} width={BAR} height={1} className="fill-muted-foreground/60">
              <title>{`${time} · no data`}</title>
            </rect>
          );
        }
        if (count === 0) return null;
        const errors = requests.errors_5xx[i] ?? 0;
        const total = Math.max(FLOOR, (count / peak) * HEIGHT);
        const bad = errors > 0 ? Math.max(2, (errors / peak) * HEIGHT) : 0;
        return (
          <g key={i}>
            <title>{`${time} · ${count} requests${errors > 0 ? ` · ${errors} 5xx` : ""}`}</title>
            <rect x={x} y={HEIGHT - total} width={BAR} height={total - bad} className="fill-muted-foreground/60" />
            {bad > 0 && <rect x={x} y={HEIGHT - bad} width={BAR} height={bad} className="fill-warning" />}
          </g>
        );
      })}
    </svg>
  );
};
