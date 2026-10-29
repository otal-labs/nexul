import { ProfileForm } from "@/components/auth/ProfileForm";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useOwnerWizardBack } from "@/hooks/useOwnerWizardBack";

interface IntroduceYourselfStepProps {
  onContinue: () => void;
}

// Fetches its own user data, like SetupWorkspaceStep, so it only needs onContinue as a prop.
export const IntroduceYourselfStep = ({ onContinue }: IntroduceYourselfStepProps) => {
  const { data, isPending, error } = useFetchMe();
  const back = useOwnerWizardBack();

  return (
    <div className="space-y-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <ProfileForm user={data.user} submitLabel="Continue" onSaved={onContinue} onBack={back} />}
    </div>
  );
};
