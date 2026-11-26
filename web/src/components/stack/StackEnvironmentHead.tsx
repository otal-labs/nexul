import { EyeIcon, EyeOffIcon } from "lucide-react";

import { microheaderClass } from "@/components/Microheader";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// Name, value and the row's last control on one grid shared with the list's column heads.
export const envGridClass = "grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)_2rem] items-center gap-2";

interface StackEnvironmentHeadProps {
  revealed: boolean;
  onReveal: (revealed: boolean) => void;
}

// The column heads, with the reveal toggle over the masked column it acts on.
export const StackEnvironmentHead = ({ revealed, onReveal }: StackEnvironmentHeadProps) => {
  const label = revealed ? "Hide values" : "Reveal values";
  return (
    <div className={`${envGridClass} pb-1.5`}>
      <span className={`${microheaderClass} px-2`}>Name</span>
      <span className={`${microheaderClass} px-3`}>Value</span>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button type="button" variant="ghost" size="icon" aria-label={label} aria-pressed={revealed} onClick={() => onReveal(!revealed)} className="size-8 text-muted-foreground hover:text-foreground">
            {revealed && <EyeOffIcon aria-hidden />}
            {!revealed && <EyeIcon aria-hidden />}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
    </div>
  );
};
