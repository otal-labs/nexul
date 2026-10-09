import { createContext, useContext } from "react";
import { Tooltip as TooltipPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

const InsideProvider = createContext(false);

// The first tooltip waits half a second; while one is open, or for 300ms after, the next shows at once and unanimated.
function TooltipProvider({
  delayDuration = 500,
  skipDelayDuration = 300,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Provider>) {
  return (
    <InsideProvider value>
      <TooltipPrimitive.Provider
        data-slot="tooltip-provider"
        delayDuration={delayDuration}
        skipDelayDuration={skipDelayDuration}
        {...props}
      />
    </InsideProvider>
  );
}

// The app's one provider groups every tooltip; a tooltip rendered outside it (a component under test) brings its own.
function Tooltip({ ...props }: React.ComponentProps<typeof TooltipPrimitive.Root>) {
  const inside = useContext(InsideProvider);
  const root = <TooltipPrimitive.Root data-slot="tooltip" {...props} />;
  return (
    <>
      {inside && root}
      {!inside && <TooltipProvider>{root}</TooltipProvider>}
    </>
  );
}

function TooltipTrigger({ ...props }: React.ComponentProps<typeof TooltipPrimitive.Trigger>) {
  return <TooltipPrimitive.Trigger data-slot="tooltip-trigger" {...props} />;
}

function TooltipContent({
  className,
  sideOffset = 6,
  children,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Content>) {
  return (
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Content
        data-slot="tooltip-content"
        sideOffset={sideOffset}
        className={cn(
          "tooltip-surface glass-menu z-50 w-fit max-w-72 origin-(--radix-tooltip-content-transform-origin) rounded-md px-2 py-1 text-xs text-balance text-popover-foreground",
          className,
        )}
        {...props}
      >
        {children}
      </TooltipPrimitive.Content>
    </TooltipPrimitive.Portal>
  );
}

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider };
