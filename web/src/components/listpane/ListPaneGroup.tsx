import type { ReactNode } from "react";

import { EnterList } from "@/components/EnterList";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface ListPaneGroupProps {
  label: string;
  children: ReactNode;
}

export const ListPaneGroup = ({ label, children }: ListPaneGroupProps) => (
  <section aria-label={label}>
    <h2 className={cn(microheaderClass, "border-b border-border px-3 pt-4 pb-1.5")}>
      {label}
    </h2>
    <EnterList className="divide-y divide-border">{children}</EnterList>
  </section>
);
