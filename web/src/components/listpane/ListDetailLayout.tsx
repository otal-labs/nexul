import type { CSSProperties, ReactNode } from "react";

import { ListPaneResizeHandle } from "@/components/listpane/ListPaneResizeHandle";
import { useListPaneStore } from "@/stores/listPaneStore";
import { cn } from "@/lib/utils";

interface ListDetailLayoutProps {
  list: ReactNode;
  detail: ReactNode;
  hasSelection: boolean;
  /** Shown in the detail pane while nothing is selected, from its top; `ListDetailPlaceholder` centres itself. Left out, the list fills the page until a row is picked (the inbox). */
  placeholder?: ReactNode;
  /** Reserves the record's scrollbar so it never shifts the record; the record then gives that width back from its right padding. */
  stableGutter?: boolean;
}

// The app sidebar, a list, and the open record, for an editor: below lg one pane shows at a time, since the editor needs the width.
export const ListDetailLayout = ({ list, detail, hasSelection, placeholder, stableGutter = false }: ListDetailLayoutProps) => {
  const width = useListPaneStore((s) => s.width);
  const split = hasSelection || placeholder !== undefined;

  return (
    <div data-pane-layout className="flex h-full gap-2">
      {/* The max-width keeps the open record readable whatever width was saved on a wider window; the bottom inset keeps the list's scrollbar off the rounded corner. */}
      <div
        style={{ "--list-pane-width": `${width}px` } as CSSProperties}
        className={cn(
          "panel relative min-h-0 pb-(--radius-xl)",
          split && "lg:block lg:w-[var(--list-pane-width)] lg:max-w-[calc(100%_-_32rem)] lg:shrink-0",
          hasSelection ? "hidden" : "block w-full",
        )}
      >
        {list}
        {split && <ListPaneResizeHandle />}
      </div>
      {/* The inner box scrolls, inset by the panel's radius so its scrollbar clears the rounded corners; relative keeps sr-only file inputs inside the clip. */}
      {/* The clip keeps the record's shadow off the list's edge, where it drew a strip under the resize handle; the ring stays. */}
      <div className={cn("panel min-w-0 flex-1 overflow-hidden [clip-path:inset(-4rem_-4rem_-4rem_-1px)]", split && "lg:block", hasSelection ? "block" : "hidden")}>
        <div className={cn("scroll-edge relative my-(--radius-xl) h-[calc(100%-2*var(--radius-xl))] overflow-y-auto", stableGutter && "[scrollbar-gutter:stable]")}>
          {hasSelection && detail}
          {!hasSelection && <div className="flex min-h-full flex-col">{placeholder}</div>}
        </div>
      </div>
    </div>
  );
};
