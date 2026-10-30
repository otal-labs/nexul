import type { ReactNode } from "react";

import { EnterList } from "@/components/EnterList";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface ListPaneGroupProps {
  label: string;
  count?: number;
  children: ReactNode;
}

export const ListPaneGroup = ({ label, count, children }: ListPaneGroupProps) => (
  <section aria-label={label}>
    <h2 className={cn(microheaderClass, "sticky-group-head flex items-end gap-2 px-3 pb-1.5")}>
      {label}
      {count !== undefined && <span className="font-mono tabular-nums">{count}</span>}
    </h2>
    <EnterList>{children}</EnterList>
  </section>
);
