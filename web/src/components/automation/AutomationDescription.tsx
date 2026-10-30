import { useState } from "react";

import { markClamped } from "@/utils/ClampUtility";
import { cn } from "@/lib/utils";

interface AutomationDescriptionProps {
  description: string;
}

// Two lines under the title, with More only when the text runs past them.
export const AutomationDescription = ({ description }: AutomationDescriptionProps) => {
  const [open, setOpen] = useState(false);
  return (
    <div className="mt-1.5 max-w-3xl text-sm text-muted-foreground">
      <div ref={markClamped} className={cn("peer overflow-hidden data-clamped:title-clamp-fade", !open && "max-h-[2lh]")}>
        <p className="text-pretty">{description}</p>
      </div>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className={cn(
          "hidden text-xs transition-colors duration-150 ease-standard peer-data-clamped:inline-flex hover:text-foreground focus-visible:text-foreground",
          open && "inline-flex",
        )}
      >
        {open ? "Less" : "More"}
      </button>
    </div>
  );
};
