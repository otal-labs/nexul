import { Slider as SliderPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

export const Slider = ({
  className,
  "aria-label": ariaLabel,
  ...props
}: React.ComponentProps<typeof SliderPrimitive.Root> & { "aria-label"?: string }) => (
  <SliderPrimitive.Root
    data-slot="slider"
    className={cn("relative flex w-full touch-none items-center select-none data-[disabled]:opacity-50", className)}
    {...props}
  >
    <SliderPrimitive.Track
      data-slot="slider-track"
      className="relative h-1.5 w-full grow overflow-hidden rounded-full bg-muted"
    >
      <SliderPrimitive.Range data-slot="slider-range" className="absolute h-full bg-primary" />
    </SliderPrimitive.Track>
    {/* Radix puts the actual role="slider" element on Thumb, not Root, so the label has to land here or single-thumb sliders are unlabeled for a11y. */}
    <SliderPrimitive.Thumb
      data-slot="slider-thumb"
      aria-label={ariaLabel}
      className="block size-4 shrink-0 rounded-full border border-primary bg-background shadow-xs transition-[box-shadow] duration-150 ease-standard outline-none focus-visible:ring-[3px] focus-visible:ring-ring/30 disabled:pointer-events-none disabled:opacity-50"
    />
  </SliderPrimitive.Root>
);
