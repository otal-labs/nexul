import { createContext, useContext, type ReactNode } from "react";

import type { RowAction } from "@/components/settings/RowActionsMenu";

// What a row in Your computers takes from its place in the order: the grip that drags it, and Move up and Move down for
// the keyboard and the menu. A row outside the order gets neither.
export interface ComputerOrder {
  grip: ReactNode;
  moveActions: RowAction[];
}

const ComputerOrderContext = createContext<ComputerOrder>({ grip: null, moveActions: [] });

export const ComputerOrderProvider = ComputerOrderContext.Provider;

export const useComputerOrder = () => useContext(ComputerOrderContext);
