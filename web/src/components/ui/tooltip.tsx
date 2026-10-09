import * as React from "react"
import { cn } from "@/lib/utils"
import { Tooltip as TooltipPrimitive } from "radix-ui"

// A tooltip outside a provider gets one of its own; inside one it shares its delays, so moving along a row opens the next at once.
const ProvidedContext = React.createContext(false)

function TooltipProvider({
  delayDuration = 0,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Provider>) {
  return (
    <ProvidedContext.Provider value>
      <TooltipPrimitive.Provider
        data-slot="tooltip-provider"
        delayDuration={delayDuration}
        {...props}
      />
    </ProvidedContext.Provider>
  )
}

function Tooltip({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Root>) {
  const provided = React.useContext(ProvidedContext)
  const root = <TooltipPrimitive.Root data-slot="tooltip" {...props} />
  if (provided) return root
  return <TooltipProvider delayDuration={400}>{root}</TooltipProvider>
}

function TooltipTrigger({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Trigger>) {
  return <TooltipPrimitive.Trigger data-slot="tooltip-trigger" {...props} />
}

// The first tooltip of a run fades in after the delay; the next ones open instantly (data-state="instant-open") while the pointer moves along a row of triggers.
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
          "z-50 w-fit origin-(--radix-tooltip-content-transform-origin) rounded-md bg-popover px-2 py-1 text-xs font-medium text-balance text-popover-foreground shadow-overlay ring-1 ring-border",
          "data-[state=delayed-open]:animate-in data-[state=delayed-open]:fade-in-0 data-[state=delayed-open]:zoom-in-97 data-[state=delayed-open]:duration-120 data-[state=delayed-open]:ease-out data-[side=right]:data-[state=delayed-open]:slide-in-from-left-1 data-[side=bottom]:data-[state=delayed-open]:slide-in-from-top-1 data-[side=top]:data-[state=delayed-open]:slide-in-from-bottom-1",
          "data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:duration-100",
          className
        )}
        {...props}
      >
        {children}
      </TooltipPrimitive.Content>
    </TooltipPrimitive.Portal>
  )
}

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider }
