import { ChevronRight } from "lucide-react";
import type { ReactNode } from "react";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

interface AdvancedFieldsProps {
  children: ReactNode;
}

// Resend-style disclosure: fields most installs never touch stay folded until asked for.
export const AdvancedFields = ({ children }: AdvancedFieldsProps) => (
  <Collapsible>
    <CollapsibleTrigger className="group flex items-center gap-1.5 text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground">
      <ChevronRight
        aria-hidden
        className="size-4 transition-transform duration-200 ease-standard group-data-[state=open]:rotate-90"
      />
      Advanced options
    </CollapsibleTrigger>
    {/* The box snaps open or shut; only the fields fade and settle, so no frame lays out. */}
    <CollapsibleContent className="ease-out data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:duration-150 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:slide-in-from-top-1 data-[state=open]:duration-200">
      <div className="space-y-4 pt-4">{children}</div>
    </CollapsibleContent>
  </Collapsible>
);
