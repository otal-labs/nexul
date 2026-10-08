import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SetupFlow } from "@/components/setup/SetupFlow";
import { useBootstrapStatus } from "@/hooks/AuthHooks";

// First run, before any user exists. Router.tsx renders it in place of the app while unconfigured, and at /setup.
export const SetupPage = () => {
  const { data: status, isPending, error } = useBootstrapStatus();

  return (
    // No frame of its own: WizardLayout brings the surface and the page gutter.
    <div className="h-full">
      {isPending && <LoadingDisplay />}
      {error && (
        <Container className="py-10">
          <ErrorDisplay error={error} />
        </Container>
      )}
      {status && <SetupFlow status={status} />}
    </div>
  );
};
