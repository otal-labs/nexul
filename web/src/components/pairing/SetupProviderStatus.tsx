import { TriangleAlert } from "lucide-react";

import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import { setupProviderLine, type SetupProviderRow } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupProviderStatusProps {
  row: SetupProviderRow;
  id?: string;
  className?: string;
}

// A provider's state as its glyph and line; a failure reads in its hue, and two lines hold a long reason before it clamps.
export const SetupProviderStatus = ({ row, id, className }: SetupProviderStatusProps) => {
  const running = row.state === "running" || row.state === "queued";
  return (
    <p id={id} className={cn("flex min-w-0 items-start gap-1.5 text-xs leading-4 text-muted-foreground", className)}>
      {row.on && row.gone && <TriangleAlert aria-hidden className="size-3.5 shrink-0 text-warning" />}
      {!row.gone && row.on && <SetupStateGlyph state={row.state} className="size-3.5 h-4" />}
      {!row.on && <span aria-hidden className="size-3.5 shrink-0" />}
      <span className={cn("line-clamp-2 min-w-0 break-words", row.on && row.state === "failed" && "text-destructive")} role={running ? "status" : undefined}>
        {setupProviderLine(row)}
      </span>
    </p>
  );
};
