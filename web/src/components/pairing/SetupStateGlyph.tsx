import { Check, X } from "lucide-react";

import type { SetupRunRow } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupStateGlyphProps {
  state: SetupRunRow["state"];
  className?: string;
}

// Status is the only colour here: success for confirmed, destructive for failed, monochrome while waiting or running.
export const SetupStateGlyph = ({ state, className }: SetupStateGlyphProps) => (
  <span className={cn("grid size-4 shrink-0 place-items-center", className)} aria-hidden>
    {state === "queued" && <span className="size-3.5 rounded-full border border-muted-foreground/50" />}
    {state === "running" && <span className="size-3.5 animate-spin rounded-full motion-reduce:animate-none border-[1.5px] border-foreground/20 border-t-foreground" />}
    {state === "confirmed" && <Check className="size-3.5 text-success" strokeWidth={2.5} />}
    {state === "failed" && <X className="size-3.5 text-destructive" strokeWidth={2.5} />}
  </span>
);
