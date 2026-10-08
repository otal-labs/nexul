import { Check as CheckIcon } from "lucide-react";
import { Checkbox as CheckboxPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

export const Checkbox = ({
  className,
  ...props
}: React.ComponentProps<typeof CheckboxPrimitive.Root>) => (
  <CheckboxPrimitive.Root
    data-slot="checkbox"
    className={cn(
      "peer size-4 shrink-0 rounded-xs border border-input shadow-xs outline-none transition-[color,box-shadow,background-color,border-color] duration-150 ease-standard focus-visible:ring-[3px] focus-visible:ring-ring/30 focus-visible:border-ring disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-brand data-[state=checked]:bg-brand data-[state=checked]:text-brand-foreground",
      className,
    )}
    {...props}
  >
    {/* Always mounted so a box that loads ticked shows still; ticking pops the mark on the small-bounce spring, unticking fades it. */}
    <CheckboxPrimitive.Indicator
      data-slot="checkbox-indicator"
      forceMount
      className="flex items-center justify-center text-current transition-[scale,opacity] duration-[350ms,150ms] ease-spring-pop data-[state=unchecked]:scale-60 data-[state=unchecked]:opacity-0 data-[state=unchecked]:duration-150 data-[state=unchecked]:ease-out"
    >
      <CheckIcon className="size-3.5" />
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
);
