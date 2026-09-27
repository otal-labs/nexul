import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SetupFlow } from "@/components/setup/SetupFlow";
import { useBootstrapStatus } from "@/hooks/AuthHooks";

// First run, before any user exists. Router.tsx renders it in place of the app while unconfigured, and at /setup.
export const SetupPage = () => {
  const { data: status, isPending, error } = useBootstrapStatus();

  return (
    // No padding of its own: WizardLayout brings the page gutter, and 320px has none to spare.
    <Container className="px-0 sm:px-0">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {status && <SetupFlow status={status} />}
    </Container>
  );
};
