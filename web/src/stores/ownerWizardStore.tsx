import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type OwnerWizardStore = {
  step: number;
  // The workspace's slug once step 2 has named it, for the last step's hand-off to the project wizard.
  workspaceSlug: string | null;
  setStep: (step: number) => void;
  setWorkspaceSlug: (slug: string) => void;
  reset: () => void;
};

// Past step 2 the server no longer asks for the wizard, so OnboardingGate reads this to keep the owner in it until the end.
export const OWNER_WIZARD_COMPLETED_STEP = 2;

// Survives step 3's full-SPA reload via sessionStorage, not localStorage — progress is per-tab scratch.
export const useOwnerWizardStore = create<OwnerWizardStore>()(
  persist(
    (set) => ({
      step: 1,
      workspaceSlug: null,
      setStep: (step) => set({ step }),
      setWorkspaceSlug: (workspaceSlug) => set({ workspaceSlug }),
      reset: () => set({ step: 1, workspaceSlug: null }),
    }),
    {
      name: "owner-wizard",
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({ step: state.step, workspaceSlug: state.workspaceSlug }),
    },
  ),
);
