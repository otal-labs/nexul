import type { ReactElement } from "react";

import { ContextMenu, ContextMenuContent, ContextMenuTrigger } from "@/components/ui/context-menu";
import { RowActionItems } from "@/components/listpane/RowActionItems";
import { contextParts, offersMenu, type RowMenu } from "@/components/listpane/rowMenu";

interface RowContextMenuProps {
  /** One DOM element: it takes the trigger's handlers, and bg-accent while the menu is open. */
  children: ReactElement;
  menu: RowMenu | undefined;
}

// Right-clicking the row opens its ⋯ items at the pointer; a row with nothing to offer keeps the browser's own menu.
export const RowContextMenu = ({ children, menu }: RowContextMenuProps) => (
  <>
    {!offersMenu(menu) && children}
    {offersMenu(menu) && (
      <ContextMenu>
        <ContextMenuTrigger asChild className="data-[state=open]:bg-accent">
          {children}
        </ContextMenuTrigger>
        <ContextMenuContent className="min-w-44">
          <RowActionItems ui={contextParts} {...menu} />
        </ContextMenuContent>
      </ContextMenu>
    )}
  </>
);
