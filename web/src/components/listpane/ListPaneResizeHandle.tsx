import type { KeyboardEvent, PointerEvent } from "react";
import { useRef } from "react";

import { clampListPaneWidth, LIST_PANE_MAX, LIST_PANE_MIN, useListPaneStore } from "@/stores/listPaneStore";

const KEY_STEP = 16;

type Drag = { pane: HTMLElement | null; startX: number; startWidth: number; x: number; frame: number };

const dragWidth = (d: Drag) => clampListPaneWidth(d.startWidth + d.x - d.startX);

const paint = (d: Drag) => d.pane?.style.setProperty("--list-pane-width", `${dragWidth(d)}px`);

// A drag handle on the list pane's right edge, shown from lg up where the list sits beside the record.
export const ListPaneResizeHandle = () => {
  const width = useListPaneStore((s) => s.width);
  const setWidth = useListPaneStore((s) => s.setWidth);
  const reset = useListPaneStore((s) => s.reset);
  const drag = useRef<Drag | null>(null);

  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    // Stops the browser starting a text selection that the drag would then extend.
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    const pane = e.currentTarget.parentElement;
    // The open record skips re-laying out its off-screen blocks until release (index.css).
    pane?.setAttribute("data-resizing", "");
    drag.current = { pane, startX: e.clientX, startWidth: width, x: e.clientX, frame: 0 };
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
    paint(d);
    d.pane?.removeAttribute("data-resizing");
    setWidth(dragWidth(d));
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    const next = { ArrowLeft: width - KEY_STEP, ArrowRight: width + KEY_STEP, Home: LIST_PANE_MIN, End: LIST_PANE_MAX }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    setWidth(next);
  };

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize list"
      aria-valuenow={width}
      aria-valuemin={LIST_PANE_MIN}
      aria-valuemax={LIST_PANE_MAX}
      tabIndex={0}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerEnd}
      onPointerCancel={onPointerEnd}
      onDoubleClick={reset}
      onKeyDown={onKeyDown}
      className="absolute inset-y-0 -right-1 z-10 hidden w-2 cursor-col-resize touch-none select-none outline-none after:absolute after:inset-y-0 after:left-1/2 after:w-px after:-translate-x-1/2 hover:after:bg-muted-foreground focus-visible:after:bg-ring active:after:bg-ring lg:block"
    />
  );
};
