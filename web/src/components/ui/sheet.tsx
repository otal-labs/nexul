import { XIcon } from "lucide-react";
import { useRef, type PointerEvent as ReactPointerEvent } from "react";
import { Dialog as SheetPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

function Sheet({ ...props }: React.ComponentProps<typeof SheetPrimitive.Root>) {
  return <SheetPrimitive.Root data-slot="sheet" {...props} />;
}

function SheetTrigger({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Trigger>) {
  return <SheetPrimitive.Trigger data-slot="sheet-trigger" {...props} />;
}

function SheetClose({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Close>) {
  return <SheetPrimitive.Close data-slot="sheet-close" {...props} />;
}

function SheetPortal({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Portal>) {
  return <SheetPrimitive.Portal data-slot="sheet-portal" {...props} />;
}

function SheetOverlay({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Overlay>) {
  return (
    <SheetPrimitive.Overlay
      data-slot="sheet-overlay"
      className={cn(
        "glass-overlay overlay-scrim sheet-scrim fixed inset-0 z-50",
        className,
      )}
      {...props}
    />
  );
}

// A sheet past this share of its width, or flicked faster than this, closes when the pointer lets go.
const DISMISS_SHARE = 0.35;
const DISMISS_SPEED = 0.5;

// Dragging a sheet's header toward its edge moves it with the pointer, 1:1, and lets go into a close or a snap back.
function useSheetDrag(side: "top" | "right" | "bottom" | "left") {
  const drag = useRef<{ x: number; t: number; dx: number; v: number } | null>(null);
  const direction = side === "left" ? -1 : 1;
  const horizontal = side === "left" || side === "right";

  const onPointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement;
    if (!horizontal || event.button !== 0 || !target.closest("[data-slot=sheet-header]") || target.closest("button, a, input, textarea, select")) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    drag.current = { x: event.clientX, t: event.timeStamp, dx: 0, v: 0 };
  };
  const onPointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const d = drag.current;
    if (!d) return;
    const raw = (event.clientX - d.x) * direction;
    // Past its open place the sheet resists instead of stopping dead.
    const dx = raw >= 0 ? raw : -8 * (1 - Math.exp(raw / 40));
    d.v = (dx - d.dx) / Math.max(1, event.timeStamp - d.t);
    d.dx = dx;
    d.t = event.timeStamp;
    const el = event.currentTarget;
    el.dataset.dragging = "";
    el.style.translate = `${dx * direction}px 0`;
    // The scrim fades with the drag, so it is always exactly as gone as the sheet.
    const scrim = el.previousElementSibling as HTMLElement | null;
    if (scrim) scrim.style.opacity = String(1 - Math.max(0, dx) / el.offsetWidth);
  };
  const onPointerUp = (event: ReactPointerEvent<HTMLDivElement>) => {
    const d = drag.current;
    drag.current = null;
    const el = event.currentTarget;
    if (!d || !("dragging" in el.dataset)) return;
    if (d.dx > el.offsetWidth * DISMISS_SHARE || d.v > DISMISS_SPEED) el.querySelector<HTMLElement>("[data-slot=sheet-close]")?.click();
    // The transition takes over from wherever the pointer left the sheet.
    delete el.dataset.dragging;
    el.style.translate = "";
    const scrim = el.previousElementSibling as HTMLElement | null;
    if (scrim) scrim.style.opacity = "";
  };
  return { onPointerDown, onPointerMove, onPointerUp, onPointerCancel: onPointerUp };
}

function SheetContent({
  className,
  children,
  side = "right",
  showCloseButton = true,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Content> & {
  side?: "top" | "right" | "bottom" | "left";
  showCloseButton?: boolean;
}) {
  const drag = useSheetDrag(side);
  return (
    <SheetPortal>
      <SheetOverlay />
      <SheetPrimitive.Content
        data-slot="sheet-content"
        data-side={side}
        {...drag}
        className={cn(
          "sheet-surface glass-popover fixed z-50 flex flex-col gap-4 outline-none",
          side === "right" && "inset-y-2 right-2 w-3/4 rounded-xl sm:max-w-sm",
          side === "left" && "inset-y-2 left-2 w-3/4 rounded-xl sm:max-w-sm",
          side === "top" && "inset-x-2 top-2 h-auto rounded-xl",
          side === "bottom" && "inset-x-2 bottom-2 h-auto rounded-xl",
          className,
        )}
        {...props}
      >
        {children}
        {showCloseButton && (
          <SheetPrimitive.Close
            data-slot="sheet-close"
            className="absolute top-3.5 right-3.5 flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors duration-[120ms] ease-standard hover:bg-accent hover:text-foreground"
          >
            <XIcon className="size-4" />
            <span className="sr-only">Close</span>
          </SheetPrimitive.Close>
        )}
      </SheetPrimitive.Content>
    </SheetPortal>
  );
}

// The header is the sheet's grip on a side sheet.
const horizontalDragHint = "in-data-[side=left]:cursor-grab in-data-[side=right]:cursor-grab in-data-dragging:cursor-grabbing";

function SheetHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-header"
      className={cn("flex flex-col gap-1.5 p-4 pr-12 select-none", horizontalDragHint, className)}
      {...props}
    />
  );
}

function SheetFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-footer"
      className={cn("mt-auto flex flex-col gap-2 p-4", className)}
      {...props}
    />
  );
}

function SheetTitle({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Title>) {
  return (
    <SheetPrimitive.Title
      data-slot="sheet-title"
      className={cn("font-semibold text-foreground", className)}
      {...props}
    />
  );
}

function SheetDescription({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Description>) {
  return (
    <SheetPrimitive.Description
      data-slot="sheet-description"
      className={cn("text-sm text-muted-foreground", className)}
      {...props}
    />
  );
}

export {
  Sheet,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
};
