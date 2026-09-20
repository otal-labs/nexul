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
    <CollapsibleContent className="overflow-hidden ease-out data-[state=closed]:animate-collapsible-up data-[state=open]:animate-collapsible-down">
      <div className="space-y-4 pt-4">{children}</div>
    </CollapsibleContent>
  </Collapsible>
);
