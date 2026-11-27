import type { ReactNode } from "react";

import { EnterList } from "@/components/EnterList";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface ListPaneGroupProps {
  label: string;
  count?: number;
  /** Drops the pinned, shaded band, for a list in a dialog column that scrolls under no fade. */
  plain?: boolean;
  children: ReactNode;
}

// The count trails in the rows' meta column, so it never reads as part of the label.
export const ListPaneGroup = ({ label, count, plain = false, children }: ListPaneGroupProps) => (
  <section aria-label={label}>
    <h2 className={cn(microheaderClass, "flex items-end gap-2 px-3 pb-1.5", plain ? "pt-4" : "sticky-group-head")}>
      {label}
      {count !== undefined && <span className="ml-auto font-mono tabular-nums">{count}</span>}
    </h2>
    <EnterList>{children}</EnterList>
  </section>
);
