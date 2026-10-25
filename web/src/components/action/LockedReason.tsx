import type { ReactElement } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface LockedReasonProps {
  reason: string;
  title?: string | undefined;
  side?: "top" | "bottom";
  children: ReactElement;
}

// The trigger carries the reason as aria-description until the tooltip opens; once open, the tooltip's aria-label copy is the one description, since Radix merges its id into aria-describedby.
export const LockedReason = ({ reason, title, side = "bottom", children }: LockedReasonProps) => (
  <Tooltip>
    <TooltipTrigger asChild>{children}</TooltipTrigger>
    <TooltipContent side={side} aria-label={reason} className="text-left">
      {title && <span className="block font-medium text-foreground">{title}</span>}
      {reason}
    </TooltipContent>
  </Tooltip>
);
