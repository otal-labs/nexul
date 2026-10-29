import { Check, EyeOff, Minus } from "lucide-react";
import { Checkbox as CheckboxPrimitive } from "radix-ui";

import { leadStrip } from "@/components/accessgrid/gridStyles";
import { leadState } from "@/models/AccessGrid";
import { cn } from "@/lib/utils";

interface AccessGridLeadCellProps {
  // How many of the rows shown hold the role (see their channel, on Public channels).
  on: number;
  rows: number;
  marks: "ticks" | "hidden";
  label: string;
  onSetAll: (on: boolean) => void;
}

// Optically on the tally's cap height, which sits a pixel above the box's centre.
const mark = "size-3.5 -translate-y-px text-foreground";

// A role on every row, some or none: a click ticks every row, or clears every row once all hold it.
export const AccessGridLeadCell = ({ on, rows, marks, label, onSetAll }: AccessGridLeadCellProps) => {
  const state = leadState(on, rows);
  const hidden = rows - on;
  return (
    <td className={cn(leadStrip, "relative p-0 [&>input]:top-0")}>
      <CheckboxPrimitive.Root
        checked={state}
        onCheckedChange={() => onSetAll(state !== true)}
        aria-label={label}
        className="group/cell block h-11 w-full cursor-pointer rounded-md p-0.5 outline-none focus-visible:!-outline-offset-2"
      >
        <span className="flex size-full items-center justify-center gap-1.5 rounded-md font-mono text-[11px] text-muted-foreground tabular-nums transition-colors duration-150 ease-standard group-hover/cell:bg-accent">
        {marks === "ticks" && state === true && <Check className={mark} strokeWidth={2.5} aria-hidden />}
        {marks === "ticks" && state === "indeterminate" && <Minus className={mark} strokeWidth={2.5} aria-hidden />}
        {marks === "ticks" && `${on}/${rows}`}
        {marks === "hidden" && hidden > 0 && (
          <>
            <EyeOff className="size-3.5 -translate-y-px" aria-hidden />
            {hidden}
          </>
        )}
        </span>
      </CheckboxPrimitive.Root>
    </td>
  );
};
