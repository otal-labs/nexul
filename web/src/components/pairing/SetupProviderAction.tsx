import { Play, RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { SetupProviderRow } from "@/models/Pairing";

interface SetupProviderActionProps {
  row: SetupProviderRow;
  busy: boolean;
  onRun: (provider: string) => void;
}

// The one thing at a row's end: Set up or Retry while it waits on the person, a quiet Re-run once confirmed, nothing while it runs.
export const SetupProviderAction = ({ row, busy, onRun }: SetupProviderActionProps) => {
  const run = () => onRun(row.provider);
  return (
    <div className="flex justify-end">
      {row.state === "unset" && (
        <Button type="button" variant="outline" size="sm" loading={busy} onClick={run} aria-label={`Set up ${row.name}`}>
          <Play className="size-3.5" aria-hidden />
          Set up
        </Button>
      )}
      {row.state === "failed" && (
        <Button type="button" variant="outline" size="sm" loading={busy} onClick={run} aria-label={`Retry ${row.name}`}>
          <RotateCcw className="size-3.5" aria-hidden />
          Retry
        </Button>
      )}
      {row.state === "confirmed" && (
        <Button type="button" variant="ghost" size="sm" className="text-muted-foreground" loading={busy} onClick={run} aria-label={`Re-run setup for ${row.name}`}>
          Re-run
        </Button>
      )}
    </div>
  );
};
