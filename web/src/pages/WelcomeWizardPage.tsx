import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { WelcomeWizard } from "@/components/welcome/WelcomeWizard";
import { useWelcomeProgress } from "@/hooks/WelcomeHooks";

export const WelcomeWizardPage = () => {
  const { data: progress, isPending, error } = useWelcomeProgress();

  return (
    <ShowcaseSurface quiet>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {progress && <WelcomeWizard progress={progress} />}
    </ShowcaseSurface>
  );
};
