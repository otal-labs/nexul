import { ConnectCloudflareFirst } from "@/components/dns/ConnectCloudflareFirst";
import { DnsSetupWizard } from "@/components/dns/DnsSetupWizard";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";
import { useActionState } from "@/hooks/useActionState";

// DNS changes after first run (the domain itself is set up before sign-in); gates only on the Cloudflare connection.
export const DnsOnboardingPage = () => {
  const { data, isPending, error } = useFetchConnectorStatus("cloudflare");
  const connected = data?.status.configured ?? false;
  const canWrite = useActionState({ permission: "dns:write" }).state === "live";

  return (
    <ShowcaseSurface quiet>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && !connected && <DnsSetupWizard gate={<ConnectCloudflareFirst />} />}
      {data && connected && canWrite && <DnsSetupWizard />}
      {data && connected && !canWrite && <DnsSetupWizard gate={<EmptyRow>Your role can't set up DNS.</EmptyRow>} />}
    </ShowcaseSurface>
  );
};
