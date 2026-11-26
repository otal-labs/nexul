import { LockIcon } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// Says the doc is read-only; Unlock is in the doc's menu, so the header holds one quiet mark instead of a word and a button.
export const DocLockedSignal = () => (
  <Tooltip>
    <TooltipTrigger asChild>
      <span role="img" aria-label="Locked" tabIndex={0} className="flex size-7 items-center justify-center rounded-md text-muted-foreground">
        <LockIcon className="size-3.5" aria-hidden />
      </span>
    </TooltipTrigger>
    <TooltipContent>Locked</TooltipContent>
  </Tooltip>
);
