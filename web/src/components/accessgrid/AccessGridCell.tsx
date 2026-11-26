import { Check, Eye, EyeOff } from "lucide-react";
import { Checkbox as CheckboxPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

interface AccessGridCellProps {
  on: boolean;
  // Unsaved: differs from what the server holds.
  changed: boolean;
  // Public channels lead with the exception: a hidden cell is an ink eye-off, a seen one a quiet eye that still reads as a control.
  marks: "ticks" | "hidden";
  label: string;
  onToggle: () => void;
}

// The whole cell is the box: a tick is a plain ink check and a hidden channel an ink eye-off on a faint tint inset 2px, the ember rings the cell only while its change is unsaved, and focus rings the cell's edge.
export const AccessGridCell = ({ on, changed, marks, label, onToggle }: AccessGridCellProps) => (
  // Inside a form Radix adds a hidden input after the box; pinned to the cell's top, the last row's stays inside the grid's scroll.
  <td className="relative border-b border-border p-0 group-hover/row:bg-accent/40 [&>input]:top-0">
    <CheckboxPrimitive.Root
      checked={on}
      onCheckedChange={onToggle}
      aria-label={label}
      data-changed={changed || undefined}
      className="group/cell block h-11 w-full cursor-pointer rounded-md p-0.5 outline-none focus-visible:!-outline-offset-2"
    >
      <span
        className={cn(
          "grid size-full place-items-center rounded-md transition-[background-color,box-shadow] duration-150 ease-standard group-hover/cell:bg-accent group-data-changed/cell:shadow-[inset_0_0_0_2px_var(--brand)]",
          marks === "hidden" && !on && "bg-foreground/[0.06] group-hover/cell:bg-foreground/10",
        )}
      >
        {marks === "hidden" && on && <Eye className="size-3.5 text-muted-foreground/60 group-hover/cell:text-foreground" aria-hidden />}
        {marks === "hidden" && !on && <EyeOff className="size-3.5 text-foreground" aria-hidden />}
        {marks === "ticks" && <Check className={cn("size-4 text-foreground", !on && "opacity-0 group-hover/cell:opacity-25")} strokeWidth={2.25} aria-hidden />}
      </span>
    </CheckboxPrimitive.Root>
  </td>
);
