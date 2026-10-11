import { createContext, use } from "react";

interface RoleDraftContextValue {
  actions: string[];
  loaded: string[];
  setActions: (actions: string[]) => void;
}

// The role dialog's held permissions, provided by the dialog so its tabs and rows edit them without threading.
export const RoleDraftContext = createContext<RoleDraftContextValue | null>(null);

export const useRoleDraft = (): RoleDraftContextValue => {
  const value = use(RoleDraftContext);
  if (!value) throw new Error("useRoleDraft needs the role dialog around it");
  return value;
};
