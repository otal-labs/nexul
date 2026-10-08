import { type ReactNode } from "react";
import { Navigate } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchMe } from "@/hooks/AuthHooks";
import { OWNER_WIZARD_COMPLETED_STEP, useOwnerWizardStore } from "@/stores/ownerWizardStore";

interface OnboardingGateProps {
  children: ReactNode;
}

// Guards the authenticated routes; consumers read useFetchMe directly, never a mirrored sessionStore copy.
export const OnboardingGate = ({ children }: OnboardingGateProps) => {
  const { data, isPending, error } = useFetchMe();
  const ownerWizardOpen = useOwnerWizardStore((s) => s.step > OWNER_WIZARD_COMPLETED_STEP);
  const needsOwnerWizard = !!data && (data.needs_owner_wizard || ownerWizardOpen);

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {needsOwnerWizard && <Navigate to="/wizard/onboarding/owner" replace />}
      {data && !needsOwnerWizard && data.needs_first_login_wizard && <Navigate to="/wizard/onboarding/profile" replace />}
      {data && !needsOwnerWizard && !data.needs_first_login_wizard && children}
    </>
  );
};
