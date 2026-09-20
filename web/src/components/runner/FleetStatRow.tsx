interface FleetStatRowProps {
  online: number;
  offline: number;
  queued: number;
}

// Only "online" is colored (a health signal); queued stays a neutral count, not a value judgement.
export const FleetStatRow = ({ online, offline, queued }: FleetStatRowProps) => (
  <div
    className="grid grid-cols-3 divide-x divide-border rounded-lg border border-border bg-card"
    aria-label="Fleet health summary"
  >
    <div className="space-y-1 px-4 py-3">
      <p className="text-xs text-muted-foreground">Online</p>
      <p className="font-mono text-xl font-semibold tabular-nums text-success">{online}</p>
    </div>
    <div className="space-y-1 px-4 py-3">
      <p className="text-xs text-muted-foreground">Offline</p>
      <p className="font-mono text-xl font-semibold tabular-nums text-muted-foreground">{offline}</p>
    </div>
    <div className="space-y-1 px-4 py-3">
      <p className="text-xs text-muted-foreground">Queued</p>
      <p className="font-mono text-xl font-semibold tabular-nums">{queued}</p>
    </div>
  </div>
);
