import { useNavigate } from "react-router";

import { WizardLayout } from "@/components/auth/WizardLayout";
import { ConnectCloudflareFirst } from "@/components/dns/ConnectCloudflareFirst";
import { DnsSetupStepper } from "@/components/dns/DnsSetupStepper";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";
import { useActionState } from "@/hooks/useActionState";

// DNS changes after first run (the domain itself is set up before sign-in); gates only on the Cloudflare connection.
export const DnsOnboardingPage = () => {
  const navigate = useNavigate();
  const { data, isPending, error } = useFetchConnectorStatus("cloudflare");
  const connected = data?.status.configured ?? false;
  const canWrite = useActionState({ permission: "dns:write" }).state === "live";

  return (
    <WizardLayout
      title="Set up DNS"
      subtitle="Optional. Give this instance a hostname and choose how traffic reaches it."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && !connected && <ConnectCloudflareFirst />}
      {data && connected && canWrite && <DnsSetupStepper />}
      {data && connected && !canWrite && <EmptyRow>Your role can't set up DNS.</EmptyRow>}
      <div className="mt-10 border-t border-border pt-4">
        <Button variant="link" className="h-auto px-0 text-muted-foreground" onClick={() => navigate("/")}>
          Skip for now
        </Button>
      </div>
    </WizardLayout>
  );
};
