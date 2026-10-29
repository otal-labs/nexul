import { CheckIcon } from "lucide-react";

import { statusStage, type StatusKind } from "@/models/Status";
import type { CommandItem } from "@/models/Command";
import { formatAge } from "@/utils/TimeUtility";
import { cn } from "@/lib/utils";

// A ticket's stage as its hue's dot beside the column's own name, never a tinted chip.
export const CommandStatus = ({ status, className }: { status: NonNullable<CommandItem["status"]>; className?: string }) => (
  <span className={cn("flex min-w-0 items-baseline gap-1.5", className)}>
    <span aria-hidden className={cn("size-1.5 shrink-0 translate-y-[-1px] self-center rounded-full", statusStage(status.kind as StatusKind).dot)} />
    <span className="truncate">{status.label}</span>
  </span>
);

export const CommandAge = ({ at }: { at: string }) => <span className="shrink-0 font-mono tabular-nums">{formatAge(at)}</span>;

export const CommandKey = ({ hint, className }: { hint: string; className?: string }) => (
  <span className={cn("shrink-0 font-mono tabular-nums", className)}>{hint}</span>
);

export const CommandChecked = () => (
  <span className="flex shrink-0 self-center">
    <CheckIcon aria-hidden className="size-3.5" />
    <span className="sr-only">current</span>
  </span>
);
