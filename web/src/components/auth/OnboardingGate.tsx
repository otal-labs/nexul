import { type ReactNode } from "react";
import { Navigate } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchMe } from "@/hooks/AuthHooks";

interface OnboardingGateProps {
  children: ReactNode;
}

// Guards the authenticated routes; consumers read useFetchMe directly, never a mirrored sessionStore copy.
export const OnboardingGate = ({ children }: OnboardingGateProps) => {
  const { data, isPending, error } = useFetchMe();

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && data.needs_owner_wizard && <Navigate to="/wizard/onboarding/owner" replace />}
      {data && !data.needs_owner_wizard && data.needs_first_login_wizard && (
        <Navigate to="/wizard/onboarding/profile" replace />
      )}
      {data && !data.needs_owner_wizard && !data.needs_first_login_wizard && children}
    </>
  );
};
