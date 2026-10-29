import type { ReactNode } from "react";

import { TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

interface BrowserTabProps {
  value: string;
  pending: boolean;
  children: ReactNode;
}

// A browser-style tab inside a dialog: the selected one joins the panel below it, its label and count share one baseline, and a dot, its slot always kept, marks changes waiting to be saved.
export const BrowserTab = ({ value, pending, children }: BrowserTabProps) => (
  <TabsTrigger
    value={value}
    className="-mb-px h-9 flex-none gap-2 rounded-t-md rounded-b-none border-b-0 px-3.5 after:hidden hover:bg-accent/40 data-[state=active]:border-border! data-[state=active]:bg-popover! data-[state=active]:hover:bg-popover"
  >
    <span className="inline-flex items-baseline gap-2">{children}</span>
    <span aria-hidden className={cn("size-1.5 rounded-full bg-foreground", !pending && "invisible")} />
    {pending && <span className="sr-only">, unsaved changes</span>}
  </TabsTrigger>
);
