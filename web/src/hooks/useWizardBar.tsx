import { createContext, use } from "react";

// The sheet's pinned action bar: a step's footer renders into it, so a long step never pushes its primary out of view.
export const WizardBarContext = createContext<HTMLElement | null>(null);

export const useWizardBar = (): HTMLElement | null => use(WizardBarContext);
