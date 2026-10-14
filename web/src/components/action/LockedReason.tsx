import type { ReactElement } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface LockedReasonProps {
  // aria-describedby target, so the reason is read on focus before the tooltip opens.
  id: string;
  reason: string;
  title?: string | undefined;
  side?: "top" | "bottom";
  children: ReactElement;
}

export const LockedReason = ({ id, reason, title, side = "bottom", children }: LockedReasonProps) => (
  <>
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent side={side} className="text-left">
        {title && <span className="block font-medium text-foreground">{title}</span>}
        {reason}
      </TooltipContent>
    </Tooltip>
    <span id={id} hidden>
      {reason}
    </span>
  </>
);
