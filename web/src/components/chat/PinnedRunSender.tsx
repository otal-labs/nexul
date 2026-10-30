import { useEffect, useRef, useState } from "react";

import type { RunLabel } from "@/utils/ChatRunUtility";

// How far under the scroller's top edge the row being read is looked for: past the 16px fade.
const PROBE_Y = 28;

// The row under the top edge names its run, unless it opened that run and any of its own header line is still on screen.
const pinnedLabel = (viewport: HTMLElement): RunLabel | null => {
  const top = viewport.getBoundingClientRect().top;
  const content = viewport.querySelector("[data-slot=message-scroller-content]");
  if (!content) return null;
  const row = document
    .elementFromPoint(content.getBoundingClientRect().left + content.clientWidth / 2, top + PROBE_Y)
    ?.closest<HTMLElement>("[data-run-name]");
  if (!row) return null;
  const header = row.querySelector("[data-slot=message-header]") ?? row;
  if (row.dataset.runHead !== undefined && header.getBoundingClientRect().bottom > top) return null;
  return { name: row.dataset.runName ?? "", day: row.dataset.runDay ?? "" };
};

// A run scrolled past its header keeps its sender and day pinned over the stream, read from the row under the top edge.
export const PinnedRunSender = () => {
  const ref = useRef<HTMLDivElement>(null);
  const [label, setLabel] = useState<RunLabel | null>(null);

  useEffect(() => {
    const viewport = ref.current?.parentElement?.querySelector<HTMLElement>("[data-slot=message-scroller-viewport]");
    if (!viewport) return;
    let frame = 0;
    const read = () => {
      frame = 0;
      const next = pinnedLabel(viewport);
      setLabel((prev) => (prev?.name === next?.name && prev?.day === next?.day ? prev : next));
    };
    const onScroll = () => {
      if (!frame) frame = requestAnimationFrame(read);
    };
    onScroll();
    viewport.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      viewport.removeEventListener("scroll", onScroll);
      cancelAnimationFrame(frame);
    };
  }, []);

  return (
    // Zero height between the header and the stream, so the pill hangs on the header's hairline over the stream's top fade, never over text.
    <div ref={ref} aria-hidden className="pointer-events-none relative z-10 h-0">
      <div className="absolute inset-x-0 -top-3 mx-auto w-full max-w-3xl px-3">
        {label && (
          <span className="ml-11 inline-flex h-6 max-w-[calc(100%-2.75rem)] items-center gap-1.5 rounded-full bg-card px-2.5 text-xs text-muted-foreground shadow-card ring-1 ring-border">
            <span className="truncate font-semibold text-foreground">{label.name}</span>
            <span aria-hidden>·</span>
            <span className="shrink-0">{label.day}</span>
          </span>
        )}
      </div>
    </div>
  );
};
