import { createContext, use } from "react";
import type { WelcomeStep } from "@nexul/client-core/welcome";

interface RoleDraftContextValue {
  actions: string[];
  loaded: string[];
  setActions: (actions: string[]) => void;
  steps: WelcomeStep[];
  loadedSteps: WelcomeStep[];
  setSteps: (steps: WelcomeStep[]) => void;
  showLocked: boolean;
  loadedShowLocked: boolean;
  setShowLocked: (on: boolean) => void;
}

// The role dialog's held permissions, welcome steps and locked-actions switch, provided by the dialog so its tabs and rows edit them without threading.
export const RoleDraftContext = createContext<RoleDraftContextValue | null>(null);

export const useRoleDraft = (): RoleDraftContextValue => {
  const value = use(RoleDraftContext);
  if (!value) throw new Error("useRoleDraft needs the role dialog around it");
  return value;
};
