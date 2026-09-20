import { useNavigate } from "react-router";

import { IntroduceYourselfStep } from "@/components/auth/IntroduceYourselfStep";
import { WizardLayout } from "@/components/auth/WizardLayout";
import { useCompleteFirstLogin, useFetchMe } from "@/hooks/AuthHooks";

export const FirstLoginWizardPage = () => {
  const navigate = useNavigate();
  const { data } = useFetchMe();
  const complete = useCompleteFirstLogin();

  // IntroduceYourselfStep saves the profile override itself; this just finishes the wizard afterward.
  const onContinue = async () => {
    try {
      await complete.mutateAsync();
      navigate("/", { replace: true });
    } catch {
      // Error is surfaced by the hook's toast; the button stays for a retry.
    }
  };

  return (
    <WizardLayout
      step={{ current: 1, total: 2 }}
      title={data ? `Welcome, ${data.user.name || data.user.login}` : "One quick thing"}
      subtitle="Set the name and avatar others will see — or leave the GitHub defaults."
      onBack={() => navigate("/login")}
    >
      <IntroduceYourselfStep onContinue={() => void onContinue()} />
    </WizardLayout>
  );
};
