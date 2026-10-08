import type { CSSProperties, ReactNode } from "react";

import { ListPaneResizeHandle } from "@/components/listpane/ListPaneResizeHandle";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useListPaneStore } from "@/stores/listPaneStore";
import { cn } from "@/lib/utils";

interface ListDetailLayoutProps {
  list: ReactNode;
  detail: ReactNode;
  hasSelection: boolean;
  /** Shown in the detail pane while nothing is selected. */
  placeholder: string;
}

// The app sidebar, a list, and the open record, for an editor: below lg one pane shows at a time, since the editor needs the width.
export const ListDetailLayout = ({ list, detail, hasSelection, placeholder }: ListDetailLayoutProps) => {
  const width = useListPaneStore((s) => s.width);

  return (
    <div data-pane-layout className="flex h-full gap-2">
      {/* The max-width keeps the open record readable whatever width was saved on a wider window. */}
      <div
        style={{ "--list-pane-width": `${width}px` } as CSSProperties}
        className={cn(
          "panel relative min-h-0 lg:block lg:w-[var(--list-pane-width)] lg:max-w-[calc(100%_-_32rem)] lg:shrink-0",
          hasSelection ? "hidden" : "block w-full",
        )}
      >
        {list}
        <ListPaneResizeHandle />
      </div>
      {/* relative keeps absolutely positioned content (sr-only file inputs) inside the scroll clip instead of stretching the page */}
      <div className={cn("panel relative min-w-0 flex-1 overflow-y-auto lg:block", hasSelection ? "block" : "hidden")}>
        {hasSelection && detail}
        {!hasSelection && (
          <div className="flex h-full items-center justify-center">
            <NoDataDisplay message={placeholder} size="compact" className="border-0" />
          </div>
        )}
      </div>
    </div>
  );
};
