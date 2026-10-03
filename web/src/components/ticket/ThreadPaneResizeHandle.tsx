import { PaneResizeHandle } from "@/components/listpane/PaneResizeHandle";
import { clampThreadPaneWidth, THREAD_PANE_MAX, THREAD_PANE_MIN, useThreadPaneStore } from "@/stores/threadPaneStore";

// Sits in the gap right of the thread column; the width lives on the ticket grid, whose first track reads it.
export const ThreadPaneResizeHandle = ({ ticketId }: { ticketId: string }) => {
  const width = useThreadPaneStore((s) => s.widths[ticketId] ?? null);
  const setWidth = useThreadPaneStore((s) => s.setWidth);
  const reset = useThreadPaneStore((s) => s.reset);
  return (
    <PaneResizeHandle
      label="Resize thread"
      width={width}
      min={THREAD_PANE_MIN}
      max={THREAD_PANE_MAX}
      clamp={clampThreadPaneWidth}
      onCommit={(next) => setWidth(ticketId, next)}
      onReset={() => reset(ticketId)}
      variable="--thread-pane-width"
      paneSelector="[data-thread-grid]"
      className="-right-5 @min-[46rem]:block"
    />
  );
};
