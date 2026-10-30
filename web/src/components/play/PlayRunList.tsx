import type { ReactNode } from "react";

import { PlayRunRow } from "@/components/play/PlayRunRow";
import type { PlayRun } from "@/models/Trail";

interface PlayRunListProps {
  label: string;
  count?: number;
  runs: PlayRun[];
  onOpen: (trailId: string) => void;
  /** Under the rows: the button that reads more. */
  children?: ReactNode;
}

export const PlayRunList = ({ label, count, runs, onOpen, children }: PlayRunListProps) => (
  <div>
    <p className="mb-1 flex items-center gap-1.5 px-3 text-xs text-muted-foreground">
      {label}
      {count !== undefined && <span className="font-mono tabular-nums">{count}</span>}
    </p>
    <ul className="divide-y divide-border overflow-hidden rounded-lg border bg-card">
      {runs.map((run) => (
        <PlayRunRow key={run.id} run={run} onOpen={onOpen} />
      ))}
    </ul>
    {children && <div className="mt-1">{children}</div>}
  </div>
);
