import { PaneResizeHandle } from "@/components/listpane/PaneResizeHandle";
import { clampListPaneWidth, LIST_PANE_MAX, LIST_PANE_MIN, useListPaneStore } from "@/stores/listPaneStore";

// A drag handle on the list pane's right edge, shown from lg up where the list sits beside the record.
export const ListPaneResizeHandle = () => {
  const width = useListPaneStore((s) => s.width);
  const setWidth = useListPaneStore((s) => s.setWidth);
  const reset = useListPaneStore((s) => s.reset);
  return (
    <PaneResizeHandle
      label="Resize list"
      width={width}
      min={LIST_PANE_MIN}
      max={LIST_PANE_MAX}
      clamp={clampListPaneWidth}
      onCommit={setWidth}
      onReset={reset}
      variable="--list-pane-width"
      className="-right-1 lg:block"
    />
  );
};
