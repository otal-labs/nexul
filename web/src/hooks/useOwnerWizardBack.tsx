import { useNavigate } from "react-router";

import { useOwnerWizardStore } from "@/stores/ownerWizardStore";

// Back from the first owner step returns to sign-in; from the others, to the step before.
export const useOwnerWizardBack = (): (() => void) => {
  const navigate = useNavigate();
  const step = useOwnerWizardStore((s) => s.step);
  const setStep = useOwnerWizardStore((s) => s.setStep);
  return () => {
    if (step === 1) {
      navigate("/login");
      return;
    }
    setStep(step - 1);
  };
};
