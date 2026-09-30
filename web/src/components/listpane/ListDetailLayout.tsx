import type { ReactNode } from "react";

import { NoDataDisplay } from "@/components/NoDataDisplay";
import { cn } from "@/lib/utils";

interface ListDetailLayoutProps {
  list: ReactNode;
  detail: ReactNode;
  hasSelection: boolean;
  /** Shown in the detail pane while nothing is selected. */
  placeholder: string;
}

// The app sidebar, a list, and the open record, for an editor: below lg one pane shows at a time, since the editor needs the width.
export const ListDetailLayout = ({ list, detail, hasSelection, placeholder }: ListDetailLayoutProps) => (
  <div className="flex h-screen">
    <div
      className={cn(
        "min-h-0 lg:block lg:w-[300px] lg:shrink-0 lg:border-r lg:border-border",
        hasSelection ? "hidden" : "block w-full",
      )}
    >
      {list}
    </div>
    <div className={cn("min-w-0 flex-1 overflow-y-auto lg:block", hasSelection ? "block" : "hidden")}>
      {hasSelection && detail}
      {!hasSelection && (
        <div className="flex h-full items-center justify-center">
          <NoDataDisplay message={placeholder} size="compact" className="border-0" />
        </div>
      )}
    </div>
  </div>
);
