import { Play } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import type { ComputerSetup, SetupRunRow } from "@/models/Pairing";
import { formatRelativeTime } from "@/utils/TimeUtility";

const overall = (setup: ComputerSetup, rows: SetupRunRow[]): { state: SetupRunRow["state"]; title: string } => {
  const running = rows.find((r) => r.state === "running");
  if (running) return { state: "running", title: `Setting up ${running.name}` };
  if (rows.some((r) => r.state === "queued")) return { state: "running", title: "Starting setup" };
  if (setup.confirmed_at) return { state: "confirmed", title: "Setup confirmed" };
  if (rows.some((r) => r.state === "failed")) return { state: "failed", title: "Setup didn't finish" };
  return { state: "queued", title: "Not set up yet" };
};

interface SetupRunHeaderProps {
  setup: ComputerSetup;
  rows: SetupRunRow[];
  busy: boolean;
  blocked: boolean;
  onStart: () => void;
}

// The run at a glance and its one action: brand until the computer is confirmed, then a quiet re-run.
export const SetupRunHeader = ({ setup, rows, busy, blocked, onStart }: SetupRunHeaderProps) => {
  const { state, title } = overall(setup, rows);
  const confirmed = rows.filter((r) => r.state === "confirmed").length;
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex min-w-0 items-start gap-2.5">
        <SetupStateGlyph state={state} className="mt-0.5" />
        <div className="min-w-0">
          <p className="text-sm font-medium">{title}</p>
          <p className="font-mono text-xs text-muted-foreground tabular-nums">
            {rows.length > 0 && <span role="status">{confirmed}/{rows.length} confirmed</span>}
            {rows.length > 0 && setup.confirmed_at && ` · ${formatRelativeTime(setup.confirmed_at)}`}
            {rows.length === 0 && "One short turn per provider"}
          </p>
        </div>
      </div>
      <Button type="button" variant={setup.confirmed_at ? "outline" : "default"} onClick={onStart} loading={busy} disabled={blocked}>
        <Play className="size-4" aria-hidden />
        {setup.turns.length > 0 ? "Re-run setup" : "Start setup"}
      </Button>
    </div>
  );
};
