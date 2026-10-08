import { RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EnterList } from "@/components/EnterList";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import { SETUP_STATE_LABEL, type SetupRunRow as Row } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupRunRowProps {
  row: Row;
  selected: boolean;
  retryDisabled: boolean;
  onSelect: (provider: string) => void;
  onRetry: (provider: string) => void;
}

// One line per provider; the whole row picks it through its stretched button, Retry sits above that overlay, and the reason and model show in the transcript.
const SetupRunRow = ({ row, selected, retryDisabled, onSelect, onRetry }: SetupRunRowProps) => {
  const done = row.state === "confirmed";
  return (
    <li
      className={cn(
        "relative flex items-center gap-2.5 px-3 py-2 transition-colors duration-150 ease-standard hover:bg-accent/40",
        selected && "bg-muted hover:bg-muted",
      )}
    >
      <SetupStateGlyph state={row.state} />
      <div className="min-w-0 flex-1">
        <button
          type="button"
          aria-pressed={selected}
          onClick={() => onSelect(row.provider)}
          className="block w-full rounded-md text-left text-sm break-words outline-none after:absolute after:inset-0 focus-visible:after:ring-2 focus-visible:after:ring-ring/30 focus-visible:after:ring-inset"
        >
          <span className={cn("transition-opacity duration-150 ease-standard", done && "text-muted-foreground line-through opacity-70")}>{row.name}</span>
          <span className="sr-only">: {SETUP_STATE_LABEL[row.state]}</span>
        </button>
      </div>
      {row.state === "failed" && (
        <Button type="button" variant="outline" size="sm" className="relative shrink-0" disabled={retryDisabled} onClick={() => onRetry(row.provider)}>
          <RotateCcw className="size-3.5" aria-hidden />
          Retry
        </Button>
      )}
    </li>
  );
};

interface SetupRunRowsProps {
  rows: Row[];
  selected: string | undefined;
  retryDisabled: boolean;
  onSelect: (provider: string) => void;
  onRetry: (provider: string) => void;
}

// One row per provider: queued, running, confirmed and struck through, or failed with Retry; the picked row fills the transcript.
export const SetupRunRows = ({ rows, selected, retryDisabled, onSelect, onRetry }: SetupRunRowsProps) => {
  const confirmed = rows.filter((r) => r.state === "confirmed").length;
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span id="setup-providers" className="text-xs font-semibold">
          Providers
        </span>
        <span role="status" className="font-mono text-xs text-muted-foreground tabular-nums">
          {confirmed}/{rows.length} confirmed
        </span>
      </div>
      <EnterList aria-labelledby="setup-providers" className="divide-y divide-border">
        {rows.map((row) => (
          <SetupRunRow
            key={row.provider}
            row={row}
            selected={row.provider === selected}
            retryDisabled={retryDisabled}
            onSelect={onSelect}
            onRetry={onRetry}
          />
        ))}
      </EnterList>
      <div className="h-0.5 overflow-hidden rounded-b-lg bg-surface-2">
        <span
          className="block h-full w-full origin-left bg-brand transition-transform duration-250 ease-standard grow-in"
          style={{ transform: `scaleX(${rows.length > 0 ? confirmed / rows.length : 0})` }}
        />
      </div>
    </div>
  );
};
