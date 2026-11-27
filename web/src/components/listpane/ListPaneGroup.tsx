import type { ReactNode } from "react";

import { EnterList } from "@/components/EnterList";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface ListPaneGroupProps {
  label: string;
  count?: number;
  /** Drops the pinned, shaded band and the mono caps: a sentence-case label at the 12px step, for a dialog column. */
  plain?: boolean;
  /** What the heading reads when it says more than the region's name, such as "Your computers · run order". */
  caption?: string;
  children: ReactNode;
}

// The count trails in the rows' meta column, so it never reads as part of the label.
export const ListPaneGroup = ({ label, count, plain = false, caption, children }: ListPaneGroupProps) => (
  <section aria-label={label}>
    <h2 className={cn("flex items-end gap-2 px-3 pb-1.5", plain ? "pt-4 text-xs font-medium text-muted-foreground" : cn(microheaderClass, "sticky-group-head"))}>
      {caption ?? label}
      {count !== undefined && <span className={cn("ml-auto tabular-nums", !plain && "font-mono")}>{count}</span>}
    </h2>
    <EnterList>{children}</EnterList>
  </section>
);
