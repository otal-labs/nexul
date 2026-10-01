import type { KeyboardEvent, PointerEvent } from "react";
import { useRef } from "react";

import { LIST_PANE_MAX, LIST_PANE_MIN, useListPaneStore } from "@/stores/listPaneStore";

const KEY_STEP = 16;

// A drag handle on the list pane's right edge, shown from lg up where the list sits beside the record.
export const ListPaneResizeHandle = () => {
  const width = useListPaneStore((s) => s.width);
  const setWidth = useListPaneStore((s) => s.setWidth);
  const reset = useListPaneStore((s) => s.reset);
  const drag = useRef<{ startX: number; startWidth: number } | null>(null);

  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    // Stops the browser starting a text selection that the drag would then extend.
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    drag.current = { startX: e.clientX, startWidth: width };
  };

  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    if (!drag.current) return;
    setWidth(drag.current.startWidth + e.clientX - drag.current.startX);
  };

  const onPointerEnd = () => {
    drag.current = null;
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
