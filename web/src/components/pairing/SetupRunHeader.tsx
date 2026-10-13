import { Play } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import type { SetupProviderRow } from "@/models/Pairing";

const overall = (rows: SetupProviderRow[]): { state: SetupProviderRow["state"]; title: string } => {
  const running = rows.filter((r) => r.state === "running" || r.state === "queued");
  if (running.length === 1) return { state: "running", title: `Setting up ${running[0]?.name}` };
  if (running.length > 1) return { state: "running", title: `Setting up ${running.length} providers` };
  if (rows.some((r) => r.state === "failed")) return { state: "failed", title: "Setup didn't finish" };
  if (rows.length > 0 && rows.every((r) => r.state === "confirmed")) return { state: "confirmed", title: "Every provider is set up" };
  return { state: "unset", title: "Not set up yet" };
};

interface SetupRunHeaderProps {
  // The switched-on providers only: a switched-off one counts toward nothing.
  rows: SetupProviderRow[];
  // Starting every provider; while one provider runs on its own the action only waits, it doesn't spin.
  busy: boolean;
  running: boolean;
  // Brand until the computer is confirmed; after that Done takes the brand and this goes quiet.
  confirmed: boolean;
  onStart: () => void;
}

// The run at a glance, one bar filling as providers confirm, and the action that runs every switched-on provider at once.
export const SetupRunHeader = ({ rows, busy, running, confirmed: computerConfirmed, onStart }: SetupRunHeaderProps) => {
  const { state, title } = overall(rows);
  const confirmed = rows.filter((r) => r.state === "confirmed").length;
  const done = rows.length > 0 && confirmed === rows.length;
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-start gap-2.5">
          <SetupStateGlyph state={state} className="mt-0.5" />
          <div className="min-w-0">
            <p className="text-sm font-medium">{title}</p>
            <p role="status" className="font-mono text-xs text-muted-foreground tabular-nums">
              {rows.length > 0 && `${confirmed}/${rows.length} confirmed`}
              {rows.length === 0 && "Switch on a provider to set it up"}
            </p>
          </div>
        </div>
        <Button type="button" variant={computerConfirmed ? "outline" : "default"} onClick={onStart} loading={busy} disabled={rows.length === 0 || running}>
          <Play className="size-4" aria-hidden />
          {done ? "Run all again" : "Set up all"}
        </Button>
      </div>
      <div className="h-0.5 overflow-hidden rounded-full bg-surface-2">
        <span
          className="block h-full w-full origin-left bg-brand transition-transform duration-250 ease-standard grow-in"
          style={{ transform: `scaleX(${rows.length > 0 ? confirmed / rows.length : 0})` }}
        />
      </div>
    </div>
  );
};
