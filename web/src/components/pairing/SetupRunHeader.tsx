import { Play } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import type { SetupProviderRow } from "@/models/Pairing";

const overall = (rows: SetupProviderRow[]): string => {
  const running = rows.filter((r) => r.state === "running" || r.state === "queued");
  if (running.length === 1) return `Setting up ${running[0]?.name}`;
  if (running.length > 1) return `Setting up ${running.length} providers`;
  if (rows.some((r) => r.state === "failed")) return "Setup didn't finish";
  if (rows.length > 0 && rows.every((r) => r.state === "confirmed")) return "Every provider switched on is set up";
  return "Not set up yet";
};

interface SetupRunHeaderProps {
  // The switched-on providers T3 Code still lists: a switched-off or vanished one counts toward nothing.
  rows: SetupProviderRow[];
  // T3 Code's provider list: until it answers there is nothing to count or run.
  providers: "pending" | "error" | "ready";
  // Starting every provider; while one provider runs on its own the action only waits, it doesn't spin.
  busy: boolean;
  running: boolean;
  // Brand until the computer is confirmed; after that Done takes the brand and this goes quiet.
  confirmed: boolean;
  onStart: () => void;
  // Asks T3 Code for its providers again after it couldn't be reached.
  onRetry: () => void;
  retrying: boolean;
}

// The run at a glance, held above the list: how many providers confirmed, what the run is doing, the action that runs them all,
// and a done meter along its bottom edge.
export const SetupRunHeader = ({ rows, providers, busy, running, confirmed: computerConfirmed, onStart, onRetry, retrying }: SetupRunHeaderProps) => {
  const confirmed = rows.filter((r) => r.state === "confirmed").length;
  const done = rows.length > 0 && confirmed === rows.length;
  const counting = providers === "ready" && rows.length > 0;
  return (
    <div className="relative flex min-h-12 flex-wrap items-center justify-between gap-3 pb-5">
      <div role="status" className="flex min-h-10 min-w-0 items-center gap-3">
        {counting && (
          <span className="mt-[3px] text-2xl leading-none font-medium tabular-nums">
            {confirmed}
            <span className="text-muted-foreground">/{rows.length}</span>
          </span>
        )}
        <span className="min-w-0 text-[13px] leading-5 font-medium">
          {counting && overall(rows)}
          {providers === "pending" && (
            <span className="flex items-center gap-2 text-muted-foreground">
              <SetupStateGlyph state="running" />
              Reading T3 Code's providers
            </span>
          )}
          {providers === "error" && <span className="text-destructive">Couldn't reach T3 Code on this computer</span>}
          {providers === "ready" && rows.length === 0 && <span className="text-muted-foreground">Switch on a provider to set it up</span>}
          {counting && <span className="block text-xs font-normal text-muted-foreground">confirmed of the providers switched on</span>}
        </span>
      </div>
      {providers === "error" && (
        <Button type="button" variant="outline" onClick={onRetry} loading={retrying}>
          Try again
        </Button>
      )}
      {providers === "ready" && (
        <Button type="button" variant={computerConfirmed ? "outline" : "default"} onClick={onStart} loading={busy} disabled={rows.length === 0 || running}>
          <Play className="size-4" aria-hidden />
          {done ? "Re-run all" : "Set up all"}
        </Button>
      )}
      <div className="absolute inset-x-0 bottom-0 h-1 overflow-hidden rounded-full bg-foreground/10">
        <span
          className="block h-full w-full origin-left bg-success transition-transform duration-250 ease-standard grow-in"
          style={{ transform: `scaleX(${counting ? confirmed / rows.length : 0})` }}
        />
      </div>
    </div>
  );
};
