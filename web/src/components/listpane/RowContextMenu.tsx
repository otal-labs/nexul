import type { ReactElement } from "react";

import { ContextMenu, ContextMenuContent, ContextMenuTrigger } from "@/components/ui/context-menu";
import { RowActionItems } from "@/components/listpane/RowActionItems";
import { contextParts, offersMenu, type RowMenu } from "@/components/listpane/rowMenu";

interface RowContextMenuProps {
  /** One DOM element: it takes the trigger's handlers, and bg-accent while the menu is open. */
  children: ReactElement;
  menu: RowMenu | undefined;
}

// Right-clicking the row opens its items at the pointer; a row with nothing to offer keeps the browser's own menu.
// The trigger stays mounted either way, so the row isn't remounted when the viewer's permissions arrive.
export const RowContextMenu = ({ children, menu }: RowContextMenuProps) => (
  <ContextMenu>
    <ContextMenuTrigger asChild disabled={!offersMenu(menu)} className="data-[state=open]:bg-accent">
      {children}
    </ContextMenuTrigger>
    {offersMenu(menu) && (
      <ContextMenuContent className="min-w-44">
        <RowActionItems ui={contextParts} {...menu} />
      </ContextMenuContent>
    )}
  </ContextMenu>
);
