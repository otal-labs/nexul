import { createContext, useContext } from "react";

import type { PanelEntity } from "@/models/PermissionPanel";

// The doc, play or project the permissions panel is open on, for every part of the dialog without passing it down.
export const PanelEntityContext = createContext<PanelEntity | null>(null);

export const usePanelEntity = (): PanelEntity => {
  const entity = useContext(PanelEntityContext);
  if (!entity) throw new Error("usePanelEntity must be used inside the permissions panel");
  return entity;
};
