import { Check, ChevronRight, RotateCcw, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import type { SetupRunRow as Row } from "@/models/Pairing";
import { cn } from "@/lib/utils";

const EMPTY: string[] = [];

interface SetupCommentaryProps {
  turnId: string;
}

// The agent's steps for the running turn, folded by default and set quieter than the rows above it.
const SetupCommentary = ({ turnId }: SetupCommentaryProps) => {
  const lines = useSetupActivityStore((s) => s.lines[turnId] ?? EMPTY);
  if (lines.length === 0) return null;
  return (
    <Collapsible className="mt-1.5">
      <CollapsibleTrigger className="group flex items-center gap-1 rounded-sm text-[11px] text-muted-foreground outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/30">
        <ChevronRight className="size-3 transition-transform duration-150 ease-standard group-data-[state=open]:rotate-90" aria-hidden />
        Agent steps ({lines.length})
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ul className="mt-1 space-y-0.5 border-l border-border pl-2.5" aria-live="polite">
          {lines.map((line, i) => (
            <li key={i} className="font-mono text-[11px] break-words text-muted-foreground">
              {line}
            </li>
          ))}
        </ul>
      </CollapsibleContent>
    </Collapsible>
  );
};

interface StateGlyphProps {
  state: Row["state"];
}

// Status is the only colour here: success for confirmed, destructive for failed, monochrome while waiting or running.
const StateGlyph = ({ state }: StateGlyphProps) => (
  <span className="mt-0.5 grid size-4 shrink-0 place-items-center" aria-hidden>
    {state === "queued" && <span className="size-3.5 rounded-full border border-muted-foreground/50" />}
    {state === "running" && <span className="size-3.5 animate-spin rounded-full motion-reduce:animate-none border-[1.5px] border-foreground/20 border-t-foreground" />}
    {state === "confirmed" && <Check className="size-3.5 text-success" strokeWidth={2.5} />}
    {state === "failed" && <X className="size-3.5 text-destructive" strokeWidth={2.5} />}
  </span>
);

const STATE_LABEL: Record<Row["state"], string> = {
  queued: "Waiting",
  running: "Running",
  confirmed: "Confirmed",
  failed: "Failed",
};

interface SetupRunRowProps {
  row: Row;
  index: number;
  retryDisabled: boolean;
  onRetry: (provider: string) => void;
}

const SetupRunRow = ({ row, index, retryDisabled, onRetry }: SetupRunRowProps) => {
  const done = row.state === "confirmed";
  return (
    <li
      className="flex animate-in items-start gap-2.5 px-3 py-2.5 fill-mode-backwards fade-in-0 slide-in-from-bottom-1 duration-200 ease-out"
      style={{ animationDelay: `${Math.min(index, 7) * 25}ms` }}
    >
      <StateGlyph state={row.state} />
      <div className="min-w-0 flex-1">
        <p className={cn("text-sm break-words transition-opacity duration-150 ease-standard", done && "text-muted-foreground line-through opacity-70")}>
          {row.name}
          <span className="sr-only">: {STATE_LABEL[row.state]}</span>
        </p>
        <p className="text-xs break-words text-muted-foreground">{row.status}</p>
        {row.state === "running" && row.turnId && <SetupCommentary turnId={row.turnId} />}
      </div>
      {row.state === "failed" && (
        <Button type="button" variant="outline" size="sm" className="shrink-0" disabled={retryDisabled} onClick={() => onRetry(row.provider)}>
          <RotateCcw className="size-3.5" aria-hidden />
          Retry
        </Button>
      )}
    </li>
  );
};

interface SetupRunRowsProps {
  rows: Row[];
  retryDisabled: boolean;
  onRetry: (provider: string) => void;
}

// One live row per provider: queued, running with its commentary, confirmed and struck through, or failed with Retry.
export const SetupRunRows = ({ rows, retryDisabled, onRetry }: SetupRunRowsProps) => {
  const confirmed = rows.filter((r) => r.state === "confirmed").length;
  return (
    <div className="rounded-lg border border-border bg-card" role="status" aria-live="polite">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-xs font-semibold">Providers</span>
        <span className="font-mono text-[11px] text-muted-foreground tabular-nums">
          {confirmed}/{rows.length} confirmed
        </span>
      </div>
      <ul className="divide-y divide-border">
        {rows.map((row, i) => (
          <SetupRunRow key={row.provider} row={row} index={i} retryDisabled={retryDisabled} onRetry={onRetry} />
        ))}
      </ul>
      <div className="h-0.5 overflow-hidden rounded-b-lg bg-surface-2">
        <span
          className="block h-full w-full origin-left bg-foreground/60 transition-transform duration-300 ease-standard"
          style={{ transform: `scaleX(${rows.length > 0 ? confirmed / rows.length : 0})` }}
        />
      </div>
    </div>
  );
};
