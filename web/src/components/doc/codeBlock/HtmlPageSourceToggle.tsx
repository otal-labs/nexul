import { CodeXmlIcon } from "lucide-react";

import { Toggle } from "@/components/ui/toggle";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface HtmlPageSourceToggleProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const HtmlPageSourceToggle = ({ open, onOpenChange }: HtmlPageSourceToggleProps) => {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Toggle
          size="sm"
          pressed={open}
          onPressedChange={onOpenChange}
          aria-label="Show code"
          className="ml-auto size-6 min-w-6 text-muted-foreground hover:text-foreground"
        >
          <CodeXmlIcon className="size-3.5" aria-hidden />
        </Toggle>
      </TooltipTrigger>
      <TooltipContent>{open ? "Hide code" : "Show code"}</TooltipContent>
    </Tooltip>
  );
};
