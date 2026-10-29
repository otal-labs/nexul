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

// One bar a minute: the answered part in a neutral tone, the 5xx part on top in `warning`; a minute nobody counted is a faint full-height track.
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
      {requests.requests.map((count, i) => {
        const x = i * SLOT + (SLOT - BAR) / 2;
        const time = formatClockTime(minuteAt(requests, i));
        if (count === null) {
          return (
            <rect key={i} x={x} y={0} width={BAR} height={HEIGHT} className="fill-muted-foreground/10">
              <title>{`${time} · no data`}</title>
            </rect>
          );
        }
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
