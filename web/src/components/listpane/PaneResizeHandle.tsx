import type { KeyboardEvent, PointerEvent } from "react";
import { useRef } from "react";

import { cn } from "@/lib/utils";

const KEY_STEP = 16;

interface PaneResizeHandleProps {
  label: string;
  /** The saved width, or null while the pane still uses its CSS default. */
  width: number | null;
  min: number;
  max: number;
  clamp: (width: number) => number;
  onCommit: (width: number) => void;
  onReset: () => void;
  /** The registered, non-inherited custom property the layout reads the width from. */
  variable: `--${string}`;
  /** Finds the element that carries the variable when it is not the handle's parent. */
  paneSelector?: string;
  /** Position and the breakpoint the handle shows from. */
  className: string;
}

type Drag = { handle: HTMLElement; pane: HTMLElement | null; startX: number; startWidth: number; x: number; frame: number };

// The width the pane shows right now; 0 when unmeasurable (hidden, or a test DOM), so the saved width wins.
const shownWidth = (el: HTMLElement) => el.parentElement?.getBoundingClientRect().width ?? 0;

// A drag handle on a pane's edge: paints the width once a frame while dragging and saves it once, on release.
export const PaneResizeHandle = ({
  label,
  width,
  min,
  max,
  clamp,
  onCommit,
  onReset,
  variable,
  paneSelector,
  className,
}: PaneResizeHandleProps) => {
  const drag = useRef<Drag | null>(null);
  const dragWidth = (d: Drag) => clamp(d.startWidth + d.x - d.startX);
  const paint = (d: Drag) => d.pane?.style.setProperty(variable, `${dragWidth(d)}px`);

  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    // Stops the browser starting a text selection that the drag would then extend.
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    const handle = e.currentTarget;
    const pane = paneSelector ? handle.closest<HTMLElement>(paneSelector) : handle.parentElement;
    // Lets index.css skip re-laying out the open record's off-screen blocks until release.
    pane?.setAttribute("data-resizing", "");
    const startWidth = shownWidth(handle) || (width ?? min);
    drag.current = { handle, pane, startX: e.clientX, startWidth, x: e.clientX, frame: 0 };
  };

  // Pointer events outpace frames and each width change reflows the open record, so paint once a frame and save on release.
  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    const d = drag.current;
    if (!d) return;
    d.x = e.clientX;
    if (d.frame) return;
    d.frame = requestAnimationFrame(() => {
      d.frame = 0;
      paint(d);
    });
  };

  const onPointerEnd = () => {
    const d = drag.current;
    if (!d) return;
    drag.current = null;
    cancelAnimationFrame(d.frame);
    d.pane?.removeAttribute("data-resizing");
    // A plain click must not pin a width the pane never had saved.
    if (d.x === d.startX) return;
    paint(d);
    // The layout may cap the pane below the dragged width (the body keeps a minimum); save what is shown.
    onCommit(Math.min(dragWidth(d), shownWidth(d.handle) || Infinity));
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    const current = shownWidth(e.currentTarget) || (width ?? min);
    const next = { ArrowLeft: current - KEY_STEP, ArrowRight: current + KEY_STEP, Home: min, End: max }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    onCommit(next);
  };

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      aria-valuenow={width ?? undefined}
      aria-valuemin={min}
      aria-valuemax={max}
      tabIndex={0}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerEnd}
      onPointerCancel={onPointerEnd}
      onDoubleClick={onReset}
      onKeyDown={onKeyDown}
      className={cn(
        "absolute inset-y-0 z-10 hidden w-2 cursor-col-resize touch-none select-none outline-none after:absolute after:inset-y-0 after:left-1/2 after:w-px after:-translate-x-1/2 hover:after:bg-muted-foreground focus-visible:after:bg-ring active:after:bg-ring",
        className,
      )}
    />
  );
};
