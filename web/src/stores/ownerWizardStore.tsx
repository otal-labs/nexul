import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import type { WorkspaceSetupData } from "@/components/auth/SetupWorkspaceStep";

export type OwnerWizardStore = {
  step: number;
  // Held here and applied right after CompleteOwnerWizard grants the permission needed to apply it.
  workspaceSetup: WorkspaceSetupData | null;
  setStep: (step: number) => void;
  setWorkspaceSetup: (data: WorkspaceSetupData) => void;
  reset: () => void;
};

// Survives step 3's full-SPA reload via sessionStorage, not localStorage — progress is per-tab scratch.
export const useOwnerWizardStore = create<OwnerWizardStore>()(
  persist(
    (set) => ({
      step: 1,
      workspaceSetup: null,
      setStep: (step) => set({ step }),
      setWorkspaceSetup: (workspaceSetup) => set({ workspaceSetup }),
      reset: () => set({ step: 1, workspaceSetup: null }),
    }),
    {
      name: "owner-wizard",
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({ step: state.step, workspaceSetup: state.workspaceSetup }),
    },
  ),
);
