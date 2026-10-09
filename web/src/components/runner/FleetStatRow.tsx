import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface FleetStatRowProps {
  online: number;
  busy: number;
  offline: number;
  queued: number;
}

interface StatProps {
  label: string;
  value: number;
  suffix?: string;
  className?: string;
  dot?: string;
}

const Stat = ({ label, value, suffix, className, dot }: StatProps) => (
  <div className="min-w-0 space-y-1.5 px-5 py-4">
    <p className={microheaderClass}>{label}</p>
    <p className={cn("flex items-baseline gap-2 font-mono text-xl font-semibold tabular-nums", className)}>
      {dot && <span aria-hidden className={cn("size-2 shrink-0 self-center rounded-full", dot)} />}
      {value}
      {suffix && <span className="text-sm font-normal text-muted-foreground">{suffix}</span>}
    </p>
  </div>
);

// The same well as a stack's live status. Only online and busy are colored (live signals); the rest stay neutral counts.
export const FleetStatRow = ({ online, busy, offline, queued }: FleetStatRowProps) => (
  <div
    className="grid grid-cols-4 divide-x divide-border rounded-lg bg-surface-2 ring-1 ring-border"
    aria-label="Fleet health summary"
  >
    <Stat label="Online" value={online} suffix={`of ${online + offline}`} dot={online > 0 ? "bg-success" : "bg-muted-foreground"} />
    <Stat label="Busy" value={busy} {...(busy > 0 && { dot: "bg-info animate-[status-pulse_2.4s_ease-standard_infinite]" })} />
    <Stat label="Offline" value={offline} className={cn(offline === 0 && "text-muted-foreground")} />
    <Stat label="Queued" value={queued} />
  </div>
);
