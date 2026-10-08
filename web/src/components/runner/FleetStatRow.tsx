import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface FleetStatRowProps {
  online: number;
  offline: number;
  queued: number;
}

const Stat = ({ label, value, className, dot }: { label: string; value: number; className?: string; dot?: string }) => (
  <div className="min-w-0 space-y-1.5 px-5 py-4">
    <p className={microheaderClass}>{label}</p>
    <p className={cn("flex items-center gap-2 font-mono text-xl font-semibold tabular-nums", className)}>
      {dot && <span aria-hidden className={cn("size-2 rounded-full", dot)} />}
      {value}
    </p>
  </div>
);

// The same well as a stack's live status. Only "online" is colored (a health signal); queued stays a neutral count.
export const FleetStatRow = ({ online, offline, queued }: FleetStatRowProps) => (
  <div className="grid grid-cols-3 divide-x divide-border rounded-lg bg-surface-2 ring-1 ring-border" aria-label="Fleet health summary">
    <Stat label="Online" value={online} dot={online > 0 ? "bg-success" : "bg-muted-foreground"} />
    <Stat label="Offline" value={offline} className="text-muted-foreground" />
    <Stat label="Queued" value={queued} />
  </div>
);
