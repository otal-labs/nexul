import { ChevronRight, RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EnterList } from "@/components/EnterList";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import { SetupTranscript } from "@/components/pairing/SetupTranscript";
import { SETUP_STATE_LABEL, type SetupRunRow as Row } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupRunRowProps {
  row: Row;
  open: boolean;
  runningName: string | undefined;
  retryDisabled: boolean;
  onToggle: (provider: string) => void;
  onRetry: (provider: string) => void;
}

// One line per provider; the whole line toggles its transcript open under it, and Retry sits above that stretched button.
const SetupRunRow = ({ row, open, runningName, retryDisabled, onToggle, onRetry }: SetupRunRowProps) => (
  <li>
    <div className="relative flex min-h-11 items-center gap-2.5 px-3 py-2 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <ChevronRight
        aria-hidden
        className={cn("size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard", open && "rotate-90")}
      />
      <SetupStateGlyph state={row.state} />
      <button
        type="button"
        aria-expanded={open}
        onClick={() => onToggle(row.provider)}
        className="quiet-focus min-w-0 flex-1 truncate rounded-md text-left text-sm font-medium after:absolute after:inset-0 focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-focus"
      >
        {row.name}
        <span className="sr-only">: {SETUP_STATE_LABEL[row.state]}</span>
      </button>
      {row.model && <span className="hidden max-w-56 truncate font-mono text-xs text-muted-foreground @md:block">{row.model}</span>}
      {row.state !== "failed" && (
        <span className="shrink-0 text-xs text-muted-foreground">
          {SETUP_STATE_LABEL[row.state]}
          {row.kind === "skills" && " · skills update"}
        </span>
      )}
      {row.state === "failed" && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="relative shrink-0"
          aria-label={`Retry ${row.name}`}
          disabled={retryDisabled}
          onClick={() => onRetry(row.provider)}
        >
          <RotateCcw className="size-3.5" aria-hidden />
          Retry
        </Button>
      )}
    </div>
    {open && (
      <div className="settle-in px-3 pb-3 pl-9">
        <SetupTranscript row={row} runningName={runningName} />
      </div>
    )}
  </li>
);

interface SetupRunRowsProps {
  rows: Row[];
  open: string | undefined;
  retryDisabled: boolean;
  onToggle: (provider: string) => void;
  onRetry: (provider: string) => void;
}

// One row per provider, queued, running, confirmed, or failed with Retry; the bar under them fills as providers confirm.
export const SetupRunRows = ({ rows, open, retryDisabled, onToggle, onRetry }: SetupRunRowsProps) => {
  const confirmed = rows.filter((r) => r.state === "confirmed").length;
  const runningName = rows.find((r) => r.state === "running")?.name;
  return (
    <div className="@container overflow-hidden rounded-lg border border-border bg-card">
      <EnterList aria-label="Providers" className="divide-y divide-border">
        {rows.map((row) => (
          <SetupRunRow
            key={row.provider}
            row={row}
            open={row.provider === open}
            runningName={runningName}
            retryDisabled={retryDisabled}
            onToggle={onToggle}
            onRetry={onRetry}
          />
        ))}
      </EnterList>
      <div className="h-0.5 bg-surface-2">
        <span
          className="block h-full w-full origin-left bg-brand transition-transform duration-250 ease-standard grow-in"
          style={{ transform: `scaleX(${rows.length > 0 ? confirmed / rows.length : 0})` }}
        />
      </div>
    </div>
  );
};
